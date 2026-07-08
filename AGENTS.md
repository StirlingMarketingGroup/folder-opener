# AGENTS.md

Folder Opener — a single-purpose local companion server (Go + tray icon) that opens folders in the system file browser for web apps, plus a TypeScript client in `packages/client` (npm: `folder-opener`). It mirrors the architecture of [Dazzle](https://github.com/StirlingMarketingGroup/dazzle), our ZPL print server; keep the two consistent (port convention, `/status` shape, client `watch()` semantics).

## Verify changes

```bash
go build ./... && go vet ./... && go test ./...
GOOS=windows go build -o /dev/null .   # windows/linux are pure Go — always cross-check both
GOOS=linux CGO_ENABLED=0 go build -o /dev/null .
cd packages/client && npm ci && npm run build
```

## Rules

- The server binds `127.0.0.1` only. Never bind other interfaces.
- One generic endpoint philosophy: no app-specific business logic, auth, or branding in this repo.
- A missing path must stay a real error (`404` + `code: "not_found"`) — never fall back to opening a different location.
- Tests must never open a real file browser window or mount a network share: only exercise invalid-path, URL-parsing, and handler-level error cases.
- Icons are generated from the SVGs in `assets/` (rsvg-convert + ImageMagick); regenerate rather than hand-editing the PNGs/ICO.
- macOS tray uses the template icon (`assets/tray-template.png`); Windows uses `icon.ico`; Linux uses `tray.png`.
