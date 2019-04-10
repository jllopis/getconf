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

var g2 *GetConf

var (
	ErrNotStructPointer    = errors.New("initializer is not a pointer to struct")
	ErrUninitializedStruct = errors.New("uninitialized struct")
	ErrKeyNotFound         = errors.New("key not found")
	ErrValueNotString      = errors.New("value is not of type string")
)

func init() {
	// Create with default, hopefully safe, values
	g2 = &GetConf{
		setName:   "gcv2",
		envPrefix: "GCV2",
		keyDelim:  "::",
	}
}

type GetConf struct {
	kvStore   backend.Backend
	options   map[string]*Option
	setName   string
	envPrefix string
	keyDelim  string
}

// Option is the struct that holds information about the Option
type Option struct {
	name      string       // name as it appears on command line
	oType     reflect.Kind // type of the option
	value     interface{}  // value as set
	defValue  string       // default value (as text); for usage message
	usage     string       // help message
	lastSetBy string       // last loader that has set the value
	updatedAt time.Time    // updated timestamp
	mu        sync.RWMutex // will keep concurrent acces safe. It is set per Option so a single operation do not block the full config set
}

// LoaderOptions holds the options that getconf will use to manage
// the configuration Options
type LoaderOptions struct {
	ConfigStruct interface{}
	SetName      string
	EnvPrefix    string
	KeyDelim     string
}

// Option implements flag.Value
func (o *Option) String() string {
	return fmt.Sprintf("%v", o.value)
}

// Set sets the value of the Option
func (o *Option) Set(s string) error {
	o.value = s
	return nil
}

// IsBoolFlag returns true if the Options is of type Bool or false otherwise
func (o *Option) IsBoolFlag() bool {
	return o.oType == reflect.Bool
}

func GetSetName() string { return g2.GetSetName() }
func (gc *GetConf) GetSetName() string {
	return gc.setName
}

// Load will read the configuration options and keep a references in its own struct.
// The Options must be accessed through the provided methods and values will not be
// binded to the provided config struct.
//
// The variables will be read in the following order:
//   1. Environment variables
//   2. command line flags
//   3. remote server (consul)
func Load(lo *LoaderOptions) {
	g2.options = make(map[string]*Option)
	// Parse client struct
	g2.parsePtrStruct(lo.ConfigStruct, "")
	loadFromEnv()
	g2.loadFromFlags()
}

// BindStruct will set the given struct fields to the values that exists in
// the getConf object.
// func (gc *getConf) BindStruct(s interface{}) error {
// 	return nil
// }

func (gc *GetConf) parsePtrStruct(s interface{}, prefix string) {
	elem := reflect.ValueOf(s).Elem()
	for i := 0; i < elem.NumField(); i++ {
		fieldValue := elem.Field(i)
		fieldType := elem.Type().Field(i)

		if fieldValue.Kind() == reflect.Struct {
			switch fieldValue.Interface().(type) {
			case time.Time:
				opt := new(Option)
				err := parseTags(fieldType, opt, prefix)
				if err != nil && err.Error() == "untrack" {
					continue
				}
				g2.options[opt.name] = opt
				continue
			}
			opt := new(Option)
			err := parseTags(fieldType, opt, prefix)
			if err != nil && err.Error() == "untrack" {
				continue
			}
			g2.parseStruct(fieldValue.Interface(), opt.name+"::")
			continue
		} else {
			opt := new(Option)
			err := parseTags(fieldType, opt, prefix)
			if err != nil && err.Error() == "untrack" {
				continue
			}
			g2.options[opt.name] = opt
		}
	}
}

func (gc *GetConf) parseStruct(s interface{}, prefix string) {
	structType := reflect.TypeOf(s)
	structValue := reflect.ValueOf(s)

	for i := 0; i < structType.NumField(); i++ {
		// fieldValue := structType.Field(i)
		fieldType := structValue.Type().Field(i)
		opt := new(Option)
		err := parseTags(fieldType, opt, prefix)
		if err != nil && err.Error() == "untrack" {
			continue
		}
		g2.options[opt.name] = opt
	}
}

