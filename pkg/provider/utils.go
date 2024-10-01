package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultTimeoutMinutes = 30

	faithfulnessOpenAiRuntimeParam      = "MODERATION_OOTB_RESPONSE_FAITHFULNESS_OPENAI_API_KEY"
	faithfulnessAzureOpenAiRuntimeParam = "MODERATION_OOTB_RESPONSE_FAITHFULNESS_AZURE_OPENAI_API_KEY"
)

type Knowable interface {
	IsUnknown() bool
	IsNull() bool
}

func IsKnown[T Knowable](t T) bool {
	return !t.IsUnknown() && !t.IsNull()
}

// retryGetRequests implements the retryable-http CheckRetry type.
func retryGetRequestsOnly(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if resp != nil && resp.Request != nil && resp.Request.Method != http.MethodGet {
		// We don't want to blindly retry anything that isn't a GET method
		// because it's possible that a different method type mutated data on
		// the server even if the response wasn't successful. Application code
		// should handle any retries where appropriate.
		return false, nil
	}

	return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
}

// leveledTFLogger implements the retryablehttp.LeveledLogger interface by adapting tflog methods.
type leveledTFLogger struct {
	baseCtx context.Context
}

func (l *leveledTFLogger) llArgsToTFLogArgs(keysAndValues []interface{}) map[string]interface{} {
	if argCount := len(keysAndValues); argCount%2 != 0 {
		tflog.Warn(l.baseCtx, fmt.Sprintf("unexpected number of log arguments: %d", argCount))
		return map[string]interface{}{}
	}
	additionalFields := make(map[string]interface{}, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		value, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		additionalFields[value] = keysAndValues[i+1]
	}
	return additionalFields
}

func (l *leveledTFLogger) Error(msg string, keysAndValues ...interface{}) {
	tflog.Error(l.baseCtx, msg, l.llArgsToTFLogArgs(keysAndValues))
}
func (l *leveledTFLogger) Info(msg string, keysAndValues ...interface{}) {
	tflog.Info(l.baseCtx, msg, l.llArgsToTFLogArgs(keysAndValues))
}
func (l *leveledTFLogger) Debug(msg string, keysAndValues ...interface{}) {
	tflog.Debug(l.baseCtx, msg, l.llArgsToTFLogArgs(keysAndValues))
}
func (l *leveledTFLogger) Warn(msg string, keysAndValues ...interface{}) {
	tflog.Warn(l.baseCtx, msg, l.llArgsToTFLogArgs(keysAndValues))
}

var _ retryablehttp.LeveledLogger = &leveledTFLogger{}

// traceAPICall is a helper for debugging which api calls are happening when to
// make it easier to determine for understanding what the provider framework is
// doing and for determining which calls will need to be mocked in our tests.
// Currently it relies on being manually called at each api call site which is
// unfortunate.
func traceAPICall(api string) {
	val, exists := os.LookupEnv("TRACE_API_CALLS")
	if exists && val == "1" {
		pc, _, _, _ := runtime.Caller(1)
		fmt.Printf("DataRobot API Call: %s (%s)\n", api, runtime.FuncForPC(pc).Name())
	}
}

// HookGlobal sets `*ptr = val` and returns a closure for restoring `*ptr` to
// its original value. A runtime panic will occur if `val` is not assignable to
// `*ptr`.
func HookGlobal[T any](ptr *T, val T) func() {
	orig := *ptr
	*ptr = val
	return func() { *ptr = orig }
}

func contains[T any](s []T, value T) bool {
	for _, v := range s {
		if reflect.DeepEqual(v, value) {
			return true
		}
	}
	return false
}

func getExponentialBackoff() backoff.BackOff {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second

	timeout, err := strconv.Atoi(os.Getenv(TimeoutMinutesEnvVar))
	if err != nil || timeout <= 0 {
		timeout = defaultTimeoutMinutes
	}
	expBackoff.MaxElapsedTime = time.Duration(timeout) * time.Minute

	return expBackoff
}

func waitForTaskStatusToComplete(ctx context.Context, s client.Service, id string) error {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("GetTaskStatus")
		task, err := s.GetTaskStatus(ctx, id)
		if err != nil {
			if err.Error() == "request was redirected" { // the task was completed, so the request got redirected
				return nil
			}
			return backoff.Permanent(err)
		}

		if task.Status == "ERROR" {
			return backoff.Permanent(errors.New(task.Message))
		}

		if task.Status != "COMPLETED" {
			return errors.New("task is not completed")
		}

		return nil
	}

	// Retry the operation using the backoff strategy
	return backoff.Retry(operation, expBackoff)
}

