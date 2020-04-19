package broadcast

type Config struct {
	Telegram Telegram
}

type Message struct {
	Title string
	Link  string
}

type Broadcast interface {
	New() (Broadcast, error)
	Send(message Message) error
}
