// Package main demonstrates shape-csv as a drop-in replacement for encoding/csv.
package main

import (
	"fmt"
	"log"

	"github.com/shapestone/shape-csv/pkg/csv"
)

func main() {
	fmt.Println("=== shape-csv: Similar API to encoding/csv ===\n")

	// CSV with special characters that need quoting
	csvData := `product,description,price
Widget,"A small, useful device",19.99
"Gadget Pro","Premium ""quality"" item",49.99
Cable,"10-foot
multi-line description",9.99`

	// Parse and render (round-trip)
	fmt.Println("Original CSV:")
	fmt.Println(csvData)
	fmt.Println()

	// Parse into AST
	node, err := csv.Parse(csvData)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Render back to CSV
	output, err := csv.Render(node)
	if err != nil {
		log.Fatalf("Render error: %v", err)
	}

	fmt.Println("Re-rendered CSV:")
	fmt.Println(string(output))

	// Demonstrate type conversion with Unmarshal
	fmt.Println("=== Unmarshal with type conversion ===")

	type Product struct {
		Name        string  `csv:"product"`
		Description string  `csv:"description"`
		Price       float64 `csv:"price"`
	}

	var products []Product
	if err := csv.Unmarshal([]byte(csvData), &products); err != nil {
		log.Fatalf("Unmarshal error: %v", err)
	}

	for _, p := range products {
		fmt.Printf("  Product: %s ($%.2f)\n", p.Name, p.Price)
		fmt.Printf("    Desc: %s\n", p.Description)
	}

	// Marshal back to CSV
	fmt.Println("\n=== Marshal back to CSV ===")
	marshaled, err := csv.Marshal(products)
	if err != nil {
		log.Fatalf("Marshal error: %v", err)
	}
	fmt.Println(string(marshaled))

	// Validation example
	fmt.Println("=== Validation ===")

	validCSV := "a,b,c\n1,2,3"
	invalidCSV := `"unclosed quote`

	if err := csv.Validate(validCSV); err == nil {
		fmt.Println("Valid CSV: ✓")
	}

	if err := csv.Validate(invalidCSV); err != nil {
		fmt.Printf("Invalid CSV: %v\n", err)
	}
}
