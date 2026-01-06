// Package main demonstrates basic CSV parsing with shape-csv.
package main

import (
	"fmt"
	"log"

	"github.com/shapestone/shape-csv/pkg/csv"
)

func main() {
	// Example CSV data
	csvData := `name,age,city
Alice,30,New York
Bob,25,Los Angeles
Charlie,35,Chicago`

	// Parse CSV into AST
	fmt.Println("=== Parsing CSV to AST ===")
	node, err := csv.Parse(csvData)
	if err != nil {
		log.Fatalf("Failed to parse CSV: %v", err)
	}
	fmt.Printf("Parsed %T successfully\n\n", node)

	// Unmarshal CSV into structs
	fmt.Println("=== Unmarshaling CSV to structs ===")
	type Person struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
		City string `csv:"city"`
	}

	var people []Person
	if err := csv.Unmarshal([]byte(csvData), &people); err != nil {
		log.Fatalf("Failed to unmarshal: %v", err)
	}

	for _, p := range people {
		fmt.Printf("  %s is %d years old, lives in %s\n", p.Name, p.Age, p.City)
	}
	fmt.Println()

	// Marshal structs back to CSV
	fmt.Println("=== Marshaling structs to CSV ===")
	output, err := csv.Marshal(people)
	if err != nil {
		log.Fatalf("Failed to marshal: %v", err)
	}
	fmt.Println(string(output))

	// Validate CSV
	fmt.Println("=== Validating CSV ===")
	if err := csv.Validate(csvData); err != nil {
		fmt.Printf("Invalid CSV: %v\n", err)
	} else {
		fmt.Println("CSV is valid!")
	}
}
