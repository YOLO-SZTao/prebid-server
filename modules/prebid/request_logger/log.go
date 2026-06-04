package request_logger

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/prebid/prebid-server/v4/logger"
)

const componentName = "pbs-hook-request-logger"

func logStage(stage string, fields map[string]any) {
	fields["component"] = componentName
	fields["stage"] = stage

	embedJSONStrings(reflect.ValueOf(fields))

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(fields); err != nil {
		logger.Infof(`{"component":"%s","stage":"%s","error":"marshal_failed"}`, componentName, stage)
		return
	}
	logger.Infof("%s", bytes.TrimSuffix(buf.Bytes(), []byte("\n")))
}

// embedJSONStrings recursively walks any map/slice tree via reflection and
// replaces string values that contain valid JSON with json.RawMessage so they
// are embedded directly rather than double-escaped.
func embedJSONStrings(rv reflect.Value) {
	switch rv.Kind() {
	case reflect.Interface, reflect.Ptr:
		if !rv.IsNil() {
			embedJSONStrings(rv.Elem())
		}
	case reflect.Map:
		for _, key := range rv.MapKeys() {
			val := rv.MapIndex(key)
			if val.Kind() == reflect.Interface {
				val = val.Elem()
			}
			if val.Kind() == reflect.String {
				s := val.String()
				if len(s) > 0 && (s[0] == '{' || s[0] == '[') && json.Valid([]byte(s)) {
					rv.SetMapIndex(key, reflect.ValueOf(json.RawMessage(s)))
				}
			} else if val.Kind() == reflect.Map || val.Kind() == reflect.Slice {
				embedJSONStrings(val)
			}
		}
	case reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			val := rv.Index(i)
			if val.Kind() == reflect.Interface {
				val = val.Elem()
			}
			if val.Kind() == reflect.Map || val.Kind() == reflect.Slice {
				embedJSONStrings(val)
			}
		}
	}
}
