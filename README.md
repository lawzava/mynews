[![MyNews](https://snapcraft.io/mynews/badge.svg)](https://snapcraft.io/mynews) ![CI](https://github.com/lawzava/mynews/workflows/CI/badge.svg)

# MyNews

Personalized news feed parser & broadcast.

Easily specify your RSS/Atom sources and broadcast preferences to get personalized news feed.

The binary is a single self-contained, pure-Go executable (no CGO). The optional embedding
scorer downloads a small static-embedding model from HuggingFace on first use; the keyword
scorer and everything else runs fully offline.

## Features

- **Sources**: RSS and Atom feeds, with per-source keyword include/exclude filters,
  optional whole-word matching, score thresholds, and a date cutoff. Feeds are fetched
  concurrently, so one slow feed never stalls the others.
- **Broadcast targets**: `stdout`, `telegram`, `discord`, `slack`, and a generic `webhook`
  (which receives the raw story JSON). Configure one or more independent "apps".
- **Relevance scoring** (optional): score story titles against your interests with local
  static embeddings (`embedding`, via [model2vec](https://github.com/MinishLab/model2vec))
  or simple `keyword` matching. Set `minScore` to drop low-relevance stories, and override
  interests per source. With embedding scoring, stories whose feed entry has no description
  get a best-effort extractive summary by default (disable with `disableArticleSummaries`).
- **Digest mode**: batch the top-N highest-scoring stories per interval instead of sending
  each one as it arrives.
- **Cross-source dedup**: the same article shared under different URLs (tracking params,
  fragments) collapses to a single story.
- **OPML import**: `mynews -import-opml feeds.opml` generates a config from a feed-reader export.
- **Health & metrics**: set `metricsAddr` (e.g. `":8080"`) to expose `/healthz` and `/metrics`.
- **Graceful shutdown** with periodic, crash-safe persistence of seen-story state.

## Installation

For `snap` users a snap package is available through `snap install mynews`.

Otherwise, pre-built binaries that are available in `releases` are recommended.

To build from source make sure you have `go` installed and run `go install`.

## Usage

Executing bare `mynews` will use in-memory DB and will print to stdout by default.

For full list of available options, see: `mynews -help`

Configuration is minimal — most fields are optional with sensible defaults (feeds parsed
every 5m, a 24h story window, stdout output). A complete config can be as small as:

```json
{ "apps": [ { "sources": [ { "url": "https://hnrss.org/newest.atom" } ] } ] }
```

Generate a starting config with `mynews -create`, or import an existing OPML export with
`mynews -import-opml feeds.opml`. See [`config.sample.json`](config.sample.json) for a
worked example covering every option (broadcast targets, scoring, per-source interests, digest).

Working examples: 

- Tech News [https://t.me/lawzava_news_tech](https://t.me/lawzava_news_tech)
- Design News [https://t.me/lawzava_news_design](https://t.me/lawzava_news_design)

## Contributions and issues

I will be actively maintaining and improving this repository until it is stated otherwise in this section. 

Feel free to create issues (questions) / PRs as you see fit for now. There are no hard rules.
