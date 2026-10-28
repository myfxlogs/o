---
schema_version: 2
id: proposal-dedup-key-cross-family-fork
kind: pitfall
risk: high
status: verified
keywords: [提案去重, direction 词表, 去重闸, RETURN NULL, 静默吞, 跨族撞键, BEFORE INSERT 触发器, 预查键对齐]
scope: [提案桥, fin_proposal, 去重幂等, 多族同 inst 结构]
summary: 同 inst 跨族共用 direction 词值→去重闸 (inst_id,direction) 键跨族撞车，BEFORE INSERT RETURN NULL 静默吞第二族提案（零错零痕）；应用层预查若键更宽（如仅 inst_id）则误吞更大。多族共享 inst 的结构族，direction 词表必须族级去分叉，应用层预查键必须与 DB 闸键全等（键列+在途态集）。
first_seen: 2026-10-29
last_verified: 2026-10-29
skills: []
evidence_refs:
  - type: task
    ref: T-0216 返修③（提案去重语叉回填）
  - type: commit
    ref: t/0216-basis（md_1074/fin_0083 双侧迁移+grammar.go 去分叉）
  - type: doc
    ref: migrations/md_1054_proposal_dedupe_gate.sql（闸键原形）
---

## 事实/症状

T-0216 basis executor 落位后，`quarter_basis_long` 与 `perp_quarter_spread` 两族在
grammar.go 中共用 `direction='basis_quarter'`，且评估同 inst（季度期货键
BTC-USD-261225 形）。提案落卡时：Go 预查（`WHERE inst_id=$1 AND state='fin_proposed'`，
单键）先拦一族；即便绕过预查，md 侧 `fin_proposal_dedupe` BEFORE INSERT 触发器
（键 `(inst_id,direction)`+双在途态）对第二族 `RETURN NULL`——**INSERT 零错零痕静默吞**，
Go 侧照打「提案落卡」假日志。净效果=两族恒互斥：先到提案在途，后到族永远无法
提案，且对账面看不到任何拒绝痕（rejects 门零计数）。

## 根因或策略解释

- **去重键没含族身份**：`(inst_id,direction)` 键在「一 inst 一族」世界里成立；
  basis 两族共享 inst（同一期货合约既可 spot-basis 也可 perp-spread）即破。
  direction 词表是封闭集（CHECK 约束），共用词值=把两族压进同一去重桶。
- **双层去重键不对齐放大**：应用层预查（inst 单键）比 DB 闸（inst+direction）
  更宽——即使闸键正确，预查先误吞；两层各自演化=「去重语义分叉」。
- **静默吞=可观测性黑洞**：触发器 RETURN NULL 不报错不计数，日志照打成功——
  病灶只能靠「为什么 B 族永远没提案」这种反面证据发现。多族结构上线前
  应以「双族同 inst 双提案并存」为验收锚（T-0216 返修 defork_test 即此锚）。

## 复用面

凡「DB 闸+应用层预查」双层去重结构：①键列必须含完整身份维度（inst+direction+
族身份/模板），新族引入共享 inst 时必查键撞；②应用层预查键=闸键全等复制
（键列+在途态集逐字对齐），单点真源在 DB 闸、预查只是省一次空插；③闸吞必须
可观测（应用层预查先行可让吞显形为 rejects 计数，禁只靠触发器静默吞）。
