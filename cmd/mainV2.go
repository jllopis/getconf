package main

import (
	"fmt"
	"time"

	"github.com/jllopis/getconf"
)

type Config struct {
	Backend     string  `getconf:", default: etcd, info: backend to use"`
	Debug       bool    `getconf:"debug, default: false, info: enable debug logging"`
	MyInt       int     `getconf:"integer, info: test int setting"`
	BigInt      int64   `getconf:"bigint, info: test int64 setting"`
	Pi          float64 `getconf:"pi, info: value of PI"`
	IgnoreMe    int     `getconf:"-"`
	IgnoreField bool    `getconf:", default: true ,info: empty field last not allowed, surprise"`
	Times       struct {
		SupportedTimeFormat  time.Time `getconf:"thetime, info: sample empty time value"`
		SupportedTimeFormat1 time.Time `getconf:"thetime1, default: 2017-10-24T22:11:12+00:00"`
		SupportedTimeFormat2 time.Time `getconf:"thetime2, default: 2017-10-24T22:21:23.159239900+00:00"`
		SupportedTimeFormat3 time.Time `getconf:"thetime3, default: 2017-10-24T22:31:34"`
		SupportedTimeFormat4 time.Time `getconf:"thetime4, default: 2017-10-24 22:41:45"`
		SupportedTimeFormat5 time.Time `getconf:"thetime5, default: 2017-10-24"`
		SupportedTimeFormat6 time.Time `getconf:"thetime6, default: 1508922049"`
	} `getconf:"times-test"`
}

// var (
// 	verbose  bool
// 	kvPrefix = "/settings/apps"
// 	kvBucket = "v1"
// )

func main() {
	getconf.Load(&getconf.LoaderOptions{
		ConfigStruct: &Config{},
		SetName:      "gc2test",
		EnvPrefix:    "GCV2",
	})

	fmt.Println("Starting test app...")

	fmt.Println("ALL OPTIONS:")
	o := getconf.GetAllV2()
	for k, v := range o {
		fmt.Printf("\tType: %T, Key: %s - Value: %v\n", v, k, v)
	}

	fmt.Printf("Full Object getconf:\n%s\n", getconf.String())
	fmt.Println("Quitting test app")
}
