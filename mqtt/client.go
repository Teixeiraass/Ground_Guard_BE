package mqtt

// MessageHandler is called when a subscribed topic receives a message.
type MessageHandler func(topic string, payload []byte)

// Client abstracts MQTT publish/subscribe operations for device communication.
type Client interface {
	Publish(topic string, payload []byte) error
	Subscribe(topic string, handler MessageHandler) error
	Close()
	TopicPrefix() string
}
