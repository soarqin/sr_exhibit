# sr_exhibit - Speedrun.com Static Page Generator

## Project Overview

sr_exhibit is a command-line tool written in Go that fetches speedrun leaderboard data from speedrun.com API and generates beautiful static HTML pages for display.

## Core Features

### 1. Game and Category Search
- Supports game abbreviation search (e.g., `sms` → Super Mario Sunshine)
- Supports full game name search
- Interactive category selection

### 2. Subcategory Support
- Automatically detects game subcategory variables
- Three ways to specify variables:
  - Command line: `--variables "var1=value1,var2=value2"`
  - Config file: `variables` field in `config.yaml`
  - Interactive selection: Program lists options for user to choose
- Only shows subcategories relevant to current category (filtered by `variable.Category` field)

### 3. Player Name Styles
- Supports speedrun.com API's name-style feature
- Supports gradient style and solid style
- Uses dark mode colors adapted for dark backgrounds

### 4. Video Link Validation
- Only displays valid video links
- Supported platforms: YouTube, Twitch, Niconico, Dailymotion, Vimeo, Bilibili

### 5. Time Formatting
- Whole seconds don't show `.00`: `14:35` instead of `14:35.00`
- Milliseconds displayed correctly: `14:35.50`

### 6. Caching System
- **Leaderboard CSV Cache**: Saves basic leaderboard data, easy to edit manually
- **Player JSON Cache**: Saves detailed player data (including name styles)
- Three cache modes:
  - Auto mode: Detects cache and prompts user
  - Force use: `--use-cache`
  - Force refresh: `--refresh-cache`
- Cache management commands:
  - `--cache-list`: List all cached leaderboards
  - `--cache-clear`: Clear all leaderboard cache

## Project Structure

```
sr_exhibit/
├── main.go              # Program entry, command line argument handling
├── models/
│   └── types.go         # Data model definitions
├── api/
│   ├── client.go        # API client
│   └── selector.go      # Interactive selector
├── cache/
│   ├── cache.go         # Player JSON cache
│   └── leaderboard.go   # Leaderboard CSV cache
├── generator/
│   ├── html.go          # HTML generator and template functions
│   └── leaderboard.html # HTML template
├── templates/
│   ├── minimal.html     # Minimal style template
│   └── leaderboard.html # Default template
├── config.yaml          # Config file example
├── .cache/              # Cache directory
│   ├── players.json     # Player data cache
│   └── *.csv            # Leaderboard cache files
└── output/              # Generated HTML output directory
```

## Core Data Models

### Variable (Subcategory Variable)
```go
type Variable struct {
    ID            string
    Name          string
    Category      string           // Parent category, empty means global
    IsSubcategory bool
    Values        VariableValues
}
```

### NameStyle (Player Name Style)
```go
type NameStyle struct {
    Style     string          // "gradient" or "solid"
    ColorFrom *NameStyleColor // Light/dark mode colors
    ColorTo   *NameStyleColor // Gradient end color
}
```

### PlayerData (Player Data)
```go
type PlayerData struct {
    Rel       string
    ID        string
    Name      string
    NameStyle *NameStyle  // Name style
    Names     struct {
        International string
    }
}
```

## API Usage

### Search Game
```
GET /api/v1/games?name={name}
GET /api/v1/games/{abbreviation}  # Direct access via abbreviation
```

### Get Variables
```
GET /api/v1/games/{game_id}/variables
```

### Get Leaderboard (with Variable Filter)
```
GET /api/v1/leaderboards/{game_id}/category/{category_id}?var-{var_id}={value_id}
```

### Get User Info
```
GET /api/v1/users/{user_id}
```

## Usage Examples

### Basic Usage
```bash
# Specify game and category
sr_exhibit --game "sms" --category "Any%"

# Specify game only, interactively select category
sr_exhibit --game "sms"

# Use abbreviation
sr_exhibit --game "sm64" --category "16 Star"
```

### Specify Subcategories
```bash
# Command line argument
sr_exhibit --game "sms" --category "Any%" --variables "9dq73k2q=mln1yv32"

# Config file
sr_exhibit --config config.yaml
```

### Config File Example (config.yaml)
```yaml
game: "sms"
category: "Any%"
output: "./output/index.html"
variables:
  version: "GCN"
api:
  baseURL: "https://www.speedrun.com/api/v1"
  timeout: "30s"
```

