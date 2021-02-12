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

	StorageFilePath string `json:"storageFilePath"`

	Elements []fileStructureElement `json:"elements"`

	// Used for backwards compatibility reasons
	// Deprecated: will be removed in v2

	LegacyBroadcastType       string `json:"broadcastType"`
	LegacyTelegramBotAPIToken string `json:"telegramBotAPIToken"`
	LegacyTelegramChatID      string `json:"telegramChatID"`

	LegacySources []fileStructureSource `json:"sources"`
}

type fileStructureElement struct {
	BroadcastType       string `json:"broadcastType"`
	TelegramBotAPIToken string `json:"telegramBotAPIToken"`
	TelegramChatID      string `json:"telegramChatID"`

	Sources []fileStructureSource `json:"sources"`
}

type fileStructureSource struct {
	URL                 string   `json:"url"`
	IgnoreStoriesBefore string   `json:"ignoreStoriesBefore"`
	MustIncludeAnyOf    []string `json:"mustIncludeAnyOf"`
	MustExcludeAnyOf    []string `json:"mustExcludeAnyOf"`
	StatusPage          bool     `json:"statusPage"`
}

func fromFile(configFilePath, storageFilePath string, log *logger.Log) (*Config, error) {
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		log.Warn(fmt.Sprintf("File '%s' does not exist", configFilePath))

		return nil, nil
	}

	configFile, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}

	defer func() { _ = configFile.Close() }()

	var file fileStructure

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&file); err != nil {
		return nil, fmt.Errorf("decoding config file (legacy): %w", err)
	}

	return file.toConfig(storageFilePath, log)
}

func (f *fileStructure) toConfig(storageFilePath string, log *logger.Log) (*Config, error) {
	var (
		config Config
		err    error
	)

	config.SleepDurationBetweenBroadcasts, err = time.ParseDuration(f.SleepDurationBetweenBroadcasts)
	if err != nil {
		return nil, fmt.Errorf("invalid broadcast sleep duration format: %w", err)
	}

	config.SleepDurationBetweenFeedParsing, err = time.ParseDuration(f.SleepDurationBetweenFeedParsing)
	if err != nil {
		return nil, fmt.Errorf("invalid feed parsing sleep duration format: %w", err)
	}

	config.Store = storage.New()
	config.StorageFilePath = f.StorageFilePath

	if config.StorageFilePath == "" {
		config.StorageFilePath = storageFilePath
	}

	if config.StorageFilePath == "" {
		config.StorageFilePath = storageFileDefaultLocation

		if e := os.Getenv(storageFilePathEnvironmentVariable); e != "" {
			config.StorageFilePath = e
		}
	}

	if config.SleepDurationBetweenBroadcasts == 0 {
		config.SleepDurationBetweenBroadcasts = defaultSleepDuration
	}

	if len(f.Elements) == 0 {
		f.Elements = append(f.Elements, fileStructureElement{
			BroadcastType:       f.LegacyBroadcastType,
			TelegramBotAPIToken: f.LegacyTelegramBotAPIToken,
			TelegramChatID:      f.LegacyTelegramChatID,
			Sources:             f.LegacySources,
		})
	}

	for _, fe := range f.Elements {
		var elementConfig App

		elementConfig, err = fe.prepareConfigElement(log)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config element: %w", err)
		}

		config.Apps = append(config.Apps, elementConfig)
	}

	if len(config.Apps) == 0 {
		return &config, nil
	}

	if err = config.Store.RecoverFromFile(config.StorageFilePath, log, config.Apps[0].Broadcast.Name()); err != nil {
		return nil, fmt.Errorf("failed to recover data from file: %w", err)
	}

	return &config, nil
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
			IgnoreStoriesBefore: time.Date(2020, 4, 20, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			MustIncludeAnyOf:    []string{"linux", "golang", "musk"},
			MustExcludeAnyOf:    []string{"windows", "trump", "apple"},
		},
		{
			URL:                 "https://hnrss.org/newest.atom",
			IgnoreStoriesBefore: time.Hour.String(),
		},
	}

	defaultFileStructure := fileStructure{
		// nolint:gomnd // allow fore defaults
		SleepDurationBetweenFeedParsing: (time.Minute * 5).String(),
		// nolint:gomnd // allow fore defaults
		SleepDurationBetweenBroadcasts: (time.Second * 10).String(),

		Elements: []fileStructureElement{
			{
				BroadcastType: "stdout",
				Sources:       sources,
			},
		},
	}

	jsonWriter := json.NewEncoder(file)
	jsonWriter.SetIndent("", "	")

	if err = jsonWriter.Encode(defaultFileStructure); err != nil {
		return fmt.Errorf("writing sample config: %w", err)
	}

	return nil
}

func (fe fileStructureElement) prepareConfigElement(log *logger.Log) (App, error) {
	var (
		cfg App
		err error
	)

	for _, source := range fe.Sources {
		s := Source{
			URL:                 source.URL,
			MustExcludeKeywords: source.MustExcludeAnyOf,
			MustIncludeKeywords: source.MustIncludeAnyOf,
		}

		s.IgnoreStoriesBefore, err = time.Parse(time.RFC3339, source.IgnoreStoriesBefore)
		if err != nil {
			log.WarnErr("failed to parse time from IgnoreStoriesBefore parameter", err)

			var dur time.Duration

			dur, err = time.ParseDuration(source.IgnoreStoriesBefore)
			if err != nil {
				log.WarnErr("failed to parse duration from IgnoreStoriesBefore parameter", err)
			}

			s.IgnoreStoriesBefore = time.Now().UTC().Add(-dur)
		}

		cfg.Sources = append(cfg.Sources, &s)
	}

	var broadcastConfig broadcast.Config
	broadcastConfig.Telegram.BotAPIToken = fe.TelegramBotAPIToken
	broadcastConfig.Telegram.ChatID = fe.TelegramChatID

	cfg.Broadcast, err = parseBroadcast(fe.BroadcastType, broadcastConfig)
	if err != nil {
		return cfg, fmt.Errorf("parsing storage: %w", err)
	}

	return cfg, nil
}
