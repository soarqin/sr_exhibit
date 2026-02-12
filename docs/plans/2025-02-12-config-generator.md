# Config Generator Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `--generate` flag that prompts the user for game/category/subcategory and generates a config.yaml file from the template.

**Architecture:** Add a new flag to main.go. When set, skip normal execution and instead prompt the user for configuration values, then read config.yaml.template, replace placeholders with user values, and write to config.yaml.

**Tech Stack:** Go 1.21+, YAML parsing (gopkg.in/yaml.v3), text/template package

---

## Task 1: Add --generate flag and constant

**Files:**
- Modify: `main.go:25-52` (flag variables section)

**Step 1: Add generateConfig constant**

```go
const (
	version = "1.0.0"
	configTemplateFile = "config.yaml.template"
)
```

**Step 2: Add generateConfig flag variable**

Add to flag variables (after line 37):
```go
	generateConfig bool   // Generate config file
```

**Step 3: Add flag definition**

Add after line 51:
```go
	flag.BoolVar(&generateConfig, "generate", false, "Generate config.yaml from template")
```

**Step 4: Verify build**

Run: `go build`
Expected: No errors

**Step 5: Commit**

```bash
git add main.go
git commit -m "feat: add --generate flag for config generation"
```

---

## Task 2: Add promptConfig function

**Files:**
- Modify: `main.go` (add after confirm function)

**Step 1: Write promptConfig function**

```go
// promptConfig prompts the user for configuration values
func promptConfig(ngame, ncategory, nsubcategory string) (game, category, subcategory string) {
	if ngame == "" {
		ngame = readLine("Enter game name or abbreviation: ")
	}
	if ncategory == "" {
		ncategory = readLine("Enter category name: ")
	}
	if nsubcategory == "" {
		ncategory = readLine("Enter subcategory value (optional, press Enter to skip): ")
	}
	return ngame, ncategory, nsubcategory
}
```

**Step 2: Verify build**

Run: `go build`
Expected: No errors

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add promptConfig helper function"
```

---

## Task 3: Add template placeholder constants

**Files:**
- Modify: `main.go` (after constants)

**Step 1: Add template placeholder constants**

```go
const (
	version              = "1.0.0"
	configTemplateFile   = "config.yaml.template"
	templateGame       = "{{.Game}}"
	templateCategory    = "{{.Category}}"
	templateSubcategory = "{{.Subcategory}}"
)
```

**Step 2: Verify build**

Run: `go build`
Expected: No errors

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add template placeholder constants"
```

---

## Task 4: Add generateConfigFile function

**Files:**
- Modify: `main.go` (add after promptConfig)

**Step 1: Write generateConfigFile function**

```go
// generateConfigFile generates a config.yaml from template and user input
func generateConfigFile(game, category, subcategory string) error {
	// Read template file
	templateContent, err := os.ReadFile(configTemplateFile)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Replace placeholders
	result := string(templateContent)
	result = strings.ReplaceAll(result, templateGame, game)
	result = strings.ReplaceAll(result, templateCategory, category)
	if subcategory != "" {
		// Remove comment and enable subcategory field
		result = strings.ReplaceAll(result, "#subcategory: \"GCN\"", "subcategory: \""+subcategory+"\"")
	} else {
		// Comment out subcategory field
		result = strings.ReplaceAll(result, "subcategory: \"GCN\"", "#subcategory: \"\"")
	}

	// Write to config.yaml
	if err := os.WriteFile("config.yaml", []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println("✓ Generated config.yaml")
	return nil
}
```

**Step 2: Verify build**

Run: `go build`
Expected: No errors

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add generateConfigFile function"
```

---

## Task 5: Wire up --generate in main

**Files:**
- Modify: `main.go:54-57` (after showVersion check)

**Step 1: Add generate mode check**

```go
	// Generate config mode
	if generateConfig {
		game, category, subcategory := promptConfig(gameName, categoryName, subcategoryStr)
		if err := generateConfigFile(game, category, subcategory); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
```

Insert after showVersion check (after line 57).

**Step 2: Verify build**

Run: `go build`
Expected: No errors

**Step 3: Test generate mode**

Run: `sr_exhibit --generate`
Expected: Prompts for input, then creates config.yaml

**Step 4: Test generate with partial args**

Run: `sr_exhibit --generate --game sms`
Expected: Prompts for category/subcategory only

**Step 5: Test generate with all args**

Run: `sr_exhibit --generate --game sms --category "Any%" --subcategory "GCN"`
Expected: No prompts, creates config.yaml directly

**Step 6: Commit**

```bash
git add main.go
git commit -m "feat: wire up --generate flag in main function"
```

---

## Task 6: Add to .gitignore

**Files:**
- Modify: `.gitignore`

**Step 1: Add config.yaml to gitignore**

```
# Ignore generated config files
config.yaml
```

**Step 2: Verify gitignore works**

Run: `git status`
Expected: config.yaml not shown as untracked

**Step 3: Commit**

```bash
git add .gitignore
git commit -m "chore: add config.yaml to gitignore"
```

---

## Task 7: Update documentation

**Files:**
- Modify: `config.yaml.template`

**Step 1: Add generate usage note**

Add to top of template:
```yaml
# sr_exhibit Configuration Template
#
# Generate config: sr_exhibit --generate
# Or: sr_exhibit --generate --game <game> --category <category> --subcategory <value>
```

**Step 2: Commit**

```bash
git add config.yaml.template
git commit -m "docs: add --generate usage to template"
```

---

## Verification

After completing all tasks:

**Test 1: Generate with no args**
```bash
sr_exhibit --generate
```
Expected: Prompts for game, category, subcategory, creates config.yaml

**Test 2: Generate with game only**
```bash
sr_exhibit --generate --game sms
```
Expected: Prompts for category, subcategory only

**Test 3: Generate with all args**
```bash
sr_exhibit --generate --game sms --category "Any%" --subcategory "GCN"
```
Expected: No prompts, creates config.yaml

**Test 4: Generated config works**
```bash
sr_exhibit  # should use generated config.yaml
```
Expected: Runs with settings from generated config

**Test 5: Normal mode still works**
```bash
sr_exhibit --game sms --category "Any%"
```
Expected: Normal execution without config generation
