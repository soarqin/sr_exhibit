package cache

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/soar/sr_exhibit/models"
)

// LeaderboardCache handles leaderboard caching
type LeaderboardCache struct {
	dir string
}

// NewLeaderboardCache creates a new leaderboard cache
func NewLeaderboardCache(dir string) *LeaderboardCache {
	if dir == "" {
		dir = DefaultCacheDir
	}
	return &LeaderboardCache{dir: dir}
}

// CacheKey uniquely identifies a leaderboard cache entry
type CacheKey struct {
	GameID       string
	GameName     string
	CategoryID   string
	CategoryName string
	Variables    map[string]string // Subcategory variables
}

// String returns the string representation of the cache key
func (k *CacheKey) String() string {
	var parts []string
	parts = append(parts, k.GameID, k.CategoryID)

	// Add variables (sorted by key for consistency)
	if len(k.Variables) > 0 {
		varVars := make([]string, 0, len(k.Variables))
		for key := range k.Variables {
			varVars = append(varVars, key)
		}
		// Simple bubble sort
		for i := 0; i < len(varVars); i++ {
			for j := i + 1; j < len(varVars); j++ {
				if varVars[i] > varVars[j] {
					varVars[i], varVars[j] = varVars[j], varVars[i]
				}
			}
		}
		for _, key := range varVars {
			parts = append(parts, key+"="+k.Variables[key])
		}
	}

	return strings.Join(parts, "_")
}

// FileName returns the cache file name
func (k *CacheKey) FileName() string {
	return k.String() + ".csv"
}

// CachedLeaderboard represents cached leaderboard data
type CachedLeaderboard struct {
	Key      CacheKey
	CachedAt time.Time
	Game     models.Game
	Category models.Category
	Runs     []models.RunEntry
	Players  map[string]models.PlayerData
}

// GetFileName returns the cache file path
func (c *LeaderboardCache) GetFileName(key *CacheKey) string {
	return filepath.Join(c.dir, key.FileName())
}

// Save saves the leaderboard to a CSV file
func (c *LeaderboardCache) Save(data *CachedLeaderboard) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	path := c.GetFileName(&data.Key)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write metadata header
	writer.Write([]string{"#META", "VERSION", "1"})
	writer.Write([]string{"#GAME", data.Key.GameID, data.Key.GameName})
	writer.Write([]string{"#CATEGORY", data.Key.CategoryID, data.Key.CategoryName})
	writer.Write([]string{"#CACHED_AT", data.CachedAt.Format(time.RFC3339)})

	// Write variables
	for key, value := range data.Key.Variables {
		writer.Write([]string{"#VARIABLE", key, value})
	}

	// Write header
	writer.Write([]string{
		"rank", "player_id", "player_name", "time_seconds",
		"date", "submit_url", "run_id", "video_links",
	})

	// Write each record
	for _, run := range data.Runs {
		for _, player := range run.Run.Players {
			var playerName string
			var playerID string

			if player.Rel == "user" {
				if pd, ok := data.Players[player.ID]; ok {
					playerName = pd.Names.International
					if playerName == "" {
						playerName = pd.Name
					}
				}
				playerID = player.ID
			} else {
				playerName = player.Name
				playerID = ""
			}

			// Video links
			var videoLinks []string
			if run.Run.Videos != nil {
				for _, link := range run.Run.Videos.Links {
					videoLinks = append(videoLinks, link.URI)
				}
			}

			writer.Write([]string{
				fmt.Sprintf("%d", run.Place),
				playerID,
				playerName,
				fmt.Sprintf("%.2f", run.Run.Times.PrimaryT), // Seconds
				run.Run.Date,
				run.Run.SubmitURL,
				run.Run.ID,
				strings.Join(videoLinks, "|"),
			})
			break // Only write first player (multiplayer games may need special handling)
		}
	}

	return nil
}

