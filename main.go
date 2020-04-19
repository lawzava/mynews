package main

import (
	"log"
)

func main() {
	cfg, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	p, err := newFeed(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err = p.run(); err != nil {
		log.Fatal(err)
	}
}
