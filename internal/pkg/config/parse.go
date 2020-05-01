package config

import (
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/store"
	"strings"
)

// Defaults to "STDOUT"
func parseBroadcast(name string, cfg broadcast.Config) (broadcast.Broadcast, error) {
	switch strings.ToUpper(name) {
	case "TELEGRAM":
		return cfg.Telegram.New()
	default:
		return cfg.StdOut.New()
	}
}

// Defaults to "MEMORY"
func parseStore(name string, cfg store.Config) (store.Store, error) {
	switch strings.ToUpper(name) {
	case "REDIS":
		return cfg.RedisDB.New()
	case "POSTGRES":
		return cfg.PostgresDB.New()
	default:
		return cfg.MemoryDB.New()
	}
}
