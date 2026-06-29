# Navidrome for Syncloud

[Navidrome](https://github.com/navidrome/navidrome) music streaming server packaged as a
[Syncloud](https://syncloud.org) app. Subsonic/OpenSubsonic API compatible, so it works with
the existing client ecosystem (Symfonium, Amperfy, play:Sub, DSub, Feishin, …).

Tracks [syncloud/platform#741](https://github.com/syncloud/platform/issues/741).

## Authentication

- **Web UI** — OpenID Connect single sign-on against the platform's Authelia, the same login as
  the rest of the Syncloud dashboard.
- **Subsonic / mobile apps** — log in with your **Syncloud username and password** (validated
  against the platform LDAP). Set the client to send the password as plaintext / BasicAuth (safe
  over the platform's HTTPS); the Subsonic token scheme can't be verified against LDAP and falls
  back to Navidrome's native login.

## Upstream version

Pinned in `.drone.jsonnet` (`local version = '...'`).
