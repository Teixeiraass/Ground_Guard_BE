package client

import (
	"fmt"
	"time"

	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/util"
	mqttgo "github.com/eclipse/paho.mqtt.golang"
)

const mqttPublishWaitTimeout = 5 * time.Second

// PahoClient implements Client using Eclipse Mosquitto via paho.mqtt.golang.
type PahoClient struct {
	client      mqttgo.Client
	topicPrefix string
}

func NewPahoClient(config util.Config) (Client, error) {
	if !config.MQTTEnabled {
		return NewNoopClient(), nil
	}

	if config.MQTTBrokerURL == "" {
		return nil, fmt.Errorf("MQTT_BROKER_URL is required when MQTT_ENABLED is true")
	}

	clientID := config.MQTTClientID
	if clientID == "" {
		clientID = "ground-guard-api"
	}

	topicPrefix := mqtt.NormalizeTopicPrefix(config.MQTTTopicPrefix)

	opts := mqttgo.NewClientOptions()
	opts.AddBroker(config.MQTTBrokerURL)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetOrderMatters(false)

	if config.MQTTUsername != "" {
		opts.SetUsername(config.MQTTUsername)
		opts.SetPassword(config.MQTTPassword)
	}

	client := mqttgo.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("cannot connect to mqtt broker: %w", token.Error())
	}

	return &PahoClient{
		client:      client,
		topicPrefix: topicPrefix,
	}, nil
}

func (c *PahoClient) Publish(topic string, payload []byte) error {
	token := c.client.Publish(topic, 1, false, payload)
	if !token.WaitTimeout(mqttPublishWaitTimeout) {
		return fmt.Errorf("mqtt publish timed out after %s", mqttPublishWaitTimeout)
	}
	return token.Error()
}

func (c *PahoClient) Subscribe(topic string, handler MessageHandler) error {
	token := c.client.Subscribe(topic, 1, func(_ mqttgo.Client, msg mqttgo.Message) {
		handler(msg.Topic(), msg.Payload())
	})
	token.Wait()
	return token.Error()
}

func (c *PahoClient) Close() {
	if c.client.IsConnected() {
		c.client.Disconnect(250)
	}
}

func (c *PahoClient) TopicPrefix() string {
	return mqtt.NormalizeTopicPrefix(c.topicPrefix)
}
