getconf
======

**v0.5.1**

Go package to load configuration variables from OS environment, command line and/or a remote backend (still not supported).

## Requirements

* **go** ~> 1.6

## Installation

    go get -u github.com/jllopis/getconf

## Quick Start

We recommend vendoring the dependency. There are nice tools out there that works with go 1.5+ (using GO15VENDOREXPERIMENT) or 1.6 (vendor enabled by default):

- [dep](https://github.com/golang/dep)
- [govendor](https://github.com/kardianos/govendor)
- [gvt](https://github.com/FiloSottile/gvt)
- [godep](https://github.com/tools/godep)

**getconf** depends on some other packages:

- github.com/coreos/etcd
- github.com/abronan/valkeyrie
- github.com/golang/protobuf
- github.com/hashicorp/consul
- google.golang.org/grpc

To start using _getconf_:

1. Include the package *github.com/jllopis/getconf* in your file
2. Create a *struct* to hold the variables. This struct will not be filled with values, it is just a convenient method to define them. Note that both the struct and the fields must be exported (uppercase)
3. Call `getconf.New("myconf", myconfstruct interface{}) *GetConf`
   where:
     - *myconf* is the name we give to the set
     - *myconfstruct* is the *struct* that define your variables
4. Now, the environment and flags are parsed for any of the variables values
6. Use the variables through the **get** methods provided

Additionally, you can check for values in a remote _Key/Val_ store such as [etcd](https://coreos.com/etcd) or [consul](https://www.consul.io). We use [valkeyrie](https://github.com/abronan/valkeyrie) so any backend supported by [valkeyrie](https://github.com/abronan/valkeyrie) should work.

To use the KV backend, you can call EnableKVStore(*getconf.KVOptions) on the **gconf** struct:

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

The **KVConfig** struct holds the configuration options specific to the backend you will use.

In the following example we will parse the command line flags, the environment and a consul backend. Then we will watch a variable change. If such a change happens, the provided function will be executed.

```go

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jllopis/getconf"
)

type Config struct {
	Backend  string  `getconf:"default: etcd, info: backend to use"`
	Debug    bool    `getconf:"debug, default: false, info: enable debug logging"`
	IgnoreMe int     `getconf:"-"`
}

var (
	conf    *getconf.GetConf
	verbose bool
)

func main() {
	var err error
	conf, err = getconf.New("test", &Config{}).EnableKVStore(&getconf.KVOptions{
		Backend: "consul",
		URLs:    []string{"127.0.0.1:8500"},
		KVConfig: &getconf.Config{
			ConnectionTimeout: 10 * time.Second,
			Bucket:            "test",
			PersistConnection: true,
		},
	})
	if err != nil {
		log.Panicf("cannot get GetConf. getconf error: %v\n", err)
	}

	// Print the options that we got
	fmt.Println(conf)

	// Get a bool value
	d, _ := conf.GetBool("debug")
	fmt.Printf("Debug = %v (Type: %T)\n", d, d)

	// Print every option that we care
	fmt.Println("ALL OPTIONS:")
	o := conf.GetAll()
	for k, v := range o {
		fmt.Printf("\tKey: %s - Value: %v\n", k, v)
	}

	fmt.Println("Testing KV watch")
	
	// A channel that will stop the watcher go routine
	stopCh := make(chan struct{})
	
	// The func passed will print the value when changed on the remote KV store. The first parameter is the value to
	// monitor. The path will be built using the setName passed to getconf.New and the Bucket passed int KVConfig so
	// it will be setname + "/" + Bucket + "/" + key
	conf.MonitFunc("integer", func(s string) { fmt.Printf("GOT NEW VALUE: %v (%T)\n", s, s) }, stopCh)
	
	// When we are no more interested in the watcher, send an empty struct to quit
	stopCh <- struct{}{}

	// We also have access to the underlying store via libkv...
	fmt.Println("Testing direct KV access:")
	var b []byte
	for _, item := range []string{"test/test/backend", "test/test/debug"} {
		if e, err := conf.KVStore.Exists(item); err == nil && e {
			pair, err := conf.KVStore.Get(item)
			if err != nil {
				fmt.Errorf("Error trying accessing value at key: %v", item)
			}
			fmt.Printf("GOT: %#+v\n", pair)
			b = pair.Value
			fmt.Printf("Key: %s, Value: %s\n", pair.Key, b)
		} else {
			fmt.Printf("Key %v not found\n", item)
		}
	}

	// We're done. Bye, bye...
	fmt.Println("Quitting test app")
}
```

And call it by

```bash
(go1.6) $ TEST_BACKEND="my.server.org" go run cmd/main.go --debug
CONFIG OPTIONS:
	Key: Backend, Default: etcd, Value: my.server.org, Type: string, LastSetBy: env, UpdatedAt: 2016-03-13 12:06:42.942103146 +0000 UTC
	Key: debug, Default: false, Value: true, Type: bool, LastSetBy: flag, UpdatedAt: 2016-03-13 12:06:42.942141204 +0000 UTC


Debug = true (Type: bool)
ALL OPTIONS:
	Key: debug - Value: true
	Key: Backend - Value: my.server.org
(go1.2) $
```

Now try changing **debug** in the KV Store:


## Conventions

The options can be defined in:

1. environment
2. command line flags
3. remote key/val store

The order is the specified, meaning that the last option will win (if you set an environment variable it can be ovewritten by a command line flag). The last value read will be from the kv store.

To be parsed, you must define a struct in your program that will define the name and the type of the variables. The struct members **must** be uppercase (exported) otherwise _reflection_ will not work.

The struct can be any length and supported types are:

* int, int8, int16, int32, int64
* float32, float64
* string
* bool
* time.Time

The type `time.Time` supports the following layouts:

* **RFC3339Nano** (_2017-10-24T22:11:12+00:00_or _2017-10-24T22:21:23.159239900+00:00_)
* **Epoch** in seconds since January 1, 1970 UTC (_1508922049_)
* _2017-10-24T22:31:34_
* _2017-10-24 22:31:34_
* _2017-10-24_

Any other type will be discarded. A `time.Time` layout different that the ones supported (i.e. epoch in miliseconds) will produce an invalid result.

If a value can not be matched to the variable type, it will be discarded and the variable set to **nil**.

Note that the values are readed as **string** in any of the three environments so if you want to store a binary value it should be **Base64** encoded. Same when reading it.

### tags

There are some tags that can be used:

- **-**: If a dash is found the variable will not be observed
- **default**: Specifies the default value for the variable if none found
- **info**: Help information about the intended use of the variable

The tags are separated by comma. It holds a `key: value` pair for every setting (key before a _colon_, value after it). Ex: `default: defaultValue, info: an example`.

The exception to the rule that if the first field does not have a colon (key only) it is assumed to be the name of the variable. This name must be used to acces it later. If a _key only_ field comes after first position, it will be ignored.

### environment

The variables must have a prefix provided by the user. This is useful to prevent collisions. So you can set

    FB_VAR1="a value"

and at the same time

    YZ_VAR1=233

being _prefixes_ "FB" and "YZ".

The variable name will be set from the struct name or from the first field of the tag if it exists. It will be UPPERCASED so wheb you define the env vars must take this into consideration. Lower and Mixed case environment variables will not be taken into account.

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

The [Backends supported by valkeyrie are](https://github.com/abronan/valkeyrie#supported-versions):

- Consul versions >= 0.5.1
- Etcd versions >= 2.0
- Zookeeper versions >= 3.4.5
- Boltdb (as local store)
- Redis versions >= 3.2.6

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

Except for **Bucket** that will be used in the path to the value (valkeyrie use it only in the Boltdb backend), all the other options match with valkeyrie: [ClientTLSConfig](https://github.com/abronan/valkeyrie/blob/master/store/store.go#L60).

# Roadmap

- [x] Read variables from flags in command line
- [x] Read variables from environment
- [x] Implement remote config service by way of [valkeyrie](https://github.com/abronan/valkeyrie)
- [x] Add documentation
- [x] Suppot all go types, mainly date
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
