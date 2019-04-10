package getconf

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jllopis/getconf/backend"
)

var g2 *GetConfV2

func init() {
	// Create with default, hopefully safe, values
	g2 = &GetConfV2{
		setName:   "gcv2",
		envPrefix: "GCV2",
		keyDelim:  "::",
	}
}

type GetConfV2 struct {
	kvStore   backend.Backend
	options   map[string]*OptionV2
	setName   string
	envPrefix string
	keyDelim  string
}

// OptionV2 is the struct that holds information about the Option
type OptionV2 struct {
	name      string       // name as it appears on command line
	oType     reflect.Kind // type of the option
	value     interface{}  // value as set
	defValue  string       // default value (as text); for usage message
	usage     string       // help message
	lastSetBy string       // last loader that has set the value
	updatedAt time.Time    // updated timestamp
	mu        sync.RWMutex // will keep concurrent acces safe. It is set per Option so a single operation do not block the full config set
}

// LoaderOptions holds the options that getconfV2 will use to manage
// the configuration Options
type LoaderOptions struct {
	ConfigStruct interface{}
	SetName      string
	EnvPrefix    string
	KeyDelim     string
}

// OptionV2 implements flag.Value
func (o *OptionV2) String() string {
	return fmt.Sprintf("%v", o.value)
}

// Set sets the value of the Option
func (o *OptionV2) Set(s interface{}) error {
	o.value = s
	return nil
}

// IsBoolFlag returns true if the Options is of type Bool or false otherwise
func (o *OptionV2) IsBoolFlag() bool {
	return reflect.ValueOf(o.value).Elem().Kind() == reflect.Bool
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
	g2.options = make(map[string]*OptionV2)
	// Parse client struct
	g2.parsePtrStruct(lo.ConfigStruct, "")
	loadFromEnvV2()
}

// BindStruct will set the given struct fields to the values that exists in
// the getConf object.
// func (gc *getConfV2) BindStruct(s interface{}) error {
// 	return nil
// }

func (gc *GetConfV2) parsePtrStruct(s interface{}, prefix string) {
	elem := reflect.ValueOf(s).Elem()
	for i := 0; i < elem.NumField(); i++ {
		fieldValue := elem.Field(i)
		fieldType := elem.Type().Field(i)

		if fieldValue.Kind() == reflect.Struct {
			switch fieldValue.Interface().(type) {
			case time.Time:
				opt := new(OptionV2)
				err := parseTagsV2(fieldType, opt, prefix)
				if err != nil && err.Error() == "untrack" {
					continue
				}
				g2.options[opt.name] = opt
				continue
			}
			opt := new(OptionV2)
			err := parseTagsV2(fieldType, opt, prefix)
			if err != nil && err.Error() == "untrack" {
				continue
			}
			g2.parseStruct(fieldValue.Interface(), opt.name+"::")
			continue
		} else {
			opt := new(OptionV2)
			err := parseTagsV2(fieldType, opt, prefix)
			if err != nil && err.Error() == "untrack" {
				continue
			}
			g2.options[opt.name] = opt
		}
	}
}

func (gc *GetConfV2) parseStruct(s interface{}, prefix string) {
	structType := reflect.TypeOf(s)
	structValue := reflect.ValueOf(s)

	for i := 0; i < structType.NumField(); i++ {
		// fieldValue := structType.Field(i)
		fieldType := structValue.Type().Field(i)
		opt := new(OptionV2)
		err := parseTagsV2(fieldType, opt, prefix)
		if err != nil && err.Error() == "untrack" {
			continue
		}
		g2.options[opt.name] = opt
	}
}

func parseTagsV2(t reflect.StructField, o *OptionV2, prefix string) error {
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

func GetAllV2() map[string]interface{} { return g2.GetAllV2() }
func (gc *GetConfV2) GetAllV2() map[string]interface{} {
	opts := make(map[string]interface{})
	for _, x := range g2.options {
		if x.value == nil {
			continue
		}
		opts[x.name] = x.value
	}
	return opts
}

func loadFromEnvV2() {
	for _, o := range g2.options {
		val := getEnvV2(g2.envPrefix, o.name, g2.keyDelim)
		if val != "" {
			g2.setOption(o.name, val, "env")
		}
	}
}

// getEnvV2 Looks up variable in environment.
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
func getEnvV2(envPrefix, flagName, keyDelim string) string {
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

func (gc *GetConfV2) setOption(name, value, setBy string) {
	gc.options[name].mu.Lock()
	defer gc.options[name].mu.Unlock()

	gc.options[name].value = getTypedValue(value, gc.options[name].oType)
	gc.options[name].updatedAt = time.Now().UTC()
	gc.options[name].lastSetBy = setBy
}

func String() string { return g2.String() }
func (gc *GetConfV2) String() string {
	var s string
	for _, o := range g2.options {
		s = s + fmt.Sprintf("\tKey: %s, Default: %v, Value: %v, Type: %v, LastSetBy: %v, UpdatedAt: %v\n", o.name, o.defValue, o.value, o.oType, o.lastSetBy, o.updatedAt)
	}
	return fmt.Sprintf("CONFIG OPTIONS:\n%s\n", s)
}
