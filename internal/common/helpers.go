package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Environment / defaults (duplicated minimally to avoid provider package import cycles during incremental refactor)
const (
	timeoutMinutesEnvVar  = "DATAROBOT_TIMEOUT_MINUTES"
	defaultTimeoutMinutes = 30
	traceAPICallsEnvVar   = "TRACE_API_CALLS"

	retrainingJobType   = "retraining"
	defaultJobType      = "default"
	notificationJobType = "notification"

	deploymentParamName         = "DEPLOYMENT"
	retrainingPolicyIDParamName = "RETRAINING_POLICY_ID"

	faithfulnessOpenAiRuntimeParam      = "MODERATION_OOTB_RESPONSE_FAITHFULNESS_OPENAI_API_KEY"
	faithfulnessAzureOpenAiRuntimeParam = "MODERATION_OOTB_RESPONSE_FAITHFULNESS_AZURE_OPENAI_API_KEY"
	nemoAzureOpenAiRuntimeParam         = "MODERATION_NEMO_GUARDRAILS_PROMPT_AZURE_OPENAI_API_KEY"
)

// ServiceAccessor is implemented by the root provider to expose the API service.
type ServiceAccessor interface {
	GetService() client.Service
}

// Knowable mirrors the Terraform framework value interface subset we need.
type Knowable interface {
	IsUnknown() bool
	IsNull() bool
}

func IsKnown[T Knowable](t T) bool {
	return !t.IsUnknown() && !t.IsNull()
}

// TraceAPICall prints a simple trace line when TRACE_API_CALLS=1.
func TraceAPICall(api string) {
	if val, ok := os.LookupEnv(traceAPICallsEnvVar); ok && val == "1" {
		if pc, _, _, ok2 := runtime.Caller(1); ok2 {
			fmt.Printf("DataRobot API Call: %s (%s)\n", api, runtime.FuncForPC(pc).Name())
		} else {
			fmt.Printf("DataRobot API Call: %s\n", api)
		}
	}
}

// internal backoff helper
func GetExponentialBackoff() backoff.BackOff {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 10 * time.Second

	timeout, err := strconv.Atoi(os.Getenv(timeoutMinutesEnvVar))
	if err != nil || timeout <= 0 {
		timeout = defaultTimeoutMinutes
	}
	expBackoff.MaxElapsedTime = time.Duration(timeout) * time.Minute
	return expBackoff
}

// WaitForDatasetToBeReady polls the dataset until completed or error state.
func WaitForDatasetToBeReady(ctx context.Context, service client.Service, datasetID string) (*client.Dataset, error) {
	expBackoff := GetExponentialBackoff()

	operation := func() error {
		TraceAPICall("GetDataset")
		dataset, err := service.GetDataset(ctx, datasetID)
		if err != nil {
			return backoff.Permanent(err)
		}
		if dataset.ProcessingState == "ERROR" {
			if dataset.Error != nil {
				return backoff.Permanent(errors.New(*dataset.Error))
			}
			return backoff.Permanent(errors.New("dataset failed"))
		}
		if dataset.ProcessingState != "COMPLETED" {
			return errors.New("dataset is not ready")
		}
		return nil
	}
	if err := backoff.Retry(operation, expBackoff); err != nil {
		return nil, err
	}
	TraceAPICall("GetDataset")
	return service.GetDataset(ctx, datasetID)
}

// AddEntityToUseCase links an entity to a use case ignoring already-linked errors.
func AddEntityToUseCase(ctx context.Context, service client.Service, useCaseID, entityType, entityID string) error {
	if err := service.AddEntityToUseCase(ctx, useCaseID, entityType, entityID); err != nil {
		if strings.Contains(err.Error(), "already linked to this Use Case") {
			return nil
		}
		return err
	}
	return nil
}

// ComputeFileHash returns the sha256 hash of a file's contents.
func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func contains[T comparable](arr []T, v T) bool {
	for _, x := range arr {
		if reflect.DeepEqual(x, v) { // allows slice element comparability fallback
			return true
		}
	}
	return false
}

func CheckCredentialNameAlreadyExists(err error, name string) string {
	return checkNameAlreadyExists(err, name, "Credential")
}

