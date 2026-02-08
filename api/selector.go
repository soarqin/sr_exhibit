package api

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/soar/sr_exhibit/models"
)

// SelectSubcategories interactively selects subcategories
func SelectSubcategories(variables []models.Variable, categoryID string) map[string]string {
	// Filter subcategory variables belonging to specified category (Category empty means global variable, applies to all categories)
	var subcats []models.Variable
	for _, v := range variables {
		if v.IsSubcategory && (v.Category == "" || v.Category == categoryID) {
			subcats = append(subcats, v)
		}
	}

	if len(subcats) == 0 {
		// No subcategories, return empty map
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	result := make(map[string]string)

	for _, variable := range subcats {
		fmt.Printf("\n%s subcategory options:\n", variable.Name)

		// List all options
		values := variable.Values.Values
		options := make([]string, 0, len(values))
		i := 1
		for valueID, value := range values {
			isDefault := valueID == variable.Values.Default
			defaultMark := ""
			if isDefault {
				defaultMark = " (default)"
			}
			fmt.Printf("  %d. %s%s\n", i, value.Label, defaultMark)
			options = append(options, valueID)
			i++
		}

		// Prompt user to select
		fmt.Printf("Select (1-%d, press Enter for default): ", len(options))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			// Use default value
			result[variable.ID] = variable.Values.Default
			fmt.Printf("Selected default: %s\n", values[variable.Values.Default].Label)
		} else {
			// Parse user input
			choice, err := strconv.Atoi(input)
			if err != nil || choice < 1 || choice > len(options) {
				fmt.Printf("Invalid input, using default value\n")
				result[variable.ID] = variable.Values.Default
			} else {
				selectedID := options[choice-1]
				result[variable.ID] = selectedID
				fmt.Printf("Selected: %s\n", values[selectedID].Label)
			}
		}
	}

	return result
}

// SelectCategory interactively selects a category
func SelectCategory(categories []models.Category) (*models.Category, error) {
	if len(categories) == 0 {
		return nil, fmt.Errorf("no available categories")
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\nAvailable categories:\n")
	for i, cat := range categories {
		fmt.Printf("  %d. %s (type: %s)\n", i+1, cat.Name, cat.Type)
	}

	fmt.Printf("Select category (1-%d): ", len(categories))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(categories) {
		return nil, fmt.Errorf("invalid input")
	}

	selected := &categories[choice-1]
	fmt.Printf("Selected: %s\n", selected.Name)
	return selected, nil
}
