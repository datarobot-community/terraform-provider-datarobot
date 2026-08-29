package client

import (
	"encoding/json"
	"testing"
)

// The memory space update endpoint applies only the keys present in the body. A nil
// field must therefore serialize to an explicit null, because an omitted key leaves
// the stored value untouched and the attribute would never be cleared.
func TestMemorySpaceRequestMarshalsNilFieldsAsNull(t *testing.T) {
	t.Parallel()

	description := "kept"

	payload, err := json.Marshal(&MemorySpaceRequest{Description: &description})
	if err != nil {
		t.Fatalf("marshal returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}

	for _, field := range []string{"llmModelName", "llmBaseUrl", "customInstructions"} {
		value, present := decoded[field]
		if !present {
			t.Errorf("%s is missing from the payload; it must be sent as null", field)
			continue
		}
		if value != nil {
			t.Errorf("%s = %v, want null", field, value)
		}
	}

	if decoded["description"] != description {
		t.Errorf("description = %v, want %q", decoded["description"], description)
	}
}

// An empty string is not a substitute for null: llmBaseUrl is parsed as a URL and
// rejects "" with 422.
func TestMemorySpaceRequestKeepsEmptyStringDistinctFromNull(t *testing.T) {
	t.Parallel()

	empty := ""

	payload, err := json.Marshal(&MemorySpaceRequest{LLMBaseURL: &empty})
	if err != nil {
		t.Fatalf("marshal returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}

	if decoded["llmBaseUrl"] != empty {
		t.Errorf("llmBaseUrl = %v, want %q", decoded["llmBaseUrl"], empty)
	}
}