func waitForDatasetToBeReady(ctx context.Context, service client.Service, datasetId string) (*client.Dataset, error) {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("GetDataset")
		dataset, err := service.GetDataset(ctx, datasetId)
		if err != nil {
			return backoff.Permanent(err)
		}
		if dataset.ProcessingState == "ERROR" {
			return backoff.Permanent(errors.New(*dataset.Error))
		}
		if dataset.ProcessingState != "COMPLETED" {
			return errors.New("dataset is not ready")
		}

		return nil
	}

	// Retry the operation using the backoff strategy
	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return nil, err
	}

	traceAPICall("GetDataset")
	return service.GetDataset(ctx, datasetId)
}

func waitForApplicationToBeReady(ctx context.Context, service client.Service, id string) (*client.Application, error) {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		traceAPICall("IsCustomApplicationReady")
		ready, err := service.IsApplicationReady(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("application is not ready")
		}
		return nil
	}

	// Retry the operation using the backoff strategy
	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return nil, err
	}

	traceAPICall("GetCustomApplication")
	return service.GetApplication(ctx, id)
}

func checkCredentialNameAlreadyExists(err error, name string) string {
	return checkNameAlreadyExists(err, name, "Credential")
}

func checkApplicationNameAlreadyExists(err error, name string) string {
	return checkNameAlreadyExists(err, name, "Application")
}

func checkNameAlreadyExists(err error, name string, resourceType string) string {
	errMessage := err.Error()
	if strings.Contains(errMessage, "already in use") ||
		strings.Contains(errMessage, "already exist") ||
		strings.Contains(errMessage, "is already used") {
		errMessage = fmt.Sprintf("%s name must be unique, and name '%s' is already in use", resourceType, name)
	}

	return errMessage
}

func formatRuntimeParameterValues(
	ctx context.Context,
	runtimeParameterValues []client.RuntimeParameter,
	parametersInPlan basetypes.ListValue,
) (
	basetypes.ListValue,
	diag.Diagnostics,
) {
	// copy parameters in stable order
	parameters := make([]RuntimeParameterValue, 0)

	if IsKnown(parametersInPlan) {
		if diags := parametersInPlan.ElementsAs(ctx, &parameters, false); diags.HasError() {
			return basetypes.ListValue{}, diags
		}
	}

	sort.SliceStable(runtimeParameterValues, func(i, j int) bool {
		return runtimeParameterValues[i].FieldName < runtimeParameterValues[j].FieldName
	})
	for _, param := range runtimeParameterValues {
		// skip the parameter if it already exists in the plan
		found := false
		for _, p := range parameters {
			if p.Key == types.StringValue(param.FieldName) {
				found = true
				break
			}
		}
		if found {
			continue
		}

		if isManagedByGuards(param) {
			// skip the parameter if it is managed by guards
			continue
		}

		parameter := RuntimeParameterValue{
			Key:   types.StringValue(param.FieldName),
			Type:  types.StringValue(param.Type),
			Value: types.StringValue(fmt.Sprintf("%v", param.CurrentValue)),
		}

		var defaultValue any = param.DefaultValue
		if param.DefaultValue == nil {
			switch param.Type {
			case "numeric":
				defaultValue = 0.0
			case "boolean":
				defaultValue = false
			}
		}

		if param.CurrentValue != defaultValue {
			parameters = append(parameters, parameter)
		}
	}

	return listValueFromRuntimParameters(ctx, parameters)
}

func isManagedByGuards(param client.RuntimeParameter) bool {
	return param.FieldName == faithfulnessOpenAiRuntimeParam ||
		param.FieldName == faithfulnessAzureOpenAiRuntimeParam
}

func formatRuntimeParameterValue(paramType, paramValue string) (any, error) {
	switch paramType {
	case "boolean":
		return strconv.ParseBool(paramValue)
	case "numeric":
		return strconv.ParseFloat(paramValue, 64)
	default:
		return paramValue, nil
	}
}

