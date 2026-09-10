package wapi

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
)

// The CLI writes the state directory too, and ships more often than the
// provider. Save re-marshals a whole struct, so without help a key the CLI
// added to config.json or manifest.json would be dropped by the next
// provider save. Load keeps every key the struct has no field for in Extra,
// and Save writes it back after the known fields, unchanged. No CLI
// counterpart: the CLI owns the format and always has every field.

// unknownKeys returns the members of the JSON object in data that the
// struct value known has no field for, or nil when there are none.
func unknownKeys(data []byte, known any) (map[string]json.RawMessage, error) {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}

	for _, key := range jsonFieldNames(reflect.TypeOf(known)) {
		delete(all, key)
	}

	if len(all) == 0 {
		return nil, nil
	}

	return all, nil
}

// jsonFieldNames lists the keys the exported fields of struct type t marshal
// to, honoring the json tag the way encoding/json does.
func jsonFieldNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		if tag, ok := field.Tag.Lookup("json"); ok {
			tagName, _, _ := strings.Cut(tag, ",")
			switch tagName {
			case "-":
				continue
			case "":
			default:
				name = tagName
			}
		}

		names = append(names, name)
	}

	return names
}

// withUnknownKeys appends extra to the JSON object in data, in key order.
// data is the compact json.Marshal output for a struct with at least one
// field, so the object always has a member to put a comma after.
func withUnknownKeys(data []byte, extra map[string]json.RawMessage) ([]byte, error) {
	if len(extra) == 0 {
		return data, nil
	}

	keys := make([]string, 0, len(extra))
	for key := range extra {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.Write(bytes.TrimSuffix(data, []byte("}")))
	for _, key := range keys {
		name, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.WriteByte(',')
		buf.Write(name)
		buf.WriteByte(':')
		buf.Write(extra[key])
	}
	buf.WriteByte('}')

	return buf.Bytes(), nil
}
