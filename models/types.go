package models

// APIResponse is a generic API response wrapper
type APIResponse[T any] struct {
	Data []T `json:"data"`
}

// GameNames supports multiple game name languages
type GameNames struct {
	International string `json:"international"`
	Japanese      string `json:"japanese,omitempty"`
}

// GameAssets represents game assets
type GameAssets struct {
	Icon       Asset `json:"icon"`
	Cover      Asset `json:"cover"`
	Logo       Asset `json:"logo,omitempty"`
	Background Asset `json:"background,omitempty"`
	Trophy1st  Asset `json:"trophy-1st,omitempty"` // 1st place trophy icon
	Trophy2nd  Asset `json:"trophy-2nd,omitempty"` // 2nd place trophy icon
	Trophy3rd  Asset `json:"trophy-3rd,omitempty"` // 3rd place trophy icon
	Trophy4th  Asset `json:"trophy-4th,omitempty"` // 4th place trophy icon (usually empty)
}

// Asset represents a single asset
type Asset struct {
	URI    string `json:"uri"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// Game represents game information
type Game struct {
	ID           string     `json:"id"`
	Names        GameNames  `json:"names"`
	Abbreviation string     `json:"abbreviation"`
	WebLink      string     `json:"weblink"`
	ReleaseDate  string     `json:"release-date"`
	Assets       GameAssets `json:"assets"`
}

// Category represents a game category
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Leaderboard represents a leaderboard
type Leaderboard struct {
	Game     string     `json:"game"`
	Category string     `json:"category"`
	Weblink  string     `json:"weblink"`
	Runs     []RunEntry `json:"runs"`
}

// RunEntry represents a leaderboard entry
type RunEntry struct {
	Place int     `json:"place"`
	Run   RunData `json:"run"`
}

// RunData represents run data
type RunData struct {
	ID        string            `json:"id"`
	Players   []Player          `json:"players"`
	Times     RunTimes          `json:"times"`
	Videos    *RunVideos        `json:"videos"`
	Comment   string            `json:"comment"`
	Date      string            `json:"date"`
	SubmitURL string            `json:"submit"`
	Values    map[string]string `json:"values"` // Subcategory variable values
}

// Player represents player information
type Player struct {
	Rel  string `json:"rel"` // "user" or "guest"
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// RunTimes represents time data
type RunTimes struct {
	Primary   string  `json:"primary"`
	PrimaryT  float64 `json:"primary_t"`
	Realtime  *string `json:"realtime,omitempty"`
	RealtimeT *float64 `json:"realtime_t,omitempty"`
	GameTime  *string `json:"gametime,omitempty"`
	GameTimeT *float64 `json:"gametime_t,omitempty"`
}

// RunVideos represents video links
type RunVideos struct {
	Links []VideoLink `json:"links"`
	Text  string      `json:"text,omitempty"`
}

// VideoLink represents a single video link
type VideoLink struct {
	URI string `json:"uri"`
}

// GameSearchResult represents game search results
type GameSearchResult struct {
	Data []Game `json:"data"`
}

// LeaderboardResponse represents leaderboard API response
type LeaderboardResponse struct {
	Data LeaderboardData `json:"data"`
}

// LeaderboardData represents leaderboard data
type LeaderboardData struct {
	Game      string                `json:"game"`      // Game ID
	Category  string                `json:"category"`  // Category ID
	Weblink   string                `json:"weblink"`
	Runs      []RunEntry            `json:"runs"`
	Players   map[string]PlayerData `json:"players"`
}

// PlayerData represents detailed player data
type PlayerData struct {
	Rel       string     `json:"rel"`
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name,omitempty"`
	NameStyle *NameStyle `json:"name-style,omitempty"`
	Names     struct {
		International string `json:"international"`
	} `json:"names,omitempty"`
}

// NameStyle represents player name style
type NameStyle struct {
	Style     string          `json:"style"`      // Style type: "solid", "gradient", etc.
	ColorFrom *NameStyleColor `json:"color-from"` // Start color
	ColorTo   *NameStyleColor `json:"color-to"`   // End color (for gradient)
}

// NameStyleColor represents name style color
type NameStyleColor struct {
	Light string `json:"light"` // Light mode color
	Dark  string `json:"dark"`  // Dark mode color
}

// Config represents config file structure
type Config struct {
	Game      string            `yaml:"game"`
	Category  string            `yaml:"category"`
	Output    string            `yaml:"output"`
	Template  string            `yaml:"template"` // Custom template file path
	API       APIConfig         `yaml:"api"`
	Cache     CacheConfig       `yaml:"cache"`    // Cache configuration
	Variables map[string]string `yaml:"variables"` // Variable filters
}

// APIConfig represents API configuration
type APIConfig struct {
	BaseURL string `yaml:"baseURL"`
	Timeout string `yaml:"timeout"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled bool   `yaml:"enabled"` // Whether to enable cache, default true
	Dir     string `yaml:"dir"`     // Cache directory, default ".cache"
	TTL     string `yaml:"ttl"`     // Cache expiration time, default "720h" (30 days)
}

// Variable represents game variable (subcategory)
type Variable struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Category      string           `json:"category,omitempty"`
	Scope         VariableScope    `json:"scope"`
	Mandatory     bool             `json:"mandatory"`
	UserDefined   bool             `json:"user-defined"`
	Obsoletes     bool             `json:"obsoletes"`
	IsSubcategory bool             `json:"is-subcategory"`
	Values        VariableValues   `json:"values"`
}

// VariableScope represents variable scope
type VariableScope struct {
	Type string `json:"type"`
}

// VariableValues represents variable values
type VariableValues struct {
	Values  map[string]VariableValue `json:"values"`
	Default string                   `json:"default"`
}

// VariableValue represents a single variable value
type VariableValue struct {
	Label string        `json:"label"`
	Rules string        `json:"rules,omitempty"`
	Flags VariableFlags `json:"flags"`
}

// VariableFlags represents variable flags
type VariableFlags struct {
	Miscellaneous *bool `json:"miscellaneous"`
}
