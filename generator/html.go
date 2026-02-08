package generator

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/soar/sr_exhibit/models"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
)

//go:embed *.html
var templateFS embed.FS

// LeaderboardData represents template data structure
type LeaderboardData struct {
	Game           models.Game
	Category       models.Category
	Leaderboard    models.LeaderboardData
	Players        map[string]models.PlayerData
	CountryCodeMap map[string]string // Country code replacement rules
}

// Generator represents the HTML generator
type Generator struct {
	templates      *template.Template
	m              *minify.M
	countryCodeMap map[string]string // Country code replacement rules
}

// NewGenerator creates a new generator
// templatePath: use embedded template if empty, otherwise load external template from specified path
// countryCodeMap: country code replacement rules, e.g. {"xk": "rs"} for Kosovo -> Serbia
func NewGenerator(templatePath string, countryCodeMap map[string]string) (*Generator, error) {
	// Create template and register custom functions
	// Include flagURL with closure over countryCodeMap
	funcMap := template.FuncMap{
		"formatTime":    formatTimeISO,
		"nameStyleAttr": GetNameStyleAttr,
		"styledName":    GetStyledPlayerName,
		"first":         firstN,
		"add":           add,
		"sub":           sub,
		"flagURL": func(code string) string {
			return CountryFlagURLWithMap(code, countryCodeMap)
		},
	}

	var tmpl *template.Template
	var err error

	if templatePath != "" {
		// Load template from external file
		content, err := os.ReadFile(templatePath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("template file not found: %s", templatePath)
			}
			if os.IsPermission(err) {
				return nil, fmt.Errorf("no permission to read template file: %s", templatePath)
			}
			return nil, fmt.Errorf("failed to read template file: %w", err)
		}

		tmpl, err = template.New("leaderboard.html").Funcs(funcMap).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse external template: %w\nHint: Please check if template syntax is correct", err)
		}
	} else {
		// Use embedded template
		tmpl, err = template.New("leaderboard.html").Funcs(funcMap).ParseFS(templateFS, "leaderboard.html")
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedded template: %w", err)
		}
	}

	// Initialize minifier
	m := minify.New()
	m.Add("text/html", &html.Minifier{
		KeepDefaultAttrVals: true,
		KeepDocumentTags:   true,
		KeepWhitespace:     false,
	})
	m.Add("text/css", &css.Minifier{})
	m.Add("text/javascript", &js.Minifier{
		KeepVarNames: false,
		Version:      2022,
	})

	return &Generator{
		templates:      tmpl,
		m:              m,
		countryCodeMap: countryCodeMap,
	}, nil
}

// formatTimeISO formats ISO 8601 duration string (e.g., "PT16M25S") to readable format
func formatTimeISO(isoTime string) string {
	// ISO 8601 format: PT16M25S means 16 minutes 25 seconds
	if len(isoTime) == 0 || isoTime == "PT0S" {
		return "0:00"
	}

	// Parse PT...S format
	remaining := isoTime
	if strings.HasPrefix(remaining, "PT") {
		remaining = remaining[2:]
	}
	if strings.HasSuffix(remaining, "S") {
		remaining = remaining[:len(remaining)-1]
	}

	var hours, minutes, seconds int

	// Parse hours
	if hIdx := strings.Index(remaining, "H"); hIdx > 0 {
		hours, _ = strconv.Atoi(remaining[:hIdx])
		remaining = remaining[hIdx+1:]
	}

	// Parse minutes
	if mIdx := strings.Index(remaining, "M"); mIdx > 0 {
		minutes, _ = strconv.Atoi(remaining[:mIdx])
		remaining = remaining[mIdx+1:]
	}

	// Parse seconds (may include decimals)
	secondsFloat, _ := strconv.ParseFloat(remaining, 64)
	seconds = int(secondsFloat)
	millis := int((secondsFloat - float64(seconds)) * 100)

	if hours > 0 {
		if millis > 0 {
			return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, millis)
		}
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	if millis > 0 {
		return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, millis)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// Generate generates static HTML page
func (g *Generator) Generate(outputPath string, data *LeaderboardData) error {
	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Set CountryCodeMap for template access
	data.CountryCodeMap = g.countryCodeMap

	// Render template to buffer first
	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "leaderboard.html", data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Minify HTML and write directly to file
	if g.m != nil {
		if err := g.m.Minify("text/html", file, &buf); err != nil {
			return fmt.Errorf("failed to minify HTML: %w", err)
		}
	} else {
		// No minify, write directly
		if _, err := file.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	}

	return nil
}

// FormatTime formats time display
func FormatTime(primaryT float64) string {
	hours := int(primaryT / 3600)
	remaining := primaryT - float64(hours*3600)
	minutes := int(remaining / 60)
	seconds := int(remaining - float64(minutes*60))
	millis := int((primaryT - float64(int(primaryT))) * 100)

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, millis)
	}
	return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, millis)
}

