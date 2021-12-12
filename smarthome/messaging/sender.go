package messaging

type Sender interface {
	Send(m Message) error
}
