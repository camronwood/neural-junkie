package main

import (
	"fmt"
	"strings"
)

func normalizeName(name string) string {
	// Remove hyphens, spaces, underscores
	normalized := strings.ReplaceAll(name, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	// Convert to lowercase
	return strings.ToLower(normalized)
}

func main() {
	name1 := "DayOneExpert"
	name2 := "day-one"

	n1 := normalizeName(name1)
	n2 := normalizeName(name2)

	fmt.Printf("DayOneExpert -> %s\n", n1)
	fmt.Printf("day-one -> %s\n", n2)
	fmt.Printf("Match: %v\n", n1 == n2)
}
