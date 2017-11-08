package getconf

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/docker/libkv/store/etcd/v3"
	// "github.com/abronan/libkv"
	// "github.com/abronan/libkv/store"
	// "github.com/abronan/libkv/store/consul"
	// "github.com/docker/libkv/store/etcd"
)

type KVOptions struct {
	Backend  string
	URLs     []string
	KVConfig *Config
}

// Config contains the options for a storage client
type Config struct {
	ClientTLS         *ClientTLSConfig
	TLS               *tls.Config
	ConnectionTimeout time.Duration
	Bucket            string
	PersistConnection bool
	Prefix            string
}

// ClientTLSConfig contains data for a Client TLS configuration in the form
// the etcd client wants it.  Eventually we'll adapt it for ZK and Consul.
type ClientTLSConfig struct {
	CertFile   string
	KeyFile    string
	CACertFile string
}

// EnableKVStore sets the backend store as resource for options.
// It will set the bucket with Prefix+setName+bucket
func (gc *GetConf) EnableKVStore(opts *KVOptions) (*GetConf, error) {
	switch strings.ToLower(opts.Backend) {
	case "consul":
		// Register consul store to libkv
		consul.Register()
		if opts.KVConfig.Prefix != "" && !strings.HasSuffix(opts.KVConfig.Prefix, "/") {
			opts.KVConfig.Prefix = opts.KVConfig.Prefix + "/"
		}
		opts.KVConfig.Prefix = opts.KVConfig.Prefix + gc.GetSetName() + "/"

		// Parse config
		c := &store.Config{
			TLS:               opts.KVConfig.TLS,
			ConnectionTimeout: opts.KVConfig.ConnectionTimeout,
			Bucket:            opts.KVConfig.Prefix + opts.KVConfig.Bucket,
			PersistConnection: opts.KVConfig.PersistConnection,
		}
		if opts.KVConfig.ClientTLS != nil {
			c.ClientTLS = &store.ClientTLSConfig{
				CertFile:   opts.KVConfig.ClientTLS.CertFile,
				KeyFile:    opts.KVConfig.ClientTLS.KeyFile,
				CACertFile: opts.KVConfig.ClientTLS.CACertFile,
			}
		}
		// Initialize a new store with consul
		kv, err := libkv.NewStore(
			store.CONSUL,
			opts.URLs,
			c,
		)
		if err != nil {
			return gc, errors.New("cannot create store consul")
		}
		gc.KVStore = kv
	case "etcd":
		etcdv3.Register()
		// Parse config
		if opts.KVConfig.Prefix != "" && opts.KVConfig.Prefix[len(opts.KVConfig.Prefix)-1] != '/' {
			opts.KVConfig.Prefix = opts.KVConfig.Prefix + "/"
		}
		opts.KVConfig.Prefix = opts.KVConfig.Prefix + gc.GetSetName() + "/"

		c := &store.Config{
			TLS:               opts.KVConfig.TLS,
			ConnectionTimeout: opts.KVConfig.ConnectionTimeout,
			Bucket:            opts.KVConfig.Prefix + opts.KVConfig.Bucket,
			PersistConnection: opts.KVConfig.PersistConnection,
		}
		if opts.KVConfig.ClientTLS != nil {
			c.ClientTLS = &store.ClientTLSConfig{
				CertFile:   opts.KVConfig.ClientTLS.CertFile,
				KeyFile:    opts.KVConfig.ClientTLS.KeyFile,
				CACertFile: opts.KVConfig.ClientTLS.CACertFile,
			}
		}
		// Initialize a new store with consul
		kv, err := libkv.NewStore(
			store.ETCDV3,
			opts.URLs,
			c,
		)
		if err != nil {
			return gc, err //ors.New("cannot create store etcd")
		}
		gc.KVStore = kv
	default:
		return gc, errors.New("unknown backend")
	}

	// Read options from KV Store
	loadFromKV(gc, opts)

	return gc, nil
}

// MonitFunc will listen for a key to change in the store. The variable must exists in the
// store prior to its use.
// If creation must be watched, use MonitTreeFunc instead.
func (gc *GetConf) MonitFunc(key string, f func(newval string), stopCh <-chan struct{}) error {
	// TODO (jllopis):  build path using setName + "/" + Bucket + "/" + key
	// and watch value using it so the key passed will not be the full path anymore.
	// Possibly we will need to add setName and Bucket to the Option struct
	if ok, err := gc.KVStore.Exists(key, nil); err != nil {
		// if ok, key exists and there was an error so we return
		// if !ok, key does not exist so we can wait for its creation
		if ok {
			return err
		}
	}
	evt, err := gc.KVStore.Watch(key, stopCh, nil)
	if err != nil {
		return err
	}
	// if changed, exec func
	go func(stop <-chan struct{}) {
		for {
			select {
			case pair := <-evt:
				if pair != nil {
					f(string(pair.Value))
				}
			case <-stopCh:
				fmt.Printf("Closed watch on %v\n", key)
				return
			}
		}
	}(stopCh)
	return nil
}

// MonitTreeFunc will listen for changes in the store refered to any variable in the tree.
// If a variable does not exist yet, it will be reported upon creation.
func (gc *GetConf) MonitTreeFunc(dir string, f func(key string, newval []byte), stopCh <-chan struct{}) error {
	// TODO (jllopis):  build path using setName + "/" + Bucket + "/" + key
	// and watch value using it so the key passed will not be the full path anymore.
	// Possibly we will need to add setName and Bucket to the Option struct
	if ok, err := gc.KVStore.Exists(dir, nil); err != nil {
		// if ok, dir exists and there was an error so we return
		// if !ok, dir does not exist so we can wait for its creation
		if ok {
			return err
		}
	}
	evt, err := gc.KVStore.WatchTree(dir, stopCh, nil)
	if err != nil {
		return err
	}
	// if changed, exec func
	go func(stop <-chan struct{}) {
		for {
			select {
			case pairList := <-evt:
				for _, pair := range pairList {
					if pair != nil {
						if dir[len(dir)-1] != '/' {
							dir += "/"
						}
						split := strings.SplitAfter(pair.Key, dir)
						key := split[len(split)-1]
						gc.setOption(key, string(pair.Value), "kvstore")
						f(pair.Key, pair.Value)
					}
				}
			case <-stopCh:
				fmt.Printf("Closed watch on %v\n", dir)
				return
			}
		}
	}(stopCh)
	return nil
}

func loadFromKV(gc *GetConf, opts *KVOptions) {
	for _, o := range gc.options {
		val := getKV(gc.KVStore, opts.KVConfig.Prefix+opts.KVConfig.Bucket, o.name)
		if val != "" {
			gc.setOption(o.name, val, "kvstore")
		}
	}
}

func getKV(kvs store.Store, bucket, key string) string {
	prefix := bucket

	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}

	if e, err := kvs.Exists(prefix+key, nil); err == nil && e {
		pair, err := kvs.Get(prefix+key, nil)
		if err != nil {
			return ""
		}
		return string(pair.Value)
	}
	return ""
}