func listValueFromRuntimParameters(ctx context.Context, runtimeParameterValues []RuntimeParameterValue) (basetypes.ListValue, diag.Diagnostics) {
	return types.ListValueFrom(
		ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":   types.StringType,
				"type":  types.StringType,
				"value": types.StringType,
			},
		}, runtimeParameterValues)
}

func prepareLocalFiles(folderPath types.String, files types.Dynamic) (localFiles []client.FileInfo, err error) {
	localFiles = make([]client.FileInfo, 0)

	if IsKnown(folderPath) {
		folder := folderPath.ValueString()
		if err = filepath.Walk(folder, func(path string, info os.FileInfo, innerErr error) error {
			if innerErr != nil {
				return innerErr
			}
			if info.IsDir() {
				return nil
			}

			pathInModel := strings.TrimPrefix(path, folder)
			pathInModel = strings.TrimPrefix(pathInModel, string(filepath.Separator))
			fileInfo, innerErr := getFileInfo(path, pathInModel)
			if innerErr != nil {
				return innerErr
			}
			localFiles = append(localFiles, fileInfo)

			return nil
		}); err != nil {
			return
		}
	}

	if IsKnown(files) && files.UnderlyingValue() != nil && IsKnown(files.UnderlyingValue()) {
		var fileTuples []FileTuple
		fileTuples, err = formatFiles(files)
		if err != nil {
			return
		}

		for _, file := range fileTuples {
			var fileInfo client.FileInfo
			fileInfo, err = getFileInfo(file.LocalPath, file.PathInModel)
			if err != nil {
				return
			}

			localFiles = append(localFiles, fileInfo)
		}
	}

	return
}

func formatFiles(files types.Dynamic) ([]FileTuple, error) {
	switch value := files.UnderlyingValue().(type) {
	case types.List:
		return handleFilesAsListOrTuple(value.Elements())
	case types.Tuple:
		return handleFilesAsListOrTuple(value.Elements())
	default:
		return nil, errors.New("files must be a list/tuple")
	}
}

func handleFilesAsListOrTuple(values []attr.Value) ([]FileTuple, error) {
	fileTuples := make([]FileTuple, 0)
	if len(values) == 0 {
		return fileTuples, nil
	}

	for i, item := range values {
		switch v := item.(type) {
		case types.List:
			var err error
			fileTuples, err = handleFileAsListOrTuple(v.Elements(), fileTuples, i)
			if err != nil {
				return nil, err
			}
		case types.Tuple:
			var err error
			fileTuples, err = handleFileAsListOrTuple(v.Elements(), fileTuples, i)
			if err != nil {
				return nil, err
			}
		case types.String:
			filePath := v.ValueString()
			fileTuples = append(fileTuples, FileTuple{
				LocalPath:   filePath,
				PathInModel: filepath.Base(filePath),
			})
		default:
			return nil, errors.New("files must be a tuple of strings or lists/tuples")
		}
	}

	return fileTuples, nil
}

func handleFileAsListOrTuple(values []attr.Value, fileTuples []FileTuple, i int) ([]FileTuple, error) {
	if len(values) < 1 || len(values) > 2 {
		return nil, fmt.Errorf("files[%d] must have 1 or 2 elements", i)
	}

	localPath, ok := values[0].(types.String)
	if !ok {
		return nil, fmt.Errorf("files[%d] has element that is not a string", i)
	}
	pathInModel := filepath.Base(localPath.ValueString())
	if len(values) == 2 {
		modelPath, ok := values[1].(types.String)
		if !ok {
			return nil, fmt.Errorf("files[%d] has element that is not a string", i)
		}
		pathInModel = modelPath.ValueString()
	}

	fileTuples = append(fileTuples, FileTuple{
		LocalPath:   localPath.ValueString(),
		PathInModel: pathInModel,
	})

	return fileTuples, nil
}

func getFileInfo(localPath, pathInModel string) (fileInfo client.FileInfo, err error) {
	var fileReader *os.File
	fileReader, err = os.Open(localPath)
	if err != nil {
		fmt.Println("Error opening file", err)
		return
	}
	defer fileReader.Close()

	var fileContent []byte
	fileContent, err = io.ReadAll(fileReader)
	if err != nil {
		return
	}

	fileInfo = client.FileInfo{
		Name:    filepath.Base(localPath),
		Path:    pathInModel,
		Content: fileContent,
	}
	return
}

