# Navidrome for Syncloud

[Navidrome](https://github.com/navidrome/navidrome) music streaming server packaged as a
[Syncloud](https://syncloud.org) app. Subsonic/OpenSubsonic API compatible, so it works with
the existing client ecosystem (Symfonium, Amperfy, play:Sub, DSub, Feishin, …).

Tracks [syncloud/platform#741](https://github.com/syncloud/platform/issues/741).

## Authentication

nginx authenticates every request against the platform's Authelia (`auth_request`) and passes the
user to Navidrome via the trusted `Remote-User` header — no app-specific auth code.

- **Web UI** — Authelia single sign-on, the same login as the rest of the Syncloud dashboard.
- **Subsonic / mobile apps** — log in with your **Syncloud username and password** using the
  client's **HTTP Basic auth** option (validated by Authelia). Open the Navidrome web UI once so
  your account is created, then mobile clients work. The Subsonic *token* scheme can't be verified
  by Authelia, so use Basic auth.

## Adding music

Navidrome has no upload UI by design — it serves an existing on-disk library. Copy your
music into `/data/navidrome` on the device (e.g. over
[SFTP / remote file access](https://github.com/syncloud/platform/wiki/Remote-file-access));
Navidrome scans it automatically.

## Upstream version

Pinned in `.drone.jsonnet` (`local version = '...'`).
