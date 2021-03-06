package config

import (
	"mynews/internal/pkg/broadcast"
	"strings"
)

// Defaults to "STDOUT".
func parseBroadcast(name string, cfg broadcast.Config) (broadcast.Broadcast, error) {
	switch strings.ToUpper(name) {
	case "TELEGRAM":
		return cfg.Telegram.New()
	default:
		return cfg.StdOut.New()
	}
}
