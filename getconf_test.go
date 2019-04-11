package getconf

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type tmpConfig struct {
	Port   int       `getconf:"port, default: 8000, info: default port to listen on"`
	AdId   string    `getconf:"adid, info: universal server id"`
	NoName time.Time `getconf:", info: hold the time"`
	Store  struct {
		Host string `getconf:"host, default: addb.acb.info, info: store server address"`
		Port int    `getconf:"port, default: 5432, info: store server port"`
		Name string `getconf:"name, default: addb, info: store database name"`
		User string `getconf:"user, default: adadm, info: store user to connect to db"`
		Pass string `getconf:"pass, default: 00000000, info: store user password"`
	}
	Broker struct {
		Amqp struct {
			AmqpUri             string `getconf:"uri, info: amqp server uri"`
			AmqpExchange        string `getconf:"exchange, default: amqp.fanout, info: amqp server exchange"`
			AmqpEventRoutingKey string `getconf:"event-routing-key, info: amqp routing key for event messages"`
		}
		Mqtt struct {
			MqttUri   string `getconf:"uri, info: mqtt server uri"`
			MqttUser  string `getconf:"user"`
			MqttPass  string `getconf:"pass"`
			MqttTopic string `getconf:"topic, default: /ad/pbp, info: mqtt topic for events"`
		}
	} `getconf:"brokers"`
	Mode string `getconf:"mode, default: dev, info: startup mode"`
}

func TestEnv(t *testing.T) {
	tests := []struct {
		envkey string
		key    string
		value  string
	}{
		{"GCV2_PORT", "port", "121212"},
		{"GCV2_STORE__HOST", "store::host", "t2_first_child"},
		{"GCV2_BROKERS__AMQP__URI", "brokers::amqp::uri", "amqp://mybroker.com"},
		{"GCV2_BROKERS__AMQP__EXCHANGE", "brokers::amqp::exchange", "amqp.fanout"},
		{"GCV2_BROKERS__AMQP__EVENT_ROUTING_KEY", "brokers::amqp::event-routing-key", "event-routing-key"},
	}
	for _, t := range tests {
		os.Setenv(t.envkey, t.value)
		defer os.Unsetenv(t.envkey)
	}
	Load(&LoaderOptions{
		ConfigStruct: &tmpConfig{},
		SetName:      "gc2test",
		EnvPrefix:    "GCV2",
	})
	for _, test := range tests {
		assert.Equal(t, test.value, GetString(test.key), "should be equal")
	}
}

func TestSet(t *testing.T) {
	if err := g2.Set("port", "10"); err != nil {
		t.Errorf("set port gives error: %s", err)
	}
}

func TestGetTypeValue(t *testing.T) {
	var result interface{}

	result = getTypedValue("9", reflect.Int)
	if reflect.ValueOf(result).Kind() != reflect.Int {
		t.Errorf("got: %T expected: int", result)
	}
	result = getTypedValue("9", reflect.Int8)
	if reflect.ValueOf(result).Kind() != reflect.Int8 {
		t.Errorf("got: %T expected: int8", result)
	}
	result = getTypedValue("9", reflect.Int16)
	if reflect.ValueOf(result).Kind() != reflect.Int16 {
		t.Errorf("got: %T expected: int16", result)
	}
	result = getTypedValue("9", reflect.Int32)
	if reflect.ValueOf(result).Kind() != reflect.Int32 {
		t.Errorf("got: %T expected: int32", result)
	}
	result = getTypedValue("9", reflect.Int64)
	if reflect.ValueOf(result).Kind() != reflect.Int64 {
		t.Errorf("got: %T expected: int64", result)
	}
	result = getTypedValue("9", reflect.Int16)
	if reflect.ValueOf(result).Kind() != reflect.Int16 {
		t.Errorf("got: %T expected: int16", result)
	}
	result = getTypedValue("false", reflect.Bool)
	if reflect.ValueOf(result).Kind() != reflect.Bool {
		t.Errorf("got: %T expected: bool", result)
	}
	result = getTypedValue("9.42", reflect.Float32)
	if reflect.ValueOf(result).Kind() != reflect.Float32 {
		t.Errorf("got: %T expected: float32", result)
	}
	result = getTypedValue("9.42", reflect.Float64)
	if reflect.ValueOf(result).Kind() != reflect.Float64 {
		t.Errorf("got: %T expected: float64", result)
	}
	result = getTypedValue("9", reflect.String)
	if reflect.ValueOf(result).Kind() != reflect.String {
		t.Errorf("got: %T expected: string", result)
	}
}