// Load loads the leaderboard from a CSV file
func (c *LeaderboardCache) Load(key *CacheKey) (*CachedLeaderboard, error) {
	path := c.GetFileName(key)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, return nil instead of error
		}
		return nil, fmt.Errorf("failed to open cache file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Allow different number of fields per line
	reader.FieldsPerRecord = -1

	result := &CachedLeaderboard{
		Key:     *key,
		Players: make(map[string]models.PlayerData),
		Game:    models.Game{ID: key.GameID, Names: models.GameNames{International: key.GameName}},
		Category: models.Category{ID: key.CategoryID, Name: key.CategoryName},
		Runs:    make([]models.RunEntry, 0),
	}

	// Read and parse
	lineCount := 0
	dataLineCount := 0
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Fprintf(os.Stderr, "CSV read error at line %d: %v\n", lineCount, err)
			}
			break
		}

		lineCount++
		if len(record) == 0 {
			continue
		}

		// Handle metadata rows
		if strings.HasPrefix(record[0], "#") {
			switch record[0] {
			case "#META":
				// Version info
			case "#GAME":
				result.Game.ID = record[1]
				if len(record) > 2 {
					result.Game.Names.International = record[2]
				}
			case "#CATEGORY":
				result.Category.ID = record[1]
				if len(record) > 2 {
					result.Category.Name = record[2]
				}
			case "#CACHED_AT":
				result.CachedAt, _ = time.Parse(time.RFC3339, record[1])
			case "#VARIABLE":
				if result.Key.Variables == nil {
					result.Key.Variables = make(map[string]string)
				}
				if len(record) > 2 {
					result.Key.Variables[record[1]] = record[2]
				}
			}
			continue
		}

		// Skip header
		if record[0] == "rank" {
			continue
		}

		// Parse data row
		dataLineCount++
		if len(record) >= 8 {
			var place int
			var primaryT float64
			fmt.Sscanf(record[0], "%d", &place)
			fmt.Sscanf(record[3], "%f", &primaryT) // time_seconds is at index 3

			// Convert seconds to ISO 8601 format
			totalSeconds := int(primaryT)
			hours := totalSeconds / 3600
			minutes := (totalSeconds % 3600) / 60
			seconds := totalSeconds % 60
			millis := int((primaryT - float64(totalSeconds)) * 100)

			var primary string
			if hours > 0 {
				if millis > 0 {
					primary = fmt.Sprintf("PT%dH%dM%d.%dS", hours, minutes, seconds, millis)
				} else {
					primary = fmt.Sprintf("PT%dH%dM%dS", hours, minutes, seconds)
				}
			} else {
				if millis > 0 {
					primary = fmt.Sprintf("PT%dM%d.%dS", minutes, seconds, millis)
				} else {
					primary = fmt.Sprintf("PT%dM%dS", minutes, seconds)
				}
			}

			// Parse video links
			var videoLinks []models.VideoLink
			if len(record) > 7 && record[7] != "" {
				links := strings.Split(record[7], "|")
				for _, link := range links {
					videoLinks = append(videoLinks, models.VideoLink{URI: link})
				}
			}

			// Collect player info - only save basic info, detailed data from player cache
			var players []models.Player
			if record[1] != "" {
				players = []models.Player{
					{Rel: "user", ID: record[1]},
				}
			} else {
				players = []models.Player{
					{Rel: "guest", Name: record[2]},
				}
			}

			run := models.RunEntry{
				Place: place,
				Run: models.RunData{
					ID:      record[6], // run_id is at index 6
					Players: players,
					Times: models.RunTimes{
						Primary:  primary, // Converted ISO 8601 format
						PrimaryT: primaryT,
					},
					Date:      record[4], // date is at index 4
					SubmitURL: record[5], // submit_url is at index 5
				},
			}

			// Add video links if any
			if len(videoLinks) > 0 {
				run.Run.Videos = &models.RunVideos{
					Links: videoLinks,
				}
			}

			result.Runs = append(result.Runs, run)
		}
	}

	_ = lineCount      // For debugging
	_ = dataLineCount  // For debugging

	return result, nil
}

// Exists checks if the cache exists
func (c *LeaderboardCache) Exists(key *CacheKey) bool {
	path := c.GetFileName(key)
	_, err := os.Stat(path)
	return err == nil
}

// GetCacheTime returns the cache modification time
func (c *LeaderboardCache) GetCacheTime(key *CacheKey) (time.Time, error) {
	path := c.GetFileName(key)
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// List lists all cache files
func (c *LeaderboardCache) List() ([]string, error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".csv") {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// Delete deletes a cache file
func (c *LeaderboardCache) Delete(key *CacheKey) error {
	path := c.GetFileName(key)
	return os.Remove(path)
}

// Clear clears all leaderboard cache
func (c *LeaderboardCache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".csv") {
			path := filepath.Join(c.dir, entry.Name())
			os.Remove(path)
		}
	}
	return nil
}
