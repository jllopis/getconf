package getconf

import (
	"reflect"
	"testing"
)

type config struct {
	ItemInteger int    `getconf:"item-integer, info: test int setting"`
	ItemString  string `getconf:"item-string, info: test string setting"`
}

func TestSet(t *testing.T) {
	gc := New("testconf", &config{})

	if len(gc.GetAll()) > 0 {
		t.Errorf("got conf size of %d expected 0", len(gc.GetAll()))
	}
	if err := gc.Set("item-integer", "10"); err != nil {
		t.Errorf("set item-integer gives error: %s", err)
	}
	if len(gc.GetAll()) != 1 {
		t.Errorf("got conf size of %d expected 1", len(gc.GetAll()))
	}
	if err := gc.Set("item-string", "IamAString"); err != nil {
		t.Errorf("set item-string gives error: %s", err)
	}
	if err := gc.Set("item-unk", "10"); err == nil {
		t.Errorf("set item-unk does not give the error expected")
	}
	if len(gc.GetAll()) != 2 {
		t.Errorf("got conf size of %d expected 2", len(gc.GetAll()))
	}
}
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
