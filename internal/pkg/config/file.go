package config

import (
	"encoding/json"
	"errors"
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

var errDuplicateBroadcast = errors.New("duplicate broadcast target across apps")

type fileStructure struct {
	SleepDurationBetweenFeedParsing string `json:"sleepDurationBetweenFeedParsing,omitempty"`
	SleepDurationBetweenBroadcasts  string `json:"sleepDurationBetweenBroadcasts,omitempty"`

	StorageFilePath string `json:"storageFilePath,omitempty"`

	MetricsAddr string `json:"metricsAddr,omitempty"`

	Elements []fileStructureElement `json:"apps"`

	Scoring *fileStructureScoring `json:"scoring,omitempty"`

	// Used for backwards compatibility reasons
	// Deprecated: will be removed in v2

	LegacyBroadcastType       string `json:"broadcastType,omitempty"`
	LegacyTelegramBotAPIToken string `json:"telegramBotAPIToken,omitempty"`
	LegacyTelegramChatID      string `json:"telegramChatID,omitempty"`

	LegacySources []fileStructureSource `json:"sources,omitempty"`
}

type fileStructureScoring struct {
	Enabled   bool     `json:"enabled"`
	Provider  string   `json:"provider,omitempty"` // "embedding" (default) or "keyword"
	Interests []string `json:"interests"`          // Topics to score stories against
	ModelName string   `json:"modelName,omitempty"`
	ModelDir  string   `json:"modelDir,omitempty"`
	MinScore  float64  `json:"minScore,omitempty"` // Stories scoring below this (0-1) are dropped

	// DisableArticleSummaries turns off the default behavior of attaching an
	// extractive summary to stories whose feed entry has no description.
	DisableArticleSummaries bool `json:"disableArticleSummaries,omitempty"`
}

type fileStructureElement struct {
	BroadcastType       string `json:"broadcastType,omitempty"`
	TelegramBotAPIToken string `json:"telegramBotAPIToken,omitempty"`
	TelegramChatID      string `json:"telegramChatID,omitempty"`
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
	URL                  string   `json:"url"`
	IgnoreStoriesBefore  string   `json:"ignoreStoriesBefore,omitempty"`
	MustIncludeAnyOf     []string `json:"mustIncludeAnyOf,omitempty"`
	MustExcludeAnyOf     []string `json:"mustExcludeAnyOf,omitempty"`
	MatchKeywordsAsWords bool     `json:"matchKeywordsAsWords,omitempty"`
	StatusPage           bool     `json:"statusPage,omitempty"`
	Interests            []string `json:"interests,omitempty"`
	MinScore             *float64 `json:"minScore,omitempty"`
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

// durationOr returns fallback when value is empty or "0", the parsed duration
// otherwise. An invalid value warns and falls back rather than failing the load.
func durationOr(value string, fallback time.Duration, log *logger.Log) time.Duration {
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.WarnErr("invalid duration in config, using default", err)

		return fallback
	}

	if duration <= 0 {
		return fallback // zero or negative would busy-loop the runner
	}

	return duration
}

// parseIgnoreBefore resolves a source's cutoff. An empty value uses the default
// window; otherwise it accepts an RFC3339 timestamp or a duration ("24h") before
// now. A non-empty but unparseable value warns and falls back to the default.
func parseIgnoreBefore(value string, log *logger.Log) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC().Add(-defaultIgnoreStoriesBefore)
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return timestamp
	}

	duration, err := time.ParseDuration(value)
	if err == nil {
		return time.Now().UTC().Add(-duration)
	}

	log.WarnErr("invalid ignoreStoriesBefore, using default window",
		fmt.Errorf("value %q is neither an RFC3339 time nor a duration", value)) //nolint:err113 // contextual

	return time.Now().UTC().Add(-defaultIgnoreStoriesBefore)
}

