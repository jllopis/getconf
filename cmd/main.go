package main

import (
	"fmt"
	"log"
	"time"

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

	//	fmt.Println("Testing Consul:")
	//	for _, item := range []string{"test/backend", "test/debug"} {
	//		if e, err := conf.KVStore.Exists(item); err == nil && e {
	//			pair, err := conf.KVStore.Get(item)
	//			if err != nil {
	//				fmt.Errorf("Error trying accessing value at key: %v", item)
	//			}
	//			fmt.Printf("GOT: %#+v\n", pair)
	//			fmt.Printf("Key: %s, Value: %s\n", pair.Key, pair.Value)
	//		} else {
	//			fmt.Printf("Key %v not found\n", item)
	//		}
	//	}

	fmt.Println("Quitting test app")
}
