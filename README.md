# Castrum Engine

## 2D Game Engine in Go, built on top of the amazing Ebiten library/Engine

Castrum provides a simple and efficient way to create 2D games with Go, leveraging the power of Ebiten for rendering and input handling. It uses an Entity Component System (ECS) architecture to manage game entities and their components efficiently. Castrum is primarily designed to build strategy and simulation games at a fixed timestep, ensuring deterministic behavior and smooth gameplay. Thanks to the powerful combination of Go and Ebiten, developers can create high-performance 2D games with minimal boilerplate code. Castrum leverages Ebiten's drawing, input, audio and platform capabilities to provide a comprehensive game development framework.

## Free, now and forever

Castrum is open-source and available for anyone to use, modify, and distribute without any cost. This ensures that developers can leverage the engine for their projects without worrying about licensing fees or restrictions. The engine is released under the [Apache License 2.0](./LICENSE), the same license used Ebiten. This guarantees that Castrum will remain free and open for the community to contribute to and benefit from indefinitely.

## How to Get the Engine

Castrum can be easily obtained and integrated into your Go projects. You can get the latest version of Castrum by running:

```bash
go get github.com/yourusername/castrum
```

After installing, you can import Castrum into your Go code as follows:

```go
import "github.com/yourusername/castrum"
```

In the future castrum aims to ship as a downloadable binary with a visual editor, cli support, and additional tooling to streamline the game development process. The engine is a work in progress and a passion project, see the [ROADMAP](./ROADMAP.md) for the current development plan and upcoming features.

# How to use Castrum

Documentation is being written and will be available soon to guide developers on how to effectively use Castrum for their game development projects. The repo comes with a set of example projects and a basic project template to help you get started quickly.

# About AI-assisted coding and its use in Castrum

Castrum code is written with the assistance of AI tools such as GitHub Copilot and other AI-assisted coding tools to help streamline development, improve code quality, and accelerate the ideation, research and implementation of features. Castrum is categorically not a vibe-coded project and every line of generated code is reviewed, modified and/or approved by the developers to ensure it meets the project's standards and requirements.

The use of AI-assisted coding tools is intended to complement the developers' expertise and not replace it, ensuring that the final codebase maintains high quality, readability, and adherence to best practices.

To aid future contributors aiming to submit AI generated code Castrum will provide dedicated SKILL.md files to guide coding agents on best practices, coding standards, and project-specific conventions to ensure consistency and maintain high-quality contributions.