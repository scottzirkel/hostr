# Roadmap

Tracking release status and future work. Order within sections is rough
priority, not commitment.

## Pending release

- **Doctor cutover wording** — `routa doctor` now reports cutover state using
  user-facing descriptions instead of internal phase labels.
- **Routing path resolution coverage** — focused tests now cover path
  resolution precedence between broader linked parents and auto-discovered
  tracked children.

## Released

### v1.8.1 — PHP public docroot detection

- **PHP public front controllers** — site auto-detection now treats
  `public/index.php` as a PHP docroot without requiring `composer.json`,
  covering non-Composer PHP apps and custom front-controller layouts.

### v1.8.0 — diagnostics, PHP tooling, and Arch packaging

- **Optional service diagnostics** — `routa doctor` adds detail for optional
  services with missing binaries, occupied ports, failed units, runtime library
  failures, and mismatched database runtimes.
- **Arch packaging path** — release builds can produce Linux `amd64`/`arm64`
  archives, GitHub releases can attach those artifacts, and AUR metadata for
  `routa-bin` lives under `packaging/aur/routa-bin/`.
- **PHP debugging toggle** — `routa php xdebug on/off/status <version>` manages
  per-version Xdebug ini settings when the installed PHP build includes Xdebug,
  and Xdebug-capable installs default to off.
- **Optional service proxy helpers** — search services and MinIO console can be
  registered as `.test` proxies with service-aware default ports.
- **PHP-FPM reliability** — PHP downloads retry interrupted transfers,
  `routa restart php [version]` targets PHP-FPM directly, generated FPM env
  values are quoted safely, and `.env` references such as
  `${FORWARD_REDIS_PORT}` expand before pool config rendering.

### v1.7.0 — optional service dashboard actions

- **TUI service visibility** — the dashboard inspector shows installed optional
  services with configured ports, data directories, and active/inactive state.
- **TUI service actions** — optional services can be selected in the dashboard
  and started/stopped or restarted after confirmation.

### v1.6.0 — managed MySQL services and DB instances

- **Managed MySQL runtime** — MySQL uses routa-owned Oracle MySQL archives under
  `~/.local/share/routa/binaries/mysql/` and rejects MariaDB-compatible
  `mysqld` binaries for `routa db ... mysql`.
- **Named MySQL instances** — MySQL instances are isolated by version and
  optional project name with their own data, config, sockets, and ports.
- **Database credentials** — MySQL application credentials can be saved and
  applied for installed or running instances.
- **Service restart coverage** — `routa restart` includes active optional
  services alongside DNS, Caddy, and PHP-FPM.

### v1.5.0 — versioned optional services

- **Versioned databases** — MariaDB and Postgres can be installed, started,
  stopped, listed, and inspected as systemd user services.
- **Search services** — Meilisearch and Typesense run as version-isolated
  systemd user services with install/start/stop/status/list commands.
- **Object storage** — MinIO runs as a version-isolated local S3-compatible
  service with configurable API and console ports.
- **Redis and Mailpit** — Redis and Mailpit user services remain the simple
  single-instance service slices, with Mailpit optionally proxied as `.test`.

### v1.4.0 — aliases, tracked roots, and PHP env pools

- **`routa alias <existing> <new>`** — registers additional `.test` hostnames
  that resolve through the target site's source, proxy, PHP, root, and HTTPS
  config. `routa unalias <name>` removes them.
- **Tracked-dir default root** — `routa track --root <path>` applies a shared
  docroot override to every immediate child of a tracked dir.
- **Per-site env file passthrough** — PHP sites with a project `.env` get a
  generated PHP-FPM pool and per-site socket with `env[FOO] = bar` entries.
- **Routing edge coverage** — added focused coverage for tracked-root overrides,
  explicit-link precedence, and alias chains.

### v1.3.0 — process-backed dev apps

- **`routa dev`** — starts a detected project dev server, waits for the port,
  and registers a reverse proxy under `.test`.
- **Dev-server detection** — supports package manager `dev` scripts, Rails,
  Phoenix, and Django defaults.
- **Manual process support** — accepts explicit command, name, and port options
  for apps that do not fit a built-in detector.
- **Proxy behavior** — reverse proxy rendering now includes WebSocket-friendly
  forwarding headers for HMR and other upgraded connections.

### v1.2.0 — routa rename and site tracking polish

- **Project rename** — completed the hostr-to-routa command, path, service, and
  documentation rename.
- **Track/untrack language** — `routa track` and `routa untrack` are now the
  primary commands, with `park` and `unpark` kept as Valet-compatible aliases.
- **Ignored tracked sites** — `routa ignore` and `routa unignore` hide or
  restore auto-discovered tracked subdirectories.
- **Static site detection** — static `public/` directories are detected, and
  static SPA routing falls back to `index.html`.

### v1.1.0 — interactive dashboard

- **Bare `routa` opens the TUI** — the dashboard is now the default entrypoint.
- **Site inspection** — the TUI has a split inspector, health strip, live probe
  status, and log previews.
