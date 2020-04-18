[![MyNews](https://snapcraft.io/mynews/badge.svg)](https://snapcraft.io/mynews) ![CI](https://github.com/lawzava/mynews/workflows/CI/badge.svg)

# MyNews

Personalized news feed parser & broadcast

Easily specify your RSS/Atom sources and broadcast preferences to get personalized news feed

## Usage

Sample call: `mynews -broadcastType=telegram -telegramBotToken=1ndsb3223j4234kasd -telegramChatID=@lawzava_news`

#### Parameters

```
  -broadcastType string
        broadcast type to use. Valid values are: 'telegram' (default "telegram")
  -interval uint
        interval in seconds between each feed parsing run (default 60)
  -sources string
        rss/atom source URLs separated by a comma (default "https://hnrss.org/newest.atom")
  -store string
        store type to use. Valid values are: 'memory' (persistent hash map), 'postgres', 'redis' (default "memory")
  -storeAccessDetails string
        store access URI if the type is not 'memory' (default "redis://localhost:6379")
  -telegramBotToken string
        telegram bot token to use with 'telegram' broadcast type
  -telegramChatID string
        telegram chatID to use with 'telegram' broadcast type

```


## TODO:

#### v1.2
- Improve the config to use keyword filtering on titles  per source

#### v1.3
- Add broadcasting as JSON to custom API every n seconds or n entities


