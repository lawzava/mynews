package news

import (
	"mynews/internal/pkg/config"
)

type News struct {
	cfg *config.Config
}

func New(cfg *config.Config) News {
	return News{cfg}
}
