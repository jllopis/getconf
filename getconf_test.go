package getconf

import (
	"reflect"
	"testing"
)

func TestGetTypeValue(t *testing.T) {

	var result interface{}

	result = getTypedValue("9", reflect.Int)
	if reflect.ValueOf(result).Kind() != reflect.Int {
		t.Errorf("got: %T expected: int", result)
	}
	result = getTypedValue("9", reflect.Int8)
	if reflect.ValueOf(result).Kind() != reflect.Int8 {
		t.Errorf("got: %T expected: int8", result)
	}
	result = getTypedValue("9", reflect.Int16)
	if reflect.ValueOf(result).Kind() != reflect.Int16 {
		t.Errorf("got: %T expected: int16", result)
	}
	result = getTypedValue("9", reflect.Int32)
	if reflect.ValueOf(result).Kind() != reflect.Int32 {
		t.Errorf("got: %T expected: int32", result)
	}
	result = getTypedValue("9", reflect.Int64)
	if reflect.ValueOf(result).Kind() != reflect.Int64 {
		t.Errorf("got: %T expected: int64", result)
	}
	result = getTypedValue("9", reflect.Int16)
	if reflect.ValueOf(result).Kind() != reflect.Int16 {
		t.Errorf("got: %T expected: int16", result)
	}
	result = getTypedValue("false", reflect.Bool)
	if reflect.ValueOf(result).Kind() != reflect.Bool {
		t.Errorf("got: %T expected: bool", result)
	}
	result = getTypedValue("9.42", reflect.Float32)
	if reflect.ValueOf(result).Kind() != reflect.Float32 {
		t.Errorf("got: %T expected: float32", result)
	}
	result = getTypedValue("9.42", reflect.Float64)
	if reflect.ValueOf(result).Kind() != reflect.Float64 {
		t.Errorf("got: %T expected: float64", result)
	}
	result = getTypedValue("9", reflect.String)
	if reflect.ValueOf(result).Kind() != reflect.String {
		t.Errorf("got: %T expected: string", result)
	}
}
