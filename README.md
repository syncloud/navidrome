# Navidrome for Syncloud

[Navidrome](https://github.com/navidrome/navidrome) music streaming server packaged as a
[Syncloud](https://syncloud.org) app. Subsonic/OpenSubsonic API compatible, so it works with
the existing client ecosystem (Symfonium, Amperfy, play:Sub, DSub, Feishin, …).

Tracks [syncloud/platform#741](https://github.com/syncloud/platform/issues/741).

## Architecture

A small Go gateway (`backend/`) sits in front of Navidrome and bridges Syncloud authentication
to Navidrome's externalized (reverse-proxy header) auth. Navidrome listens on a unix socket with
`ND_EXTAUTH_USERHEADER=Remote-User` and `ND_EXTAUTH_TRUSTEDSOURCES=@`; the gateway is the only
thing that talks to it and injects a trusted `Remote-User` header after authenticating the caller.

```
platform nginx ──▶ web.socket ──▶ nginx ──▶ backend.sock ─┬─▶ navidrome.sock
                                                           │   (Remote-User: <user>)
  Web UI (/...)        → OIDC (Authelia) → session cookie ─┘
  Subsonic (/rest/*)   → LDAP bind (platform slapd) of the
                         client-supplied user+password ────┘   (token-auth clients fall
                                                                 through to Navidrome native)
```

- **Web UI**: OpenID Connect against the platform's Authelia. Seamless single sign-on with the
  rest of the Syncloud dashboard.
- **Subsonic / mobile apps**: the gateway validates the username/password the client sends against
  the platform LDAP, so users log in with their **Syncloud credentials**. Set the mobile client to
  send the password as plaintext / BasicAuth (the platform terminates HTTPS) — the Subsonic *token*
  scheme (`md5(password+salt)`) can't be verified against LDAP and falls through to Navidrome's
  native auth.

## Layout

| Path | What |
|---|---|
| `backend/` | Go auth gateway (OIDC + LDAP-Subsonic + reverse proxy) |
| `cli/` | Cobra install/configure/refresh hooks + `bin/cli` lifecycle commands |
| `navidrome/` | downloads & vendors the upstream Navidrome binary |
| `nginx/` | static nginx build, fronts `web.socket` |
| `config/` | templated `nginx.conf`, `oidc.env` |
| `test/` | pytest integration tests |
| `web/e2e/` | Playwright UI tests |

## Upstream version

Pinned in `.drone.jsonnet` (`local version = '...'`).

## Build

CI builds via `.drone.jsonnet` on Drone. Locally each step is a script: `./nginx/build.sh`,
`./navidrome/build.sh <version>`, `./backend/build.sh`, `./cli/build.sh`, then
`./package.sh navidrome <build-number>`.
