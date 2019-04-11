package getconf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tmpConfig struct {
	Port  int    `getconf:"port, default: 8000, info: default port to listen on"`
	AdId  string `getconf:"adid, info: universal server id"`
	Store struct {
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
	g2 = &GetConf{
		setName:   "gcv2",
		envPrefix: "GCV2",
		keyDelim:  "::",
	}

	tests := []struct {
		key   string
		value string
	}{
		{"GCV2_PORT", "121212"},
		{"GCV2_STORE::HOST", "t2_first_child"},
		{"GCV2_BROKERS::AMQP::URI", "amqp://mybroker.com"},
		{"GCV2_BROKERS::AMQP::USER", "sample_user"},
		{"GCV2_BROKERS::AMQP::PASS", "the-pass"},
	}
	for _, t := range tests {
		os.Setenv(t.key, t.value)
		defer os.Unsetenv(t.key)
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

// type Config struct {
// 	Port                string `getconf:"port, default: 8000, info: default port to listen on"`
// 	AdId                string `getconf:"adid, info: universal server id"`
// 	StoreHost           string `getconf:"store-host, default: addb.acb.info, info: store server address"`
// 	StorePort           string `getconf:"store-port, default: 5432, info: store server port"`
// 	StoreName           string `getconf:"store-name, default: addb, info: store database name"`
// 	StoreUser           string `getconf:"store-user, default: adadm, info: store user to connect to db"`
// 	StorePass           string `getconf:"store-pass, default: 00000000, info: store user password"`
// 	AmqpUri             string `getconf:"amqp-uri, info: amqp server uri"`
// 	AmqpExchange        string `getconf:"amqp-exchange, default: amqp.fanout, info: amqp server exchange"`
// 	AmqpEventRoutingKey string `getconf:"amqp-event-routing-key, info: amqp routing key for event messages"`
// 	MqttUri             string `getconf:"mqtt-uri, info: mqtt server uri"`
// 	MqttUser            string `getconf:"mqtt-user"`
// 	MqttPass            string `getconf:"mqtt-pass"`
// 	MqttTopic           string `getconf:"mqtt-topic, default: /ad/pbp, info: mqtt topic for events"`
// 	Mode                string `getconf:"mode, default: dev, info: startup mode"`
// }
