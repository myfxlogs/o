---
schema_version: 2
id: armed-ttl-expired-misread-claim-stall
kind: pitfall
risk: medium
status: verified
keywords: [armed_ttl_expired, 晋升领取, 停摆定性, 吸收径, 墓碑锁, claim-lag]
scope: [lifecycle, portfolio, diagnosis]
summary: armed_ttl_expired 批量过期≠领取件停摆——armed=已领/吸收终态；定性先查领取时滞+proposal_ref 孤兒率，真墓碑锁在 paper 面无退役径
first_seen: 2026-11-01
last_seen: 2026-11-01
evidence_refs:
  - type: provenance
    ref: docs/tasks/T-0291-armed晋升领取件停摆定位复起.md
---

## armed_ttl_expired 批量签名误读为「晋升领取件停摆」

- **实证**：T-0200 二期凭 `fin_proposal.retire_reason='armed_ttl_expired'` 7d×347 件定性「晋升领取器停摆：合格机会在架 4h 无人接」。T-0286 取证反转：347 件中 323 件有 fin_strategy 行（proposal_ref 回指），领取时滞中位 **2.2min**（p90 9.9min）；7d 全链通量 proposed→paper 2062 / paper→probe 1191 / probe→live 1190——链全段大规模在工作。
- **语义根**：码面 armed=「已领取/被吸收」终态标记（admitProposals 领取成功或同键吸收都翻 fin_armed），armed 能立=已被领。「armed 4h 过期」是 TTL 卫生闸对已领提案的例行回收，非停摆信号。
- **教训**：判「领取停摆」先查三件——①孤兒率（`NOT EXISTS fin_strategy.proposal_ref`）；②领取时滞分布（`strategy.created_at - proposal.created_at`，中位分钟级=健康）；③transition 通量（7d 逐 from→to 计数）。另注意 live 行 `entered_at` 是**迁移时刻**（每次 Transition 刷新），非「扫描器刷新」伪证。
- **真病灶藏在对侧**：24 孤兒全=同键吸收环（paper 行无退役径永占同键槽→上游重起草被吸收→4h TTL 退，BTC 季度对儿 3 天 24 连环）。paper 墓碑锁=T-0286 根修（lifecycle.paper_ttl_h 缺省 168h——慢毕业行 139h 实证在案，72h 会误杀；T-0098 degraded 同病先例）；上游产供断流（72h 零 funding-hedge 起草）是另一独立层，勿与领取面混判。
