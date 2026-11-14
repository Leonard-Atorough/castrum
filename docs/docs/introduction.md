---
sidebar_position: 1
---

# Introduction

If you're here then you've at least got some interest in learning about Castrum. This documentation page is here to help you understand what Castrum is, how it works and whether it's a good fit for what you might want to do. If you want to build 2D games or applications, Castrum might be the right tool for you.

## Castrum is an open-source Framework

We like to call Castrum an engine but really it's a framework, built around Ebitengine (https://ebiten.org/). We chose Ebiten because it provides a solid foundation for 2D game development in Go, allowing us to focus on higher-level abstractions and features that make game development easier and more efficient.

Castrum aims to simplify common tasks in game development, such as handling input, managing game states, rendering graphics, and organizing game entities. By providing these abstractions, Castrum allows developers to spend more time on the creative aspects of their projects rather than dealing with low-level implementation details.

## Castrum is a passion project

To break the fourth wall a bit, Hi, I'm Leonard, the lead developer and creator of Castrum. Chances are like me you love the creative process and outlet of making games and interactive experiences. Castrum is my way of contributing to this community and doing something meaningful with my passion for game development. My hope is that this framework will empower other developers to bring their creative visions to life with greater ease and enjoyment.

## Design goals and non-goals

This is a good time to be clear about the design goals, and non-goals, of Castrum. They can cleanly be grouped into three key categories:

### Core goals

- Data-driven design: A developer should be able to define game data separately from the game logic, making it easier to manage and modify game content without changing the underlying code.
- Capable 2D game development: Castrum is a 2D first framework, and aims to support the full spectrum of 2D game development needs.
- Simple and intuitive API: One of the main goals, and sources of sleepless nights, is aiming to provide APIs that feel intuitive and good to use.
-

### Stretch goals

- Editor driven: The framework doesn't currently include a built-in editor. In the long term this is a goal but it comes after the core functionality and stability of the framework are well established.
- Scripting support: We'd like to eventually support a scripting language alongside Go, allowing developers to write game logic in a more flexible and dynamic manner.

### Non-goals

- 3D game development: Maybe in the distant future but this would require an entirely new set of tools, abstractions and a new rendering engine integration or custom pipeline. At the moment we simply don't have enough resources or expertise to pursue this direction.

Castrum is built in the open and the core will be free forever, under the Apache 2.0 license. This ensures that developers can freely use, modify, and distribute the framework while contributing to its ongoing development.

:::warning

## A Note on Stability

_yes, this is nicked off Bevy, but it's just as relevant here_
Castrum is still in active development and so there is currently no promise of stability version to version and documentation is a continuing challenge with a small team and rapidly evolving codebase.
Expect frequent changes, potential breaking updates, sudden shifts in API design and overall chaos in the pre 1.0 stage.

That said, we are committed to improving stability and documentation over time, and we welcome contributions and feedback from the community to help guide the development of Castrum.
:::

:::info

## Another note on AI

Up till now Castrum has pretty haphazardly and heavily used AI. This has brought some real boons but also led to some significant wild goose chasing, churn and occasional frustration.
Moving forward, we aim to adopt a structured, nuanced approach to AI usage that prevents unnecessary complexity, reduces wild goose chases, and ensures that AI contributes meaningfully to the development process rather than causing frustration.
:::

Okay, with all those caveats out of the way, and hopefully we haven't bored/scared you off, let's dive into what Castrum is all about and how it can help you build amazing 2D games.
