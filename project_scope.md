# YouTube Studio upload service (d2hle-yt-automation)

Lightweight **Go** service using **Playwright + Chromium** to automate YouTube Studio video uploads. Intended as a **git submodule** consumed by a parent project via **docker-compose**; the parent owns job queues and calls this service over HTTP.

## Stack

- **Go** + [playwright-go](https://github.com/playwright-community/playwright-go)
- **Chromium** (Playwright-managed; Docker image installs via Playwright)
- **Persistent browser profile** per configured channel (`userDataDir` per profile)
- **Concurrency**: default `1`, configurable (`MAX_CONCURRENT_JOBS`)
- **Job state**: JSON files on mounted volume (v1)
- **HTTP API** primary interface; optional CLI for ops (login bootstrap, debug)

## Goals

- Playwright + Chromium only; no PyAutoGUI, screen coordinates, or image matching.
- DOM/state-based selectors and explicit waits (load state, visibility, upload progress, processing, publish confirmation).
- Persistent profiles so Google/YouTube login survives container restarts.
- Multi-profile design (generic `profile_id`); do not hardcode a single channel.
- Each profile gets an isolated `userDataDir` under persistent storage.
- Headless/headed configurable; Docker runs headless by default.
- One persistent browser context per active job; always close pages/context cleanly.
- No duplicate queue inside this service—the **parent consumer** queues work; this service executes one upload at a time (by default) and reports state via API.
- Stable long-running process; avoid leaks and unbounded in-memory state.
- Configuration via environment variables and optional config file.
- On failure: structured logs, optional Playwright trace, screenshot artifact paths in job record.

## Deployment model (docker-compose submodule)

- Parent runs **LocalStack (S3)** and/or a shared media volume; this service supports **either** reference style (see Video sources).
- Browser profiles and job JSON live on a **dedicated persistent volume** (e.g. `/data/state/profiles`, `/data/state/jobs`).
- Ephemeral **staging** directory on disk (e.g. `/data/staging/{job_id}/`) for S3-sourced files; always deleted after job terminal state (success or failure).
- Do **not** stream multi-GB bodies through the upload HTTP API; use path or S3 reference instead.

### Video sources (one canonical copy)

Parent should keep **one canonical object** per render (typically **S3 in LocalStack**). This service never holds a second long-term copy.

| Source | When to use | Behavior |
|--------|-------------|----------|
| **`local_path`** | Parent already bind-mounts media into this container | Playwright `SetInputFiles` reads file directly; no extra persistent copy |
| **`s3`** | Canonical file is only in S3 (post-MVP; in scope, implement after shared mount path) | Service downloads via **presigned GET** or S3 API (LocalStack endpoint + env credentials), **streaming `io.Copy` to staging file** (not into RAM), then upload to Studio, then **delete staging** |

**MVP default:** parent uses **shared bind-mount** + `video.local_path` only.

- Thumbnail: same pattern (`local_path` or `s3`).
- Memory: at most one active staging download per job; respect `MAX_CONCURRENT_JOBS` so peak disk = N × file size on staging volume, not RAM.
- Optional: parent passes presigned URL in job payload; alternatively service generates GET from bucket/key using configured LocalStack endpoint (parent and service agree on bucket layout).

## HTTP API (v1)

### Profiles

Each `profile_id` maps to an isolated Chromium **`userDataDir`** (e.g. `/data/state/profiles/{profile_id}/`). Concurrent jobs on **different** profiles may run in parallel when `MAX_CONCURRENT_JOBS` > 1; never two jobs on the same profile at once.

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/profiles` | Register profile (`profile_id`, display name); creates profile directory |
| `GET` | `/v1/profiles` | List profiles with **session_status** enum (for UI / parent) |
| `GET` | `/v1/profiles/{profile_id}` | Profile detail + session status + last check time |
| `POST` | `/v1/profiles/{profile_id}/tokens` | Issue **service API token** for this profile (parent automation); store hash only |
| `DELETE` | `/v1/profiles/{profile_id}/tokens/{token_id}` | Revoke service token |
| `POST` | `/v1/profiles/{profile_id}/session/check` | Studio probe: open Studio, update **session_status** (no upload) |
| `POST` | `/v1/profiles/{profile_id}/session/refresh` | Headed session warm-up (Studio + youtube.com); sets `last_refreshed_at` — cookie profile, not OAuth |
| `POST` | `/v1/profiles/{profile_id}/login` | Start Google/YouTube bootstrap (MVP: manual completion; see Authentication) |
| `GET` | `/v1/profiles/{profile_id}/status` | Same as session status summary (alias for convenience) |

**Profile `session_status` (enum)** — Google session in `userDataDir`, not a JWT we mint:

| Status | Meaning |
|--------|---------|
| `unconfigured` | Profile record exists; no successful login yet |
| `active` | Studio reachable; signed in |
| `expiring_soon` | Optional: heuristic (e.g. re-auth prompt seen recently) |
| `needs_reauth` | Session invalid; uploads blocked until login flow completes |
| `needs_user_action` | CAPTCHA, unusual challenge, or MFA step blocking automation |
| `login_in_progress` | Bootstrap or re-login flow started, not finished |
| `check_failed` | Probe error (browser crash, timeout); retry check |

**Service API tokens** (this microservice): bearer token per profile (or scoped global) so the parent consumer can call upload APIs without exposing the service on the public internet. These are **not** YouTube OAuth tokens; YouTube auth remains cookie-based in `userDataDir`.

### Upload jobs

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/jobs` | Submit upload (see payload); returns `job_id` |
| `GET` | `/v1/jobs/{job_id}` | Job state, step, errors, artifact paths |
| `GET` | `/v1/jobs/current` | Active job if any (for parent polling) |
| `GET` | `/v1/docs` | List all configured API routes (discovery) |
| `GET` | `/v1/health` | Liveness |
| `GET` | `/v1/ready` | Ready if browser slot available (optional for parent) |

**Busy behavior**: If `MAX_CONCURRENT_JOBS` is saturated, respond **`409 Conflict`** with body describing active `job_id`. Parent retries with backoff. No second persisted queue in this service (parent is source of truth). Optional: `GET /v1/jobs/current` for explicit polling.

**Idempotency**: Parent may send `Idempotency-Key` header (or `client_request_id` in body) to avoid duplicate uploads on retry; store mapping in job JSON.

### Upload job payload

Exactly one video source; same for thumbnail (optional).

```json
{
  "profile_id": "main",
  "video": {
    "local_path": "/data/media/out/render.mp4"
  },
  "title": "...",
  "description": "...",
  "tags": ["dota2", "..."],
  "thumbnail": { "s3": { "bucket": "renders", "key": "out/thumb.jpg" } },
  "visibility": "private|unlisted|public",
  "schedule_at": "2026-09-27T18:00:00+05:30",
  "category": "gaming",
  "game_title": "Dota 2",
  "audience": { "made_for_kids": false }
}
```

S3 variant (presigned or bucket/key):

```json
"video": {
  "s3": {
    "bucket": "renders",
    "key": "out/render.mp4",
    "presigned_get_url": "https://..."
  }
}
```

MVP implements core fields first; category, game selection (gaming), and made-for-kids are required in the model and implemented in Studio wizard as soon as selectors are stable.

### Path validation

- Reject missing files, directories, and **zero-byte** video.
- Restrict paths to configured **`ALLOWED_MEDIA_ROOTS`** (e.g. `/data/media`) to prevent path traversal.

## Architecture

```
UploadJob          — input + persisted state machine
ProfileSession     — profile_id, persistent_profile_path, auth metadata
YouTubeStudioAdapter — Playwright-only Studio automation
BrowserLauncher    — LaunchPersistentContext, timeouts, teardown
```

Keep business logic out of Playwright selectors (adapter behind interfaces).

### Job states

`accepted` → `running` → (`awaiting_auth` if needed) → `processing` → `published` | `failed` | `cancelled`

Expose `current_step` string for parent logs (e.g. `uploading_file`, `metadata`, `checks`, `publish`).

## Authentication

- Never bypass CAPTCHA, MFA, or Google security.
- If session expired mid-job: set profile **`needs_reauth`**, job **`awaiting_auth`**, stop automation; parent surfaces “login again” from `GET /v1/profiles`.
- **MVP**: user completes login when bootstrap or re-auth is triggered (`POST .../login`); MFA via phone/email as today.
- **Google sign-in**: use **installed Google Chrome** (`BROWSER_CHANNEL=chrome`), not Playwright-bundled Chromium alone—Google blocks browsers it treats as automation ([supported browser help](https://support.google.com/accounts/answer/7675428?hl=en)). Login uses stealth launch flags; cookies live only in the per-profile `userDataDir`, not your normal Chrome profile.
- **Later (home automation, not MVP)**: optional credential-assisted re-login—password from parent **secret store** (env/vault/LocalStack Secrets Manager pattern), automated navigation to Google sign-in, then **block until MFA push approved** (detect post-login Studio URL). High fragility and account-risk; feature-flagged. No stored Google password in this repo.
- **Proactive**: daily `POST .../session/refresh` (headed on host) or `POST .../session/check` before unattended uploads; catches `needs_reauth` before the consumer wastes a queue slot.
- **Docker steady state**: headless upload per profile `userDataDir`; bootstrap may use headed compose profile or host CLI on same profile volume.

## Errors

Classify: retryable (transient UI/network), non-retryable (bad file, validation), user-action (auth). Parent decides retry policy; this service returns structured error codes.

## Testing

- Unit: validation, state machine, config.
- Integration (tagged): Studio flow with real profile on a test channel.
- Keep dependencies and repo structure small.

## Personal use

Operator accepts YouTube/Google terms and account risk. Profiles and traces are sensitive; backup `DATA_ROOT` accordingly.

## Network exposure

- Default: HTTP listens on **localhost** inside compose network only; parent reaches service by service name.
- If a host port is mapped for debugging, require **service API token** on mutating routes.
