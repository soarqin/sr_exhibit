package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/soar/sr_exhibit/api"
	"github.com/soar/sr_exhibit/cache"
	"github.com/soar/sr_exhibit/generator"
	"github.com/soar/sr_exhibit/models"
	"gopkg.in/yaml.v3"
)

const (
	version = "1.0.0"
)

func main() {
	// Define command line flags
	var (
		configFile    string
		gameName      string
		categoryName  string
		outputDir     string
		variablesStr  string
		templatePath  string
		showVersion   bool
		timeout       string
		useCache      bool   // Force use cache
		refreshCache  bool   // Force refresh cache
		showCacheList bool   // Show cache list
		clearCache    bool   // Clear cache
	)

	flag.StringVar(&configFile, "config", "", "Config file path (YAML)")
	flag.StringVar(&gameName, "game", "", "Game name or ID")
	flag.StringVar(&categoryName, "category", "", "Category name or ID")
	flag.StringVar(&outputDir, "output", "./output", "Output directory")
	flag.StringVar(&variablesStr, "variables", "", "Subcategory filter (format: var1=value1,var2=value2)")
	flag.StringVar(&templatePath, "template", "", "Custom template file path")
	flag.StringVar(&timeout, "timeout", "30s", "API request timeout")
	flag.BoolVar(&showVersion, "version", false, "Show version info")
	flag.BoolVar(&useCache, "use-cache", false, "Force use cached data")
	flag.BoolVar(&refreshCache, "refresh-cache", false, "Force refresh and update cache")
	flag.BoolVar(&showCacheList, "cache-list", false, "List all cached leaderboards")
	flag.BoolVar(&clearCache, "cache-clear", false, "Clear all leaderboard cache")
	flag.Parse()

	if showVersion {
		fmt.Printf("sr_exhibit v%s\n", version)
		os.Exit(0)
	}

	// Load config
	var config models.Config
	configFileToUse := configFile

	// Default to config.yaml if not specified
	if configFileToUse == "" {
		configFileToUse = "config.yaml"
	}

	// Try to read config file
	data, err := os.ReadFile(configFileToUse)
	if err == nil {
		// Config file exists, parse it
		if err := yaml.Unmarshal(data, &config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to parse config file: %v\n", err)
			os.Exit(1)
		}
		// Command line args override config file
		if gameName != "" {
			config.Game = gameName
		}
		if categoryName != "" {
			config.Category = categoryName
		}
		if outputDir != "./output" {
			config.Output = outputDir
		}
		if templatePath != "" {
			config.Template = templatePath
		}
		if timeout != "30s" {
			config.API.Timeout = timeout
		}
	} else {
		// Config file doesn't exist, use command line args or defaults
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Failed to read config file: %v\n", err)
			os.Exit(1)
		}
		if gameName == "" && !showCacheList && !clearCache {
			fmt.Fprintf(os.Stderr, "Error: Game name must be specified (use -game flag or config file)\n")
			fmt.Fprintf(os.Stderr, "Use -h to see help\n")
			os.Exit(1)
		}
		config.Game = gameName
		config.Category = categoryName
		config.Output = outputDir
		config.API.BaseURL = "https://www.speedrun.com/api/v1"
		config.API.Timeout = timeout
	}

	// Parse timeout duration
	duration, err := time.ParseDuration(config.API.Timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid timeout format: %v\n", err)
		os.Exit(1)
	}

	// Parse command line specified variables
	var varFilters map[string]string
	if variablesStr != "" {
		varFilters = make(map[string]string)
		pairs := strings.Split(variablesStr, ",")
		for _, pair := range pairs {
			kv := strings.Split(strings.TrimSpace(pair), "=")
			if len(kv) == 2 {
				varFilters[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	// Determine template path to use
	finalTemplatePath := templatePath
	if finalTemplatePath == "" && config.Template != "" {
		finalTemplatePath = config.Template
	}

	// Initialize leaderboard cache
	cacheDir := cache.DefaultCacheDir
	if config.Cache.Dir != "" {
		cacheDir = config.Cache.Dir
	}
	leaderboardCache := cache.NewLeaderboardCache(cacheDir)

	// Handle cache related commands
	if showCacheList {
		if err := listCaches(leaderboardCache); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if clearCache {
		if err := clearAllCaches(leaderboardCache); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Cleared all leaderboard cache")
		os.Exit(0)
	}

	// Execute generation
	if err := run(context.Background(), config, duration, varFilters, finalTemplatePath, leaderboardCache, useCache, refreshCache); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Page generated successfully!")
	fmt.Printf("Output: %s\n", config.Output)
}

func listCaches(lbCache *cache.LeaderboardCache) error {
	files, err := lbCache.List()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("No cached leaderboards")
		return nil
	}

	fmt.Printf("Found %d cache files:\n", len(files))
	for _, file := range files {
		// Parse filename to get info
		parts := strings.Split(strings.TrimSuffix(file, ".csv"), "_")
		if len(parts) >= 2 {
			fmt.Printf("  - Game:%s Category:%s\n", parts[0], parts[1])
			if len(parts) > 2 {
				fmt.Printf("    Variables: %s\n", strings.Join(parts[2:], ", "))
			}
		}
	}
	return nil
}

func clearAllCaches(lbCache *cache.LeaderboardCache) error {
	return lbCache.Clear()
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func confirm(prompt string) bool {
	for {
		answer := strings.ToLower(readLine(prompt + " (y/n): "))
		if answer == "y" || answer == "yes" {
			return true
		} else if answer == "n" || answer == "no" {
			return false
		}
		fmt.Println("Please enter y or n")
	}
}

// isInteractive checks if running in an interactive terminal environment
func isInteractive() bool {
	// Check if stdin is a terminal
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func run(ctx context.Context, config models.Config, timeout time.Duration, varFilters map[string]string, templatePath string, lbCache *cache.LeaderboardCache, useCache, refreshCache bool) error {
	client := api.NewClient(config.API.BaseURL, timeout)

	// Initialize player cache
	playerCache, err := cache.NewPlayerCache(cache.DefaultCacheDir, cache.DefaultTTL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize cache, caching disabled: %v\n", err)
	} else {
		client.SetPlayerCache(playerCache)
		if removed := playerCache.CleanExpired(); removed > 0 {
			fmt.Printf("Cache: Cleaned %d expired entries\n", removed)
		}
		if total, expired := playerCache.Stats(); total > 0 {
			fmt.Printf("Cache: %d entries (%d expired)\n", total, expired)
		}
	}

	fmt.Printf("Searching game: %s\n", config.Game)
	game, err := client.SearchGameByName(ctx, config.Game)
	if err != nil {
		return fmt.Errorf("failed to search game: %w", err)
	}
	fmt.Printf("  Found game: %s (ID: %s)\n", game.Names.International, game.ID)

	var category *models.Category
	if config.Category != "" {
		fmt.Printf("Getting category: %s\n", config.Category)
		cat, err := client.GetCategoryByName(ctx, game.ID, config.Category)
		if err != nil {
			return fmt.Errorf("failed to get category: %w", err)
		}
		category = cat
		fmt.Printf("  Found category: %s (ID: %s)\n", category.Name, category.ID)
	} else {
		fmt.Println("Getting game categories...")
		categories, err := client.GetCategories(ctx, game.ID)
		if err != nil {
			return fmt.Errorf("failed to get categories: %w", err)
		}
		fmt.Printf("  Found %d categories\n", len(categories))

		cat, err := api.SelectCategory(categories)
		if err != nil {
			return fmt.Errorf("failed to select category: %w", err)
		}
		category = cat
	}

	// Get game variables
	variables, err := client.GetVariables(ctx, game.ID)
	if err != nil {
		return fmt.Errorf("failed to get variables: %w", err)
	}

	// Check if there are subcategories
	hasSubcategories := false
	for _, v := range variables {
		if v.IsSubcategory && (v.Category == "" || v.Category == category.ID) {
			hasSubcategories = true
			break
		}
	}

	// Determine variable filters to use
	selectedVars := make(map[string]string)

	if len(varFilters) > 0 {
		selectedVars = varFilters
		fmt.Printf("Using command line specified variables: %v\n", selectedVars)
	} else if len(config.Variables) > 0 {
		selectedVars = config.Variables
		fmt.Printf("Using config file specified variables: %v\n", selectedVars)
	} else if hasSubcategories {
		if isInteractive() {
			fmt.Println("\nDetected subcategory options...")
			selectedVars = api.SelectSubcategories(variables, category.ID)
		}
	}

	if len(selectedVars) == 0 {
		for _, v := range variables {
			if v.IsSubcategory && (v.Category == "" || v.Category == category.ID) && v.Values.Default != "" {
				selectedVars[v.ID] = v.Values.Default
			}
		}
		if len(selectedVars) > 0 {
			fmt.Printf("Using default subcategory\n")
		}
	}

	// Create cache key
	cacheKey := &cache.CacheKey{
		GameID:       game.ID,
		GameName:     game.Names.International,
		CategoryID:   category.ID,
		CategoryName: category.Name,
		Variables:    selectedVars,
	}

	var leaderboard *models.LeaderboardData

	// Check if using cache
	if refreshCache {
		// Force refresh
		fmt.Println("Force refresh mode: Fetching latest data...")
		leaderboard, err = client.GetLeaderboard(ctx, game.ID, category.ID, selectedVars)
		if err != nil {
			return fmt.Errorf("failed to get leaderboard: %w", err)
		}
		// Save cache
		if err := saveToCache(lbCache, cacheKey, game, category, leaderboard, playerCache); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save cache: %v\n", err)
		} else {
			fmt.Println("✓ Cache updated")
		}
	} else if useCache {
		// Force use cache
		if !lbCache.Exists(cacheKey) {
			return fmt.Errorf("cache does not exist, please run once to create cache")
		}
		fmt.Println("Use cache mode: Loading cached data...")
		cachedData, err := lbCache.Load(cacheKey)
		if err != nil {
			return fmt.Errorf("failed to load cache: %w", err)
		}

		// Collect all player IDs that need to be fetched
		playerIDs := make(map[string]bool)
		for _, run := range cachedData.Runs {
			for _, p := range run.Run.Players {
				if p.Rel == "user" {
					playerIDs[p.ID] = true
				}
			}
		}

		// Fill player data: first from cache, then from API if cache miss
		cachedData.Players = make(map[string]models.PlayerData)

		for playerID := range playerIDs {
			if playerCache != nil {
				// Try to get from cache first
				if data, found := playerCache.Get(playerID); found {
					cachedData.Players[playerID] = *data
					continue
				}
			}

			// Cache miss, fetch from API
			fmt.Printf("  Fetching player data: %s\n", playerID)
			playerData, err := client.GetUser(ctx, playerID)
			if err == nil {
				cachedData.Players[playerID] = *playerData
				// Save to cache
				if playerCache != nil {
					playerCache.Set(playerID, *playerData)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Failed to fetch player %s: %v\n", playerID, err)
			}
		}

		// Save cache to file
		if playerCache != nil {
			if err := playerCache.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to save cache: %v\n", err)
			}
		}

		leaderboard = &models.LeaderboardData{
			Game:      cachedData.Game.ID,
			Category:  cachedData.Category.ID,
			Weblink:   "",
			Runs:      cachedData.Runs,
			Players:   cachedData.Players,
		}
		cacheTime, _ := lbCache.GetCacheTime(cacheKey)
		fmt.Printf("✓ Loaded cache (cache time: %s)\n", cacheTime.Format("2006-01-02 15:04:05"))
	} else {
		// Auto mode: check cache and prompt
		if lbCache.Exists(cacheKey) && isInteractive() {
			cacheTime, _ := lbCache.GetCacheTime(cacheKey)
			fmt.Printf("\nFound local cache (cache time: %s)\n", cacheTime.Format("2006-01-02 15:04:05"))
			if confirm("Use cached data?") {
				cachedData, err := lbCache.Load(cacheKey)
				if err != nil {
					return fmt.Errorf("failed to load cache: %w", err)
				}

				// Collect all player IDs that need to be fetched
				playerIDs := make(map[string]bool)
				for _, run := range cachedData.Runs {
					for _, p := range run.Run.Players {
						if p.Rel == "user" {
							playerIDs[p.ID] = true
						}
					}
				}

				// Fill player data: first from cache, then from API if cache miss
				cachedData.Players = make(map[string]models.PlayerData)

				for playerID := range playerIDs {
					if playerCache != nil {
						// Try to get from cache first
						if data, found := playerCache.Get(playerID); found {
							cachedData.Players[playerID] = *data
							continue
						}
					}

					// Cache miss, fetch from API
					fmt.Printf("  Fetching player data: %s\n", playerID)
					playerData, err := client.GetUser(ctx, playerID)
					if err == nil {
						cachedData.Players[playerID] = *playerData
						// Save to cache
						if playerCache != nil {
							playerCache.Set(playerID, *playerData)
						}
					} else {
						fmt.Fprintf(os.Stderr, "Warning: Failed to fetch player %s: %v\n", playerID, err)
					}
				}

				// Save cache to file
				if playerCache != nil {
					if err := playerCache.Save(); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to save cache: %v\n", err)
					}
				}

				leaderboard = &models.LeaderboardData{
					Game:      cachedData.Game.ID,
					Category:  cachedData.Category.ID,
					Weblink:   "",
					Runs:      cachedData.Runs,
					Players:   cachedData.Players,
				}
				fmt.Println("✓ Using cached data")
			} else {
				fmt.Println("Fetching latest data...")
				leaderboard, err = client.GetLeaderboard(ctx, game.ID, category.ID, selectedVars)
				if err != nil {
					return fmt.Errorf("failed to get leaderboard: %w", err)
				}
				// Save cache
				if err := saveToCache(lbCache, cacheKey, game, category, leaderboard, playerCache); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to save cache: %v\n", err)
				} else {
					fmt.Println("✓ Data cached")
				}
			}
		} else {
			// No cache or no stdin, fetch directly
			fmt.Println("Fetching leaderboard data...")
			leaderboard, err = client.GetLeaderboard(ctx, game.ID, category.ID, selectedVars)
			if err != nil {
				return fmt.Errorf("failed to get leaderboard: %w", err)
			}
			// Save cache
			if err := saveToCache(lbCache, cacheKey, game, category, leaderboard, playerCache); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to save cache: %v\n", err)
			} else {
				fmt.Println("✓ Data cached")
			}
		}
	}

	fmt.Printf("  Got %d records\n", len(leaderboard.Runs))

	fmt.Println("Generating page...")
	gen, err := generator.NewGenerator(templatePath)
	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}

	outputPath := config.Output
	if outputPath == "./output" {
		outputPath = "./output/index.html"
	}

	data := &generator.LeaderboardData{
		Game:         *game,
		Category:     *category,
		Leaderboard: *leaderboard,
		Players:      leaderboard.Players,
	}

	if err := gen.Generate(outputPath, data); err != nil {
		return fmt.Errorf("failed to generate page: %w", err)
	}

	return nil
}

func saveToCache(lbCache *cache.LeaderboardCache, key *cache.CacheKey, game *models.Game, category *models.Category, leaderboard *models.LeaderboardData, playerCache *cache.PlayerCache) error {
	cachedData := &cache.CachedLeaderboard{
		Key:       *key,
		CachedAt:  time.Now(),
		Game:      *game,
		Category:  *category,
		Runs:      leaderboard.Runs,
		Players:   make(map[string]models.PlayerData),
	}

	// Collect all player data
	for _, run := range leaderboard.Runs {
		for _, player := range run.Run.Players {
			if player.Rel == "user" {
				if pd, ok := leaderboard.Players[player.ID]; ok {
					cachedData.Players[player.ID] = pd
				}
			}
		}
	}

	return lbCache.Save(cachedData)
}
