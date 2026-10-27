---
schema_version: 2
id: domain-switch-cross-domain-read-face-audit
kind: pitfall
risk: high
status: verified
keywords: ["42P01", "relation does not exist", "分域切换", "PG-md", "dedupe 闸", "静默 fail-open", "err==nil &&", "T-0169", "提案管线堵"]
scope: [internal/projector, internal/research, internal/review, migrations/domain.tsv, PG 分域]
summary: 分域切换按「写面」盘表漏「读面」——研究臂跨域读 fin 域表（dedupe 闸 fin_strategy）42P01 堵死提案管线 2h 2,293 条，且黑名单读败被 `err==nil &&` 静默吞成 fail-open；切域清单必须逐进程枚举触表×库对拍，吞错读面单独过一遍
first_seen: 2026-10-27
last_verified: 2026-10-27
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0172-研究侧fin域读面分域断供.md
  - type: commit
    ref: 4dfb4a2
  - type: doc
    ref: internal/research/funding/persist.go:91
  - type: server_fact
    source: postgres
    query: "journalctl --user -u o-research | grep -c 42P01（切换后 24h=2988，修复后=0）"
    observed_at: 2026-10-27T09:52:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

T-0169 分域切换（research 主池 SG→PG-md）后：`persist.go:91`/`dcd/eval.go:171`
dedupe 闸读 `fin_strategy` 恒 `SQLSTATE 42P01`——eval 落、提案死（提案管线断流
2h+，报错 2,293 条）；review 臂 boot 首拍 `fin_position` 42P01 全盲。更隐蔽面：
`fin_venue_untradeable` 黑名单查询写形 `err == nil && banned > 0`，读败静默走
「未拉黑」=起草闸 fail-open（日志零痕）。

## 根因或策略解释

切换盘面按「谁写哪张表」分域（写面对册 domain.tsv），漏盘「谁读哪张表」——
dedupe 闸/评史官/黑名单是跨域读面：表归 fin 域（SG），读者在 md 域（dev）。
读面断供两态：显性（闸查询 42P01 管线死）与隐性（吞错读面降级 fail-open），
后者更危险（无报错、判据悄然失效）。

## 处置

反向桥（T-0172）：读者需要的 fin 域表升 `bridge-reverse`，projector 反向镜像
SG 正本→PG-md 只读副本（全表 diff+事件帧）；写臂（review 钱径面）不改数据所
在地、加 FIN_PG_DSN 小池直连正本。判别式：**切域票验收前逐进程跑触表清单
（grep FROM/INTO/UPDATE/JOIN）× 目标库 information_schema 对拍，缺表=反向桥
或 DSN 分池二选一，且每处 `err == nil` 形读面单独标 fail-open 风险**。

## 证据

- check: `journalctl --user -u o-research --since '-24h' | grep -c 42P01`
- result: 修复前 2,988（24h）；修复后（09:52Z 轮换）0
- check: PG-md `SELECT count(*) FROM fin_strategy` vs SG 同查
- result: 1770==1770 全行 md5 指纹一致（462bbc6b…）；黑名单 5==5
- ref: T-0172；commit 4dfb4a2；blacklisted 起草闸判决恢复（XPD-USDT-SWAP）

## 反证条件

- check: 研究侧日志再现 42P01 或 proposed 卡过闸率归零而 eval 照产
- falsifies_when: 反向桥断供（projector 岗终/镜像陈）时闸再次 42P01——条目
  仍活；若未来单库回收（分域废弃）则本条目随架构退役

## 适用边界

适用于多库分域/分实例拓扑下的进程迁移与角色切换（不限 PG）；核心律=
「表域账管写权，读面自证」——任何切换后首验是读者视角的表可达性对拍，
不是写者视角的落位对拍。吞错读面（`err==nil &&` 判据形）在任何拓扑下都
是 fail-open 候选，切域只是把它从理论变成现实。
