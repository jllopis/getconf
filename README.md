getconf
======

**v0.4.0**

Go package to load configuration variables from OS environment, command line and/or a remote backend (still not supported).

## Requirements

* **go** ~> 1.6

## Installation

    go get github.com/jllopis/getconf

## Quick Start

We recommend vendoring the dependency. There are nice tools out there that works with go 1.5+ (using GO15VENDOREXPERIMENT) or 1.6 (vendor enabled by default):

- [govendor](https://github.com/kardianos/govendor)
- [gvt](https://github.com/FiloSottile/gvt)
- [godep](https://github.com/tools/godep)

To start using _getconf_:

1. Include the package *github.com/jllopis/getconf* in your file
2. Create a *struct* to hold the variables. This struct will not be filled with values, it is just a convenient method to define them
3. Call `getconf.New("myconf", myconfstruct interface{}) *GetConf`
   where:
     - *myconf* is the name we give to the set
     - *myconfstruct* is the *struct* that define your variables
4. Now, the environment and flags are parsed for any of the variables values
6. Use the variables through the **get** methods provided

```go
package main

import (
	"fmt"
	"log"

	"github.com/jllopis/getconf"
)

type Config struct {
	Backend  string `getconf:"default etcd, info backend to use"`
	Debug    bool   `getconf:"debug, default false, info enable debug logging"`
	IgnoreMe int    `getconf:"-"`
}

var (
	conf    *getconf.GetConf
	verbose bool
)

func main() {
	var err error
	conf, err = getconf.New("test", &Config{})
	if err != nil {
		log.Panicf("cannot get GetConf. getconf error: %v\n", err)
	}

	fmt.Println(conf)

	d, _ := conf.GetBool("debug")
	fmt.Printf("Debug = %v (Type: %T)\n", d, d)

	fmt.Println("ALL OPTIONS:")
	o := conf.GetAll()
	for k, v := range o {
		fmt.Printf("\tKey: %s - Value: %v\n", k, v)
	}
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

## Conventions

The options can be defined in:

1. environment
2. command line flags

The order is the specified, meaning that the last option will win (if you set an environment variable it can be ovewritten by a command line flag).

To be parsed, you must define a struct in your program that will define the name and the type of the variables. The struct members **must** be uppercase (exported) otherwise _reflection_ will not work.

The struct can be any length and supported types are:

* int, int8, int16, int32, int64
* float32, float64
* string
* bool

Any other type will be discarded.

If a value can not be matched to the variable type, it will be discarded and the variable set to **nil**.

### tags

There are some tags that can be used:

- **-**: If a dash is found the variable will not be observed
- **default**: Specifies the default value for the variable if none found
- **info**: Help information about the intended use of the variable

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

# Roadmap

- [x] Read variables from flags in command line
- [x] Read variables from environment
- [ ] Implement remote config service by way of [libkv](https://github.com/docker/libkv)
- [ ] Add test cases
- [ ] Add documentation

