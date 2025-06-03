package provider

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	nemoAzureOpenAiRuntimeParam         = "MODERATION_NEMO_GUARDRAILS_PROMPT_AZURE_OPENAI_API_KEY"
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

func setStringValueIfKnown(target *string, source basetypes.StringValue) {
	if IsKnown(source) {
		*target = source.ValueString()
	}
}

func getExponentialBackoff() backoff.BackOff {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 10 * time.Second

	timeout, err := strconv.Atoi(os.Getenv(TimeoutMinutesEnvVar))
	if err != nil || timeout <= 0 {
		timeout = defaultTimeoutMinutes
	}
	expBackoff.MaxElapsedTime = time.Duration(timeout) * time.Minute

	return expBackoff
}

func waitForGenAITaskStatusToComplete(ctx context.Context, s client.Service, id string) error {
	return waitForTaskStatusToCompleteGeneric(ctx, s, id, true)
}

func waitForTaskStatusToComplete(ctx context.Context, s client.Service, id string) error {
	return waitForTaskStatusToCompleteGeneric(ctx, s, id, false)
}

func waitForTaskStatusToCompleteGeneric(ctx context.Context, s client.Service, id string, isGenAI bool) error {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		var task *client.TaskStatusResponse
		var err error
		if isGenAI {
			traceAPICall("GetGenAITaskStatus")
			task, err = s.GetGenAITaskStatus(ctx, id)
		} else {
			traceAPICall("GetTaskStatus")
			task, err = s.GetTaskStatus(ctx, id)
		}

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
		traceAPICall("GetCustomApplication")
		customApplication, err := service.GetApplication(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}
		if customApplication.Status == "failed" {
			return backoff.Permanent(errors.New("application failed to create, review the logs for more details"))
		}

		if customApplication.Status != "running" {
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

func waitForRegisteredModelVersionToBeReady(ctx context.Context, service client.Service, registeredModelId string, versionId string) error {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		ready, err := service.IsRegisteredModelVersionReady(ctx, registeredModelId, versionId)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
			return errors.New("registered model version is not ready")
		}
		return nil
	}

	// Retry the operation using the backoff strategy
	err := backoff.Retry(operation, expBackoff)
	if err != nil {
		return err
	}
	return nil
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
	return formatRuntimeParameterValuesInternal(ctx, runtimeParameterValues, parametersInPlan, false)
}

func formatRuntimeParameterValuesForRetrainingJob(
	ctx context.Context,
	runtimeParameterValues []client.RuntimeParameter,
	parametersInPlan basetypes.ListValue,
) (
	basetypes.ListValue,
	diag.Diagnostics,
) {
	return formatRuntimeParameterValuesInternal(ctx, runtimeParameterValues, parametersInPlan, true)
}

func formatRuntimeParameterValuesInternal(
	ctx context.Context,
	runtimeParameterValues []client.RuntimeParameter,
	parametersInPlan basetypes.ListValue,
	isRetrainingJob bool,
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

		if isRetrainingJob && isManagedByRetrainingPolicy(param) {
			// skip the parameter if it is managed by retraining policy
			continue
		}

		parameter := RuntimeParameterValue{
			Key:   types.StringValue(param.FieldName),
			Type:  types.StringValue(param.Type),
			Value: types.StringValue(fmt.Sprintf("%v", param.CurrentValue)),
		}

		var defaultValue = param.DefaultValue
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
		param.FieldName == faithfulnessAzureOpenAiRuntimeParam ||
		param.FieldName == nemoAzureOpenAiRuntimeParam
}

