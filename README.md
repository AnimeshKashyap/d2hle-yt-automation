# d2hle-yt-automation

Go service that uploads videos to **YouTube Studio** via **Playwright** and a persistent Chrome profile per channel. Exposes an HTTP API for a parent job queue; includes a CLI for login and debugging.

**API reference:** start the server and open `http://127.0.0.1:8081/v1/docs` (or `python scripts/test_upload.py docs`).

More design detail: [project_scope.md](project_scope.md).

## Prerequisites

- **Go 1.22+**
- **Google Chrome** installed (recommended; set `BROWSER_CHANNEL=chrome` in env)
- **Playwright browser drivers** (once per machine):

```bash
go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6201.1 install chromium
```

- **Python 3** (optional) for `scripts/test_upload.py` — stdlib only, no pip packages

## Configuration

Copy [.env.example](.env.example) to `.env` beside the binary or in the directory you run from (`.env` is gitignored; the example file uses placeholder paths only).

Every command (`serve`, `profile …`, `job …`) loads configuration on startup. Precedence (highest wins):

1. **`.env` file** — if present (`./.env` in the current working directory, otherwise `.env` next to the executable)
2. **OS environment variables**
3. **Built-in defaults** in code (see `.env.example`)

Important defaults for local dev:

| Variable | Typical local value |
|----------|---------------------|
| `DATA_ROOT` | `./data` |
| `HTTP_HOST` | `127.0.0.1` (use `0.0.0.0` for LAN) |
| `HTTP_PORT` | `8081` |
| `ALLOWED_MEDIA_ROOTS` | `./data/media` |
| `UPLOAD_HEADLESS` | `false` (match how you logged in) |
| `SESSION_CHECK_HEADLESS` | `false` |
| `SESSION_REFRESH_HEADLESS` | `false` (morning session warm-up) |

Video paths in API/CLI must resolve under `ALLOWED_MEDIA_ROOTS` on the **server** host.

**Session refresh** keeps the Google/YouTube **browser profile** (cookies in `userDataDir`) warm. It is not an OAuth refresh token. Run it headed once per day before unattended uploads.

## Build

```bash
go build -o bin/d2hle-yt ./cmd/d2hle-yt
```

On Windows: `bin\d2hle-yt.exe`.

## Run the HTTP server

```bash
./bin/d2hle-yt serve
```

On Windows: `bin\d2hle-yt.exe serve` (place `.env` in the same folder or `cd` there first).

## CLI — profiles

```bash
# Register a profile (isolated browser user data under DATA_ROOT/state/profiles/)
d2hle-yt profile create --id main --name "My channel"

# List profiles and session status
d2hle-yt profile list

# Headed Google/YouTube login (complete MFA in the browser window)
d2hle-yt profile login --id main

# Probe Studio session (updates session_status)
d2hle-yt profile check --id main

# Refresh session (headed browser; Studio + youtube.com, sets last_refreshed_at)
d2hle-yt profile refresh --id main
```

## CLI — jobs

Upload jobs are normally created via **POST `/v1/jobs`** (async on the server). To run a stored job synchronously in the foreground (debug):

```bash
d2hle-yt job run --id job_<id>
```

## Test script (`scripts/test_upload.py`)

Environment:

- `YT_UPLOAD_BASE_URL` — default `http://127.0.0.1:8081`
- `API_TOKEN` — optional `Authorization: Bearer …` if `API_TOKEN` is set on the server

```bash
# Health and API catalog
python scripts/test_upload.py health
python scripts/test_upload.py ready
python scripts/test_upload.py docs

# Profiles
python scripts/test_upload.py profiles
python scripts/test_upload.py profile --profile-id main
python scripts/test_upload.py session-check --profile-id main
python scripts/test_upload.py session-refresh --profile-id main

# Jobs
python scripts/test_upload.py job-current
python scripts/test_upload.py job --job-id job_<id>

# Upload (POST /v1/jobs, then poll until published/failed)
python scripts/test_upload.py upload \
  --video "./data/media/Test Video.mp4" \
  --title "Test upload" \
  --tags "dota2,test" \
  --visibility private \
  --profile-id main

# Accept job without polling
python scripts/test_upload.py upload --video "./data/media/sample.mp4" --no-poll

# Long uploads: increase poll window (seconds)
python scripts/test_upload.py upload --video "./data/media/big.mp4" --poll-timeout 7200
```

Optional upload flags: `--description`, `--thumbnail <path>`, `--made-for-kids`, `--idempotency-key`, `--poll-interval`.

## Typical first-time flow

1. Copy `.env.example` to `.env` (Chrome channel, `DATA_ROOT`, media roots).
2. `go build -o bin/d2hle-yt ./cmd/d2hle-yt` and install Playwright drivers (above).
3. `d2hle-yt profile create --id main`
4. `d2hle-yt profile login --id main` — sign in in the opened browser.
5. `d2hle-yt profile check --id main` — expect `active`.
6. `d2hle-yt serve`
7. Put a video under `data/media/` and run `python scripts/test_upload.py upload ...`

## Headless vs headed

Use the **same** headless mode for login, session check, and upload when possible. Headless upload with a profile only validated in headed mode often fails at `open_studio` (30s step timeout, job may report `needs_auth`).

## Tests

```bash
go test ./...
```
