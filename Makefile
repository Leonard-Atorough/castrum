.PHONY: build test lint bench run dev install clean release version bump

BIN_DIR := bin
BINARY  := castrum

build:
	@echo "Building the project..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/castrum$(shell go env GOEXE) ./cmd/game

test:
	@echo "Running tests..."
	@go test -v -race ./...

lint:
	@echo "Linting the code..."
	@golangci-lint run ./...

bench:
	@echo "Running benchmarks..."
	@cd benchmark && go test -bench=. -benchmem -benchtime=200ms -count=8 -run='^$$' ./...

run:
	@echo "Running the project..."
	@./$(BIN_DIR)/castrum$(shell go env GOEXE) run

dev:
	@echo "Running the project in development mode..."
	@go run ./cmd/game run --watch

install:
	@echo "Installing to GOPATH/bin..."
	@go install ./cmd/game

clean:
	@echo "Cleaning up build artifacts..."
	@rm -rf $(BIN_DIR)

version:
	@git describe --tags --abbrev=0 2>/dev/null || echo "no tags yet"

bump:
	@bash scripts/bump.sh $(or $(BUMP),auto)

release:
	@echo "Cutting a release (see .github/workflows/release.yml for the canonical path)..."
	@if [ -z "$(BUMP)" ]; then echo "Error: BUMP not specified. Usage: make release BUMP=[auto|patch|minor|major]"; exit 1; fi
	@NEXT=$$(bash scripts/bump.sh $(BUMP)) && \
	  git tag -a $$NEXT -m "Release $$NEXT" && \
	  echo "Tagged $$NEXT — push with: git push origin $$NEXT"