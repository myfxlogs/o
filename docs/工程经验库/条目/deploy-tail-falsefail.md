---
schema_version: 2
id: deploy-tail-falsefail
kind: pitfall
risk: high
status: observed
keywords: [deploy_image, deploy tail, 健康闸, rc=1 假败, ops_event 部署行, deployed-sha 钉陈, deploy_watch, selfstate_beat, tail=, deploy_tail_ 升件, pg_md_migrate]
scope: [deploy_image.sh, ops_event DEPLOY/部署账, deploy_watch.sh, selfstate_beat.go, SG/OCI 尾步保洁与 PG-md 迁移]
summary: 健康闸后尾步败不再翻部署 rc——部署主体=健康闸，尾步欠账走 tail=/deploy_tail_ 升件；修前 SG 已 healthy 仍 rc=1 钉陈 deployed-sha
first_seen: 2026-11-15
last_verified: 2026-11-15
skills: []
evidence_refs:
  - type: doc
    ref: scripts/deploy_image.sh
  - type: doc
    ref: scripts/cognition/escalate.sh
  - type: server_fact
    source: postgres
    query: SELECT id,seat,event,note FROM ops_event WHERE id=17757
    observed_at: 2026-11-15T12:00:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

deploy_image.sh 在 o-core 健康闸通过后仍把后续保洁/迁移尾步失败当部署失败：SG image prune、SG image_tag_gc、OCI PG-md 迁移/水位断言、本地 tag_gc、builder prune 任一 set -e/|| 链败→脚本非零退出→EXIT trap 记 `deploy <sha> rc=1`。生产实证：部署事件 #17757 在 SG 已跑 `o-core:3a9eb1965afd` healthy（StartedAt=01:59:48Z）时仍落 `rc=1`，deploy_watch/selfstate_beat 不认成功部署、自报 deployed-sha 钉陈并误发 deploy_blocked。

## 根因或策略解释

脚本把「健康闸=主体成」与「尾步=部署结果」混在同一 rc：健康闸后没有语义边界，`set -euo pipefail` 下任何尾步非零即整体 rc=1；部署账只按 `rc=0`+`o-core:<sha>` 认成功，故尾步欠账反噬主体成功。

## 处置

健康闸通过即锚定部署主体成功；其后尾步统一走 `tail_fail`：累加 `TAIL_FAIL`、stderr 留痕、经 `scripts/cognition/escalate.sh` 自报 `deploy_tail_<step>` 升件，返回 0 不翻 rc；`deploy_event` 在 relay/feishu/ops_event 部署行追加 `tail=<逗号步名>`，rc 仍锚健康闸。PG-md 迁移封装 `pg_md_tail`，build/scp/remote/水位断言任败 `return 1` 交 `tail_fail`；临时 migrate 二进制在 build/scp 败径清理。保洁件（sg_image_prune/sg_tag_gc/local_tag_gc/builder_prune）同径。

## 证据

- check: `bash -n scripts/deploy_image.sh && bash scripts/check_tolerate_report.sh && bash scripts/check-lines.sh && bash scripts/check_no_forbidden_patterns.sh`
- result: 全绿；新增 `|| true` 升件写径带 `# tolerate:` 锚
- ref: 工单#17818 / scripts/deploy_image.sh

## 反证条件

- check: 若未来部署账消费面改为「rc=0 且无 tail= 才算干净成功」或尾步失败必须阻断上线，则本条目需重审
- falsifies_when: 出现健康闸后尾步失败仍被记 rc=1 的新部署事件，或 `tail=` 标记破坏 `rc=0`/`o-core:<sha>` 解析

## 适用边界

只规 deploy_image.sh 健康闸后的尾步会计；不改锁/出处闸/健康闸判据/钱径；`O_MD_MIGRATE_SKIP=1` 仍为显式演习跳步（哨兵照红），不记 tail_fail；升件写败不翻 rc（结构性限度同 escalate.sh）。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（部署账实一致是会计口径坑非量化阈值病灶；tail= 升件由 ops_esc_pending 面承接）
- class: n/a
- query: n/a
- threshold: n/a
