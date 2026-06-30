package config

import (
	"encoding/json"
	"fmt"
	"mynews/internal/pkg/broadcast"
	"mynews/internal/pkg/logger"
	"mynews/internal/pkg/storage"
	"os"
	"strings"
	"time"
)

const (
	sampleFeedURL       = "https://hnrss.org/newest.atom"
	stdoutBroadcastType = "stdout"
)

type fileStructure struct {
	SleepDurationBetweenFeedParsing string `json:"sleepDurationBetweenFeedParsing"`
	SleepDurationBetweenBroadcasts  string `json:"sleepDurationBetweenBroadcasts"`

	StorageFilePath string `json:"storageFilePath"`

	MetricsAddr string `json:"metricsAddr,omitempty"`

	Elements []fileStructureElement `json:"apps"`

	Scoring *fileStructureScoring `json:"scoring,omitempty"`

	// Used for backwards compatibility reasons
	// Deprecated: will be removed in v2

	LegacyBroadcastType       string `json:"broadcastType"`
	LegacyTelegramBotAPIToken string `json:"telegramBotAPIToken"`
	LegacyTelegramChatID      string `json:"telegramChatID"`

	LegacySources []fileStructureSource `json:"sources"`
}

type fileStructureScoring struct {
	Enabled   bool     `json:"enabled"`
	Provider  string   `json:"provider"`  // "embedding" or "keyword"
	Interests []string `json:"interests"` // Topics to score stories against
	ModelName string   `json:"modelName,omitempty"`
	ModelDir  string   `json:"modelDir,omitempty"`
	MinScore  float64  `json:"minScore,omitempty"` // Stories scoring below this (0-1) are dropped
}

type fileStructureElement struct {
	BroadcastType       string `json:"broadcastType"`
	TelegramBotAPIToken string `json:"telegramBotAPIToken"`
	TelegramChatID      string `json:"telegramChatID"`
	DiscordWebhookURL   string `json:"discordWebhookURL,omitempty"`
	SlackWebhookURL     string `json:"slackWebhookURL,omitempty"`
	WebhookURL          string `json:"webhookURL,omitempty"`

	Digest *fileStructureDigest `json:"digest,omitempty"`

	Sources []fileStructureSource `json:"sources"`
}

type fileStructureDigest struct {
	Every string `json:"every"`
	TopN  int    `json:"topN"`
}

type fileStructureSource struct {
	URL                 string   `json:"url"`
	IgnoreStoriesBefore string   `json:"ignoreStoriesBefore"`
	MustIncludeAnyOf    []string `json:"mustIncludeAnyOf"`
	MustExcludeAnyOf    []string `json:"mustExcludeAnyOf"`
	StatusPage          bool     `json:"statusPage"`
	Interests           []string `json:"interests,omitempty"`
}

func fromFile(configFilePath, storageFilePath string, log *logger.Log) (*Config, error) {
	_, err := os.Stat(configFilePath) //nolint:gosec // path is the user-provided CLI config location
	if os.IsNotExist(err) {
		log.Warn(fmt.Sprintf("File '%s' does not exist", configFilePath))

		return nil, fmt.Errorf("file '%s' does not exist: %w", configFilePath, err)
	}

	configFile, err := os.Open(configFilePath) //nolint:gosec // path is the user-provided CLI config location
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}

	defer func() { _ = configFile.Close() }()

	var file fileStructure

	jsonParser := json.NewDecoder(configFile)

	err = jsonParser.Decode(&file)
	if err != nil {
		return nil, fmt.Errorf("decoding config file (legacy): %w", err)
	}

	return file.toConfig(storageFilePath, log)
}

