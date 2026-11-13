# Writing Castrum documentation

This guide is the editorial contract for everyone writing or reviewing Castrum documentation, whether working by hand or with an agent. It describes the purpose of each kind of page, how to keep content accurate and maintainable, and what to check before proposing a change. It applies to documentation for game authors; [Contributing to Castrum](../CONTRIBUTING.md) covers the project-wide contribution and release process.

The current documentation is being reorganized gradually. These principles guide new work and scoped improvements; they do not require a contributor to move or rewrite unrelated pages.

## Start with the reader

Most readers know Go, but may be new to game development, entity-component systems, or Castrum. Do not assume they know the engine's terminology or internal packages. State prerequisites that matter, introduce necessary terms in context, and link to deeper explanations.

Before writing, answer:

1. Who is the reader, and which engine version are they using?
2. What do they want to learn, do, understand, or look up?
3. What do they need before starting, and how will they recognize success?
4. Which existing page, public API contract, example, or test already owns the relevant information?

Write for the reader's goal, not the shape of the source tree. A useful task may cross several packages. Keep necessary steps together even when their APIs live in different places.

## Choose the page's job

Each page should have one primary purpose. These purposes are an editorial tool, not labels a reader must understand before navigating.

| Purpose | Reader's question | What the page does |
| --- | --- | --- |
| Tutorial | How do I learn to build something? | Takes the reader through a prescribed outcome, with prerequisites, runnable milestones, and observable results. |
| How-to | How do I accomplish this task? | Shows a practical path and task-relevant choices and constraints. It assumes the reader knows their goal. |
| Explanation | Why does this work this way? | Develops a mental model or explains relationships and trade-offs, without becoming a full procedure. |
| Reference | What exactly does this guarantee? | Describes facts, defaults, errors, and contracts for lookup. Go documentation is the primary reference for exported APIs. |
| Migration | What must I change to upgrade? | Describes changes between specific versions, including when a change is needed and how to verify it. |

A brief explanation or warning can make a how-to usable, and a tutorial can link to a reference. Do not turn either into an API inventory. Markdown reference is for necessary cross-cutting contracts or navigation that cannot be owned clearly by one Go API comment.

Name pages and sections for the reader's goal. For example, “Frame the world with a camera” tells a reader what they can accomplish; “Camera API” suggests a symbol catalog. Organize related tasks together, but do not assume one folder per package or one page per exported type.

## Keep one authoritative home

Place the detailed contract with the public Go API that owns it. Put shared mental models in explanation, task procedures in how-tos, and prescribed learning sequences in tutorials. Link to that home instead of repeating a complete description elsewhere.

Readers should not have to leave a task to discover a prerequisite, a relevant limitation, or an error that could invalidate their result. Summarize essential constraints at the point of use, then link to the detailed contract. For example, a sheet guide should say when a missing image or invalid region can fail, while the detailed validation rules remain with the owning API.

When an API changes, inspect its Go documentation, tests, examples, affected workflows, and any cross-cutting contracts. Update each affected item; do not add a guide section merely because a symbol was added. A new page is warranted when it serves a distinct reader goal or makes a real navigation problem easier. Prefer improving an existing page otherwise.

When splitting or removing content, map each useful section and contract to its new home before removing it. Preserve important limitations and update links; do not discard information simply because it does not fit the first proposed outline.

## Write for people

- Be friendly, direct, and respectful. Use “you” and active voice when natural. Avoid marketing, jokes at the reader's expense, unnecessary jargon, and claims that a task is “easy” or “obvious.”
- Lead with the outcome and essential prerequisites. Use descriptive headings and link text, and keep sections focused enough to scan.
- Explain the reason for a decision when it helps the reader act. State current limitations and meaningful errors where they affect the task; distinguish setup validation from failures that can occur later.
- Use consistent names and explain unfamiliar concepts the first time they matter. Prefer plain language without obscuring exact API names.
- Write for accessibility and future web publishing: use a logical heading hierarchy, meaningful link text, language-tagged code fences, and descriptive alternative text for informative images. Do not rely on color, position, or an image alone to convey an instruction.
- Use Markdown that works in the repository without requiring a particular site generator. Use relative links for repository pages and examples, and link to the matching version of external API documentation.
- Do not hard-wrap Markdown prose. Keep each paragraph or list item on one source line; preserve meaningful line breaks in code blocks, tables, and other structured content.

## Examples and runnable steps

Prefer an existing, version-matched runnable example or test to invented code. A guide may use an illustrative excerpt rather than a complete program, but say what is assumed: relevant imports, variables, setup, files, working directory, and error-returning context. Link to the complete example when one exists. Do not present an incomplete excerpt as a copy-and-run program.

For commands and tutorials, state the prerequisites, where to run the command, and what result to expect. Give each tutorial milestone a runnable state and an observable outcome. Keep displayed code and completed examples aligned; if the full example builds but intermediate steps were not checked, say so.

Surface relevant errors instead of ignoring them or substituting success-shaped defaults. Show the supported API and current limitations, not speculative behavior. Do not use implementation details as public guarantees unless the contract makes them one.

## Versions, links, and moves

Development documentation belongs under `docs/unreleased`. Released snapshots describe their matching tagged API; do not silently rewrite history or point a release reader at incompatible development code. A factual correction to a released guide must remain accurate for that release.

Link released guides to the matching API version, example source or release tag, and asset bundle when those are needed. Development documentation may link to development source, but identify it as such. Migration notes identify both the source and destination versions.

Before moving or renaming a page or heading, identify incoming links and anchors, update the affected links, and preserve access for readers of published URLs where practical. A future website should provide redirects for published paths; repository links or compatibility notes may be needed separately. Do not assume a website redirect exists until it has been configured and checked.

## Review and validation

Use the smallest checks that demonstrate the actual change. Record the exact commands and results in the pull request, and state what could not be verified.

1. **Read for the reader.** Can the intended reader find the page, follow it from its stated starting point, recognize the result, and recover from relevant failures?
2. **Check accuracy.** Compare changed claims, names, signatures, lifecycle rules, and examples with the public API and relevant tests or runnable code for the target version.
3. **Check navigation.** Verify changed links and anchors, including incoming links affected by a move, and confirm examples and API links match the documented version.
4. **Run focused executable checks when applicable.** For example, `go test ./asset` checks the root asset package. Examples and the starter are separate modules: run `go test ./...` from the relevant module directory using the [development workspace](../CONTRIBUTING.md#develop-examples-against-the-checkout), or the matching release checkout as appropriate. Root `go test ./...` does not test nested modules.
5. **Verify the promised output.** A passing package or example test does not compile an independent Markdown snippet or prove a visual or audio result. Check changed snippets and tutorial milestones in their stated context where feasible; report any compile, interactive, platform, or version checks not performed.
6. **Check the change.** Run `git diff --check` and any applicable repository documentation checks. There is currently no general Markdown validation gate in the repository; do not claim one was run.

Documentation-only edits do not require the full engine test suite by default. Choose checks based on what changed and report limitations rather than claiming unverified behavior.
