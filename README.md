# sr_exhibit

A command-line tool written in Go that fetches speedrun leaderboard data from speedrun.com API and generates beautiful static HTML pages for display.

## Features

- Search games by name or abbreviation
- Interactive category and subcategory selection
- Custom HTML template support
- Leaderboard caching for faster subsequent generation
- Player name style support (gradient, solid colors)
- Trophy icons and video link validation
- HTML/CSS/JS minification for optimized output

## Installation

### Build from source

```bash
git clone https://github.com/soar/sr_exhibit.git
cd sr_exhibit
go build -o sr_exhibit
```

### Requirements

- Go 1.23 or later

## Usage

### Basic usage

```bash
# Specify game and category
sr_exhibit --game "sms" --category "Any%"

# Specify game only, interactively select category
sr_exhibit --game "sms"

# Use game abbreviation
sr_exhibit --game "sm64" --category "16 Star"
```

### Using config file

Create a `config.yaml` file:

```yaml
game: "sms"
category: "Any%"
output: "./output/index.html"
variables:
  version: "GCN"
api:
  baseURL: "https://www.speedrun.com/api/v1"
  timeout: "30s"
cache:
  enabled: true
  dir: ".cache"
  ttl: "720h"
```

Then run:

```bash
sr_exhibit --config config.yaml
```

### Command-line options

```
--game string          Game name or abbreviation
--category string      Category name
--output string        Output HTML file path (default "./output/index.html")
--template string      Custom HTML template file path
--variables string     Variable filters (format: "var1=value1,var2=value2")
--config string        Config file path (default "config.yaml")
--use-cache           Force use cached data
--refresh-cache       Force refresh cached data
--cache-list          List all cached leaderboards
--cache-clear         Clear all leaderboard cache
--help                Show help
```

### Cache management

```bash
# List cached leaderboards
sr_exhibit --cache-list

# Clear all cache
sr_exhibit --cache-clear

# Force use cache (skip API)
sr_exhibit --game "sm64" --category "16 Star" --use-cache

# Force refresh cache
sr_exhibit --game "sm64" --category "16 Star" --refresh-cache
```

## Examples

Generate a leaderboard for Super Mario Sunshine Any%:

```bash
sr_exhibit --game "sms" --category "Any%" --output "./output/sms_any.html"
```

Generate with custom template:

```bash
sr_exhibit --game "celeste" --category "Any%" --template "./templates/custom.html"
```

## Output

The program generates a self-contained HTML file that can be:
- Opened directly in a web browser
- Embedded in OBS for streaming overlays
- Hosted on any static web server

## License

MIT License - see [LICENSE](LICENSE) for details.