//nolint:cyclop,funlen // allow higher complexity on config setup for now
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
	config.MetricsAddr = f.MetricsAddr

	if config.StorageFilePath == "" {
		config.StorageFilePath = storageFilePath
	}

	if config.StorageFilePath == "" {
		config.StorageFilePath = os.ExpandEnv(storageFileDefaultLocation)

		if e := os.Getenv(storageFilePathEnvironmentVariable); e != "" {
			config.StorageFilePath = e
		}
	}

	if config.SleepDurationBetweenBroadcasts == 0 {
		config.SleepDurationBetweenBroadcasts = defaultSleepDuration
	}

	if config.SleepDurationBetweenFeedParsing == 0 {
		config.SleepDurationBetweenFeedParsing = defaultSleepDuration
	}

	if len(f.Elements) == 0 {
		f.Elements = append(f.Elements, fileStructureElement{
			BroadcastType:       f.LegacyBroadcastType,
			TelegramBotAPIToken: f.LegacyTelegramBotAPIToken,
			TelegramChatID:      f.LegacyTelegramChatID,
			DiscordWebhookURL:   "",
			SlackWebhookURL:     "",
			WebhookURL:          "",
			Digest:              nil,
			Sources:             f.LegacySources,
		})
	}

	for idx := range f.Elements {
		var elementConfig App

		elementConfig, err = f.Elements[idx].prepareConfigElement(log)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config element: %w", err)
		}

		config.Apps = append(config.Apps, elementConfig)
	}

	if len(config.Apps) == 0 {
		return &config, nil
	}

	err = config.Store.RecoverFromFile(config.StorageFilePath, log, config.Apps[0].Broadcast.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to recover data from file: %w", err)
	}

	// Parse scoring config
	if f.Scoring != nil && f.Scoring.Enabled {
		config.Scoring = &ScoringConfig{
			Enabled:   f.Scoring.Enabled,
			Provider:  f.Scoring.Provider,
			Interests: f.Scoring.Interests,
			ModelName: f.Scoring.ModelName,
			ModelDir:  f.Scoring.ModelDir,
			MinScore:  f.Scoring.MinScore,
		}
	}

	return &config, nil
}

