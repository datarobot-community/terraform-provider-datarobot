package common

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "fmt"
    "io"
    "os"
    "reflect"
    "runtime"
    "strconv"
    "strings"
    "time"

    "github.com/cenkalti/backoff/v4"
    "github.com/datarobot-community/terraform-provider-datarobot/internal/client"
)

// Environment / defaults (duplicated minimally to avoid provider package import cycles during incremental refactor)
const (
    timeoutMinutesEnvVar   = "DATAROBOT_TIMEOUT_MINUTES"
    defaultTimeoutMinutes  = 30
    traceAPICallsEnvVar    = "TRACE_API_CALLS"
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
func getExponentialBackoff() backoff.BackOff {
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
    expBackoff := getExponentialBackoff()

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

// UpdateUseCasesForEntity performs diff add/remove logic for use case links.
func UpdateUseCasesForEntity(
    ctx context.Context,
    service client.Service,
    entityType string,
    entityID string,
    stateUseCaseIDs []string,
    planUseCaseIDs []string,
) error {
    // additions
    for _, id := range planUseCaseIDs {
        if !contains(stateUseCaseIDs, id) {
            TraceAPICall(fmt.Sprintf("Add%sToUseCase", strings.ToUpper(entityType)))
            if err := AddEntityToUseCase(ctx, service, id, entityType, entityID); err != nil {
                return err
            }
        }
    }
    // removals
    for _, id := range stateUseCaseIDs {
        if !contains(planUseCaseIDs, id) {
            TraceAPICall(fmt.Sprintf("Remove%sFromUseCase", strings.ToUpper(entityType)))
            if err := service.RemoveEntityFromUseCase(ctx, id, entityType, entityID); err != nil {
                if _, ok := err.(*client.NotFoundError); ok { // already gone
                    continue
                }
                return err
            }
        }
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
