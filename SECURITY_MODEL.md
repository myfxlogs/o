# Threat model: myfxlogs/o (O arbitrage fund system)

_Threat model for Devin security scans (profile "o — threat-model scan"). Owner-confirmed facts are marked "owner"; the rest is drawn from the code. Edit freely; the next scan starts from this file._

## 1. System overview & trust boundaries

O is a single-owner automated cross-market arbitrage/carry fund: it discovers opportunities, sizes and executes trades on OKX (CEX), MT5 (CFD via mtapi gRPC gateway) and TRON (on-chain voting/rewards), and keeps a ledger. Real funds are at stake.

Hosts / trust zones (see `docs/数据流.md`, `docker-compose*.yml`, `deploy/`):

- **SG production host**: `o-core` container (Go, `cmd/core`) and `o-postgres` (PG-fin, `127.0.0.1:15433`). `o-core` holds the master key file (`./secrets/o_master_key` → `/run/secrets/o_master_key`), decrypts `app_secrets` and places real orders.
- **OCI collection box**: `o-collecto` (`cmd/collector`), `o-research` (`cmd/research`), projector/sweeper, PG-md (`:5435`). Reaches SG via SSH tunnels with a restricted `tunnel@` key (`deploy/o-collecto/tunnel.service`, `deploy/o-readcred-tunnel.service`: `permitopen` 15433 and 8095).
- **Dev box**: builds images and ships them to SG over root SSH (`scripts/deploy_image.sh`, `O_SG_HOST=root@…`). AI agent sessions (decision/construction roles, 研究员, 告警分诊) work here with repo and sometimes production access.
- **External**: OKX REST/WS, mtapi MT5 gateway (TLS gRPC, `internal/mt5/client.go`), TRON gRPC/HTTP APIs, CMC, Feishu (alerts), GitHub Actions CI.

**Owner fact:** other people/processes on the SG host or dev box **can reach** o-core's loopback ports. The `127.0.0.1` binding of `:50051` and `:8095` is **not** a sufficient boundary.

## 2. Major components

| Component | Code it covers |
|---|---|
| **core-api** | `cmd/core` ConnectRPC `CoreService` + SPA on `:50051` (`service.go`, `*_face.go`, `trx_faces.go`), `proto/dashboardv1`, `web/` |
| **secrets** | `cmd/core/readcred.go` (`:8095 GET /v1/readcred`), `cmd/core/boot_secrets.go`, `cmd/research/readcred.go`, `internal/platform/config` (master key, `app_config`), `internal/platform/cryptoutil`, `signutil`, `app_secrets` table |
| **trading** | `internal/trading` (execution, okxexec, mtexec), `internal/risk`, `internal/guard`, `internal/earn`, `internal/portfolio` (incl. `trxvote` signer), `internal/ledger`, `internal/store`, `cmd/core` open/close/trx*/bills, `migrations/` |
| **data-collection** | `internal/data` (okxws, okxpriv, trx, cmc, collect, fact), `internal/mt5`, `internal/projector`, `cmd/collector`, `cmd/projector`, `cmd/sweeper` |
| **research** | `internal/research`, `internal/review`, `cmd/research`, AI agent role contracts in `skills/研究员`, `skills/告警分诊` |
| **deploy-ci** | `deploy/`, `docker-compose*.yml`, `Dockerfile.core`, `scripts/`, `.github/workflows/ci.yml`, `go.mod`/`web/package*.json` |
| **tools** | `tools/*` one-off operator CLIs (dustsweep, okxunwind, mt5close, credx, keypipe, …) |

## 3. Entry points & who can reach them

