package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var memoryPattern = regexp.MustCompile(`^(\d+\.?\d*)\s*([a-zA-Z]*)$`)

// parseMemoryBytes converts a human-readable memory string or raw integer string to bytes.
// Supports SI units (KB, MB, GB, TB — 1000-based) and IEC units (Ki/KiB, Mi/MiB, Gi/GiB, Ti/TiB — 1024-based).
func parseMemoryBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("memory value must not be empty")
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}

	m := memoryPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid memory format %q: expected a number with optional unit (e.g. \"4GB\", \"512MB\", \"4096Mi\")", s)
	}

	value, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value in %q", s)
	}

	var multiplier float64
	switch strings.ToLower(m[2]) {
	case "", "b":
		multiplier = 1
	case "kb":
		multiplier = 1e3
	case "mb":
		multiplier = 1e6
	case "gb":
		multiplier = 1e9
	case "tb":
		multiplier = 1e12
	case "k", "ki", "kib":
		multiplier = 1 << 10
	case "m", "mi", "mib":
		multiplier = 1 << 20
	case "g", "gi", "gib":
		multiplier = 1 << 30
	case "t", "ti", "tib":
		multiplier = 1 << 40
	default:
		return 0, fmt.Errorf("unknown memory unit %q in %q: use B, KB, MB, GB, TB (1000-based) or Ki/Mi/Gi/Ti (1024-based)", m[2], s)
	}

	return int64(value * multiplier), nil
}

type memoryStringValidator struct{}

var _ validator.String = memoryStringValidator{}

func (memoryStringValidator) Description(_ context.Context) string {
	return "Validates a memory value string (e.g. \"4GB\", \"512MB\", \"4096Mi\", or raw bytes integer)."
}

func (memoryStringValidator) MarkdownDescription(_ context.Context) string {
	return "Validates a memory value string (e.g. `\"4GB\"`, `\"512MB\"`, `\"4096Mi\"`, or raw bytes integer)."
}

func (memoryStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := parseMemoryBytes(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid memory value", err.Error())
	}
}
