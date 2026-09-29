#!/usr/bin/env python3
"""
HTTP integration helper for d2hle-yt-automation (replaces manual curl).

Examples:
  python scripts/test_upload.py health
  python scripts/test_upload.py profiles
  python scripts/test_upload.py upload --video ./data/media/sample.mp4 --title "Test" --visibility private

Environment (optional):
  YT_UPLOAD_BASE_URL   default http://127.0.0.1:8081
  API_TOKEN            sent as Authorization: Bearer ...
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


def base_url() -> str:
    return os.environ.get("YT_UPLOAD_BASE_URL", "http://127.0.0.1:8081").rstrip("/")


def api_request(
    method: str,
    path: str,
    body: dict[str, Any] | None = None,
    idempotency_key: str | None = None,
    timeout: float = 300.0,
) -> tuple[int, Any]:
    url = f"{base_url()}{path}"
    data = None
    headers = {"Accept": "application/json"}
    token = os.environ.get("API_TOKEN", "").strip()
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if idempotency_key:
        headers["Idempotency-Key"] = idempotency_key

    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            if not raw:
                return resp.status, None
            return resp.status, json.loads(raw)
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8")
        try:
            payload = json.loads(raw) if raw else {"message": e.reason}
        except json.JSONDecodeError:
            payload = {"message": raw or e.reason}
        return e.code, payload


def pp(obj: Any) -> None:
    print(json.dumps(obj, indent=2, sort_keys=True))


def cmd_health(_: argparse.Namespace) -> int:
    code, data = api_request("GET", "/v1/health")
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def cmd_ready(_: argparse.Namespace) -> int:
    code, data = api_request("GET", "/v1/ready")
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def cmd_docs(_: argparse.Namespace) -> int:
    code, data = api_request("GET", "/v1/docs")
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def cmd_profiles(_: argparse.Namespace) -> int:
    code, data = api_request("GET", "/v1/profiles")
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def cmd_profile_status(args: argparse.Namespace) -> int:
    code, data = api_request("GET", f"/v1/profiles/{args.profile_id}")
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def cmd_session_check(args: argparse.Namespace) -> int:
    code, data = api_request("POST", f"/v1/profiles/{args.profile_id}/session/check")
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def cmd_session_refresh(args: argparse.Namespace) -> int:
    code, data = api_request(
        "POST",
        f"/v1/profiles/{args.profile_id}/session/refresh",
        timeout=900.0,
    )
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def poll_job(job_id: str, interval: float, timeout: float) -> int:
    deadline = time.time() + timeout
    terminal = {"published", "failed", "cancelled", "awaiting_auth"}
    while time.time() < deadline:
        code, data = api_request("GET", f"/v1/jobs/{job_id}")
        if code != 200:
            print(f"poll HTTP {code}")
            pp(data)
            return 1
        job = data if isinstance(data, dict) and "job_id" in data else data.get("job", data)
        status = job.get("status", "")
        step = job.get("current_step", "")
        print(f"[{time.strftime('%H:%M:%S')}] status={status} step={step}")
        if status in terminal:
            print("\nFinal job:")
            pp(job)
            if job.get("screenshot_path") or job.get("trace_path"):
                print("\nArtifacts:")
                print(f"  screenshot: {job.get('screenshot_path')}")
                print(f"  trace:      {job.get('trace_path')}")
            return 0 if status == "published" else 1
        time.sleep(interval)
    print("poll timed out", file=sys.stderr)
    return 1


def cmd_job_get(args: argparse.Namespace) -> int:
    code, data = api_request("GET", f"/v1/jobs/{args.job_id}")
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def cmd_job_current(_: argparse.Namespace) -> int:
    code, data = api_request("GET", "/v1/jobs/current")
    print(f"HTTP {code}")
    pp(data)
    return 0 if code == 200 else 1


def cmd_upload(args: argparse.Namespace) -> int:
    video = Path(args.video).resolve()
    if not video.is_file():
        print(f"video not found: {video}", file=sys.stderr)
        return 1
    if video.stat().st_size == 0:
        print(f"video is empty: {video}", file=sys.stderr)
        return 1

    payload: dict[str, Any] = {
        "profile_id": args.profile_id,
        "video": {"local_path": str(video)},
        "title": args.title,
        "description": args.description,
        "tags": [t.strip() for t in args.tags.split(",") if t.strip()],
        "visibility": args.visibility,
        "audience": {"made_for_kids": args.made_for_kids},
    }
    if args.thumbnail:
        thumb = Path(args.thumbnail).resolve()
        payload["thumbnail"] = {"local_path": str(thumb)}

    code, data = api_request("POST", "/v1/jobs", body=payload, idempotency_key=args.idempotency_key)
    print(f"HTTP {code}")
    pp(data)
    if code not in (200, 202):
        return 1

    job = data.get("job") if isinstance(data, dict) else None
    if not job:
        return 0
    job_id = job.get("job_id")
    if not job_id or args.no_poll:
        return 0
    print(f"\nPolling job {job_id} ...\n")
    return poll_job(job_id, args.poll_interval, args.poll_timeout)


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="Test client for d2hle-yt-automation API")
    sub = p.add_subparsers(dest="command", required=True)

    sub.add_parser("health", help="GET /v1/health").set_defaults(func=cmd_health)
    sub.add_parser("ready", help="GET /v1/ready").set_defaults(func=cmd_ready)
    sub.add_parser("docs", help="GET /v1/docs").set_defaults(func=cmd_docs)
    sub.add_parser("profiles", help="GET /v1/profiles").set_defaults(func=cmd_profiles)

    ps = sub.add_parser("profile", help="GET /v1/profiles/{id}")
    ps.add_argument("--profile-id", default="main")
    ps.set_defaults(func=cmd_profile_status)

    sc = sub.add_parser("session-check", help="POST /v1/profiles/{id}/session/check")
    sc.add_argument("--profile-id", default="main")
    sc.set_defaults(func=cmd_session_check)

    sr = sub.add_parser(
        "session-refresh",
        help="POST /v1/profiles/{id}/session/refresh (headed cookie warm-up on server)",
    )
    sr.add_argument("--profile-id", default="main")
    sr.set_defaults(func=cmd_session_refresh)

    jc = sub.add_parser("job-current", help="GET /v1/jobs/current")
    jc.set_defaults(func=cmd_job_current)

    jg = sub.add_parser("job", help="GET /v1/jobs/{id}")
    jg.add_argument("--job-id", required=True)
    jg.set_defaults(func=cmd_job_get)

    up = sub.add_parser("upload", help="POST /v1/jobs and poll until done")
    up.add_argument("--profile-id", default="main")
    up.add_argument("--video", required=True, help="Path to video (must be under ALLOWED_MEDIA_ROOTS on server)")
    up.add_argument("--thumbnail", default="", help="Optional thumbnail path")
    up.add_argument("--title", default="Test upload", help="Video title (not tags)")
    up.add_argument("--description", default="Uploaded via scripts/test_upload.py")
    up.add_argument("--tags", default="dota2,test", help="Comma-separated; filled in Tags field only")
    up.add_argument("--visibility", choices=("private", "unlisted", "public"), default="private")
    up.add_argument("--made-for-kids", action="store_true")
    up.add_argument("--idempotency-key", default="", help="Optional Idempotency-Key header")
    up.add_argument("--no-poll", action="store_true", help="Return after job accept without polling")
    up.add_argument("--poll-interval", type=float, default=5.0)
    up.add_argument("--poll-timeout", type=float, default=7200.0, help="Seconds (default 2h for large files)")
    up.set_defaults(func=cmd_upload)

    return p


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
