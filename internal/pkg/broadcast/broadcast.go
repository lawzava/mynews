package broadcast

type Config struct {
	StdOut   Broadcast
	Telegram Broadcast
}

type Story struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Broadcast interface {
	Send(message Story) error
	Name() string
}
