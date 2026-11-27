---
sidebar_position: 1
---

# Getting Setup

Before we can start coding anything we need to ensure we have some prerequisites installed, like Go and an IDE.

## Go Programming Language

Castrum is built in Go. A basic knowledge of Go is recommended when using the engine. Fortunately, Go is a relatively easy language to pick up.

You can download Go from the official site, [here](https://go.dev) and follow the instructions. Castrum requires Go 1.27.2 or newer and we aim to always keep up with the latest Go releases.

Once Go is installed, you can verify it by running `go version` in your terminal.

On Linux and FreeBSD, you'll also need a graphical environment and runtime libraries. See the [Ebitengine installation guide](https://ebitengine.org/en/documents/install.html) for the platform-specific requirements.

### New to Go?

If you're new to Go we recommend checking out a few learning resources:

- [A Tour of Go](https://go.dev/tour/welcome/1)
- [Go by Example](https://gobyexample.com)
- [Effective Go](https://go.dev/doc/effective_go)
- [Official learning resources](https://go.dev/learn)

You don't need to be an expert or digest the full Go ecosystem to get started but understanding the memory model, pointers, and slices will prevent you from coding yourself into a conundrum. Fortunately, Go is also a fairly forgiving language. Its compiler gives helpful errors and its runtime checks help catch mistakes early.

## IDE

Castrum doesn't require a specific IDE, but using one with Go support will make development easier. Popular choices include VS Code, with the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go), and [GoLand](https://www.jetbrains.com/go/) for a more integrated experience. VS Code is lightweight, customisable and free but this comes at the cost of some advanced features found in GoLand. GoLand offers more built-in tools for Go development and debugging but is not free.

## Your first Castrum project

### Initialize a new Go project

We firstly need to create a new Go project. The following examples use the terminal. We'll start by creating an empty directory and initializing a Go module there.

```bash
mkdir castrumProject && cd castrumProject
go mod init castrumProject
```

### Add Castrum dependency

Go makes adding dependencies easy. You can add the latest published version of Castrum by running:

```bash
go get github.com/Leonard-Atorough/castrum
```

If you want to install an older version of castrum you can append `@vX.Y.Z` to the `go get` command.

For example:

```bash
go get github.com/Leonard-Atorough/castrum@v0.1.0
```

### Create main.go

Finally, create a file called `main.go` at the root of your project. Paste the following into the file:

```go
package main

import (
	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/ebitrun"
)

func main() {
	game:= castrum.New(castrum.WithTitle("Hello World"))
	// You can add systems, configure the world, or set up other game settings here before running the game.
	ebitrun.New().Run(game)
}
```

Don't worry too much about the syntax right now. Run `go mod tidy` from the project root to download the remaining dependencies, then run the program with `go run main.go`. An empty window titled "Hello World" should open. Close it to stop the program.

That's it. We have now set up a basic Castrum project. Next, we'll talk about the main entry points of every castrum project: the `Game` and `World` structs.
