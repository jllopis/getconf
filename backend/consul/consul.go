package consul

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/jllopis/getconf/backend"
)

var (
	// ErrMultipleEndpointsUnsupported is thrown when there are
	// multiple endpoints specified for Consul
	ErrMultipleEndpointsUnsupported = errors.New("consul does not support multiple endpoints")

	watchTimeDuration = 15 * time.Second
)

type ConsulBackend struct {
	sync.Mutex
	config *api.Config
	client *api.Client
	prefix string
	bucket string
}

func New(endpoints []string, cnf *backend.Config) (*ConsulBackend, error) {
	if len(endpoints) > 1 {
		return nil, ErrMultipleEndpointsUnsupported
	}

	s := &ConsulBackend{}
	s.prefix = cnf.Prefix
	s.bucket = cnf.Bucket

	// Create Consul client
	config := api.DefaultConfig()
	s.config = config
	config.HttpClient = http.DefaultClient
	config.Address = endpoints[0]

	// Set options
	if cnf != nil {
		if cnf.TLS != nil {
			s.setTLS(cnf.TLS)
		}
		if cnf.ConnectionTimeout != 0 {
			s.setTimeout(cnf.ConnectionTimeout)
		}
	}

	// Creates a new client
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	s.client = client

	return s, nil
}

func (s *ConsulBackend) SetPrefix(p string) { s.prefix = p }
func (s *ConsulBackend) SetBucket(b string) { s.bucket = b }
func (s *ConsulBackend) GetPrefix() string  { return s.prefix }
func (s *ConsulBackend) GetBucket() string  { return s.bucket }

// SetTLS sets Consul TLS options
func (s *ConsulBackend) setTLS(tls *tls.Config) {
	s.config.HttpClient.Transport = &http.Transport{
		TLSClientConfig: tls,
	}
	s.config.Scheme = "https"
}

// setTimeout sets the timeout for connecting to ConsulBackend
func (s *ConsulBackend) setTimeout(time time.Duration) {
	s.config.WaitTime = time
}

// SetWatchTimeDuration sets the wait time for a watch connection to ConsulBackend
func (s *ConsulBackend) SetWatchTimeDuration(time time.Duration) {
	watchTimeDuration = time
}

// Exists return true if key exists in backend and false otherwise
func (s *ConsulBackend) Exists(key string) (bool, error) {
	_, err := s.Get(context.TODO(), key)
	if err != nil {
		if err == backend.ErrKeyNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Get a value given its key
// It is safe to provide a timeout by using context.Timeout.
func (s *ConsulBackend) Get(ctx context.Context, key string) ([]byte, error) {
	key = strings.TrimPrefix(key, "/")
	kv, _, err := s.client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, backend.ErrKeyNotFound
	}
	return kv.Value, nil
}

// List will get all keypairs under a prefix
// It is safe to provide a timeout by using context.Timeout.
func (s *ConsulBackend) List(ctx context.Context, key string) ([]*backend.KVPair, error) {
	key = strings.TrimPrefix(key, "/")
	pairs, _, err := s.client.KV().List(key, nil)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, backend.ErrKeyNotFound
	}
	ret := []*backend.KVPair{}
	for _, kv := range pairs {
		// if pair.Key == directory {
		// 	continue
		// }
		ret = append(ret, &backend.KVPair{Key: kv.Key, Value: kv.Value, LastIndex: kv.ModifyIndex})
	}
	return ret, nil
}

// Watch listens for changes on a "key"
// It returns a channel that will receive changes or pass on errors.
// When created, the current value will be sent to the channel.
// You can stop by using a context.Cancel when calling the method.
func (s *ConsulBackend) Watch(ctx context.Context, key string) (<-chan []byte, error) {
	respChan := make(chan []byte, 0)
	keypair, meta, err := s.client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}
	if keypair == nil && err == nil {
		return nil, backend.ErrKeyNotFound
	}
	go func(wi uint64) { // TODO: quit con cxt.Done()
		defer close(respChan)
		waitIndex := wi
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("Closed watch on %v\n", key)
				return
			default:
			}
			opts := api.QueryOptions{
				WaitIndex: waitIndex,
			}
			keypair, meta, err := s.client.KV().Get(key, &opts)
			if err != nil {
				return
			}
			// If LastIndex didn't change then it means `Get` returned
			// because of the WaitTime and the key didn't changed.
			if opts.WaitIndex == meta.LastIndex {
				continue
			}
			waitIndex = meta.LastIndex
			respChan <- keypair.Value
		}
	}(meta.LastIndex)
	return respChan, nil
}

// Copyright 2014-2016 Docker, Inc.
// WatchTree listens for changes on a "tree".
// It returns a channel that will receive changes or pass on errors.
// When created, the current values will be sent to the channel.
// You can stop by using a context.Cancel when calling the method.
func (s *ConsulBackend) WatchTree(ctx context.Context, directory string) (<-chan []*backend.KVPair, error) {
	respCh := make(chan []*backend.KVPair)
	if directory[len(directory)-1] != '/' {
		directory += "/"
	}

	go func() {
		defer close(respCh)
		var waitIndex uint64
		opts := &api.QueryOptions{WaitTime: watchTimeDuration}
		for {
			// Check if we should quit
			select {
			case <-ctx.Done():
				fmt.Printf("Closed watch on %v\n", directory)
				return
			default:
			}

			// Get all the childrens
			directory = strings.TrimPrefix(directory, "/")
			pairs, meta, err := s.client.KV().List(directory, opts)
			if err != nil {
				return
			}

			// If LastIndex didn't change then it means `Get` returned
			// because of the WaitTime and the child keys didn't change.
			if waitIndex == meta.LastIndex {
				time.Sleep(watchTimeDuration)
				continue
			}
			waitIndex = meta.LastIndex

			// Return children KV pairs to the channel
			kvpairs := []*backend.KVPair{}
			for _, pair := range pairs {
				if pair.Key == directory {
					continue
				}
				kvpairs = append(kvpairs, &backend.KVPair{
					Key:       pair.Key,
					Value:     pair.Value,
					LastIndex: pair.ModifyIndex,
				})
			}
			respCh <- kvpairs
		}
	}()

	return respCh, nil
}