// resolveStorageFilePath picks the storage path from the config, then the CLI
// flag, then the env var, then the default location.
func resolveStorageFilePath(configured, cliFlag string) string {
	if configured != "" {
		return configured
	}

	if cliFlag != "" {
		return cliFlag
	}

	if env := os.Getenv(storageFilePathEnvironmentVariable); env != "" {
		return env
	}

	return os.ExpandEnv(storageFileDefaultLocation)
}

func (f *fileStructure) toConfig(storageFilePath string, log *logger.Log) (*Config, error) {
	var (
		config Config
		err    error
	)

	config.SleepDurationBetweenFeedParsing = durationOr(f.SleepDurationBetweenFeedParsing, defaultFeedParsingInterval, log)
	config.SleepDurationBetweenBroadcasts = durationOr(f.SleepDurationBetweenBroadcasts, defaultBroadcastInterval, log)

	config.Store = storage.New()
	config.MetricsAddr = f.MetricsAddr
	config.StorageFilePath = resolveStorageFilePath(f.StorageFilePath, storageFilePath)

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

	seenBroadcasts := make(map[string]bool, len(f.Elements))

	for idx := range f.Elements {
		var elementConfig App

		elementConfig, err = f.Elements[idx].prepareConfigElement(log)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config element: %w", err)
		}

		// Apps share dedup/cleanup state per broadcast target, so two apps with
		// the same target would corrupt each other's seen-story tracking.
		name := elementConfig.Broadcast.Name()
		if seenBroadcasts[name] {
			return nil, fmt.Errorf("%w: %q (merge their sources into one app)", errDuplicateBroadcast, name)
		}

		seenBroadcasts[name] = true

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
			Enabled:           f.Scoring.Enabled,
			Provider:          f.Scoring.Provider,
			Interests:         f.Scoring.Interests,
			ModelName:         f.Scoring.ModelName,
			ModelDir:          f.Scoring.ModelDir,
			MinScore:          f.Scoring.MinScore,
			SummarizeArticles: !f.Scoring.DisableArticleSummaries,
		}
	}

	return &config, nil
}

// createSampleFile writes a minimal starter config that relies on defaults: feed
// parsing every 5m, a 24h story window, stdout output, scoring off. Optional
// fields are omitted (see config.sample.json / README for the full surface).
func createSampleFile(filePath string) error {
	sources := []fileStructureSource{
		{
			URL:                  sampleFeedURL,
			IgnoreStoriesBefore:  "",
			MustIncludeAnyOf:     []string{"linux", "golang"},
			MustExcludeAnyOf:     nil,
			MatchKeywordsAsWords: false,
			MinScore:             nil,
			StatusPage:           false,
			Interests:            nil,
		},
	}

	sample := leanFileStructure(sources)

	return writeConfigFile(filePath, &sample)
}

// leanFileStructure wraps sources in a single stdout app and leaves every
// optional field at its default, producing a minimal config file.
func leanFileStructure(sources []fileStructureSource) fileStructure {
	return fileStructure{
		SleepDurationBetweenFeedParsing: "",
		SleepDurationBetweenBroadcasts:  "",
		StorageFilePath:                 "",
		MetricsAddr:                     "",
		Elements:                        []fileStructureElement{stdoutElement(sources)},
		Scoring:                         nil,
		LegacyBroadcastType:             "",
		LegacyTelegramBotAPIToken:       "",
		LegacyTelegramChatID:            "",
		LegacySources:                   nil,
	}
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
			URL:                  fe.Sources[sourceIdx].URL,
			IgnoreStoriesBefore:  parseIgnoreBefore(fe.Sources[sourceIdx].IgnoreStoriesBefore, log),
			MustIncludeKeywords:  fe.Sources[sourceIdx].MustIncludeAnyOf,
			MustExcludeKeywords:  fe.Sources[sourceIdx].MustExcludeAnyOf,
			MatchKeywordsAsWords: fe.Sources[sourceIdx].MatchKeywordsAsWords,
			StatusPage:           fe.Sources[sourceIdx].StatusPage,
			Interests:            fe.Sources[sourceIdx].Interests,
			MinScore:             fe.Sources[sourceIdx].MinScore,
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
