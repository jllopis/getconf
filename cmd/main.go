package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jllopis/getconf"
)

type Config struct {
	Backend              string    `getconf:"default: etcd, info: backend to use"`
	Debug                bool      `getconf:"debug, default: false, info: enable debug logging"`
	MyInt                int       `getconf:"integer, info: test int setting"`
	BigInt               int64     `getconf:"bigint, info: test int64 setting"`
	Pi                   float64   `getconf:"pi, info: value of PI"`
	IgnoreMe             int       `getconf:"-"`
	IgnoreField          bool      `getconf:"info: empty field last not allowed, surprise"`
	SupportedTimeFormat1 time.Time `getconf:"thetime1, default: 2017-10-24T22:11:12+00:00"`
	SupportedTimeFormat2 time.Time `getconf:"thetime2, default: 2017-10-24T22:21:23.159239900+00:00"`
	SupportedTimeFormat3 time.Time `getconf:"thetime3, default: 2017-10-24T22:31:34"`
	SupportedTimeFormat4 time.Time `getconf:"thetime4, default: 2017-10-24 22:41:45"`
	SupportedTimeFormat5 time.Time `getconf:"thetime5, default: 2017-10-24"`
	SupportedTimeFormat6 time.Time `getconf:"thetime6, default: 1508922049"`
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

	nullValue, err := conf.GetInt("integer")
	if err != nil {
		fmt.Printf("integer Type = %T, integer=%d | error=%s\n", nullValue, nullValue, err)
	}

	d, _ := conf.GetBool("debug")
	fmt.Printf("Debug = %v (Type: %T)\n", d, d)

	t := conf.GetTime("thetime")
	fmt.Printf("thetime = %v (Type: %T)\n", t, t)

	fmt.Println("ALL OPTIONS:")
	o := conf.GetAll()
	for k, v := range o {
		fmt.Printf("\tType: %T, Key: %s - Value: %v\n", v, k, v)
	}

	fmt.Println("Testing Consul:")
	var b []byte
	for _, item := range []string{"test/test/Backend", "test/test/debug", "test/test/integer", "test/test/pi"} {
		if e, err := conf.KVStore.Exists(item); err == nil && e {
			pair, err := conf.KVStore.Get(item)
			if err != nil {
				fmt.Printf("Error trying accessing value at key: %v\n", item)
			}
			fmt.Printf("GOT: %#+v\n", pair)
			b = pair.Value
			fmt.Printf("Type: %T, Key: %s, Value: %s\n", b, pair.Key, b)
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
