package broadcast

type Config struct {
	StdOut   Broadcast
	Telegram Broadcast
}

type Story struct {
	Title  string  `json:"title"`
	URL    string  `json:"url"`
	Score  float64 `json:"score,omitempty"`
	Reason string  `json:"scoreReason,omitempty"`
}

type Broadcast interface {
	Send(message Story) error
	Name() string
}
