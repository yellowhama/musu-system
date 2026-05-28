package publisher

type Publisher interface {
	Publish(topic, content string) (string, error)
}
