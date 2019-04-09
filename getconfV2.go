package getconf

import (
	"errors"
	"fmt"
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
			k := strings.Split(tag, ",")
			if len(k[0]) > 0 {
				if strings.TrimSpace(k[0]) == "-" {
					return errors.New("untrack")
				}
				o.name = strings.ToLower(prefix + k[0])
			}
			for _, sk := range k[1:] {
				kv := strings.SplitN(sk, ":", 2)
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
		}
	}
	// return strings.ToLower(prefix + name)
	return nil
}

// func (gc *GetConfV2) parseStruct(structPtr interface{}, prefix string) error {
// 	s := reflect.ValueOf(structPtr).Elem()
// 	for i := 0; i < s.NumField(); i++ {
// 		fmt.Printf("[getconf#parseStruct] iteration over struct fields: %d - %v -> %v\n", i, s.Type().Field(i).Name, s.String())
// 		fieldType := s.Type().Field(i)
// 		// fieldPtr := s.Field(i).Addr().Interface()

// 		opt := &OptionV2{
// 			name:  strings.ToLower(prefix + fieldType.Name),
// 			oType: s.Field(i).Kind(),
// 		}

// 		tag := fieldType.Tag
// 		if t := tag.Get("getconf"); t != "" {
// 			err := parseTagsV2(opt, t)
// 			if err != nil && err.Error() == "untrack" {
// 				//log.Printf("getconf.parse: option %s not tracked!", o.name)
// 				continue
// 			}
// 		}

// 		switch fieldType.Type.Kind() {
// 		case reflect.Struct:
// 			switch reflect.ValueOf(structPtr).Interface().(type) {
// 			case time.Time:
// 				continue
// 			default:
// 				fmt.Printf("[GetConfV2#parseStruct] found nested struct: %s (%v)\n", fieldType.Name, s.Field(i).Kind())
// 				gc.parseStruct(fieldType.Type, opt.name+g2.keyDelim)
// 			}
// 		}

// 		g2.options[opt.name] = opt
// 	}

// 	return nil
// }

// // parseTags read the tags and set the corresponding variables in the Option struct
// func parseTagsV2(o *OptionV2, t string) error {
// 	for i, k := range strings.Split(t, ",") {
// 		if strings.TrimSpace(k) == "-" {
// 			return errors.New("untrack")
// 		}
// 		kv := strings.SplitN(k, ":", 2)
// 		if len(kv) == 1 {
// 			if i == 0 {
// 				o.name = strings.TrimSpace(kv[0])
// 			}
// 			continue
// 		}
// 		switch strings.TrimSpace(kv[0]) {
// 		case "default":
// 			o.defValue = strings.TrimSpace(kv[1])
// 			o.value = getTypedValue(o.defValue, o.oType)
// 			o.updatedAt = time.Now().UTC()
// 			o.lastSetBy = "default"
// 		case "info":
// 			o.usage = strings.TrimSpace(kv[1])
// 		}
// 	}
// 	return nil
// }

func GetAllV2() map[string]interface{} { return g2.GetAllV2() }
func (gc *GetConfV2) GetAllV2() map[string]interface{} {
	fmt.Printf("getconf: have %d options defined\n", len(g2.options))
	opts := make(map[string]interface{})
	for _, x := range g2.options {
		// if x.value == nil {
		// 	continue
		// }
		opts[x.name] = x.value
	}
	return opts
}
