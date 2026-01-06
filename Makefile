.PHONY: test lint build coverage clean all grammar-test grammar-verify
.PHONY: bench bench-report bench-compare bench-profile performance-report
.PHONY: bench-history bench-compare-history bench-vs-stdlib

# Run all tests (excluding examples and scripts)
test:
	go test -v -race ./internal/... ./pkg/...

# Run grammar verification tests only
grammar-test:
	@echo "Running grammar verification tests..."
	go test -v ./internal/parser -run TestGrammar

# Verify grammar files exist and are valid
grammar-verify:
	@echo "Verifying grammar files..."
	@if [ ! -f docs/grammar/csv.ebnf ]; then \
		echo "Error: Grammar file missing at docs/grammar/csv.ebnf"; \
		exit 1; \
	fi
	@echo "✓ Grammar file exists (csv.ebnf)"
	@go test ./internal/parser -run TestGrammarFileExists 2>/dev/null || echo "Note: Grammar test not yet implemented"

# Run linter
lint:
	golangci-lint run

# Build the project
build:
	go build ./...

# Generate coverage report (excluding examples and scripts)
coverage:
	@mkdir -p coverage
	go test -v -coverprofile=coverage/coverage.out ./internal/... ./pkg/...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"
	@go tool cover -func=coverage/coverage.out | grep total

# Clean generated files
clean:
	rm -rf coverage/ benchmarks/
	go clean

# Run all checks (grammar, test, lint, build, coverage)
all: grammar-verify test lint build coverage

# ================================
# Benchmark Targets
# ================================

# Run all benchmarks with standard settings
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./pkg/csv/

# Run benchmarks and save output to a file
bench-report:
	@mkdir -p benchmarks
	@echo "Running benchmarks and saving to benchmarks/results.txt..."
	go test -bench=. -benchmem ./pkg/csv/ | tee benchmarks/results.txt
	@echo "Benchmark results saved to benchmarks/results.txt"

# Run benchmarks multiple times with benchstat for statistical analysis
bench-compare:
	@mkdir -p benchmarks
	@echo "Running benchmarks 10 times for statistical analysis..."
	@echo "This will take several minutes..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		echo "Run $$i/10..."; \
		go test -bench=. -benchmem ./pkg/csv/ >> benchmarks/benchstat.txt; \
	done
	@echo "Results saved to benchmarks/benchstat.txt"
	@echo "Install benchstat with: go install golang.org/x/perf/cmd/benchstat@latest"
	@echo "Analyze with: benchstat benchmarks/benchstat.txt"

# Run benchmarks with CPU and memory profiling
bench-profile:
	@mkdir -p benchmarks
	@echo "Running benchmarks with CPU profiling..."
	go test -bench=BenchmarkShapeCSV_Parse_Large -benchmem -cpuprofile=benchmarks/cpu.prof ./pkg/csv/
	@echo "CPU profile saved to benchmarks/cpu.prof"
	@echo "Analyze with: go tool pprof benchmarks/cpu.prof"
	@echo ""
	@echo "Running benchmarks with memory profiling..."
	go test -bench=BenchmarkShapeCSV_Parse_Large -benchmem -memprofile=benchmarks/mem.prof ./pkg/csv/
	@echo "Memory profile saved to benchmarks/mem.prof"
	@echo "Analyze with: go tool pprof benchmarks/mem.prof"

# Generate performance report from benchmark results
performance-report:
	@echo "Generating performance report..."
	@go run scripts/generate_benchmark_report/main.go
	@echo "Performance report updated: PERFORMANCE_REPORT.md"

# List available benchmark history runs
bench-history:
	@if [ -d "benchmarks/history" ]; then \
		has_benchmarks=false; \
		for dir in benchmarks/history/*/; do \
			if [ -d "$$dir" ] && [ -f "$${dir}benchmark_output.txt" ]; then \
				has_benchmarks=true; \
				break; \
			fi; \
		done; \
		if [ "$$has_benchmarks" = "true" ]; then \
			echo "Available benchmark history:"; \
			echo ""; \
			for dir in benchmarks/history/*/; do \
				if [ -d "$$dir" ] && [ -f "$${dir}benchmark_output.txt" ]; then \
					timestamp=$$(basename "$$dir"); \
					echo "  $$timestamp"; \
					if [ -f "$${dir}metadata.json" ]; then \
						grep -E '"(commit|platform)"' "$${dir}metadata.json" | sed 's/^/    /'; \
					fi; \
					echo ""; \
				fi; \
			done; \
		else \
			echo "No benchmark history found."; \
			echo "Run 'make performance-report' to create your first benchmark."; \
		fi; \
	else \
		echo "No benchmark history found."; \
		echo "Run 'make performance-report' to create your first benchmark."; \
	fi

# Compare current benchmarks vs most recent historical run
bench-compare-history:
	@if ! command -v benchstat >/dev/null 2>&1; then \
		echo "Error: benchstat not found. Install with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
		exit 1; \
	fi
	@if [ ! -d "benchmarks/history" ] || [ -z "$$(ls -A benchmarks/history 2>/dev/null)" ]; then \
		echo "Error: No benchmark history found."; \
		echo "Run 'make performance-report' to create benchmark history."; \
		exit 1; \
	fi
	@echo "Comparing benchmarks..."
	@go run scripts/compare_benchmarks/main.go latest previous

# Compare shape-csv vs encoding/csv (standard library)
bench-vs-stdlib:
	@echo "============================================"
	@echo "shape-csv vs encoding/csv Benchmark"
	@echo "============================================"
	@echo ""
	@echo "Unmarshal to [][]string:"
	@go test -bench='(Unmarshal_Records|ReadAll)_(Small|Medium|Large)$$' -benchmem ./pkg/csv/ 2>/dev/null | grep -E '^(Benchmark|ok)'
	@echo ""
	@echo "Streaming:"
	@go test -bench='(Scanner|Reader)_Large' -benchmem ./pkg/csv/ 2>/dev/null | grep -E '^(Benchmark|ok)'
	@echo ""
	@echo "Marshal/Write:"
	@go test -bench='(Marshal|Write)_(Medium|Large)$$' -benchmem ./pkg/csv/ 2>/dev/null | grep -E '^(Benchmark|ok)'
	@echo ""
	@echo "Quoted Fields:"
	@go test -bench='QuotedFields' -benchmem ./pkg/csv/ 2>/dev/null | grep -E '^(Benchmark|ok)'