func isManagedByRetrainingPolicy(param client.RuntimeParameter) bool {
	return param.FieldName == deploymentParamName ||
		param.FieldName == retrainingPolicyIDParamName
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

func convertRuntimeParameterValues(
	ctx context.Context,
	tfRuntimeParameterValues basetypes.ListValue,
) (
	jsonParamsStr string,
	err error,
) {
	params, err := convertRuntimeParameterValuesToList(ctx, tfRuntimeParameterValues)
	if err != nil {
		return
	}

	jsonParams, err := json.Marshal(params)
	if err != nil {
		return
	}
	jsonParamsStr = string(jsonParams)

	return
}

func convertRuntimeParameterValuesToList(
	ctx context.Context,
	tfRuntimeParameterValues basetypes.ListValue,
) (
	params []client.RuntimeParameterValueRequest,
	err error,
) {
	runtimeParameterValues := make([]RuntimeParameterValue, 0)
	if diags := tfRuntimeParameterValues.ElementsAs(ctx, &runtimeParameterValues, false); diags.HasError() {
		err = errors.New("Error converting runtime parameter values")
		return
	}

	params = make([]client.RuntimeParameterValueRequest, len(runtimeParameterValues))
	for i, param := range runtimeParameterValues {
		var value any
		value, err = formatRuntimeParameterValue(param.Type.ValueString(), param.Value.ValueString())
		if err != nil {
			return
		}
		params[i] = client.RuntimeParameterValueRequest{
			FieldName: param.Key.ValueString(),
			Type:      param.Type.ValueString(),
			Value:     &value,
		}
	}

	return
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

func prepareLocalFiles(folderPath types.String, files []FileTuple) (localFiles []client.FileInfo, err error) {
	localFiles = make([]client.FileInfo, 0)

	if IsKnown(folderPath) {
		folder := folderPath.ValueString()
		if err = WalkSymlinkSafe(folder, func(path string, info os.FileInfo, innerErr error) error {
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

	if len(files) > 0 {
		for _, file := range files {
			var fileInfo client.FileInfo
			fileInfo, err = getFileInfo(file.Source.ValueString(), file.Destination.ValueString())
			if err != nil {
				return
			}

			localFiles = append(localFiles, fileInfo)
		}
	}

	return
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
		if err = WalkSymlinkSafe(folder, func(path string, info os.FileInfo, innerErr error) error {
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
		hash = types.StringValue(computeHash([]byte(hashValue)))
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

func computeFilesHashes(ctx context.Context, files []FileTuple) (hashes types.List, err error) {
	hashValues := make([]string, 0)

	for _, file := range files {
		var hash string
		hash, err = computeFileHash(file.Source.ValueString())
		if err != nil {
			return
		}
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

func computeHash(value []byte) (hash string) {
	sha256 := sha256.New()
	sha256.Write(value)
	hash = hex.EncodeToString(sha256.Sum(nil))
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

func BoolValuePointerOptional(value basetypes.BoolValue) *bool {
	if value.IsUnknown() {
		return nil
	}

	return value.ValueBoolPointer()
}

func convertTfStringMap(tfMap types.Map) map[string]string {
	convertedMap := make(map[string]string)
	for k, v := range tfMap.Elements() {
		if strVal, ok := v.(types.String); ok {
			convertedMap[k] = strVal.ValueString()
		}
	}
	return convertedMap
}

func convertTfStringList(input []types.String) []string {
	output := make([]string, len(input))
	for i, value := range input {
		output[i] = value.ValueString()
	}

	return output
}

func convertToTfStringList(input []string) []types.String {
	output := make([]types.String, len(input))
	for i, value := range input {
		output[i] = types.StringValue(value)
	}

	return output
}

func convertTfStringListToPtr(input []types.String) *[]string {
	output := make([]string, len(input))
	for i, value := range input {
		output[i] = value.ValueString()
	}

	return &output
}

func convertDynamicType(tfType types.Dynamic) any {
	switch t := tfType.UnderlyingValue().(type) {
	case types.String:
		return t.ValueString()
	case types.Int64:
		return t.ValueInt64()
	default:
		return nil
	}
}

func addEntityToUseCase(
	ctx context.Context,
	service client.Service,
	useCaseID,
	entityType,
	entityID string,
) (err error) {
	if err = service.AddEntityToUseCase(ctx, useCaseID, entityType, entityID); err != nil {
		if strings.Contains(err.Error(), "already linked to this Use Case") {
			err = nil
		}
	}

	return
}

func updateUseCasesForEntity(
	ctx context.Context,
	service client.Service,
	entityType string,
	entityID string,
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
			traceAPICall(fmt.Sprintf("Add%sToUseCase", strings.ToUpper(entityType)))
			if err = addEntityToUseCase(ctx, service, useCaseID, entityType, entityID); err != nil {
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
			traceAPICall(fmt.Sprintf("Remove%sFromUseCase", strings.ToUpper(entityType)))
			if err = service.RemoveEntityFromUseCase(ctx, useCaseID, entityType, entityID); err != nil {
				if _, ok := err.(*client.NotFoundError); ok {
					err = nil
					continue
				}
				return
			}
		}
	}

	return
}

func zipDirectory(source, target string) (content []byte, err error) {
	zipFile, err := os.Create(target)
	if err != nil {
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = WalkSymlinkSafe(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	if content, err = os.ReadFile(target); err != nil {
		return
	}
	defer os.Remove(target)

	return
}

func convertSchedule(schedule Schedule) (clientSchedule client.Schedule, err error) {
	clientSchedule = client.Schedule{}
	minute, err := convertScheduleExpression(schedule.Minute)
	if err != nil {
		return
	}
	clientSchedule.Minute = minute

	hour, err := convertScheduleExpression(schedule.Hour)
	if err != nil {
		return
	}
	clientSchedule.Hour = hour

	dayOfMonth, err := convertScheduleExpression(schedule.DayOfMonth)
	if err != nil {
		return
	}
	clientSchedule.DayOfMonth = dayOfMonth

	month, err := convertScheduleExpression(schedule.Month)
	if err != nil {
		return
	}
	clientSchedule.Month = month

	dayOfWeek, err := convertScheduleExpression(schedule.DayOfWeek)
	if err != nil {
		return
	}
	clientSchedule.DayOfWeek = dayOfWeek

	return
}

func convertScheduleExpression(expression []types.String) (any, error) {
	if len(expression) == 0 {
		return nil, nil
	}

	if expression[0].ValueString() == "*" {
		return []string{"*"}, nil
	}

	convertedExpression := make([]int, 0, len(expression))
	for _, i := range expression {
		converted, err := strconv.Atoi(i.ValueString())
		if err != nil {
			return nil, err
		}
		convertedExpression = append(convertedExpression, converted)
	}

	return convertedExpression, nil
}
func convertScheduleFromAPI(clientSchedule client.Schedule) (schedule Schedule, err error) {
	schedule = Schedule{}

	convertScheduleExpressionFromAPI := func(expression any) ([]types.String, error) {
		if expression == nil {
			return nil, nil
		}

		switch v := expression.(type) {
		case []string:
			convertedExpression := make([]types.String, len(v))
			for i, val := range v {
				convertedExpression[i] = types.StringValue(val)
			}
			return convertedExpression, nil
		case []int:
			convertedExpression := make([]types.String, len(v))
			for i, val := range v {
				convertedExpression[i] = types.StringValue(strconv.Itoa(val))
			}
			return convertedExpression, nil
		case []interface{}:
			convertedExpression := make([]types.String, len(v))
			for i, val := range v {
				switch val := val.(type) {
				case string:
					convertedExpression[i] = types.StringValue(val)
				case int:
					convertedExpression[i] = types.StringValue(strconv.Itoa(val))
				case float64:
					convertedExpression[i] = types.StringValue(fmt.Sprintf("%.0f", val))
				default:
					return nil, fmt.Errorf("unsupported schedule expression type: %T", val)
				}
			}
			return convertedExpression, nil
		default:
			return nil, fmt.Errorf("unsupported schedule expression type: %T", expression)
		}
	}

	// Convert each schedule field
	minute, err := convertScheduleExpressionFromAPI(clientSchedule.Minute)
	if err != nil {
		return
	}
	schedule.Minute = minute

	hour, err := convertScheduleExpressionFromAPI(clientSchedule.Hour)
	if err != nil {
		return
	}
	schedule.Hour = hour

	dayOfMonth, err := convertScheduleExpressionFromAPI(clientSchedule.DayOfMonth)
	if err != nil {
		return
	}
	schedule.DayOfMonth = dayOfMonth

	month, err := convertScheduleExpressionFromAPI(clientSchedule.Month)
	if err != nil {
		return
	}
	schedule.Month = month

	dayOfWeek, err := convertScheduleExpressionFromAPI(clientSchedule.DayOfWeek)
	if err != nil {
		return
	}
	schedule.DayOfWeek = dayOfWeek

	return
}

func prepareTestFolder(folderPath string) (string, error) {
	// Check if the directory exists
	if _, err := os.Stat(folderPath); err == nil {
		// Remove the directory if it exists
		if err := os.RemoveAll(folderPath); err != nil {
			return "", err
		}
	}

	// Create the directory
	if err := os.Mkdir(folderPath, 0755); err != nil {
		return "", err
	}

	// Return the path to the created directory
	return filepath.Abs(folderPath)
}
