package client

import (
	"errors"
	"fmt"
	"testing"
)

func TestNotFoundErrorIs(t *testing.T) {
	err := NewNotFoundError("https://staging.datarobot.com/api/v2/customModels/abc123/")

	if !errors.Is(err, &NotFoundError{}) {
		t.Error("errors.Is should return true for *NotFoundError")
	}
	wrapped := fmt.Errorf("wrapped: %w", err)
	if !errors.Is(wrapped, &NotFoundError{}) {
		t.Error("errors.Is should return true for wrapped *NotFoundError")
	}
	if errors.Is(err, &UnauthorizedError{}) {
		t.Error("errors.Is should return false for wrong type")
	}
}

func TestUnauthorizedErrorIs(t *testing.T) {
	err := NewUnauthorizedError("some resource")
	if !errors.Is(err, &UnauthorizedError{}) {
		t.Error("errors.Is should return true for *UnauthorizedError")
	}
	if errors.Is(err, &NotFoundError{}) {
		t.Error("errors.Is should return false for wrong type")
	}
}
