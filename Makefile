.PHONY build test lint install clean run dev bench release

build:
	@echo "Building the project..."
	@mkdir -p bin
	@go build -o bin/castrum.exe ./cmd/game

test:
	@echo "Running tests..."
	@go test -v -race ./...

lint:
	@echo "Linting the code..."
	@golangci-lint run ./...

bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem -run=^$$ ./benchmark/...

install:
	@echo "Installing the project to the GOPATH/bin..."
	@go install ./cmd/game

clean:
	@echo "Cleaning up build artifacts..."
	@rm -rf bin

run:
	@echo "Running the project..."
	@./bin/castrum.exe run

dev:
	@echo "Running the project in development mode..."
	@go run ./cmd/game/main.go run --watch

release:
	@echo "Cutting a release..."
	@if [ -z "$(BUMP)" ]; then echo "Error: BUMP not specified. Usage: make release BUMP=[patch|minor|major]"; exit 1; fi
	@bash scripts/release.sh $(BUMP)