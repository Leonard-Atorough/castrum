# Castrum documentation website

From this directory, install dependencies and start the local website:

```sh
npm ci
npm start
```

Add Markdown pages under `docs/`. Folders containing pages appear as groups in
the sidebar. Set `sidebar_position` in a page's front matter to change its
order.

To check the website builds:

```sh
npm run build
```

The website is published to GitHub Pages from `main`. Pull requests build and
type-check the website; publishing requires the repository's Pages source to
be set to GitHub Actions. Its address is
`https://leonard-atorough.github.io/castrum/`.

After a release without an explicit pre-release suffix, the release workflow
snapshots documentation from the release tag and commits it to `main`. The
latest release is the default; the version menu also links to development docs.
Tags such as `v0.4.0-rc.1` do not create a documentation snapshot.
The v0.3.0 documentation was imported from the earlier archive; its tag
predates this website and cannot be regenerated with the versioning workflow.

If snapshotting fails after a release is published, fix the reported problem
and run the **Version documentation** workflow manually with the existing
release tag. It is safe to rerun for an already-created matching snapshot.
The workflow uses the release environment and the same GitHub App credentials
as the release workflow, so it may require a separate release-environment
approval. The app must be allowed to push to `main`.
