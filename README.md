# Folder Opener

<img src="assets/icon.png" width="96" align="right" alt="Folder Opener icon">

A tiny cross-platform local companion server that does exactly one thing: **open a folder in the system's file browser** (Explorer / Finder / your Linux file manager) when a web app asks it to.

Web pages can't open local folders — browsers rightly forbid it. Folder Opener bridges that gap the same way [Dazzle](https://github.com/StirlingMarketingGroup/dazzle) bridges label printing: a small background app listening on `localhost` that any of your web apps can call, with a thin npm client and a tray icon so you can see it's alive.

It replaces fragile custom URL-protocol handlers (`sterling:` and friends) that are Windows-only, flash console windows, and silently open your Documents folder when the target path doesn't exist.

## How it works

- Runs silently in the background with a system tray icon (Windows, macOS, Linux)
- Listens on **`127.0.0.1:29101`** only — never exposed to the network
- `POST /open` with a path: directories open in the file browser, files are revealed (selected) in their parent folder
- A **missing path returns a real 404 error** the calling app can show to the user — no silent fallback
- `GET /status` lets web apps poll whether it's running (to show an "install Folder Opener" banner)

## HTTP API

### `GET /status`

```json
{ "status": "running", "version": "v0.1.0" }
```

### `POST /open`

```bash
curl -X POST http://localhost:29101/open \
  -H 'Content-Type: application/json' \
  -d '{"path": "/Users/me/Documents/projects"}'
```

Success — `200`:

```json
{ "path": "/Users/me/Documents/projects", "action": "opened" }
```

`action` is `"opened"` for a directory, `"revealed"` when the path was a file that got selected in its parent folder.

Errors are JSON with a stable `code`:

```json
{ "error": "path does not exist: \"/Users/me/nope\"", "code": "not_found" }
```

| Status | `code` | Meaning |
|---|---|---|
| 400 | `bad_request` | Empty/relative path or invalid JSON |
| 404 | `not_found` | Path doesn't exist on this machine |
| 500 | `internal` | The file browser failed to launch |

Paths must be absolute. Windows UNC paths (`\\server\share\...`) count as absolute.

### SMB share URLs

`path` can also be an `smb://server/share/...` URL, so one payload works on every OS even when the share isn't mounted yet:

- **macOS** mounts the share on demand the same way Finder does (NetFS: mounts under `/Volumes`, uses Keychain credentials, shows the standard authentication dialog only when needed) — no extra Finder window at the share root.
- **Windows** opens the equivalent UNC path (`\\server\share\...`) directly; Windows connects and authenticates natively.
- **Linux** mounts the share through gvfs (`gio mount`) and opens the path inside the user's gvfs FUSE mount.

```bash
curl -X POST http://localhost:29101/open \
  -H 'Content-Type: application/json' \
  -d '{"path": "smb://storage/Signature Coins/12345"}'
```

Percent-encoding is optional — literal spaces are accepted. A missing folder below the share is still a real `404`/`not_found`; a share that can't be mounted is a `500` with the mount error.

CORS is permissive (any origin): the server only ever opens the local file browser, and it answers Chrome's Private Network Access preflight so pages on public origins can reach `localhost`. Chrome will still ask the user once for "local network access" permission per origin — same as Dazzle.

## Client library

```bash
npm install folder-opener
```

```ts
import { FolderOpener, FolderOpenerError } from 'folder-opener';

const folderOpener = new FolderOpener();

try {
    await folderOpener.open('\\\\storage\\Art\\12345');
} catch (err) {
    if (err instanceof FolderOpenerError && err.code === 'not_found') {
        // the folder doesn't exist — tell the user instead of silently failing
    }
    throw err;
}

// Show a banner while the companion app isn't running
const unwatch = folderOpener.watch(running => {
    banner.classList.toggle('d-none', running);
}, { interval: 1_000 });
```

## Installing the server

Grab the installer for your platform from [Releases](https://github.com/StirlingMarketingGroup/folder-opener/releases):

| OS | Artifact | Autostart |
|---|---|---|
| Windows | `.msi` (per-machine, GPO-deployable) | baked in — HKLM `Run` value for all users |
| macOS | `.dmg` (universal, signed + notarized + stapled) | drag to Applications, launch once, tray → "Start at Login" |
| Linux | `.deb` | baked in — `/etc/xdg/autostart` entry for all desktop sessions |

A bare `.exe` is also attached for Windows setups that don't want the MSI. When running the bare binary, start-at-login can be managed from the tray menu ("Start at Login") or the CLI:

```bash
folder-opener autostart enable    # enable start at login
folder-opener autostart disable
folder-opener autostart status
```

The CLI/tray manage the per-user mechanism (HKCU `Run` value / `~/Library/LaunchAgents/com.stirlingmarketinggroup.folder-opener.plist` / `~/.config/autostart/folder-opener.desktop`); the MSI and deb install machine-wide autostart on their own. The Windows release binary is built with `-H windowsgui` so it never shows a console, which also means the `autostart` CLI subcommands print nothing on Windows.

The port can be overridden with the `FOLDER_OPENER_PORT` environment variable (default `29101`); the client takes a matching `port` option.

## Development

```bash
go run .                          # run server + tray
go test ./...                     # tests (handler + validation only; they never pop a real window)
cd packages/client && npm ci && npm run build   # build the npm client
```

Releases: push a `v*` tag; GitHub Actions builds Windows (amd64/arm64), Linux (amd64/arm64), and a macOS universal binary, attaches them to the release, and publishes the npm client at the tag's version.

npm publishing uses [trusted publishing](https://docs.npmjs.com/trusted-publishers) (GitHub Actions OIDC) — no token to rotate; the trusted publisher is configured on the npm package settings for this repo's `release.yml`.

Release secrets (org-level, shared with Dazzle; the signing/notarization steps are skipped when missing):

| Secret | Purpose |
|---|---|
| `APPLE_CERTIFICATE`, `APPLE_CERTIFICATE_PASSWORD`, `APPLE_SIGNING_IDENTITY` | Developer ID code signing of the macOS binary |
| `APPLE_ID`, `APPLE_PASSWORD`, `APPLE_TEAM_ID` | Notarization via `notarytool` |

With signing + notarization in place, the macOS binary runs without any Gatekeeper "Open Anyway" dance; without them, unsigned builds need a right-click → Open (or `xattr -d com.apple.quarantine`) on first launch.