// GetPlayerNames gets player name list
func GetPlayerNames(run models.RunData, players map[string]models.PlayerData) []string {
	var names []string
	for _, p := range run.Players {
		if p.Rel == "user" {
			if playerData, ok := players[p.ID]; ok {
				names = append(names, playerData.Names.International)
			} else {
				names = append(names, "Unknown")
			}
		} else {
			names = append(names, p.Name)
		}
	}
	return names
}

// ValidateVideoURI validates if video URI is valid
func ValidateVideoURI(uri string) bool {
	if uri == "" {
		return false
	}

	// Basic URL format check
	if !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") {
		return false
	}

	// Check if contains known video platforms
	knownPlatforms := []string{
		"youtube.com", "youtu.be",
		"twitch.tv", "twitch.com",
		"nicovideo.jp", "nico.ms",
		"dailymotion.com", "dai.ly",
		"vimeo.com",
		"bilibili.com",
	}

	uriLower := strings.ToLower(uri)
	for _, platform := range knownPlatforms {
		if strings.Contains(uriLower, platform) {
			return true
		}
	}

	// Accept if not a known platform but format is correct
	return strings.Contains(uri, ".")
}

// GetValidVideoURI gets the first valid video link
func GetValidVideoURI(run models.RunData) string {
	if run.Videos != nil && len(run.Videos.Links) > 0 {
		for _, link := range run.Videos.Links {
			if ValidateVideoURI(link.URI) {
				return link.URI
			}
		}
	}
	return ""
}

// GetVideoURI gets video link
func GetVideoURI(run models.RunData) string {
	return GetValidVideoURI(run)
}

// GetNameStyleAttr generates HTML style attribute based on name style
func GetNameStyleAttr(nameStyle *models.NameStyle) string {
	if nameStyle == nil {
		return ""
	}

	var style strings.Builder

	switch nameStyle.Style {
	case "gradient":
		// Gradient style
		fromColor := "#ffffff"
		toColor := "#888888"
		if nameStyle.ColorFrom != nil && nameStyle.ColorFrom.Dark != "" {
			fromColor = nameStyle.ColorFrom.Dark
		}
		if nameStyle.ColorTo != nil && nameStyle.ColorTo.Dark != "" {
			toColor = nameStyle.ColorTo.Dark
		}
		style.WriteString(fmt.Sprintf("background: linear-gradient(90deg, %s, %s); ", fromColor, toColor))
		style.WriteString("-webkit-background-clip: text; ")
		style.WriteString("-webkit-text-fill-color: transparent; ")
		style.WriteString("background-clip: text; ")

	case "solid":
		// Solid style
		color := "#ffffff"
		if nameStyle.ColorFrom != nil && nameStyle.ColorFrom.Dark != "" {
			color = nameStyle.ColorFrom.Dark
		}
		style.WriteString(fmt.Sprintf("color: %s; ", color))
	}

	return style.String()
}

// StyledPlayerName represents styled player name
type StyledPlayerName struct {
	Name  string
	Style string
}

// GetStyledPlayerName gets styled player name structure
func GetStyledPlayerName(playerData models.PlayerData) StyledPlayerName {
	name := playerData.Names.International
	if name == "" {
		name = playerData.Name
	}
	if name == "" {
		name = "Unknown"
	}

	style := GetNameStyleAttr(playerData.NameStyle)
	return StyledPlayerName{Name: name, Style: style}
}

// firstN returns the first n elements of a slice
func firstN(v interface{}, n int) interface{} {
	switch val := v.(type) {
	case []models.RunEntry:
		if n > len(val) {
			n = len(val)
		}
		return val[:n]
	default:
		return v
	}
}

// add returns the sum of two integers
func add(a, b int) int {
	return a + b
}

// sub returns the difference of two integers
func sub(a, b int) int {
	return a - b
}

// CountryFlagURL returns the speedrun.com official flag image URL
// e.g., "US" -> "https://www.speedrun.com/images/flags/us.png"
func CountryFlagURL(code string) string {
	return CountryFlagURLWithMap(code, nil)
}

// CountryFlagURLWithMap returns the speedrun.com official flag image URL with country code replacement
// countryCodeMap: optional map for replacing country codes, e.g. {"xk": "rs"} for Kosovo -> Serbia
// e.g., "US" -> "https://www.speedrun.com/images/flags/us.png"
// If code is in replacement map, uses the replacement code instead
func CountryFlagURLWithMap(code string, countryCodeMap map[string]string) string {
	if len(code) < 2 {
		return ""
	}

	// Convert to lowercase for lookup and URL
	codeLower := strings.ToLower(code)

	// Check if there's a replacement rule
	if countryCodeMap != nil {
		if replacement, ok := countryCodeMap[codeLower]; ok {
			codeLower = strings.ToLower(replacement)
		}
	}

	return fmt.Sprintf("https://www.speedrun.com/images/flags/%s.png", codeLower)
}