func CheckApplicationNameAlreadyExists(err error, name string) string {
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

func Int64ValuePointerOptional(value basetypes.Int64Value) *int64 {
	if value.IsUnknown() || value.IsNull() {
		return nil
	}

	return value.ValueInt64Pointer()
}

func BoolValuePointerOptional(value basetypes.BoolValue) *bool {
	if value.IsUnknown() || value.IsNull() {
		return nil
	}

	return value.ValueBoolPointer()
}

func StringValuePointerOptional(value basetypes.StringValue) *string {
	if value.IsUnknown() || value.IsNull() {
		return nil
	}

	return value.ValueStringPointer()
}

func FormatRuntimeParameterValue(paramType, paramValue string) (any, error) {
	switch paramType {
	case "boolean":
		return strconv.ParseBool(paramValue)
	case "numeric":
		return strconv.ParseFloat(paramValue, 64)
	default:
		return paramValue, nil
	}
}

func FormatRuntimeParameterValues(
	ctx context.Context,
	runtimeParameterValues []client.RuntimeParameter,
	parametersInPlan basetypes.ListValue,
) (
	basetypes.ListValue,
	diag.Diagnostics,
) {
	return formatRuntimeParameterValuesInternal(ctx, runtimeParameterValues, parametersInPlan, false)
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
	parameters := make([]models.RuntimeParameterValue, 0)

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

		parameter := models.RuntimeParameterValue{
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
func listValueFromRuntimParameters(ctx context.Context, runtimeParameterValues []models.RuntimeParameterValue) (basetypes.ListValue, diag.Diagnostics) {
	return types.ListValueFrom(
		ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":   types.StringType,
				"type":  types.StringType,
				"value": types.StringType,
			},
		}, runtimeParameterValues)
}

// Helper functions for converting pointer values back to Terraform types.
func Int64PointerValue(ptr *int64) types.Int64 {
	if ptr == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*ptr)
}

func BoolPointerValue(ptr *bool) types.Bool {
	if ptr == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*ptr)
}

func StringPointerValue(ptr *string) types.String {
	if ptr == nil {
		return types.StringNull()
	}
	return types.StringValue(*ptr)
}


func ComputeFilesHashes(ctx context.Context, files types.Dynamic) (hashes types.List, err error) {
	hashValues := make([]string, 0)
	localFiles, err := PrepareLocalFiles(types.StringUnknown(), files)
	if err != nil {
		return
	}

	for _, file := range localFiles {
		hashValues = append(hashValues, computeHash(file.Content))
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


func PrepareLocalFiles(folderPath types.String, files types.Dynamic) (localFiles []client.FileInfo, err error) {
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

	if IsKnown(files) && files.UnderlyingValue() != nil && IsKnown(files.UnderlyingValue()) {
		var fileTuples []models.FileTuple
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


func formatFiles(files types.Dynamic) ([]models.FileTuple, error) {
	switch value := files.UnderlyingValue().(type) {
	case types.List:
		return handleFilesAsListOrTuple(value.Elements())
	case types.Tuple:
		return handleFilesAsListOrTuple(value.Elements())
	default:
		return nil, errors.New("files must be a list/tuple")
	}
}

func handleFilesAsListOrTuple(values []attr.Value) ([]models.FileTuple, error) {
	fileTuples := make([]models.FileTuple, 0)
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
			fileTuples = append(fileTuples, models.FileTuple{
				LocalPath:   filePath,
				PathInModel: filepath.Base(filePath),
			})
		default:
			return nil, errors.New("files must be a tuple of strings or lists/tuples")
		}
	}

	return fileTuples, nil
}

func handleFileAsListOrTuple(values []attr.Value, fileTuples []models.FileTuple, i int) ([]models.FileTuple, error) {
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

	fileTuples = append(fileTuples, models.FileTuple{
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


func ComputeFolderHash(folderPath types.String) (hash types.String, err error) {
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

func ConvertToTfStringList(input []string) []types.String {
	output := make([]types.String, len(input))
	for i, value := range input {
		output[i] = types.StringValue(value)
	}

	return output
}

func ConvertDynamicType(tfType types.Dynamic) any {
	switch t := tfType.UnderlyingValue().(type) {
	case types.String:
		return t.ValueString()
	case types.Int64:
		return t.ValueInt64()
	default:
		return nil
	}
}

func ConvertTfStringList(input []types.String) []string {
	output := make([]string, len(input))
	for i, value := range input {
		output[i] = value.ValueString()
	}

	return output
}


func ConvertTfStringMap(tfMap types.Map) map[string]string {
	convertedMap := make(map[string]string)
	for k, v := range tfMap.Elements() {
		if strVal, ok := v.(types.String); ok {
			convertedMap[k] = strVal.ValueString()
		}
	}
	return convertedMap
}


func Float64ValuePointerOptional(value basetypes.Float64Value) *float64 {
	if value.IsUnknown() || value.IsNull() {
		return nil
	}

	return value.ValueFloat64Pointer()
}


func ConvertSchedule(schedule models.Schedule) (clientSchedule client.Schedule, err error) {
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



func WaitForApplicationToBeReady(ctx context.Context, service client.Service, id string) (*client.Application, error) {
	expBackoff := GetExponentialBackoff()

	operation := func() error {
		TraceAPICall("GetCustomApplication")
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

	TraceAPICall("GetCustomApplication")
	return service.GetApplication(ctx, id)
}

func UpdateUseCasesForEntity(
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
			TraceAPICall(fmt.Sprintf("Add%sToUseCase", strings.ToUpper(entityType)))
			if err = AddEntityToUseCase(ctx, service, useCaseID, entityType, entityID); err != nil {
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
			TraceAPICall(fmt.Sprintf("Remove%sFromUseCase", strings.ToUpper(entityType)))
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
