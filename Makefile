.PHONY: build test lint bench release version bump

build:
	@echo "Building the project..."
	@go build ./...

test:
	@echo "Running tests..."
	@go test -v -race ./...

lint:
	@echo "Linting the code..."
	@go vet ./...

bench:
	@echo "Running benchmarks..."
	@cd benchmark && go test -bench=. -benchmem -benchtime=200ms -count=8 -run='^$$' ./...

version:
	@git describe --tags --abbrev=0 2>/dev/null || echo "no tags yet"

bump:
	@bash .github/scripts/bump.sh $(or $(BUMP),auto)

release:
	@echo "Cutting a release (see .github/workflows/release.yaml for the canonical path)..."
	@if [ -z "$(BUMP)" ]; then echo "Error: BUMP not specified. Usage: make release BUMP=[auto|patch|minor|major]"; exit 1; fi
	@NEXT=$$(bash .github/scripts/bump.sh $(BUMP)) && \
	  git tag -a $$NEXT -m "Release $$NEXT" && \
	  echo "Tagged $$NEXT - push with: git push origin $$NEXT"