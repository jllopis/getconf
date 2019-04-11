package getconf

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/cast"
)

func GetAll() map[string]interface{} { return g2.GetAll() }
func (gc *GetConf) GetAll() map[string]interface{} {
	opts := make(map[string]interface{})
	for _, x := range g2.options {
		if x.value == nil {
			continue
		}
		opts[x.name] = x.value
	}
	return opts
}

// Get return the value associated to the key
func Get(key string) interface{} { return g2.Get(key) }
func (gc *GetConf) Get(key string) interface{} {
	if o, ok := gc.options[key]; ok != false {
		return o.value
	}
	return nil
}

// GetString returns the value associated with the key as a string.
func GetString(key string) string { return g2.GetString(key) }
func (gc *GetConf) GetString(key string) string {
	return cast.ToString(gc.Get(key))
}

func GetInt(key string) int { return g2.GetInt(key) }
func (gc *GetConf) GetInt(key string) int {
	return cast.ToInt(gc.Get(key))
}

func GetInt8(key string) int8 { return g2.GetInt8(key) }
func (gc *GetConf) GetInt8(key string) int8 {
	return cast.ToInt8(gc.Get(key))
}

func GetInt16(key string) int16 { return g2.GetInt16(key) }
func (gc *GetConf) GetInt16(key string) int16 {
	return cast.ToInt16(gc.Get(key))
}

func GetInt32(key string) int32 { return g2.GetInt32(key) }
func (gc *GetConf) GetInt32(key string) int32 {
	return cast.ToInt32(gc.Get(key))
}

func GetInt64(key string) int64 { return g2.GetInt64(key) }
func (gc *GetConf) GetInt64(key string) int64 {
	return cast.ToInt64(gc.Get(key))
}

func GetUInt(key string) uint { return g2.GetUInt(key) }
func (gc *GetConf) GetUInt(key string) uint {
	return cast.ToUint(gc.Get(key))
}

func GetUint8(key string) uint8 { return g2.GetUint8(key) }
func (gc *GetConf) GetUint8(key string) uint8 {
	return cast.ToUint8(gc.Get(key))
}

func GetUint16(key string) uint16 { return g2.GetUint16(key) }
func (gc *GetConf) GetUint16(key string) uint16 {
	return cast.ToUint16(gc.Get(key))
}

func GetUint32(key string) uint32 { return g2.GetUint32(key) }
func (gc *GetConf) GetUint32(key string) uint32 {
	return cast.ToUint32(gc.Get(key))
}

func GetUint64(key string) uint64 { return g2.GetUint64(key) }
func (gc *GetConf) GetUint64(key string) uint64 {
	return cast.ToUint64(gc.Get(key))
}

func GetFloat32(key string) float32 { return g2.GetFloat32(key) }
func (gc *GetConf) GetFloat32(key string) float32 {
	return cast.ToFloat32(gc.Get(key))
}

func GetFloat64(key string) float64 { return g2.GetFloat64(key) }
func (gc *GetConf) GetFloat64(key string) float64 {
	return cast.ToFloat64(gc.Get(key))
}

func GetBool(key string) bool { return g2.GetBool(key) }
func (gc *GetConf) GetBool(key string) bool {
	return cast.ToBool(gc.Get(key))
}

func GetTime(key string) time.Time { return g2.GetTime(key) }
func (gc *GetConf) GetTime(key string) time.Time {
	return cast.ToTime(gc.Get(key))
}

// func GetDuration(key string) time.Duration { return g2.GetDuration(key) }
// func (gc *GetConf) GetDuration(key string) time.Duration {
// 	return cast.ToDuration(gc.Get(key))
// }

func getTypedValue(opt string, t reflect.Kind) interface{} {
	switch t {
	case reflect.Int:
		if value, err := strconv.ParseInt(opt, 10, 0); err == nil {
			return int(value)
		}
		return 0
	case reflect.Int8:
		if value, err := strconv.ParseInt(opt, 10, 8); err == nil {
			return int8(value)
		}
		return 0
	case reflect.Int16:
		if value, err := strconv.ParseInt(opt, 10, 16); err == nil {
			return int16(value)
		}
		return 0
	case reflect.Int32:
		if value, err := strconv.ParseInt(opt, 10, 32); err == nil {
			return int32(value)
		}
		return 0
	case reflect.Int64:
		if value, err := strconv.ParseInt(opt, 10, 64); err == nil {
			return int64(value)
		}
		return 0
	case reflect.Uint:
		if value, err := strconv.ParseUint(opt, 10, 0); err == nil {
			return uint(value)
		}
		return 0
	case reflect.Uint8:
		if value, err := strconv.ParseUint(opt, 10, 8); err == nil {
			return uint8(value)
		}
		return 0
	case reflect.Uint16:
		if value, err := strconv.ParseUint(opt, 10, 16); err == nil {
			return uint16(value)
		}
		return 0
	case reflect.Uint32:
		if value, err := strconv.ParseUint(opt, 10, 32); err == nil {
			return uint32(value)
		}
		return 0
	case reflect.Uint64:
		if value, err := strconv.ParseUint(opt, 10, 64); err == nil {
			return uint64(value)
		}
		return 0
	case reflect.Float32:
		if value, err := strconv.ParseFloat(opt, 32); err == nil {
			return float32(value)
		}
		return 0
	case reflect.Float64:
		if value, err := strconv.ParseFloat(opt, 64); err == nil {
			return value
		}
		return 0
	case reflect.Bool:
		if value, err := strconv.ParseBool(string(opt)); err == nil {
			return value
		}
		return false
	case reflect.String:
		return string(opt)
	case reflect.Struct:
		if t, err := StringToDate(opt); err == nil {
			return t
		}
		if sec, err := strconv.ParseInt(opt, 10, 64); err == nil {
			return time.Unix(sec, 0)
		}
		return time.Time{}
	}
	return nil
}

// From https://github.com/spf13/cast/blob/master/caste.go
// Copyright © 2014 Steve Francia <spf@spf13.com>.
// StringToDate attempts to parse a string into a time.Time type using a
// predefined list of formats.  If no suitable format is found, an error is
// returned.
func StringToDate(s string) (time.Time, error) {
	return parseDateWith(s, []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05", // iso8601 without timezone
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC850,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		"2006-01-02 15:04:05.999999999 -0700 MST", // Time.String()
		"2006-01-02",
		"02 Jan 2006",
		"2006-01-02T15:04:05-0700", // RFC3339 without timezone hh:mm colon
		"2006-01-02 15:04:05 -07:00",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05Z07:00", // RFC3339 without T
		"2006-01-02 15:04:05Z0700",  // RFC3339 without T or timezone hh:mm colon
		"2006-01-02 15:04:05",
		time.Kitchen,
		time.Stamp,
		time.StampMilli,
		time.StampMicro,
		time.StampNano,
	})
}

// From https://github.com/spf13/cast/blob/master/caste.go
// Copyright © 2014 Steve Francia <spf@spf13.com>.
func parseDateWith(s string, dates []string) (d time.Time, e error) {
	for _, dateType := range dates {
		if d, e = time.Parse(dateType, s); e == nil {
			return
		}
	}
	return d, fmt.Errorf("unable to parse date: %s", s)
}