func parseTags(t reflect.StructField, o *Option, prefix string) error {
	o.name = strings.ToLower(prefix + t.Name)
	o.oType = t.Type.Kind()
	if tag, exists := t.Tag.Lookup("getconf"); exists {
		if tag = strings.TrimSpace(tag); tag != "" {
			if tag == "-" {
				return errors.New("untrack")
			}

			name, opts := parseTag(tag)
			if name != "" {
				o.name = strings.ToLower(prefix + name)
			}
			k := strings.Split(opts, ",")

			for _, sk := range k {
				key, value := getKeyValFromTagOption(sk)
				switch key {
				case "default":
					o.defValue = value
					o.value = getTypedValue(o.defValue, o.oType)
					o.updatedAt = time.Now().UTC()
					o.lastSetBy = "default"
				case "info":
					o.usage = value
				}
			}
		}
	}
	return nil
}

func getKeyValFromTagOption(opt string) (string, string) {
	if idx := strings.Index(opt, ":"); idx != -1 {
		return strings.TrimSpace(opt[:idx]), strings.TrimSpace(opt[idx+1:])

	}
	fmt.Printf("ilegal option: %s\n\n", opt)
	return "", ""
}

func parseTag(tag string) (string, string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], strings.TrimSpace(tag[idx+1:])
	}
	return tag, ""
}

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

func loadFromEnv() {
	for _, o := range g2.options {
		val := getEnv(g2.envPrefix, o.name, g2.keyDelim)
		if val != "" {
			g2.setOption(o.name, val, "env")
		}
	}
}

// getEnv Looks up variable in environment.
// env variables must be uppercase and the only separator allowed in the underscore. Dots and middle score
// will be changed to underscore.
//
// The variable name must be preceded by a prefix. If no prefix is provided the variable will be ignored.
//
// There can be nested variables by using 'separator'. By defalut, separator is '__'. They will be transformed
// from '::' when the key is set for reading from ENV:
//
//     ex: parent::child -> GC2_PARENT__CHILD
//
// Taken from https://github.com/rakyll/globalconf/blob/master/globalconf.go#L159
func getEnv(envPrefix, flagName, keyDelim string) string {
	// If we haven't set an EnvPrefix, don't lookup vals in the ENV
	if envPrefix == "" {
		return ""
	}
	if !strings.HasSuffix(envPrefix, "_") {
		envPrefix += "_"
	}
	flagName = strings.Replace(flagName, ".", "_", -1)
	flagName = strings.Replace(flagName, "-", "_", -1)
	flagName = strings.Replace(flagName, keyDelim, "__", -1)
	envKey := strings.ToUpper(envPrefix + flagName)

	return os.Getenv(envKey)
}

func (gc *GetConf) setOption(name, value, setBy string) {
	gc.options[name].mu.Lock()
	defer gc.options[name].mu.Unlock()

	gc.options[name].value = getTypedValue(value, gc.options[name].oType)
	gc.options[name].updatedAt = time.Now().UTC()
	gc.options[name].lastSetBy = setBy
}

func String() string { return g2.String() }
func (gc *GetConf) String() string {
	var s string
	for _, o := range g2.options {
		s = s + fmt.Sprintf("\tKey: %s, Default: %v, Value: %v, Type: %v, LastSetBy: %v, UpdatedAt: %v\n", o.name, o.defValue, o.value, o.oType, o.lastSetBy, o.updatedAt)
	}
	return fmt.Sprintf("CONFIG OPTIONS:\n%s\n", s)
}

func (gc *GetConf) loadFromFlags() {
	// Register flags in flagSet and parse it
	flagConfigSet := flag.NewFlagSet(gc.setName, flag.ContinueOnError) //  flag.ExitOnError
	for _, o := range g2.options {
		flagConfigSet.Var(o, o.name, o.usage)
	}
	flagConfigSet.Parse(os.Args[1:])
	flagConfigSet.Visit(g2.setConfigFromFlag)
}

func (g2 *GetConf) setConfigFromFlag(f *flag.Flag) {
	g2.setOption(f.Name, f.Value.String(), "flag")
}

// Set adds the value received as the value of the key.
// If the key does not exist, an error ErrKeyNotFound is returned
func Set(key, value string) error { return g2.Set(key, value) }
func (gc *GetConf) Set(key, value string) error {
	if reflect.TypeOf(value).String() != "string" {
		return ErrValueNotString
	}
	if _, ok := g2.options[key]; !ok {
		return ErrKeyNotFound
	}
	g2.setOption(key, value, "user")
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
