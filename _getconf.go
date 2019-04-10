// +build ignore

// Package getconf load the variables to be used in a program from different sources:
//
//   1. Environment variables
//   2. command line
//   3. remote server (consul, etcd)
//
// This is also the precedence order.
//
// The package know about the options by way of a struct definition that must be passed.
//
// Example:
//
//    type Config struct {
//        key1 int,
//        key2 string
//    }
//    config := getconf.New("default", &Config{})
//    fmt.Printf("Key1 = %d\nKey2 = %s\n", config.Get(key1), config.GetString(key2))
//
// The default names and behaviour can be modified by the use of defined tags in the variable
// declaration. This way you can state if a var have to be watched for changes in etcd or
// if must be ignored for example. Also it is possible to define the names to look for.
// As the package parse the command line options, it should be called at the program start,
// it must be the first action to call.
package getconf

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jllopis/getconf/backend"
)

var g *GetConf

func init() {
	g = &GetConf{}
}

var (
	ErrNotStructPointer    = errors.New("initializer is not a pointer to struct")
	ErrUninitializedStruct = errors.New("uninitialized struct")
	ErrKeyNotFound         = errors.New("key not found")
	ErrValueNotString      = errors.New("value is not of type string")

	setName string
)

type Option struct {
	name      string       // name as it appears on command line
	usage     string       // help message
	oType     reflect.Kind // type of the option
	value     interface{}  // value as set
	defValue  string       // default value (as text); for usage message
	lastSetBy string       // last loader that has set the value
	updatedAt time.Time    // updated timestamp
}

// Option implements flag.Value
func (o *Option) String() string {
	return fmt.Sprintf("%v", o.value)
}

func (o *Option) Set(s string) error {
	o.value = s
	return nil
}

func (o *Option) IsBoolFlag() bool {
	return o.oType == reflect.Bool
}

type GetConf struct {
	KVStore backend.Backend
	options map[string]*Option
	mu      sync.RWMutex
}

type GetConfOptions struct {
	ConfigStruct interface{}
	SetName      string
	EnableEnv    bool
	EnableFlag   bool
	EnvPrefix    string
}

func LoadConfig(setName string, clientStruct interface{}) {
	New(setName, clientStruct)
}

// env then flags then remote (etcd, consul)
func New(setName string, clientStruct interface{}) *GetConf {
	opts := &GetConfOptions{ConfigStruct: clientStruct, EnableFlag: true}
	if setName != "" {
		opts.SetName = setName
		opts.EnableEnv = true
		opts.EnvPrefix = setName
	}
	g = NewWithOptions(opts)
	return g
}

func NewWithOptions(opts *GetConfOptions) *GetConf {
	setName = opts.SetName
	g := &GetConf{}
	elem := reflect.ValueOf(opts.ConfigStruct).Elem()
	if elem.Kind() == reflect.Invalid {
		return g
	}
	if elem.Kind() != reflect.Struct {
		return g
	}

	g.options = make(map[string]*Option)
	// Parse client struct
	if err := g.parseStruct(elem); err != nil {
		return g
	}

	// Check env
	if opts.EnableEnv {
		loadFromEnv(g, opts)
	}

	// Register flags in flagSet and parse it
	if opts.EnableFlag {
		flagConfigSet := flag.NewFlagSet(opts.SetName, flag.ContinueOnError) //  flag.ExitOnError
		for _, o := range g.options {
			flagConfigSet.Var(o, o.name, o.usage)
		}
		flagConfigSet.Parse(os.Args[1:])
		flagConfigSet.Visit(g.setConfigFromFlag)
	}

	return g
}

func GetSetName() string { return g.GetSetName() }
func (g *GetConf) GetSetName() string {
	return setName
}

// parseStruct parses the struct provided and load the options array in the GetConf object
func (g *GetConf) parseStruct(s reflect.Value) error {
	for i := 0; i < s.NumField(); i++ {
		f := s.Type().Field(i)
		o := &Option{name: f.Name,
			oType: s.Field(i).Kind(),
		}
		tag := f.Tag
		if t := tag.Get("getconf"); t != "" {
			err := parseTags(o, t)
			if err != nil && err.Error() == "untrack" {
				//log.Printf("getconf.parse: option %s not tracked!", o.name)
				continue
			}
		}
		g.options[o.name] = o
	}
	return nil
}

func (g *GetConf) setConfigFromFlag(f *flag.Flag) {
	g.setOption(f.Name, f.Value.String(), "flag")
}

func (g *GetConf) setOption(name, value, setBy string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.options[name].value = getTypedValue(value, g.options[name].oType)
	g.options[name].updatedAt = time.Now().UTC()
	g.options[name].lastSetBy = setBy
}

