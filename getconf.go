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

	"github.com/docker/libkv/store"
)

var (
	ErrNotStructPointer    = errors.New("initializer is not a pointer to struct")
	ErrUninitializedStruct = errors.New("uninitialized struct")

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
	KVStore store.Store
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

// env then flags then remote (etcd, consul)
func New(setName string, clientStruct interface{}) *GetConf {
	opts := &GetConfOptions{ConfigStruct: clientStruct, EnableFlag: true}
	if setName != "" {
		opts.SetName = setName
		opts.EnableEnv = true
		opts.EnvPrefix = setName
	}
	g := NewWithOptions(opts)
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

func (c *GetConf) setConfigFromFlag(f *flag.Flag) {
	c.setOption(f.Name, f.Value.String(), "flag")
}

func (c *GetConf) setOption(name, value, setBy string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.options[name].value = getTypedValue(value, c.options[name].oType)
	c.options[name].updatedAt = time.Now().UTC()
	c.options[name].lastSetBy = setBy
}

// Get return the value associated to the key
func (c *GetConf) Get(key string) interface{} {
	if o, ok := c.options[key]; ok != false {
		return o.value
	}
	return nil
}

// GetString will return the value associated to the key as a string
func (c *GetConf) GetString(key string) string {
	if val, ok := c.options[key]; ok {
		return val.value.(string)
	}
	return ""
}

// GetInt will return the value associated to the key as an int
func (c *GetConf) GetInt(key string) (int, error) {
	if val, ok := c.options[key]; ok {
		return val.value.(int), nil
	}
	return 0, errors.New("Key not found")
}

// GetBool will return the value associated to the key as a bool
func (c *GetConf) GetBool(key string) (bool, error) {
	if val, ok := c.options[key]; ok {
		return val.value.(bool), nil
	}
	return false, errors.New("Key not found")
}

// GetFloat will return the value associated to the key as a float64
func (c *GetConf) GetFloat(key string) (float64, error) {
	if val, ok := c.options[key]; ok {
		return val.value.(float64), nil
	}
	return 0, errors.New("Key not found")
}

// GetAll return a map with the options and its values
// The values are of type interface{} so they have to be casted
func (c *GetConf) GetAll() map[string]interface{} {
	opts := make(map[string]interface{})
	for _, x := range c.options {
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
	for _, k := range strings.Split(t, ",") {
		if strings.TrimSpace(k) == "-" {
			return errors.New("untrack")
		}
		switch strings.Fields(k)[0] {
		case "default":
			o.defValue = strings.TrimSpace(strings.Fields(k)[1])
			o.value = getTypedValue(o.defValue, o.oType)
			o.updatedAt = time.Now().UTC()
			o.lastSetBy = "default"
		case "info":
			o.usage = strings.TrimSpace(strings.Join(strings.Fields(k)[1:], " "))
		default:
			o.name = strings.TrimSpace(strings.Fields(k)[0])
		}
	}
	return nil
}

func getTypedValue(opt string, t reflect.Kind) interface{} {
	switch t {
	case reflect.Int:
		if value, err := strconv.ParseInt(opt, 10, 0); err == nil {
			return value
		}
		return 0
	case reflect.Int8:
		if value, err := strconv.ParseInt(opt, 10, 8); err == nil {
			return value
		}
		return 0
	case reflect.Int16:
		if value, err := strconv.ParseInt(opt, 10, 16); err == nil {
			return value
		}
		return 0
	case reflect.Int32:
		if value, err := strconv.ParseInt(opt, 10, 32); err == nil {
			return value
		}
		return 0
	case reflect.Int64:
		if value, err := strconv.ParseInt(opt, 10, 64); err == nil {
			return value
		}
		return 0
	case reflect.Float32:
		if value, err := strconv.ParseFloat(opt, 32); err == nil {
			return value
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
	}
	return nil
}
