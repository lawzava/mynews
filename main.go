package main

import (
	"flag"
	"fmt"
	"log"
	"news/internal/app/parser"
	"news/internal/pkg/storage"
	"strings"
	"time"
)

func main() {
	var config parser.Config

	if err := parseFlagsToConfig(&config); err != nil {
		log.Fatal(err)
	}

	p, err := parser.New(config)
	if err != nil {
		log.Fatal(err)
	}

	if err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func parseFlagsToConfig(config *parser.Config) error {
	var (
		sources             string
		storageType         string
		storageDetails      string
		intervalBetweenRuns uint64
	)

	flag.StringVar(&sources, "sources", "https://hnrss.org/newest.atom", "rss/atom source URLs separated by a comma")
	flag.Uint64Var(&intervalBetweenRuns, "interval", 60, "interval in seconds between each feed parsing run")

	flag.StringVar(&storageType, "storage", "memory",
		"storage type to use valid values are: 'memory' (persistent hash map), 'postgres', 'redis'")

	flag.StringVar(&storageDetails, "storageDetails", "redis://localhost:6379",
		"storage access URI if the type is not 'memory'")

	flag.Parse()

	config.Sources = strings.Split(sources, ",")
	config.SleepDurationBetweenRuns = time.Second * time.Duration(intervalBetweenRuns)

	st, err := storage.ParseType(storageType)
	if err != nil {
		return fmt.Errorf("parsing storage type: %w", err)
	}

	config.StorageType = st
	config.StorageAccessDetails = storageDetails

	return nil
}