func computeFolderHash(folderPath types.String) (hash types.String, err error) {
	hash = types.StringNull()
	if IsKnown(folderPath) {
		hashValue := ""
		filesInFolder := make([]string, 0)
		folder := folderPath.ValueString()
		if err = filepath.Walk(folder, func(path string, info os.FileInfo, innerErr error) error {
			if innerErr != nil {
				return innerErr
			}
			if info.IsDir() {
				return nil
			}

			filesInFolder = append(filesInFolder, path)
			return nil
		}); err != nil {
			return
		}

		// sort files to ensure consistent hash
		sort.Strings(filesInFolder)

		for _, file := range filesInFolder {
			var fileHash string
			if fileHash, err = computeFileHash(file); err != nil {
				return
			}
			hashValue += fileHash
		}

		// calculate hash of all file hashes
		sha256 := sha256.New()
		sha256.Write([]byte(hashValue))
		hashValue = hex.EncodeToString(sha256.Sum(nil))

		hash = types.StringValue(hashValue)
	}

	return
}

func computeFileHash(file string) (hash string, err error) {
	// calculate hash of file contents
	var fileReader *os.File
	if fileReader, err = os.Open(file); err != nil {
		return
	}
	defer fileReader.Close()

	sha256 := sha256.New()
	if _, err = io.Copy(sha256, fileReader); err != nil {
		return
	}
	hash = hex.EncodeToString(sha256.Sum(nil))

	return
}

func computeFilesHashes(ctx context.Context, files types.Dynamic) (hashes types.List, err error) {
	hashValues := make([]string, 0)
	localFiles, err := prepareLocalFiles(types.StringUnknown(), files)
	if err != nil {
		return
	}

	for _, file := range localFiles {
		sha256 := sha256.New()
		sha256.Write(file.Content)
		hash := hex.EncodeToString(sha256.Sum(nil))
		hashValues = append(hashValues, hash)
	}

	// convert hashValues to types.List
	hashes, diags := types.ListValueFrom(ctx, types.StringType, hashValues)
	if diags.HasError() {
		err = errors.New(diags.Errors()[0].Detail())
		return
	}

	return
}

func Int64ValuePointerOptional(value basetypes.Int64Value) *int64 {
	if value.IsUnknown() {
		return nil
	}

	return value.ValueInt64Pointer()
}

func Float64ValuePointerOptional(value basetypes.Float64Value) *float64 {
	if value.IsUnknown() {
		return nil
	}

	return value.ValueFloat64Pointer()
}

func StringValuePointerOptional(value basetypes.StringValue) *string {
	if value.IsUnknown() {
		return nil
	}

	return value.ValueStringPointer()
}

func ConvertTfStringListToPtr(input []types.String) *[]string {
	output := make([]string, len(input))
	for i, value := range input {
		output[i] = value.ValueString()
	}

	return &output
}

func UpdateUseCasesForDataset(
	ctx context.Context,
	service client.Service,
	datasetID string,
	stateUseCaseIDs []types.String,
	planUseCaseIDs []types.String,
) (err error) {
	if !reflect.DeepEqual(stateUseCaseIDs, planUseCaseIDs) {
		useCasesToAdd := make([]string, 0)
		for _, useCaseID := range planUseCaseIDs {
			found := false
			for _, oldUseCaseID := range stateUseCaseIDs {
				if useCaseID.ValueString() == oldUseCaseID.ValueString() {
					break
				}
			}
			if !found {
				useCasesToAdd = append(useCasesToAdd, useCaseID.ValueString())
			}
		}

		for _, useCaseID := range useCasesToAdd {
			traceAPICall("AddDatasetToUseCase")
			err = service.AddDatasetToUseCase(ctx, useCaseID, datasetID)
			if err != nil {
				return
			}
		}

		useCasesToRemove := make([]string, 0)
		for _, oldUseCaseID := range stateUseCaseIDs {
			found := false
			for _, useCaseID := range planUseCaseIDs {
				if useCaseID.ValueString() == oldUseCaseID.ValueString() {
					break
				}
			}
			if !found {
				useCasesToRemove = append(useCasesToRemove, oldUseCaseID.ValueString())
			}
		}

		for _, useCaseID := range useCasesToRemove {
			traceAPICall("RemoveDatasetFromUseCase")
			err = service.RemoveDatasetFromUseCase(ctx, useCaseID, datasetID)
			if err != nil {
				return
			}
		}
	}

	return
}
