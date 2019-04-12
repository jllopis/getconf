package backend

import (
	"context"
	"crypto/tls"
	"errors"
	"time"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

// Backend defines the interface that every Backend should implement
type Backend interface {
	// Get a value given its key
	// It is safe to provide a timeout by using context.Timeout.
	Get(ctx context.Context, key string) ([]byte, error)
	// List will get all keypairs under a prefix
	// It is safe to provide a timeout by using context.Timeout.
	List(ctx context.Context, key string) ([]*KVPair, error)
	// Watch listens for changes on a "key"
	// It returns a channel that will receive changes or pass on errors.
	// When created, the current value will be sent to the channel.
	// You can stop by using a context.Cancel when calling the method.
	Watch(ctx context.Context, key string) (<-chan []byte, error)
	// WatchTree listens for changes on a "tree".
	// It returns a channel that will receive changes or pass on errors.
	// When created, the current values will be sent to the channel.
	// You can stop by using a context.Cancel when calling the method.
	WatchTree(ctx context.Context, directory string) (<-chan []*KVPair, error)
	// Exists return true if key exists in backend and false otherwise
	Exists(key string) (bool, error)
	// SetWatchTimeDuration sets the wait time for a watch connection to Backend
	SetWatchTimeDuration(time time.Duration)
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

// KVPair represents a key/value pair stored in the Backend
type KVPair struct {
	Key       string
	Value     []byte
	LastIndex uint64
}
