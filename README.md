getconf
======

[![version-2.0.0-alpha](https://img.shields.io/badge/Release-2.0.0--alpha-informational.svg)](https://github.com/jllopis/getconf) 
[![Godoc](https://godoc.org/github.com/jllopis/getconf?status.svg)](https://godoc.org/github.com/jllopis/getconf) 
[![Go Report Card](https://goreportcard.com/badge/github.com/jllopis/getconf)](https://goreportcard.com/report/github.com/jllopis/getconf)
[![GitHub Open Issues](https://img.shields.io/github/issues/jllopis/getconf.svg)](https://github.com/jllopis/getconf/issues)

Simple config management for your Go application.

## What is GetConf?

The main goal of `GetConf` is to provide an easy to use configuration manager that is able to load from environment, command line and/or a remote backend .

It works nice with [12-Factor apps](https://12factor.net). What can be done?:

* load config at startup
* set defaults
* read from environment variables
* read from command line flags
* read from remote config systems
* monitor remote config systems for changes (only [Consul](https://www.consul.io) is supported right now)

As it is intended to work mainly in [12-Factor apps](https://12factor.net), it does not support configuration files at this time. This could be added if really needed.

## Installation

    go get -u github.com/jllopis/getconf

We recommend using `go mod` to manage dependencies. _GetConf_ works with it and simplify dependency management. It is recommended to use **>=go1.12** in this case.

**getconf** itself has few direct dependencies:

- github.com/hashicorp/consul
- github.com/spf13/cast
- github.com/stretchr/testify (to run the tests)

## How to work with it

To start using _getconf_ is really simple:

1. Include the package *github.com/jllopis/getconf* in your _go_ file
2. Create a *struct* to hold the variables. This struct will not be filled with values, it is just a convenient method to define them. Note that both the struct and the fields must be exported (uppercase)
3. Call `getconf.Load( LoaderOptions )`
   where `LoaderOptions` is a struct to provide some data to `GetConf`:
     * `ConfigStruct interface{}` will carry the defined config struct. **This is mandatory**
     * `SetName string` is the name for the _Options Set_ used in a remote config server
     * `EnvPrefix string` sets the prefix prepended to the variable names in the environment (to preven collisions)
     * `KeyDelim string` sets the delimiter string to allow for embedded configuration _structs_
4. Now, the environment and flags are parsed for any of the config variables values
6. Use the variables through the **Get** methods provided
7. It will cast to the required type by the **Get** method so you can request a `GetString(string)` variable that is defined as `int`. Just be sure they are convertible

Additionally, you can check for values in a remote [consul](https://www.consul.io) store. To use the KV backend, you should call `EnableKVStore(*getconf.KVOptions)` on **getconf**:

```go
conf.EnableKVStore(&getconf.KVOptions{
	Backend: "consul",
	URLs:    []string{"127.0.0.1:8500"},
	KVConfig: &getconf.Config{
		ConnectionTimeout: 10 * time.Second,
		Bucket:            "test",
		PersistConnection: true,
	},
})
```

The **KVConfig** struct holds the configuration options specific to the backend.

### Trivial use case

The simplest use case will be to define the configuration _struct_ and get the values from enviroment and/or flags. Just to calls are needed:

```go

package main

import (
	"fmt"

	"github.com/jllopis/getconf"
)

type Config struct {
	Host     string `getconf:", default: etcd, info: this is the hostname"`   // just use the lowercase var name
	Port     int    `getconf:"default-port, info: service port"`   // rename the variable and add some info about it
	Debug    bool   `getconf:"debug, default: false, info: enable debug logging"`   // add a default
	IgnoreMe string `getconf:"-"`   // ignore this variable
}

func main() {
	fmt.Println("Starting test app...")

	// Load and set the variables defined in Config struct
	getconf.Load(&getconf.LoaderOptions{
		ConfigStruct: &Config{},
	})

	// just use the value
	fmt.Printf("Host=%s (%T)\n", getconf.GetString("host"), getconf.GetString("host"))

	// use it as the type defined...
	fmt.Printf("default-port=%d (%T)\n", getconf.GetInt("default-port"), getconf.GetInt("default-port"))
	// ...or get the type you need...
	fmt.Printf("default-port as string = %s (%T)\n", getconf.GetString("default-port"), getconf.GetString("default-port"))

	// ...for every supported type
	fmt.Printf("Debug = %t (Type: %T)\n", getconf.GetBool("debug"), getconf.GetBool("debug"))
	fmt.Printf("bool as string = %s (%T) ... and as int = %d (%T)\n", getconf.GetString("debug"), getconf.GetString("debug"), getconf.GetInt("debug"), getconf.GetInt("debug"))

	// just print the options that have taken some value
	fmt.Println("ALL OPTIONS SET:")
	o := getconf.GetAll()
	for k, v := range o {
		fmt.Printf("\tKey: %s (%T)- Value: %v\n", k, v, v)
	}

	// or the full getconf
	fmt.Printf("\nThe options as we know'em:\n%s\n", getconf.String())

	fmt.Println("Quitting test app")
}
```

Lets run it an see what happens...

```bash

ᐅ GCV2_DEFAULT_PORT=1112 go run littltest.go
Starting test app...
Host=etcd (string)
default-port=1112 (int)
default-port as string = 1112 (string)
Debug = false (Type: bool)
bool as string = false (string) ... and as int = 0 (int)
ALL OPTIONS SET:
	Key: debug (bool)- Value: false
	Key: host (string)- Value: etcd
	Key: default-port (int)- Value: 1112

The options as we know'em:
CONFIG OPTIONS:
	Key: default-port, Default: , Value: 1112, Type: int, LastSetBy: env, UpdatedAt: 2019-04-11 18:37:21.417653 +0000 UTC
	Key: debug, Default: false, Value: false, Type: bool, LastSetBy: default, UpdatedAt: 2019-04-11 18:37:21.417638 +0000 UTC
	Key: host, Default: etcd, Value: etcd, Type: string, LastSetBy: default, UpdatedAt: 2019-04-11 18:37:21.417632 +0000 UTC


Quitting test app
```

### Nested variables

Sometimes could be, in long configs, useful to define the config struct with support for nested variables. That way will be easy to manage.

The use is the same as the simple case but, nested structures have some specificities:

* the variable name in the config struct can be any valid name and can include chars, numbers or hyphen
* when the variables are loaded from the environment, use '__' as separator between father and child. All chars must be uppercase
* when the variables are loaded from the flags, use also '__' as separator but use the same case as it is defined in the struct
* when accessing a variable you must use their name. With nested variables you must use the separator '::'

So lets see the same example as before but adding some nested variables.

```go
package main

import (
	"fmt"

	"github.com/jllopis/getconf"
)

type Config struct {
	Server struct {
		Host string `getconf:", default: https://localhost, info: this is the hostname"`
		Port int    `getconf:"default-port, info: service port"`
	}
	Debug    bool   `getconf:"debug, default: false, info: enable debug logging"`
	IgnoreMe string `getconf:"-"`
}

func main() {
	fmt.Println("Starting test app...")

	// Load and set the variables defined in Config struct
	getconf.Load(&getconf.LoaderOptions{
		ConfigStruct: &Config{},
	})

	fmt.Printf("Host=%s (%T)\n", getconf.GetString("server::host"), getconf.GetString("server::host"))

	fmt.Printf("default-port=%d (%T)\n", getconf.GetInt("server::default-port"), getconf.GetInt("server::default-port"))

	// see what we have..
	fmt.Printf("\nThe options as we know'em:\n%s\n", getconf.String())

	fmt.Println("Quitting test app")
}
```

And here it is:

```bash

ᐅ GCV2_SERVER__DEFAULT_PORT=8484 go run littltest.go
Starting test app...
Host=https://localhost (string)
default-port=8484 (int)

The options as we know'em:
CONFIG OPTIONS:
	Key: debug, Default: false, Value: false, Type: bool, LastSetBy: default, UpdatedAt: 2019-04-11 18:57:52.718259 +0000 UTC
	Key: server::host, Default: https://localhost, Value: https://localhost, Type: string, LastSetBy: default, UpdatedAt: 2019-04-11 18:57:52.71824 +0000 UTC
	Key: server::default-port, Default: , Value: 8484, Type: int, LastSetBy: env, UpdatedAt: 2019-04-11 18:57:52.718282 +0000 UTC


Quitting test app
```

### Read from Consul

So we will create a connection to a consul backend. Then we will request a variable again. We have set the variable `debug` to `true` in Consul:

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jllopis/getconf"
	"github.com/jllopis/getconf/backend"
)

type Config struct {
	Server struct {
		Host string `getconf:", default: localhost, info: this is the hostname"`
		Port int    `getconf:"default-port, info: service port"`
	}
	Debug    bool   `getconf:"debug, default: false, info: enable debug logging"`
	IgnoreMe string `getconf:"-"`
}

func main() {
	fmt.Println("Starting test app...")

	// Load and set the variables defined in Config struct
	getconf.Load(&getconf.LoaderOptions{
		ConfigStruct: &Config{},
	})

	// see what value Debug have..
	fmt.Printf("[Pre Consul] Debug = %t (Type: %T)\n", getconf.GetBool("debug"), getconf.GetBool("debug"))

	fmt.Println("Enabling consul...")
	if err := getconf.EnableKVStore(&getconf.KVOptions{
		Backend: "consul",
		URLs:    []string{getconf.GetString("server::host") + ":" + getconf.GetString("server::default-port")},
		// URLs: []string{"localhost:8500"},
		KVConfig: &backend.Config{
			ConnectionTimeout: 10 * time.Second,
			Bucket:            "/settings/apps",
			PersistConnection: true,
			Prefix:            "v1",
		},
	}); err != nil {
		log.Panicf("cannot get bind to kv store. getconf error: %v\n", err)
	}

	// and after binding to consul...
	fmt.Printf("[Post Consul] Debug = %t (Type: %T)\n", getconf.GetBool("debug"), getconf.GetBool("debug"))

	fmt.Println("Quitting test app")
}
```

Let's see what happened...

```

ᐅ GCV2_SERVER__DEFAULT_PORT=8500 go run littltest.go
Starting test app...
[Pre Consul] Debug = false (Type: bool)
Enabling consul...
[Post Consul] Debug = true (Type: bool)
Quitting test app
```

## Whatch a variable in config server

The separator for nested values in _Consul_ is translated to '/', so you have to do it for setting the key.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jllopis/getconf"
	"github.com/jllopis/getconf/backend"
)

type Config struct {
	Server struct {
		Host string `getconf:", default: localhost, info: this is the hostname"`
		Port int    `getconf:"default-port, info: service port"`
	}
	Debug    bool   `getconf:"debug, default: false, info: enable debug logging"`
	IgnoreMe string `getconf:"-"`
}

func main() {
	fmt.Println("Starting test app...")

	// Load and set the variables defined in Config struct
	getconf.Load(&getconf.LoaderOptions{
		ConfigStruct: &Config{},
	})

	fmt.Println("Enabling consul...")
	if err := getconf.EnableKVStore(&getconf.KVOptions{
		Backend: "consul",
		URLs:    []string{getconf.GetString("server::host") + ":" + getconf.GetString("server::default-port")},
		// URLs: []string{"localhost:8500"},
		KVConfig: &backend.Config{
			ConnectionTimeout: 10 * time.Second,
			Prefix:            "/settings/apps",
			PersistConnection: true,
			Bucket:            "v1",
		},
	}); err != nil {
		log.Panicf("cannot get bind to kv store. getconf error: %v\n", err)
	}

	// and after binding to consul...
	fmt.Printf("[Port value] server::default-port = %d (Type: %T)\n", getconf.GetInt("default-port"), getconf.GetInt("default-port"))

	key := "server::default-port"
	fmt.Printf("Monitoring key '%s'\n", key)
	ctx, cancel := context.WithCancel(context.Background())
	err := getconf.WatchWithFunc(ctx, key, func(s []byte) {
		fmt.Printf("%s value changed on store: %s (%T)\n", key, s, s)
	})
	if err != nil {
		fmt.Printf("Error trying to watch value at key: %v\tError: %s\n", key, err.Error())
	}
	time.Sleep(10 * time.Second)
	cancel()

	fmt.Println("Quitting test app")
}
```

And call it by

```
ᐅ GCV2_SERVER__DEFAULT_PORT=8500 go run littltest.go
Starting test app...
Enabling consul...
[Port value] server::default-port = 0 (Type: int)
Monitoring key 'server::default-port'

```

Now, go to your Consul admin page and change the value for `server::default-port`. You will see something like:

```

server::default-port value changed on store: 80 ([]uint8)
Quitting test app
Closed watch on /settings/apps/gcv2/v1/server/default-port
```

Nice. The monitorin function `getconf.WatchWithFunc` take a func as param that will be executed when the monitored variable changes value. This func has the signature `func(s []byte)` where `s` is the `string` value got from the store.

We can also see that a context is provided in the first param. This allows to cancel the function anytime (in the sample we had a fixed timer of 10s).

## How it works

The options can be defined in:

1. environment
2. command line flags
3. remote key/val store

The order is the specified, meaning that the last option will win (if you set an environment variable it can be ovewritten by a command line flag). The last value read will be from the kv store.

To be parsed, you must define a struct in your program that will define the name and the type of the variables. The struct members **must** be uppercase (exported) otherwise _reflection_ will not work.

The struct can be any length and supported types are:

* int, int8, int16, int32, int64
* uint, uint8, uint16, uint32, uint64
* float32, float64
* string
* bool
* time.Time

The type `time.Time` supports different layouts (see godoc), like:

* **RFC3339Nano** (_2017-10-24T22:11:12+00:00_or _2017-10-24T22:21:23.159239900+00:00_)
* **Epoch** in seconds since January 1, 1970 UTC (_1508922049_)
* _2017-10-24T22:31:34_
* _2017-10-24 22:31:34_
* _2017-10-24_

Any other type will be discarded. A `time.Time` layout different that the ones supported (i.e. epoch in miliseconds) will produce an invalid result.

If a value can not be matched to the variable type, it will be discarded and the variable set to **nil**.

### struct tags

There are some tags that can be used:

- **-**: If a dash is found the variable will not be observed. Should be the only element in the tag
- **default**: Specifies the default value for the variable if none found
- **info**: Help information about the intended use of the variable

The tags are separated by comma. It holds a `key: value` pair for every setting (key before a _colon_, value after it). Ex: `default: defaultValue, info: an example`. Because _colon_ is used as a separater, a value can not contain a _colon_ in it.

The exception to the rule that is the first field that is the name of the variable. This name must be used to acces it later. If no name is assigned the tag must still start with a _colon_.

If a _key only_ field comes after first position, it will be ignored.

### environment

The variables must have a prefix provided by the user (defaults to `GCV2`). This is useful to prevent collisions. So you can set

    GCV2_VAR1="a value"

and at the same time

    YZ_VAR1=233

being _prefixes_ "GCV2" and "YZ".

The variable name will be set from the struct name or from the first field of the tag if it exists. It will be UPPERCASED so when you define the env vars must take this into consideration. Lower and Mixed case environment variables will not be taken into account.

Nested variables shoud use `__` as separator.

### command line flags

Command line flags are standard variables from the _go_ **flag** package. As before, the variable name will be set from the struct name or from the first field of the tag if it exists.

In command line, a _boolean_ flag acts as a switch, that is, it will take the value of **true** if present and **false** otherwise. You can force a boolean flag to _false_.

### remote kv store

In order to use the kv store, we need to use two structs to pass the configuration options. The first one is for **getconf** itself and inform about the backend to be user, the server URLs and the configuration needed to operate.

```go

type KVOptions struct {
	Backend  string
	URLs     []string
	KVConfig *Config
}
```

The Backends supported by GetConf now:

- Consul versions >= 0.5.1

The second struct is meant to be passed to the backend.

```go

type Config struct {
	ClientTLS         *ClientTLSConfig
	TLS               *tls.Config
	ConnectionTimeout time.Duration
	Bucket            string
	PersistConnection bool
}

type ClientTLSConfig struct {
	CertFile   string
	KeyFile    string
	CACertFile string
}

```

# Roadmap

- [x] Read variables from flags in command line
- [x] Read variables from environment
- [x] Implement remote config service
- [x] Add documentation
- [x] Suppot all go basic types plus time.Time
- [x] Support for nested options
- [x] Suport for auto cast on Get
- [ ] Add test cases

# Similar projects

- [stevenroose/gonfig](https://github.com/stevenroose/gonfig)
- [kelseyhightower/confd](https://github.com/kelseyhightower/confd)
- [spf13/viper](https://github.com/spf13/viper)
- [containous/staert](https://github.com/containous/staert)
- [bsideup/configo](https://github.com/bsideup/configo)
- [tomazk/envcfg](https://github.com/tomazk/envcfg)
- [jimmysawczuk/go-config](https://github.com/jimmysawczuk/go-config)
- [jinzhu/configor](https://github.com/jinzhu/configor)
- [zpatrick/go-config](https://github.com/zpatrick/go-config)
- [hashicorp/hcl](https://github.com/hashicorp/hcl)
- [JeremyLoy/config](https://github.com/JeremyLoy/config)
