// Package main demonstrates streaming CSV parsing with shape-csv.
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/shapestone/shape-csv/pkg/csv"
)

func main() {
	// Simulating a large CSV file with a reader
	csvData := `id,name,email,department
1,Alice,alice@example.com,Engineering
2,Bob,bob@example.com,Marketing
3,Charlie,charlie@example.com,Sales
4,Diana,diana@example.com,Engineering
5,Eve,eve@example.com,HR`

	fmt.Println("=== Streaming CSV with Scanner ===")

	// Create a reader (in real usage, this would be os.Open("file.csv"))
	reader := strings.NewReader(csvData)

	// Create a scanner for streaming
	scanner := csv.NewScanner(reader)
	scanner.SetHasHeaders(true)

	// Get headers
	fmt.Printf("Headers: %v\n\n", scanner.Headers())

	// Stream through records one at a time
	fmt.Println("Records:")
	rowNum := 0
	for scanner.Scan() {
		rowNum++
		record := scanner.Record()

		// Access by index
		id, _ := record.Get(0)

		// Access by column name (requires headers)
		name, _ := record.GetByName("name")
		email, _ := record.GetByName("email")
		dept, _ := record.GetByName("department")

		fmt.Printf("  Row %d: ID=%s, Name=%s, Email=%s, Dept=%s\n",
			rowNum, id, name, email, dept)
	}

	// Check for errors
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading CSV: %v", err)
	}

	fmt.Printf("\nTotal rows processed: %d\n", rowNum)

	// Example with ParseReader for full AST
	fmt.Println("\n=== Using ParseReader for AST ===")
	reader2 := strings.NewReader(csvData)
	node, err := csv.ParseReader(reader2)
	if err != nil {
		log.Fatalf("Failed to parse: %v", err)
	}
	fmt.Printf("Parsed into: %T\n", node)
}
