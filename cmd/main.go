package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jllopis/getconf"
)

type Config struct {
	Backend  string  `getconf:"default etcd, info backend to use"`
	Debug    bool    `getconf:"debug, default false, info enable debug logging"`
	MyInt    int     `getconf:"integer, info test int setting"`
	Pi       float64 `getconf:"pi, info value of PI"`
	IgnoreMe int     `getconf:"-"`
}

func init() {
}

var (
	conf    *getconf.GetConf
	verbose bool
)

func main() {
	var err error
	conf, err = getconf.New("test", &Config{}).EnableKVStore(&getconf.KVOptions{
		Backend: "consul",
		URLs:    []string{"b2d:8500"},
		KVConfig: &getconf.Config{
			ConnectionTimeout: 10 * time.Second,
			Bucket:            "test",
			PersistConnection: true,
		},
	})
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

	fmt.Println("Testing Consul:")
	var b []byte
	for _, item := range []string{"test/test/Backend", "test/test/debug", "test/test/integer", "test/test/pi"} {
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

	fmt.Println("Testing watch on integer")
	stopCh := make(chan struct{})
	conf.MonitFunc("test/test/integer", func(s string) { fmt.Printf("GOT NEW VALUE: %v (%T)\n", s, s) }, stopCh)
	time.Sleep(20 * time.Second)
	stopCh <- struct{}{}
	time.Sleep(1 * time.Second)
	fmt.Println("Quitting test app")
}
