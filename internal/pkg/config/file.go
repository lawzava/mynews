package config

import (
	"encoding/json"
	"fmt"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/storage"
	"os"
	"time"
)

type fileStructure struct {
	SleepDurationBetweenFeedParsing string `json:"sleepDurationBetweenFeedParsing"`
	SleepDurationBetweenBroadcasts  string `json:"sleepDurationBetweenBroadcasts"`

	BroadcastType       string `json:"broadcastType"`
	TelegramBotAPIToken string `json:"telegramBotAPIToken"`
	TelegramChatID      string `json:"telegramChatID"`

	Sources []fileStructureSource `json:"sources"`
}

type fileStructureSource struct {
	URL                 string    `json:"url"`
	IgnoreStoriesBefore time.Time `json:"ignoreStoriesBefore"`
	MustIncludeAnyOf    []string  `json:"mustIncludeAnyOf"`
	MustExcludeAnyOf    []string  `json:"mustExcludeAnyOf"`
}

func fromFile(filePath string, log *logger.Log) (*Config, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Warn(fmt.Sprintf("File '%s' does not exist", filePath))
		return nil, nil
	}

	configFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}

	defer func() { _ = configFile.Close() }()

	var file fileStructure

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&file); err != nil {
		return nil, fmt.Errorf("decoding config file: %w", err)
	}

	return file.toConfig()
}

func (f *fileStructure) toConfig() (*Config, error) {
	var (
		cfg Config
		err error
	)

	for _, source := range f.Sources {
		cfg.Sources = append(cfg.Sources, &Source{
			URL:                 source.URL,
			IgnoreStoriesBefore: source.IgnoreStoriesBefore,
			MustExcludeKeywords: source.MustExcludeAnyOf,
			MustIncludeKeywords: source.MustIncludeAnyOf,
		})
	}

	cfg.SleepDurationBetweenBroadcasts, err = time.ParseDuration(f.SleepDurationBetweenBroadcasts)
	if err != nil {
		return nil, fmt.Errorf("invalid broadcast sleep duration format: %w", err)
	}

	cfg.SleepDurationBetweenFeedParsing, err = time.ParseDuration(f.SleepDurationBetweenFeedParsing)
	if err != nil {
		return nil, fmt.Errorf("invalid feed parsing sleep duration format: %w", err)
	}

	var broadcastConfig broadcast.Config
	broadcastConfig.Telegram.BotAPIToken = f.TelegramBotAPIToken
	broadcastConfig.Telegram.ChatID = f.TelegramChatID

	cfg.Store = storage.New()

	cfg.Broadcast, err = parseBroadcast(f.BroadcastType, broadcastConfig)
	if err != nil {
		return nil, fmt.Errorf("parsing storage: %w", err)
	}

	return &cfg, nil
}

func createSampleFile(filePath string) error {
	if _, err := os.Stat(filePath); err != nil && os.IsExist(err) {
		return nil
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("initializing config file: %w", err)
	}

	defer func() { _ = file.Close() }()

	sources := []fileStructureSource{
		{
			URL:                 "https://hnrss.org/newest.atom",
			IgnoreStoriesBefore: time.Date(2020, 4, 20, 0, 0, 0, 0, time.UTC),
			MustIncludeAnyOf:    []string{"linux", "golang", "musk"},
			MustExcludeAnyOf:    []string{"windows", "trump", "apple"},
		},
	}

	defaultFileStructure := fileStructure{
		SleepDurationBetweenFeedParsing: (time.Minute * 5).String(),  // nolint:nomnd used for sample file
		SleepDurationBetweenBroadcasts:  (time.Second * 10).String(), // nolint:nomnd used for sample file

		BroadcastType: "stdout",
		Sources:       sources,
	}

	jsonWriter := json.NewEncoder(file)
	jsonWriter.SetIndent("", "	")

	if err = jsonWriter.Encode(defaultFileStructure); err != nil {
		return fmt.Errorf("writing sample config: %w", err)
	}

	return nil
}