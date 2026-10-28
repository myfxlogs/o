---
schema_version: 2
id: pipefail-ls-empty-match-silent-chain-death
kind: pitfall
risk: high
status: verified
keywords: [pipefail, set -e, ls 空匹配, 静默炸链, 滚动裁尾, grep 空匹配, exit 2]
scope: [ops, script, backup]
summary: set -euo pipefail 下 ls/grep 空匹配（exit 2/1）×管道=整链静默死亡——「无输出无日志 exit 2」；滚动裁尾/守卫行一律 find 径或 (ls||true) 径，首次运行零存量即触发
first_seen: 2026-10-29
last_seen: 2026-10-29
evidence_refs:
  - type: ticket
    ref: T-0234
  - type: file
    ref: scripts/arm/o-backup-md.sh
---

## pipefail 下 ls 空匹配把整链静默炸死（零存量首跑必触）

- **实证**：T-0234 OCI md 备份链首跑（systemd o-backup-md.service）——
  journal 仅 `status=2/INVALIDARGUMENT` 一行，**零业务输出**，同秒即折。
  病灶=脚本头部的 escrow 残件裁尾行：
  `ls -1t "$BACKUP_DIR"/o-md-escrow-*.tar.age 2>/dev/null | tail | xargs -r rm`
  首跑时 glob 零匹配→ls 收到字面量路径→exit 2（stderr 已吞）→
  `pipefail` 把管道状态定为 2→`set -e` 整链退。同族：裁尾用
  `ls|grep '^pattern$'|tail` 在 grep 零命中（exit 1）时同炸；
  `latest=$(ls … | head -1)` 赋值位同炸且**绕过下方 [ -n ] 守卫**。
- **病灶**：`ls|grep` 的「零匹配=失败」语义与「无残件/空库=正常态」
  语义相撞；pipefail 把静默的 2 变成整链死因。存量常在的旧链
  （SG fin 每日有件）永触不到——**只有新链首跑/零存量路径才炸**，
  复现窗口天然窄。
- **律**：`set -euo pipefail` 脚本内，一切「可为空集合」的列举行：
  ①find 径（空匹配 exit 0）——`find DIR -maxdepth 1 -name 'pat' | sort -r | tail -n +15 | xargs -r rm`（时间序靠文件名内嵌定宽 UTC 戳）；②或 `(ls … 2>/dev/null || true) | head -1` 保守卫语义可达；③grep 收罗行改 find/`|| true`；④语句级 `[ -f x ] && cmd` 同雷（x 缺=exit 1 炸链）——一律 if 形。
- **同族**：`shell-tz-vs-system-tz-cron-clock`——同为「调度面常态
  假设」翻车谱系；本条特化在 bash 管道语义层。