- **Navigation controls** — filters, sorting, collapsible subdomain groups,
  help prompts, and selected-site actions are available inline.
- **Compatibility** — `routa tui` remains available as a hidden alias.

### v1.0.0 — stable Linux local dev server

Goal: make the Linux-focused workflow stable, recoverable, and supportable enough
to treat the CLI and config shape as a real contract. This milestone was not
trying to become a full-stack desktop dev suite.

- **Installation and rollback confidence**
  - `routa install` now checks required commands before side effects and has
    pure unit-rendering coverage.
  - `routa uninstall --purge` has helper coverage for purge scope and PHP-FPM
    unit discovery.
  - Cutover/rollback now has partial-state helper coverage and sudo block
    ordering checks.
  - `routa init` now treats missing required dependencies as blocking
    diagnostics instead of reporting a pass before `routa install` fails.
  - `v0.7.0` tightened prerequisite diagnostics and `routa doctor` service
    failure output.
  - `v0.5.1` added baseline hardening for proxy target validation,
    PHP-FPM cleanup during uninstall, safer rollback resolver restoration,
    existing systemd-resolved detection, and cutover refusal when no
    systemd-networkd `.network` files are available.
  - Document the required host assumptions: systemd user services,
    systemd-resolved, systemd-networkd `.network` files for per-link routing,
    Caddy, and p11-kit trust store behavior.
- **Config/schema stability**
  - Treat `~/.config/routa/state.json` as a stable contract.
  - Current state files are versioned; future shape changes
    require explicit migrations instead of silent guessing.
- **Core routing correctness**
  - Custom roots, linked-site overrides, secure toggle rendering, and
    missing-docroot status output now have focused coverage.
  - Proxy targets now validate before state is saved or Caddy fragments render.
  - Site detection, parked directory resolution, proxy target validation, and
    Caddy fragment rendering have focused coverage for the v1 contract.
- **Migration reliability**
  - Missing/malformed config, relative symlinks, quoted Nginx roots, whitespace,
    custom roots, and isolated PHP versions now have focused coverage.
- **Supportability**
  - Service failure diagnostics now preserve `systemctl` error details in
    `routa doctor`.
  - DNS failures now preserve raw query details in `routa doctor`.
  - Cert trust errors now name the missing Caddy root or failed `trust anchor`
    action with a p11-kit/system trust store hint.
  - Port diagnostics now call out likely ownership conflicts when HTTPS ports
    are bound while `routa-caddy` is not active.
- **Distribution**
  - Current policy: GitHub releases are source/tag-only until a binary artifact
    policy is chosen.
  - Tagged releases with proper semver; `routa version` already prints
    `git describe`.
- **Docs pass**
  - README troubleshooting covers install, migration, rollback, DNS, port,
    certificate, and source/tag-only release behavior.
  - Command help covers the v1 workflows that should be usable without reading
    implementation details.

## Near-term (small, well-scoped)

- **More routing edge coverage** — keep adding unusual tracked-dir, linked-site,
  proxy, dev-server, and path-combination cases as they appear.

## Mid-term

- **Distribution**
  - Publish and maintain the AUR package (`routa-bin`) after the first binary
    release artifacts are attached.
- **Optional service polish**
  - Per-service proxy helpers where they make sense, such as named Mailpit
    inboxes.
  - Consistent lifecycle output and list/status formatting across Redis,
    Mailpit, databases, search, and storage.
  - Backup/export guidance for stateful service data.
- **PHP extension variants** — `routa php ext list <ver>` exists today for the
  compiled-in upstream bulk profile. Add finer-grained variant selection or
  custom static-php-cli builds for users who need a different extension set.
- **Xdebug variants** — install xdebug-enabled PHP variants alongside the
  default bulk profile for users whose installed PHP build does not include
  Xdebug.

## Next logical steps

1. **PHP debugging workflow** — after packaging/service polish, tackle xdebug
   and extension variants as a focused PHP developer-experience milestone.
2. **Optional service polish** — keep smoothing lifecycle output and proxy
   helpers for search dashboards, MinIO console, and named Mailpit inboxes.

## Backlog / ideas

- **More TLD support** — currently hardcoded `.test`. Allow `.localhost` or arbitrary local TLDs.
- **Multi-host (LAN sharing)** — bind routa-caddy to LAN IP, have other
  devices on the network resolve `*.test` against your machine. Useful for
  testing on phones/tablets.
- **Caddy admin API integration** — drive site changes via the admin API instead
  of file fragments + reload (faster, atomic).
- **Plugin / driver system** — Laravel-style "drivers" for unusual project
  layouts so the auto-detect can be extended without touching core.
- **Web dashboard** — small local web UI (in addition to TUI) for users who prefer a browser.
- **macOS support** — most of the stack (Caddy, php-fpm, miekg/dns) is portable; the resolver bits are Linux-specific. Not a near-term priority.

## Won't do

- **GUI app** — explicit project rejection from day one.
- **Auto-updating the binary in place** — leave to OS package managers (AUR, brew, deb, rpm) and `git pull && bash install.sh`.
