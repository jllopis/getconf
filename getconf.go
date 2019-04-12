package getconf

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
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
	kvPrefix  string // ej: "/settings/apps"
	kvBucket  string // ej: "v1"
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
	x := indirect(s)
	elem := reflect.ValueOf(x)
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
			g2.parsePtrStruct(fieldValue.Interface(), opt.name+"::")
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

// From html/template/content.go
// Copyright 2011 The Go Authors. All rights reserved.
// Returns de Value after dereferencing when needed
func indirect(a interface{}) interface{} {
	if a == nil {
		return nil
	}
	if t := reflect.TypeOf(a); t.Kind() != reflect.Ptr {
		// Avoid creating a reflect.Value if it's not a pointer.
		return a
	}
	v := reflect.ValueOf(a)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
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
