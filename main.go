package main

import (
	"log"
	"mynews/config"
	"mynews/feed"
	"os"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	if cfg == nil {
		os.Exit(0)
	}

	if err = feed.New(cfg).Run(); err != nil {
		log.Fatal(err)
	}
}
