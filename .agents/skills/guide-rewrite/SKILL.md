---
name: guide-rewrite
description: "Plan, write, improve, or review Castrum tutorials and how-to guides. Use for reader-centered documentation structure, API-grounded workflows, examples, and focused validation."
argument-hint: "[plan|implement|review] <reader-goal-or-guide-path> [owning-package-or-folder]"
user-invocable: true
---

# Guide Rewrite

Use this workflow for Castrum tutorials and how-to guides. It also applies when planning or reviewing their structure. Ground every behavioral claim in the public API and relevant working examples or tests.

## Modes and scope

Choose a mode from the request. Default to `implement` for a request to write or update documentation and to `plan` for a request about structure or approach.

- `plan`: read-only; propose an outcome, structure, ownership, and checks.
- `implement`: make the requested or agreed change and validate it.
- `review`: read-only; report actionable findings and residual risk.

Identify the reader's goal, intended audience, target version, affected guide or area, owning public API, and closest runnable example or test. If the guide exists, read it first and preserve valid user-authored material. Inspect only the related material needed to understand the task; a cross-cutting workflow may involve more than one package.

Keep changes within the requested scope. Do not move or split pages, change released snapshots, or alter examples or Go docs unless requested or necessary to correct an affected link or claim. If a necessary change would materially broaden scope, explain it and ask before proceeding. Follow the repository's versioning policy: development documentation lives under `docs/unreleased`, while released snapshots describe their matching tagged API.

## Establish purpose and ownership

Assign each page a primary reader need:

- **Tutorial:** guide learning through a prescribed, end-to-end outcome, with prerequisites, working milestones, and observable results.
- **How-to:** help a reader who has a specific goal complete it; cover the relevant choices and constraints without cataloging every API.
- **Explanation:** clarify a mental model, relationship, or design decision; link to tasks rather than embedding a full procedure.
- **Reference:** describe exact behavior and contracts for lookup. Go documentation is the primary API reference; Markdown reference is for necessary cross-cutting contracts or navigation.

Group related tasks by what the reader is trying to do, not by source package. Prefer an existing page or section over creating another. Split only when readers have distinct goals or navigation clearly benefits; keep dependent steps together. A folder can bring related tasks and concepts together without making every page serve the same purpose.

Give each detailed contract or shared explanation one authoritative home. Summarize a safety-critical or task-essential constraint where needed and link to its home; do not duplicate an API catalog. An API addition warrants a guide change when it changes the reader's task or recommended path, not merely because the symbol exists. Preserve important existing behavior and limitations when moving material.

## Procedure

### 1. Establish the contract

Read the current guide, owning public API, and the closest relevant example or test. Determine:

- The outcome, prerequisites, setup order, and observable result.
- Relevant ownership, lifecycle, scheduling, caching, validation, and error timing.
- Which example can be reused and whether it matches the target version.
- Which claims are stale, which constraints matter to this task, and where related details already belong.

Distinguish verified facts from assumptions. State a concise implementation hypothesis and a focused validation check before editing.

### 2. Plan

Return the proposed page purpose and heading hierarchy, reader workflow, API facts and limitations, existing material and examples to reuse, ownership and links, and focused validation. For a structural plan, map affected existing material to its proposed home and identify link and version compatibility. Do not edit files in this mode.

### 3. Implement

Write for people using the engine:

- Use a warm, clear, direct voice. Explain why a choice matters when it helps the reader act; avoid filler, jargon without context, marketing, and forced enthusiasm.
- Lead with the goal and essential prerequisites. Give a tutorial a sequence of runnable milestones; give a how-to the shortest useful path and task-relevant alternatives.
- Explain outcomes and meaningful failure modes at the point of use. Distinguish setup-time validation from lazy or runtime resolution and state limitations rather than implying unsupported features.
- Prefer focused snippets adapted from working code. Say whether a snippet is complete or illustrative, identify required surrounding setup, and link to the matching runnable example where available.
- Use descriptive headings and link text, short coherent sections, and relative links that work in repository Markdown. Do not depend on a particular website renderer, visual layout, or navigation state.
- Preserve useful user-authored material. Do not hard-wrap Markdown prose; keep each paragraph and list item on one source line, while preserving meaningful structure in code blocks and tables. Use ASCII by default.

After editing, check the page end to end for a reader's ability to complete the task. Verify API names, signatures, links and anchors, example version, error handling, and claims against the implementation. Run the smallest relevant executable check for the owning package or example when available, followed by existing documentation checks and `git diff --check`. State explicitly when a snippet, interactive step, link, or command was not verified. Do not claim that a package test proves independent Markdown snippets compile.

### 4. Review

Do not edit. Report findings first, ordered by severity: incorrect behavior or examples; missing constraints, ownership, or failure timing; broken links or version mismatches; then confusing reader flow, duplicated ownership, and prose. Cite the affected location and practical consequence. End with checks performed and residual risk; say clearly if there are no findings.

## Completion

A rewrite is complete when its primary purpose and reader outcome are clear, its claims match the target API, necessary constraints and examples are accurate, links resolve to appropriate versions, and focused checks pass or their limitations are reported. Do a final prose and navigation check without expanding scope.
