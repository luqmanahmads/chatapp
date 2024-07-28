package entity

type PublishParam struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Message  string `json:"message"`
}

type SubscribeParam struct {
	Subscriber string `json:"subscriber"`
}

type ChatMessage struct {
	Sender   string
	Receiver string
	Message  string
}
