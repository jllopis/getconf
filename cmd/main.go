package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jllopis/getconf"
	"github.com/jllopis/getconf/backend"
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
	conf     *getconf.GetConf
	verbose  bool
	kvPrefix = "/settings/apps"
	kvBucket = "v1"
)

func main() {
	var err error
	conf, err = getconf.New("testgetconf", &Config{}).EnableKVStore(&getconf.KVOptions{
		Backend: "consul",
		URLs:    []string{"localhost:8500"},
		KVConfig: &backend.Config{
			ConnectionTimeout: 10 * time.Second,
			Bucket:            kvBucket,
			PersistConnection: true,
			Prefix:            kvPrefix,
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

	fmt.Println("Testing consul:")
	for _, item := range []string{
		kvPrefix + "/testgetconf/" + kvBucket + "/Backend",
		kvPrefix + "/testgetconf/" + kvBucket + "/debug",
		kvPrefix + "/testgetconf/" + kvBucket + "/integer",
		kvPrefix + "/testgetconf/" + kvBucket + "/pi",
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		go func() {
			select {
			case <-ctx.Done():
				fmt.Println(item + " -> " + ctx.Err().Error()) // prints "context deadline exceeded"
			}
		}()

		if e, err := conf.KVStore.Exists(item); err == nil && e {
			val, err := conf.KVStore.Get(ctx, item)
			if err != nil {
				fmt.Printf("Error trying accessing value at key: %v\n", item)
			}
			fmt.Printf("GOT: %#+v\n", val)
			fmt.Printf("Type: %T, Key: %s, Value: %s\n", val, item, val)
		} else {
			fmt.Printf("Key %v not found\n", item)
		}
		cancel()
	}

	fmt.Println("Testing KV List:")
	optlist, err := conf.ListKV("/settings")
	if err != nil {
		fmt.Printf("error calling ListKV: %s\n", err.Error())
		os.Exit(1)
	}
	for _, item := range optlist {
		fmt.Printf("Key: %s, Value: %v, LastIndex: %d\n", item.Key, item.Value, item.LastIndex)
	}

	key := kvPrefix + "/testgetconf/" + kvBucket + "/integer"
	// key := "integer"
	fmt.Printf("Monitoring key %s\n", key)
	ctx, cancel := context.WithCancel(context.Background())
	err = conf.WatchWithFunc(ctx, key, func(s []byte) {
		fmt.Printf("GOT NEW VALUE: %s (%T)\n", s, s)
	})
	if err != nil {
		fmt.Printf("Error trying to watch value at key: %v\tError: %s\n", key, err.Error())
	}
	time.Sleep(10 * time.Second)
	cancel()

	// fmt.Println("Testing MonitFunc:")
	// stop := make(chan struct{})
	// err = conf.MonitFunc(key, func(s []byte) {
	// 	fmt.Printf("GOT NEW VALUE: %s (%T)\n", s, s)
	// }, stop)
	// if err != nil {
	// 	fmt.Printf("Error trying to watch value at key: %v\tError: %s\n", key, err.Error())
	// }
	// time.Sleep(30 * time.Second)
	// stop <- struct{}{}

	fmt.Println("Testing WatchTree")
	key = kvPrefix + "/testgetconf/" + kvBucket
	fmt.Printf("Monitoring tree %s\n", key)
	ctx, cancel = context.WithCancel(context.Background())
	conf.SetWatchTimeDuration(1 * time.Second)
	err = conf.WatchTreeWithFunc(ctx, key, func(kv *backend.KVPair) {
		fmt.Printf("GOT NEW VALUE: %s = %s\n", kv.Key, kv.Value)
	})
	if err != nil {
		fmt.Printf("Error trying to watch value at key: %v\tError: %s\n", key, err.Error())
	}

	time.Sleep(10 * time.Second)
	cancel()

	/*
		fmt.Println("Testing watch on dir " + kvPrefix + "/testgetconf/" + kvBucket)
		stopCh = make(chan struct{})
		conf.MonitTreeFunc("/settings/apps/testgetconf/v1", func(k string, v []byte) { fmt.Printf("GOT NEW VALUE FOR %s: %#+v (%T)\n", k, v, v) }, stopCh)
		time.Sleep(20 * time.Second)
		stopCh <- struct{}{}
		time.Sleep(1 * time.Second)
	*/

	intval, err := conf.GetInt("integer")
	if err != nil {
		fmt.Printf("integer Type = %T, integer=%d | error=%s\n", intval, intval, err)
	}
	fmt.Printf("\tType: %T, Key: %s - Value: %v\n", intval, "integer", intval)
	fmt.Println("Quitting test app")
}
