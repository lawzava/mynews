package broadcast

type Config struct {
	StdOut   StdOut
	Telegram Telegram
}

type Story struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Broadcast interface {
	New() (Broadcast, error)
	Send(message Story) error
}
