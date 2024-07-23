package pubsub

type subscriber struct {
	name  string
	msgCh chan []byte
}
