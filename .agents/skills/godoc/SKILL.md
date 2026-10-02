---
name: godoc
description: >
  Write and review Go doc comments for this repository. Stdlib-like
  brevity following go.dev/doc/comment. Use for "document this",
  "godoc pass", "fix these docs", or before finishing a new public API.
---

# godoc

A doc comment answers "what does this do and how do I use it" without
reading the code. The stdlib is the style bar: precise, quiet, never
an essay.

## Rules

1. Every exported name gets a doc comment starting with its name.
   Unexported names get none, unless the comment carries a *why* the
   code cannot show (a decision, a contract, an ownership rule) —
   never a what.
2. Purpose, not mechanics. Document what the thing does, its
   contract, and what it is for. Restate the body only where the
   caller cannot see it. Length is earned: NewMixer needs one line;
   a state machine's Sync earns its paragraphs. A contract is
   documented once, where it lives — on the field, the method, or
   the type that owns it; a doc that repeats another name's
   contract is a drift bug waiting. Contract words are exact:
   "not finished" and "not playing" are different rules, and an
   implementer will be wrong about the difference.
3. Structs describe their purpose; fields carry a short comment
   saying what the field is for. That comment is the IDE hover in a
   struct literal — where the user actually writes the value — so it
   exists even when the name and type feel obvious. One line is the
   norm; longer is earned by a non-obvious rule (a rejected or
   inverted zero, ownership, who writes it). Field docs do not
   repeat the struct doc.
4. Package docs live in the main file for small packages, in doc.go
   for large ones. One paragraph: what the package provides and how
   its pieces relate.
5. Follow go.dev/doc/comment: complete sentences, no first person,
   [T] links for exported names in the same module, names written
   exactly as they appear, a blank comment line between the summary
   and the detail.
6. In a pass over existing code, change only what is wrong or noise.
   Do not rewrite docs that already follow these rules.
