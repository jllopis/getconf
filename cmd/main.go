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

func init() {
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
	fmt.Println("Starting test app...")

	fmt.Println(conf)

	d, _ := conf.GetBool("debug")
	fmt.Printf("Debug = %v (Type: %T)\n", d, d)

	fmt.Println("ALL OPTIONS:")
	o := conf.GetAll()
	for k, v := range o {
		fmt.Printf("\tKey: %s - Value: %v\n", k, v)
	}
	fmt.Println("Quitting test app")
}
