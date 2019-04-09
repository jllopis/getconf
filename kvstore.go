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

func GetKVStore() backend.Backend {
	return g.KVStore
}

// EnableKVStore sets the backend store as resource for options.
// It will set the bucket with Prefix+setName+bucket
func EnableKVStore(opts *KVOptions) (*GetConf, error) { return g.EnableKVStore(opts) }
func (gc *GetConf) EnableKVStore(opts *KVOptions) (*GetConf, error) {
	switch strings.ToLower(opts.Backend) {
	case "consul":
		if opts.KVConfig.Prefix != "" && !strings.HasSuffix(opts.KVConfig.Prefix, "/") {
			opts.KVConfig.Prefix = opts.KVConfig.Prefix + "/"
		}
		opts.KVConfig.Prefix = opts.KVConfig.Prefix + gc.GetSetName() + "/"

		// Initialize a new store with consul
		kv, err := consul.New(opts.URLs, opts.KVConfig)
		if err != nil {
			return gc, errors.New("cannot create store consul")
		}
		gc.KVStore = kv
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
	// 	gc.KVStore = kv
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
// Deprecated. This function is deprecated and will be removed in the next release. Use WatchWithFunc instead
func MonitFunc(key string, f func(newval []byte), stopCh <-chan struct{}) error {
	return g.MonitFunc(key, f, stopCh)
}
func (gc *GetConf) MonitFunc(key string, f func(newval []byte), stopCh <-chan struct{}) error {
	// TODO (jllopis):  build path using setName + "/" + Bucket + "/" + key
	// and watch value using it so the key passed will not be the full path anymore.
	// Possibly we will need to add setName and Bucket to the Option struct
	if ok, err := gc.KVStore.Exists(key); err != nil {
		// if ok, key exists and there was an error so we return
		// if !ok, key does not exist so we can wait for its creation
		if ok {
			return err
		}
	}
	evt, err := gc.KVStore.Watch(context.TODO(), key)
	if err != nil {
		return err
	}
	// if changed, exec func
	go func(stop <-chan struct{}) {
		for {
			select {
			case value := <-evt:
				if value != nil {
					f(value)
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
func MonitTreeFunc(dir string, f func(key string, newval []byte), stopCh <-chan struct{}) error {
	return g.MonitTreeFunc(dir, f, stopCh)
}
func (gc *GetConf) MonitTreeFunc(dir string, f func(key string, newval []byte), stopCh <-chan struct{}) error {
	// TODO (jllopis):  build path using setName + "/" + Bucket + "/" + key
	// and watch value using it so the key passed will not be the full path anymore.
	// Possibly we will need to add setName and Bucket to the Option struct
	if ok, err := gc.KVStore.Exists(dir); err != nil {
		// if ok, dir exists and there was an error so we return
		// if !ok, dir does not exist so we can wait for its creation
		if ok {
			return err
		}
	}
	ctx, ctxCancel := context.WithCancel(context.Background()) // debe llegar como parámetro a la función para que le cliente lo pueda cancelar
	evt, err := gc.KVStore.WatchTree(ctx, dir)
	if err != nil {
		ctxCancel()
		return err
	}
	// if changed, exec func
	go func() {
		defer ctxCancel()
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
			default:
			}
		}
	}()
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

func getKV(kvs backend.Backend, bucket, key string) string {
	prefix := bucket

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

func ListKV(path string) ([]*backend.KVPair, error) { return g.ListKV(path) }
func (gc *GetConf) ListKV(path string) ([]*backend.KVPair, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
			// fmt.Println(item + " -> " + ctx.Err().Error()) // prints "context deadline exceeded"
		}
	}()

	e, err := gc.KVStore.List(ctx, path)
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
	return g.WatchWithFunc(ctx, key, f)
}
func (gc *GetConf) WatchWithFunc(ctx context.Context, key string, f func(newval []byte)) error {
	// TODO (jllopis):  build path using setName + "/" + Bucket + "/" + key
	// and watch value using it so the key passed will not be the full path anymore.
	// Possibly we will need to add setName and Bucket to the Option struct
	if ok, err := gc.KVStore.Exists(key); err != nil {
		// if ok, key exists and there was an error so we return
		// if !ok, key does not exist so we can wait for its creation
		if ok {
			return err
		}
	}
	evt, err := gc.KVStore.Watch(ctx, key)
	if err != nil {
		return err
	}
	// if changed, exec func
	go func() {
		split := strings.SplitAfter(key, "/")
		k := split[len(split)-1]
		for {
			select {
			case val := <-evt:
				if val != nil {
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

func WatchTreeWithFunc(ctx context.Context, dir string, f func(*backend.KVPair)) error {
	return g.WatchTreeWithFunc(ctx, dir, f)
}
func (gc *GetConf) WatchTreeWithFunc(ctx context.Context, dir string, f func(*backend.KVPair)) error {
	evt, err := gc.KVStore.WatchTree(ctx, dir)
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
func SetWatchTimeDuration(time time.Duration) { g.SetWatchTimeDuration(time) }
func (g *GetConf) SetWatchTimeDuration(time time.Duration) {
	g.KVStore.SetWatchTimeDuration(time)
}
