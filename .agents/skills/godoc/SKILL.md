---
name: godoc
description: >
  Write and review Go doc comments for this repository. Stdlib-like
  brevity following go.dev/doc/comment. Use for "document this",
  "godoc pass", "fix these docs", or before finishing a new public API.
---

# godoc

A doc comment says what a name does and how a caller uses it, without
the reader opening the code. Write like the standard library: precise,
short, and never an essay. Follow go.dev/doc/comment.

## Procedure

Follow these steps in order. Do not skip the checklist.

1. Find the names that need a doc comment.
   - Every exported name needs one: funcs, methods, types, consts, vars, and struct fields.
   - Unexported names need one only if the comment explains a reason the code cannot show (see "Unexported names").
2. For each name, decide whether its doc passes every item in the checklist.
3. Edit only the docs that fail. Leave passing docs exactly as they are.
4. Verify (see "Verification").

## Checklist

Each item must be true for every doc you touch.

- [ ] The first sentence starts with the exact name: `Collect resolves...`, not `This method resolves...`.
- [ ] The first sentence is a complete sentence in the third person, present tense.
- [ ] No first person ("I", "we", "our") and no "this function" or "this type".
- [ ] The doc is attached to the declaration. No blank line sits between the comment and the `func`, `type`, or field it describes.
- [ ] The doc states purpose and contract, not the implementation steps. Restating the body is allowed only when a caller could not learn it from the signature.
- [ ] Each contract is stated once, on the name that owns it. Do not copy another name's contract into this doc.
- [ ] Contract words are exact (see "Contract words").
- [ ] Links use the link syntax in "Links".
- [ ] Punctuation is plain: no spaced hyphens (` - `) used as dashes. Use a semicolon, colon, or new sentence.
- [ ] The length fits the rules in "Length".

## Structs and fields

- The type doc says what the type is for.
- Each exported field gets a one-line comment that starts with the field name, for example `Asset is the texture ID.` Field comments show in IDE hovers when a caller writes a struct literal, so they are required even when the name looks obvious.
- Add a second line only for a non-obvious rule: a zero value that is rejected or means something special, who writes the field, or who owns it.
- A field comment does not repeat the type doc.

## Length

- Default to one sentence for the summary. Add a paragraph only when the caller needs a rule they cannot see.
- A method with no contract beyond its name needs one line. For example: `Len returns the number of queued items.`
- Separate paragraphs with a blank `//` line.

## Contract words

These words have exact meanings. Use them only when the meaning matches.

- Use a contract word only when its meaning matches the code. Two words that sound alike can mean different rules, such as "not finished" and "not playing". Check the code before you write either one.
- "returns an error" means the call reports failure through its error result. "panics" means it panics.

## Links

- Link exported names in the same package with `[Name]`, for example `[Collector.Collect]`.
- Link exported names in another package of this module with `[pkg.Name]`, for example `[sprite.Sprite]`.
- Do not link struct fields. Doc links resolve types, functions, methods, and packages only. Name the field in plain text, or link its owning type.
- Link the package with `[pkg]`.

## Packages

- Package docs start with `Package name` (or `Command name` for a `main` package).
- Small packages put the doc in the main file. Large packages put it in `doc.go`.
- Only one file in a package carries the package doc.

## Unexported names

Do not document an unexported name unless the comment states a reason the code cannot show: a decision, a contract, or an ownership rule. Never document what an unexported name does.

- Keep: `// view is the published resource; every Collect writes into it.` (It states ownership.)
- Remove: `// helper that loops over entities.` (It restates the code.)

## Pitfalls

- Doc on the wrong symbol. A comment placed above a method that describes the type is a bug. Move the type doc to the type, and give the method its own doc.
- Duplicated contracts. A method and its type both describing the same rule will drift apart. Keep the rule on one name and refer to it from the other.
- Long algorithm walkthroughs. Describe the observable result and the contract, not each step.
- Deprecation notes. Use a separate paragraph that starts with `Deprecated:` and names the replacement.

## Pass over existing code

- Change only docs that break a checklist item or contain noise. Do not rewrite a doc that already follows the rules, even if you would phrase it differently.
- Do not add docs to unexported names unless they carry a reason (see "Unexported names").

## Verification

Run these after editing:

1. `gofmt -l <dir>` should print nothing for the files you changed.
2. `go vet <pkg>` should pass.
3. `go doc <pkg> <Name>` should show the doc you wrote, starting with the name.
4. Search for exported names that still have no doc. Use the grep tool with `-B 1` on `^func [A-Z]|^func \([^)]*\) [A-Z]|^type [A-Z]` and check that each match follows a comment line.

Report to the user which docs you changed and why. If you changed nothing, say so.
