package getconf

import (
	"crypto/tls"
	"errors"
	"strings"
	"time"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
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
}

// ClientTLSConfig contains data for a Client TLS configuration in the form
// the etcd client wants it.  Eventually we'll adapt it for ZK and Consul.
type ClientTLSConfig struct {
	CertFile   string
	KeyFile    string
	CACertFile string
}

func (gc *GetConf) EnableKVStore(opts *KVOptions) (*GetConf, error) {
	switch strings.ToLower(opts.Backend) {
	case "consul":
		// Register consul store to libkv
		consul.Register()
		// Parse config
		c := &store.Config{
			TLS:               opts.KVConfig.TLS,
			ConnectionTimeout: opts.KVConfig.ConnectionTimeout,
			Bucket:            opts.KVConfig.Bucket,
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
	default:
		return gc, errors.New("unknown backend")
	}

	// Read options from KV Store
	loadFromKV(gc, opts)

	return gc, nil
}

func loadFromKV(gc *GetConf, opts *KVOptions) {
	for _, o := range gc.options {
		val := getKV(gc.KVStore, gc.GetSetName(), opts.KVConfig.Bucket, o.name)
		if val != "" {
			gc.setOption(o.name, val, "kvstore")
		}
	}
}

func getKV(kvs store.Store, setName, bucket, key string) string {
	var prefix string
	if setName != "" {
		prefix = setName
	}
	if bucket != "" {
		prefix = prefix + "/" + bucket
		if prefix[len(prefix)-1] != '/' {
			prefix += "/"
		}
	}

	if e, err := kvs.Exists(prefix + key); err == nil && e {
		pair, err := kvs.Get(prefix + key)
		if err != nil {
			return ""
		}
		return string(pair.Value)
	}
	return ""
}