- `:50051` ConnectRPC + UI (core-api): **no app-level authentication**; reachable by anyone on the SG host / with a tunnel (owner confirmed). Mostly read RPCs (`Overview`, `Watch`, `List*`, `Get*`) exposing positions, equity, cashflows; `TrxAccountRefresh` triggers venue reads + DB writes.
- `:8095 GET /v1/readcred` (secrets): returns **decrypted** `<venue>_read.*` credentials, gated only by the `readcred.entitlement` whitelist in `app_config`; no caller auth. Reachable by SG host users and anyone holding the tunnel key.
- Inbound external data (data-collection, trading, research): OKX WS/REST payloads, MT5 gateway responses, TRON API responses, CMC, research feeds. These drive sizing, risk and execution decisions and are stored in PG.
- AI agent inputs (research): alert text, external research material, potential prompt injection; controls in the role skill files are allowlisted write paths.
- PG instances: `o-postgres` on loopback with `o:o` default creds; `collector_rw` role for the collector (`deploy/o-collecto/collector_rw.sql`).
- CI: GitHub Actions on push/PR to master; deps from Go modules and npm.

## 4. Assets (owner: all equally important)

1. Master key file and everything it decrypts (`app_secrets`: OKX trade keys, `<venue>_read.*` keys, Feishu app secret, `trx.vote_private_key`).
2. TRON signing key (`cmd/core/trxvote.go`, `internal/portfolio/trxvote`, arb permission id 3 on a multisig account) and the on-chain funds.
3. Exchange accounts and funds (OKX, MT5 incl. live account), reachable via trade keys.
4. Integrity of trading decisions: risk guard hard lines (`internal/guard`, `internal/risk`), sizing, open/close paths. A wrong trade means a loss.
5. Ledger/NAV integrity (`internal/ledger`, ledger hash scripts) and confidentiality of positions/equity.
6. SG/OCI host access (root SSH from the dev box, tunnel keys).

## 5. Attacker profiles (owner: all matter)

- **A1 Local/adjacent user** on SG host or dev box reaching loopback services (`:50051`, `:8095`, PG ports).
- **A2 Internet attacker** against any service that is or becomes exposed.
- **A3 Malicious external data**: hostile or spoofed venue/API payloads (OKX, MT5, TRON, CMC, research feeds) and prompt injection into AI agent roles.
- **A4 Compromised dev box / tunnel key**: pivot to SG via tunnels or root SSH deploy path.
- **A5 Supply chain**: Go/npm dependencies, GitHub Actions, base images.
- **A6 AI agent session overreach**: agent sessions with repo/prod access acting beyond their role contract.

## 6. Existing controls

- Master key only as a host file, mounted read-only, never in env or DB (`internal/platform/config/masterkey.go`, D-0005). Secrets are AES-256-GCM sealed with random nonces (`cryptoutil`).
- Services bind to loopback via compose `ports` (`127.0.0.1:…`). Not sufficient by itself (owner fact).
- readcred is fail-closed (503 on missing key, whitelist or decrypt failure) and only enumerates `_read.` rows, so trade keys are structurally excluded. It runs on a separate mux from `:50051`.
- Tunnel user restricted (`ForceCommand /bin/false`, `permitopen`) per `deploy/o-collecto/tunnel.service`.
- TRON executor uses a single-sig arb permission rather than owner; the vote key is dormant unless its flag is enabled.
- MT5 gateway over TLS. Image tags pinned to the build hash for collector/core deploys.
- AI agent roles: allowlisted write paths and "injection double-layer defense" in `skills/研究员`, `skills/告警分诊`.
- Risk hard lines (30-day −3% stop, exposure caps) in guard/risk.

## 7. Out of scope / accepted risk (owner)

_Owner selected "nothing out of scope" together with the three items below. They are treated as accepted risk; confirm or correct._
- `tools/` one-off operator CLIs.
- Default local DB creds (`o:o`) on the loopback-only Postgres (note: with A1 in play this is weaker than it looks).
- `docs/` and `skills/` markdown.

## 8. Open questions

- Should `:50051` / `:8095` get caller authentication (e.g. mTLS or token), given that local users can reach them?
- Who are the "other people" on the SG host and dev box (other tenants, other projects like alphaforge, AI sessions)?
- Is `trx.vote_private_key` enabled in production?
- Confirm the scope answer in §7.