func createSampleFile(filePath string) error {
	sources := []fileStructureSource{
		{
			URL:                 sampleFeedURL,
			IgnoreStoriesBefore: time.Date(2020, 4, 20, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			MustIncludeAnyOf:    []string{"linux", "golang", "musk"},
			MustExcludeAnyOf:    []string{"windows", "trump", "apple"},
			StatusPage:          false,
			Interests:           nil,
		},
		{
			URL:                 sampleFeedURL,
			IgnoreStoriesBefore: time.Hour.String(),
			MustIncludeAnyOf:    nil,
			MustExcludeAnyOf:    nil,
			StatusPage:          false,
			Interests:           nil,
		},
	}

	defaultFileStructure := fileStructure{
		//nolint:mnd // allow fore defaults
		SleepDurationBetweenFeedParsing: (time.Minute * 5).String(),
		//nolint:mnd // allow fore defaults
		SleepDurationBetweenBroadcasts: (time.Second * 10).String(),
		StorageFilePath:                "",
		MetricsAddr:                    "",
		Elements:                       []fileStructureElement{stdoutElement(sources)},
		Scoring: &fileStructureScoring{
			Enabled:  false,
			Provider: "embedding",
			Interests: []string{
				"artificial intelligence and machine learning",
				"geopolitical conflicts and international relations",
				"stock market and financial technology",
			},
			ModelName: "",
			ModelDir:  "",
			MinScore:  0,
		},
		LegacyBroadcastType:       "",
		LegacyTelegramBotAPIToken: "",
		LegacyTelegramChatID:      "",
		LegacySources:             nil,
	}

	return writeConfigFile(filePath, &defaultFileStructure)
}

// stdoutElement builds a stdout broadcast element with the given sources.
func stdoutElement(sources []fileStructureSource) fileStructureElement {
	return fileStructureElement{
		BroadcastType:       stdoutBroadcastType,
		Sources:             sources,
		TelegramBotAPIToken: "",
		TelegramChatID:      "",
		DiscordWebhookURL:   "",
		SlackWebhookURL:     "",
		WebhookURL:          "",
		Digest:              nil,
	}
}

// writeConfigFile writes fileStruct as indented JSON to filePath, refusing to
// clobber an existing file.
func writeConfigFile(filePath string, fileStruct *fileStructure) error {
	_, err := os.Stat(filePath) //nolint:gosec // path is the user-provided CLI config location
	if err == nil {
		return nil // config already exists; do not clobber it
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("checking config file: %w", err)
	}

	file, err := os.Create(filePath) //nolint:gosec // path is the user-provided CLI config location
	if err != nil {
		return fmt.Errorf("initializing config file: %w", err)
	}

	defer func() { _ = file.Close() }()

	jsonWriter := json.NewEncoder(file)
	jsonWriter.SetIndent("", "	")

	err = jsonWriter.Encode(fileStruct)
	if err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func (fe *fileStructureElement) prepareConfigElement(log *logger.Log) (App, error) {
	var (
		cfg App
		err error
	)

	cfg.Sources = make([]*Source, len(fe.Sources))

	for sourceIdx := range fe.Sources {
		cfg.Sources[sourceIdx] = &Source{
			URL:                 fe.Sources[sourceIdx].URL,
			IgnoreStoriesBefore: time.Time{},
			MustIncludeKeywords: fe.Sources[sourceIdx].MustIncludeAnyOf,
			MustExcludeKeywords: fe.Sources[sourceIdx].MustExcludeAnyOf,
			StatusPage:          fe.Sources[sourceIdx].StatusPage,
			Interests:           fe.Sources[sourceIdx].Interests,
		}

		cfg.Sources[sourceIdx].IgnoreStoriesBefore, err = time.Parse(time.RFC3339, fe.Sources[sourceIdx].IgnoreStoriesBefore)
		if err != nil {
			dur, errDur := time.ParseDuration(fe.Sources[sourceIdx].IgnoreStoriesBefore)
			if errDur != nil {
				log.WarnErr("failed to parse time from IgnoreStoriesBefore parameter", err)
				log.WarnErr("failed to parse duration from IgnoreStoriesBefore parameter", errDur)
			}

			cfg.Sources[sourceIdx].IgnoreStoriesBefore = time.Now().UTC().Add(-dur)
		}
	}

	cfg.Broadcast, err = fe.broadcastClient()
	if err != nil {
		return App{}, fmt.Errorf("failed to create broadcast client: %w", err)
	}

	cfg.Digest, err = fe.digestConfig()
	if err != nil {
		return App{}, err
	}

	return cfg, nil
}

// digestConfig parses the optional digest settings for an element. A nil result
// means immediate (non-digest) broadcasting.
func (fe *fileStructureElement) digestConfig() (*DigestConfig, error) {
	if fe.Digest == nil {
		return nil, nil //nolint:nilnil // nil config legitimately means "no digest"
	}

	every, err := time.ParseDuration(fe.Digest.Every)
	if err != nil {
		return nil, fmt.Errorf("invalid digest interval %q: %w", fe.Digest.Every, err)
	}

	if every <= 0 || fe.Digest.TopN <= 0 {
		return nil, nil //nolint:nilnil // incomplete digest config disables it
	}

	return &DigestConfig{Every: every, TopN: fe.Digest.TopN}, nil
}

// broadcastClient builds the broadcast target for an element. The type is matched
// case-insensitively; an unknown or empty type falls back to stdout.
//
//nolint:ireturn // factory deliberately returns the Broadcast impl selected by type
func (fe *fileStructureElement) broadcastClient() (broadcast.Broadcast, error) {
	var (
		client broadcast.Broadcast
		err    error
	)

	switch strings.ToLower(fe.BroadcastType) {
	case "telegram":
		client, err = broadcast.NewTelegramClient(fe.TelegramBotAPIToken, fe.TelegramChatID)
	case "discord":
		client, err = broadcast.NewDiscordClient(fe.DiscordWebhookURL)
	case "slack":
		client, err = broadcast.NewSlackClient(fe.SlackWebhookURL)
	case "webhook":
		client, err = broadcast.NewWebhookClient(fe.WebhookURL)
	default:
		return broadcast.NewStdOutClient(), nil
	}

	if err != nil {
		return nil, fmt.Errorf("creating %q broadcast client: %w", fe.BroadcastType, err)
	}

	return client, nil
}
