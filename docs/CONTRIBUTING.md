# Contributing to the documentation

Thanks for helping make Castrum easier to learn and use! Fixes to a typo, a confusing explanation, or a broken example are just as welcome as a new guide. For the project-wide contribution process, see [Contributing to Castrum](../CONTRIBUTING.md).

## Write for the reader

Think about what the reader is trying to do and what they need to know first. They may know Go but be new to game development or Castrum, so introduce unfamiliar terms and link to more detail where it helps.

- Start with the goal, prerequisites, and what success looks like.
- Use clear headings, short explanations, and a friendly, direct tone. Explain choices and limitations when they affect the task.
- Keep related steps together, even if they use different packages. Link to existing explanations and API documentation rather than repeating everything.
- Make pages accessible: use descriptive link text, language-tagged code blocks, and alternative text for informative images. Don't rely on color or an image alone to convey an instruction.

A tutorial can walk readers through building something; a how-to can help them solve a specific problem. Explanations provide context, while reference material helps readers look up details. These are useful ways to think about a page, not rigid templates.

## Use practical examples

When possible, start with an existing example or test for the version you're documenting. Show the setup and commands a reader needs, where to run them, and what to expect. If a snippet is only part of a program, make its assumptions clear and link to a complete example when available.

Check API names, behavior, errors, and important constraints against the public API and relevant examples or tests. Go documentation is the primary reference for exported APIs; guides should give readers the details they need for the task and link to the full contract where useful.

## Keep versions and links in mind

Write documentation for the in-development API under `docs/unreleased`. Released documentation describes its matching version, so corrections to a release should remain accurate for that release. Link readers to examples and API documentation that match the version they're using.

Use relative links to pages and examples in the repository. If you move or rename a page or heading, check for incoming links and update them where needed. Keep Markdown readable in the repository without relying on a particular website generator.

## Before opening a pull request

Read your changes as the intended reader: can you follow the steps and recognize the result? Check links and examples, and run the focused tests or commands relevant to your change. If a snippet or an interactive result can't be verified, mention that in the pull request.

Examples and the starter are separate Go modules; root-level tests don't cover them. See [Develop examples against the checkout](../CONTRIBUTING.md#develop-examples-against-the-checkout) for setup. Documentation-only changes don't normally need the full engine test suite.
