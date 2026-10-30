---
schema_version: 2
id: import-direction-gate-silent-fail-unprovisioned
kind: pitfall
risk: low
status: verified
keywords: [check_import_direction, provision_worktree, worktree, 静默失败, go list, internal/gen]
scope: [worktree, 机检门禁, pre-commit]
summary: 未 provision 的 worktree 里 check_import_direction.sh 静默 rc=1（go list 败被 set -e 吞、stderr 已 /dev/null）——零输出误导成「门禁违例」，真因是 internal/gen 缺件；先跑 provision_worktree.sh 即解
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: doc
    ref: scripts/check_import_direction.sh:141
  - type: doc
    ref: scripts/provision_worktree.sh:2
  - type: ticket
    ref: T-0263
---

## import 方向门禁在未 provision worktree 静默红——先补台架件再判

- **实证**：决策席五票并码 session（2026-10-30），新 worktree
  `/opt/o-wt/merge5` 直接 merge+commit——pre-commit 报
  「FAIL: import 方向门禁违例」；直跑 `scripts/check_import_direction.sh`
  **零输出 rc=1**，一度误判有真实违例。
- **病灶**：行 141 `RAW=$(go list ... ./internal/... 2>/dev/null)`——
  gitignored 生成件 `internal/gen` 未 provision 时 `go list` 整包解析败，
  stderr 吞、rc 经命令替换触发 `set -e` 静默退。脚本头注
  「无 Go 面=条件未备直通」只覆盖 go.mod/internal 缺席，不覆盖
  「internal 在但 gen 缺」这一中间态。
- **处置**：新 worktree 任何 commit/机检前先 `scripts/provision_worktree.sh`
  （部署树拷 gen×2+dist 秒级）——prov 后门禁输出正常
  （OK 112/121 边）。同类坑 canon `worktree-provision-missing-web-gen`
  属同族（gitignored 生成件缺位的另一显形面）。
- **修法向（未立票）**：脚本可在 `go list` 败时 echo 一句「gen 件疑缺——
  先 provision」再退 1，一行消歧；立票与否由后续决策裁。
