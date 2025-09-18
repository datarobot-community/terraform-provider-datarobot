package provider

import (
	"encoding/json"
	"strconv"
)

// NumberFromInt64 converts int64 to json.Number
func NumberFromInt64(val int64) json.Number {
	return json.Number(strconv.FormatInt(val, 10))
}

// NumberPtrFromInt64 converts int64 to *json.Number
func NumberPtrFromInt64(val int64) *json.Number {
	n := NumberFromInt64(val)
	return &n
}

// NumberPtrFromInt64Ptr converts *int64 to *json.Number
func NumberPtrFromInt64Ptr(val *int64) *json.Number {
	if val == nil {
		return nil
	}
	return NumberPtrFromInt64(*val)
}

// Int64FromNumberPtr safely converts *json.Number to int64, returns 0 if nil
func Int64FromNumberPtr(n *json.Number) int64 {
	if n == nil {
		return 0
	}
	val, err := n.Int64()
	if err != nil {
		return 0
	}
	return val
}

// Int64FromNumber safely converts json.Number to int64
func Int64FromNumber(n json.Number) int64 {
	val, err := n.Int64()
	if err != nil {
		return 0
	}
	return val
}