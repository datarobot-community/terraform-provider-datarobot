package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultTimeoutMinutes = 30
)

func addConfigureProviderErr(diagnostics *diag.Diagnostics) {
	diagnostics.AddError(
		"Provider not configured",
		"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
	)
}

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

func waitForDatasetToBeReady(ctx context.Context, service client.Service, datasetId string) (*client.DatasetResponse, error) {
	expBackoff := getExponentialBackoff()

	operation := func() error {
		ready, err := service.IsDatasetReady(ctx, datasetId)
		if err != nil {
			return backoff.Permanent(err)
		}
		if !ready {
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

func waitForApplicationToBeReady(ctx context.Context, service client.Service, id string) (*client.ApplicationResponse, error) {
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
