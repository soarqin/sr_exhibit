package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/soar/sr_exhibit/cache"
	"github.com/soar/sr_exhibit/models"
)

const (
	DefaultBaseURL = "https://www.speedrun.com/api/v1"
	DefaultTimeout = 30 * time.Second
)

// Client represents the API client
type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	playerCache *cache.PlayerCache
	cacheOnce   sync.Once // Ensures cache is initialized only once
}

// NewClient creates a new API client
func NewClient(baseURL string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetPlayerCache sets the player cache
func (c *Client) SetPlayerCache(pc *cache.PlayerCache) {
	c.playerCache = pc
}

// GetPlayerCache returns the player cache
func (c *Client) GetPlayerCache() *cache.PlayerCache {
	return c.playerCache
}

// GetGames gets the game list
func (c *Client) GetGames(ctx context.Context, offset int) ([]models.Game, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/games", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add parameters
	q := req.URL.Query()
	q.Add("max", "200")
	if offset > 0 {
		q.Add("offset", fmt.Sprintf("%d", offset))
	}
	req.URL.RawQuery = q.Encode()

	var result models.GameSearchResult
	if err := c.doRequest(req, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetAllGames gets all games (handles pagination)
func (c *Client) GetAllGames(ctx context.Context) ([]models.Game, error) {
	var allGames []models.Game
	offset := 0

	for {
		games, err := c.GetGames(ctx, offset)
		if err != nil {
			return nil, err
		}

		allGames = append(allGames, games...)

		if len(games) < 200 {
			break
		}

		offset += len(games)
	}

	return allGames, nil
}

// SearchGameByName searches for a game by name
func (c *Client) SearchGameByName(ctx context.Context, name string) (*models.Game, error) {
	// First try direct abbreviation access (speedrun.com supports /games/{abbreviation})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/games/"+url.PathEscape(name), nil)
	if err == nil {
		var result struct {
			Data models.Game `json:"data"`
		}
		// If direct access succeeds (HTTP 200 or redirect), it's a valid abbreviation/ID
		if err := c.doRequest(req, &result); err == nil {
			return &result.Data, nil
		}
	}

	// If direct access fails, try using API search function
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/games", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("name", name)
	q.Add("max", "1")
	req.URL.RawQuery = q.Encode()

	var result models.GameSearchResult
	if err := c.doRequest(req, &result); err != nil {
		return nil, err
	}

	// If exact search finds result, return it
	if len(result.Data) > 0 {
		return &result.Data[0], nil
	}

	// Otherwise get all games for fuzzy matching
	games, err := c.GetAllGames(ctx)
	if err != nil {
		return nil, err
	}

	// First try exact abbreviation match
	for _, game := range games {
		if strings.EqualFold(game.Abbreviation, name) {
			return &game, nil
		}
	}

	// Then try fuzzy name match
	for _, game := range games {
		if strings.Contains(strings.ToLower(game.Names.International), strings.ToLower(name)) {
			return &game, nil
		}
	}

	return nil, fmt.Errorf("game not found: %s", name)
}

// GetCategories gets game categories
func (c *Client) GetCategories(ctx context.Context, gameID string) ([]models.Category, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/games/"+url.PathEscape(gameID)+"/categories", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result models.APIResponse[models.Category]
	if err := c.doRequest(req, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetCategoryByName gets a category by name
func (c *Client) GetCategoryByName(ctx context.Context, gameID string, categoryName string) (*models.Category, error) {
	categories, err := c.GetCategories(ctx, gameID)
	if err != nil {
		return nil, err
	}

	for _, cat := range categories {
		if strings.EqualFold(cat.Name, categoryName) {
			return &cat, nil
		}
	}

	return nil, fmt.Errorf("category not found: %s", categoryName)
}

// GetLeaderboard gets leaderboard data
func (c *Client) GetLeaderboard(ctx context.Context, gameID, categoryID string, varFilters map[string]string) (*models.LeaderboardData, error) {
	reqURL := fmt.Sprintf("%s/leaderboards/%s/category/%s",
		c.BaseURL, url.PathEscape(gameID), url.PathEscape(categoryID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add parameters to get more data
	q := req.URL.Query()
	q.Add("top", "100")      // Get top 100
	q.Add("players", "true") // Get player data

	// Add variable filter parameters
	for varID, varValue := range varFilters {
		q.Add("var-"+varID, varValue)
	}

	req.URL.RawQuery = q.Encode()

	var result models.LeaderboardResponse
	if err := c.doRequest(req, &result); err != nil {
		return nil, err
	}

	// If API didn't return player data, we need to fetch it manually
	if len(result.Data.Players) == 0 {
		result.Data.Players = make(map[string]models.PlayerData)

		// Collect all unique player IDs
		playerIDs := make(map[string]bool)
		for _, run := range result.Data.Runs {
			for _, p := range run.Run.Players {
				if p.Rel == "user" {
					playerIDs[p.ID] = true
				}
			}
		}

		// Use goroutines to fetch player info concurrently, but limit concurrency
		// Prioritize cache, only fetch players not in or expired in cache
		type playerResult struct {
			id   string
			data *models.PlayerData
		}

		semaphore := make(chan struct{}, 10) // Limit to 10 concurrent
		results := make(chan playerResult, len(playerIDs))

		// Player IDs that need to be fetched from API
		var idsToFetch []string

		if c.playerCache != nil {
			// First try to get from cache
			for playerID := range playerIDs {
				if data, found := c.playerCache.Get(playerID); found {
					// Cache hit
					result.Data.Players[playerID] = *data
				} else {
					// Cache miss, need to fetch from API
					idsToFetch = append(idsToFetch, playerID)
				}
			}
		} else {
			// No cache, all players need to be fetched
			for playerID := range playerIDs {
				idsToFetch = append(idsToFetch, playerID)
			}
		}

		// Concurrently fetch player info not in cache
		for _, playerID := range idsToFetch {
			go func(id string) {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				data, err := c.GetUser(ctx, id)
				if err == nil {
					results <- playerResult{id: id, data: data}
					// Save to cache
					if c.playerCache != nil {
						c.playerCache.Set(id, *data)
					}
				} else {
					results <- playerResult{id: id, data: nil}
				}
			}(playerID)
		}

		// Collect results
		for i := 0; i < len(idsToFetch); i++ {
			r := <-results
			if r.data != nil {
				result.Data.Players[r.id] = *r.data
			}
		}

		// Save cache to file
		if c.playerCache != nil {
			if err := c.playerCache.Save(); err != nil {
				// Cache save failure shouldn't affect main flow
				fmt.Fprintf(os.Stderr, "Warning: Failed to save cache: %v\n", err)
			}
		}
	}

	return &result.Data, nil
}

// GetVariables gets game variables (subcategories)
func (c *Client) GetVariables(ctx context.Context, gameID string) ([]models.Variable, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/games/"+url.PathEscape(gameID)+"/variables", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result struct {
		Data []models.Variable `json:"data"`
	}

	if err := c.doRequest(req, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetUser gets user info
func (c *Client) GetUser(ctx context.Context, userID string) (*models.PlayerData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/users/"+url.PathEscape(userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result struct {
		Data models.PlayerData `json:"data"`
	}

	if err := c.doRequest(req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// doRequest executes HTTP request
func (c *Client) doRequest(req *http.Request, result any) error {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sr_exhibit/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned error status code %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// GetRunDetails gets run details
func (c *Client) GetRunDetails(ctx context.Context, runID string) (*models.RunData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.BaseURL+"/runs/"+url.PathEscape(runID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result struct {
		Data models.RunData `json:"data"`
	}

	if err := c.doRequest(req, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
