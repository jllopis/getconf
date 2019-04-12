package getconf

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jllopis/getconf/backend"
	"github.com/jllopis/getconf/backend/consul"
)

type KVOptions struct {
	Backend  string
	URLs     []string
	KVConfig *backend.Config
}

func GetKVStore() backend.Backend { return g2.GetKVStore() }
func (gc *GetConf) GetKVStore() backend.Backend {
	return g2.kvStore
}

// EnableKVStore sets the backend store as resource for options.
// It will set the bucket with Prefix+setName+bucket
func EnableKVStore(opts *KVOptions) error { return g2.EnableKVStore(opts) }
func (gc *GetConf) EnableKVStore(opts *KVOptions) error {
	switch strings.ToLower(opts.Backend) {
	case "consul":
		// if opts.KVConfig.Prefix != "" && !strings.HasSuffix(opts.KVConfig.Prefix, "/") {
		// 	opts.KVConfig.Prefix = opts.KVConfig.Prefix + "/"
		// }
		g2.kvPrefix = opts.KVConfig.Prefix
		g2.kvBucket = opts.KVConfig.Bucket

		// Initialize a new store with consul
		kv, err := consul.New(opts.URLs, opts.KVConfig)
		if err != nil {
			return errors.New("cannot create store consul")
		}
		gc.kvStore = kv
	// case "etcd":
	// 	etcdv3.Register()
	// 	// Parse config
	// 	if opts.KVConfig.Prefix != "" && !strings.HasSuffix(opts.KVConfig.Prefix, "/") {
	// 		opts.KVConfig.Prefix = opts.KVConfig.Prefix + "/"
	// 	}
	// 	opts.KVConfig.Prefix = opts.KVConfig.Prefix + gc.GetSetName() + "/"

	// 	c := &store.Config{
	// 		TLS:               opts.KVConfig.TLS,
	// 		ConnectionTimeout: opts.KVConfig.ConnectionTimeout,
	// 		Bucket:            opts.KVConfig.Prefix + opts.KVConfig.Bucket,
	// 		PersistConnection: opts.KVConfig.PersistConnection,
	// 	}
	// 	if opts.KVConfig.ClientTLS != nil {
	// 		c.ClientTLS = &store.ClientTLSConfig{
	// 			CertFile:   opts.KVConfig.ClientTLS.CertFile,
	// 			KeyFile:    opts.KVConfig.ClientTLS.KeyFile,
	// 			CACertFile: opts.KVConfig.ClientTLS.CACertFile,
	// 		}
	// 	}
	// 	// Initialize a new store with consul
	// 	kv, err := valkeyrie.NewStore(
	// 		store.ETCDV3,
	// 		opts.URLs,
	// 		c,
	// 	)
	// 	if err != nil {
	// 		return gc, err //ors.New("cannot create store etcd")
	// 	}
	// 	gc.kvStore = kv
	default:
		return errors.New("unknown backend")
	}

	// Read options from KV Store
	loadFromKV(opts)

	return nil
}

func loadFromKV(opts *KVOptions) {
	for _, o := range g2.options {
		name := strings.Replace(o.name, g2.keyDelim, "/", -1)
		val := getKV(g2.kvStore, g2.kvPrefix+"/"+g2.setName+"/"+g2.kvBucket, name)
		if val != "" {
			g2.setOption(o.name, val, "kvstore")
		}
	}
}

func getKV(kvs backend.Backend, path, key string) string {
	prefix := path

	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}

	if e, err := kvs.Exists(prefix + key); err == nil && e {
		value, err := kvs.Get(context.TODO(), prefix+key)
		if err != nil {
			return ""
		}
		return string(value)
	}
	return ""
}

func ListKV(path string) ([]*backend.KVPair, error) { return g2.ListKV(path) }
func (gc *GetConf) ListKV(path string) ([]*backend.KVPair, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
			// fmt.Println(item + " -> " + ctx.Err().Error()) // prints "context deadline exceeded"
		}
	}()

	e, err := gc.kvStore.List(ctx, path)
	if err != nil {
		return nil, err
	}
	if len(e) < 1 {
		return nil, backend.ErrKeyNotFound
	}

	return e, nil
}

// WatchWithFunc will listen for a key to change in the store. The variable must exist in the
// store prior to its use.
// If creation must be watched, use MonitTreeFunc instead.
func WatchWithFunc(ctx context.Context, key string, f func(newval []byte)) error {
	return g2.WatchWithFunc(ctx, key, f)
}
func (gc *GetConf) WatchWithFunc(ctx context.Context, name string, f func(newval []byte)) error {
	key := getKVKey(name)
	if ok, err := gc.kvStore.Exists(key); err != nil {
		// if ok, key exists and there was an error so we return
		// if !ok, key does not exist so we can wait for its creation
		if ok {
			return err
		}
	}
	evt, err := gc.kvStore.Watch(ctx, key)
	if err != nil {
		return err
	}
	// if changed, exec func
	go func() {
		k := getGCKey(key)
		for {
			select {
			case val := <-evt:
				if val != nil {
					fmt.Printf("changed value for key %s -> %s\nOrig key: %s\n", k, val, key)
					gc.setOption(k, string(val), "kvstore")
					f(val)
				}
			case <-ctx.Done():
				fmt.Printf("Closed watch on %v\n", key)
				return
			}
		}
	}()
	return nil
}

func getKVKey(nm string) string {
	name := strings.Replace(nm, g2.keyDelim, "/", -1)
	fmt.Printf("kvPrefix='%s', g2.setName='%s', g2kvBucket='%s', name='%s'\n", g2.kvPrefix, g2.setName, g2.kvBucket, name)
	return g2.kvPrefix + "/" + g2.setName + "/" + g2.kvBucket + "/" + name
}

func getGCKey(k string) string {
	split := strings.SplitAfter(k, g2.kvPrefix+"/"+g2.setName+"/"+g2.kvBucket+"/")
	return strings.Replace(split[len(split)-1], "/", g2.keyDelim, -1)
}

func WatchTreeWithFunc(ctx context.Context, dir string, f func(*backend.KVPair)) error {
	return g2.WatchTreeWithFunc(ctx, dir, f)
}
func (gc *GetConf) WatchTreeWithFunc(ctx context.Context, dir string, f func(*backend.KVPair)) error {
	evt, err := gc.kvStore.WatchTree(ctx, dir)
	if err != nil {
		return err
	}
	go func() {
		dir = strings.TrimPrefix(dir, "/")
		if dir[len(dir)-1] != '/' {
			dir += "/"
		}
		for {
			select {
			case pairList := <-evt:
				for _, pair := range pairList {
					if pair != nil {
						split := strings.SplitAfter(pair.Key, dir)
						key := split[len(split)-1]
						gc.setOption(key, string(pair.Value), "kvstore")
						f(pair)
					}
				}
			}
		}
	}()
	return nil
}

// SetWatchTimeDuration sets the wait time for a watch connection to ConsulBackend
func SetWatchTimeDuration(time time.Duration) { g2.SetWatchTimeDuration(time) }
func (gc *GetConf) SetWatchTimeDuration(time time.Duration) {
	gc.kvStore.SetWatchTimeDuration(time)
}
