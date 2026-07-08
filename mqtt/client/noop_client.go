package client

import "github.com/Teixeiraass/ground_guard_be/mqtt"

// NoopClient is used when MQTT is disabled or in unit tests.
type NoopClient struct {
	topicPrefix string
}

func NewNoopClient() Client {
	return &NoopClient{
		topicPrefix: mqtt.DefaultTopicPrefix,
	}
}

func (c *NoopClient) Publish(topic string, payload []byte) error {
	return nil
}

func (c *NoopClient) Subscribe(topic string, handler MessageHandler) error {
	return nil
}

func (c *NoopClient) Close() {}

func (c *NoopClient) TopicPrefix() string {
	return mqtt.DefaultTopicPrefix
}
