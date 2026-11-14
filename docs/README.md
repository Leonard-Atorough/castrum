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

The site URL and base path in `docusaurus.config.ts` are for local use and
must be updated before deployment.