### Cache Management
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

## Caching System

### Leaderboard CSV Cache
Leaderboard data is saved in `.cache/{game_id}_{category_id}_{variables}.csv`:

```csv
#META,VERSION,1
#GAME,o1y9j9v6,Celeste
#CATEGORY,7kjpl1gk,Any%
#CACHED_AT,2026-02-08T15:27:40+08:00
#VARIABLE,e8m7em86,9qj7z0oq
rank,player_id,player_name,time_seconds,date,submit_url,run_id,video_links
1,8rpk9dgj,secureaccount,1491.04,2026-02-02,,mr5p4e2y,https://www.youtube.com/watch?v=0fT1lHHQ0xs
```

**CSV Format Notes**:
- Only saves basic data, easy to edit manually
- Does NOT save player name styles (styles are read from JSON cache)
- `time_seconds`: Floating-point seconds
- `video_links`: Multiple links separated by `|`

### Player JSON Cache
Detailed player data is saved in `.cache/players.json`:

```json
{
  "players": {
    "8rpk9dgj": {
      "id": "8rpk9dgj",
      "name": "secureaccount",
      "names": {
        "international": "secureaccount"
      },
      "nameStyle": {
        "style": "solid",
        "color": {
          "light": "#ff6b6b",
          "dark": "#ff6b6b"
        }
      }
    }
  }
}
```

**Cache Loading Logic**:
1. First check playerCache (JSON) for player data
2. If not found, fetch from API and save to cache
3. This keeps CSV simple while managing player styles separately

## Important Implementation Details

### Search Logic Priority
1. First try direct abbreviation/ID access via `/games/{name}`
2. If failed, use API search function `/games?name={name}`
3. Finally fetch all games for fuzzy matching

### Subcategory Filter Logic
```go
// Only select subcategories belonging to current category
if v.IsSubcategory && (v.Category == "" || v.Category == categoryID) {
    // v.Category == "" means global variable, applies to all categories
}
```

### Name Style Handling
```go
// Generate CSS style attribute
func GetNameStyleAttr(nameStyle *models.NameStyle) string {
    switch nameStyle.Style {
    case "gradient":
        // Use background-clip: text for text gradient
    case "solid":
        // Simple color setting
    }
}
```

### Time Formatting Logic
```go
// Parse ISO 8601 format PT16M25S
// Check if has fractional seconds
if millis > 0 {
    return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, millis)
}
return fmt.Sprintf("%d:%02d", minutes, seconds)
}
```

## Development Notes

### Documentation Rule
**IMPORTANT**: After completing any task, always update CLAUDE.md to reflect the changes made. This ensures documentation stays in sync with the codebase.

### Interactive Input Detection
```go
func isInteractive() bool {
    fileInfo, _ := os.Stdin.Stat()
    return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Only enable interactive selection when TTY is available
// This prevents hanging in CI/CD environments
```

### Template Function Registration
```go
funcMap := template.FuncMap{
    "formatTime":     formatTimeISO,
    "nameStyleAttr":  GetNameStyleAttr,
    "styledName":     GetStyledPlayerName,
}
```

### HTML Template Variable Types
Note that `index $.Players $p.ID` returns a value type, not a pointer:
```go
func GetStyledPlayerName(playerData models.PlayerData) StyledPlayerName
// Not *models.PlayerData
```

## Recent Updates

### 2026-02-08
- **Documentation**: Added README.md with program description, usage, and build instructions
- **Licensing**: Added LICENSE file (MIT License)
- **HTML Minification**: Output HTML (including embedded CSS and JavaScript) is minified using tdewolff/minify library
- **Cache System Improvement**: CSV only saves basic data, player styles read from JSON cache
- **Cache Modes**: Added --use-cache, --refresh-cache, --cache-list, --cache-clear parameters
- **Auto Config**: Default to reading config.yaml configuration file
- **Top 3 Icons**: Display trophy icons (fetched from speedrun.com API)
- **Minimal Template**: New minimal style template, supports OBS overlay display
- **Font Optimization**: JetBrains Mono for numbers, tabular-nums prevents jitter
- Fixed interactive subcategory selection showing all categories
- Improved time formatting (whole seconds don't show .00)
- **.gitignore**: Added `.cache/` directory to gitignore

### Original Features
- Variable data model support
- API variable filtering
- Video link validation
- Name style support
- Search optimization