// Set adds the value received as the value of the key.
// If the key does not exist, an error ErrKeyNotFound is returned
func Set(key, value string) error { return g.Set(key, value) }
func (g *GetConf) Set(key, value string) error {
	if reflect.TypeOf(value).String() != "string" {
		return ErrValueNotString
	}
	if _, ok := g.options[key]; !ok {
		return ErrKeyNotFound
	}
	g.setOption(key, value, "user")
	return nil
}

// Get return the value associated to the key
func Get(key string) interface{} { return g.Get(key) }
func (g *GetConf) Get(key string) interface{} {
	if o, ok := g.options[key]; ok != false {
		return o.value
	}
	return nil
}

// GetString will return the value associated to the key as a string
func GetString(key string) string { return g.GetString(key) }
func (g *GetConf) GetString(key string) string {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(string)
	}
	return ""
}

func GetTime(key string) time.Time { return g.GetTime(key) }
func (g *GetConf) GetTime(key string) time.Time {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(time.Time)
	}
	return time.Time{}
}

// GetInt will return the value associated to the key as an int
func GetInt(key string) (int, error) { return g.GetInt(key) }
func (g *GetConf) GetInt(key string) (int, error) {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(int), nil
	}
	return 0, errors.New("Key not found")
}

// GetInt8 will return the value associated to the key as an int8
func GetInt8(key string) (int8, error) { return g.GetInt8(key) }
func (g *GetConf) GetInt8(key string) (int8, error) {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(int8), nil
	}
	return 0, errors.New("Key not found")
}

// GetInt16 will return the value associated to the key as an int16
func GetInt16(key string) (int16, error) { return g.GetInt16(key) }
func (g *GetConf) GetInt16(key string) (int16, error) {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(int16), nil
	}
	return 0, errors.New("Key not found")
}

// GetInt32 will return the value associated to the key as an int32
func GetInt32(key string) (int32, error) { return g.GetInt32(key) }
func (g *GetConf) GetInt32(key string) (int32, error) {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(int32), nil
	}
	return 0, errors.New("Key not found")
}

// GetInt64 will return the value associated to the key as an int64
func GetInt64(key string) (int64, error) { return g.GetInt64(key) }
func (g *GetConf) GetInt64(key string) (int64, error) {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(int64), nil
	}
	return 0, errors.New("Key not found")
}

// GetBool will return the value associated to the key as a bool
func GetBool(key string) (bool, error) { return g.GetBool(key) }
func (g *GetConf) GetBool(key string) (bool, error) {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(bool), nil
	}
	return false, errors.New("Key not found")
}

// GetFloat will return the value associated to the key as a float64
func GetFloat(key string) (float64, error) { return g.GetFloat(key) }
func (g *GetConf) GetFloat(key string) (float64, error) {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(float64), nil
	}
	return 0, errors.New("Key not found")
}

// GetFloat32 will return the value associated to the key as a float32
func GetFloat32(key string) (float32, error) { return g.GetFloat32(key) }
func (g *GetConf) GetFloat32(key string) (float32, error) {
	if val, ok := g.options[key]; ok && val.value != nil {
		return val.value.(float32), nil
	}
	return 0, errors.New("Key not found")
}

// GetAll return a map with the options and its values
// The values are of type interface{} so they have to be casted
func GetAll() map[string]interface{} { return g.GetAll() }
func (g *GetConf) GetAll() map[string]interface{} {
	opts := make(map[string]interface{})
	for _, x := range g.options {
		if x.value == nil {
			continue
		}
		opts[x.name] = x.value
	}
	return opts
}

func (g *GetConf) String() string {
	var s string
	for _, o := range g.options {
		s = s + fmt.Sprintf("\tKey: %s, Default: %v, Value: %v, Type: %v, LastSetBy: %v, UpdatedAt: %v\n", o.name, o.defValue, o.value, o.oType, o.lastSetBy, o.updatedAt)
	}
	return fmt.Sprintf("CONFIG OPTIONS:\n%s\n", s)
}

// parseTags read the tags and set the corresponding variables in the Option struct
func parseTags(o *Option, t string) error {
	for i, k := range strings.Split(t, ",") {
		if strings.TrimSpace(k) == "-" {
			return errors.New("untrack")
		}
		kv := strings.SplitN(k, ":", 2)
		if len(kv) == 1 {
			if i == 0 {
				o.name = strings.TrimSpace(kv[0])
			}
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "default":
			o.defValue = strings.TrimSpace(kv[1])
			o.value = getTypedValue(o.defValue, o.oType)
			o.updatedAt = time.Now().UTC()
			o.lastSetBy = "default"
		case "info":
			o.usage = strings.TrimSpace(kv[1])
		}
	}
	return nil
}

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
