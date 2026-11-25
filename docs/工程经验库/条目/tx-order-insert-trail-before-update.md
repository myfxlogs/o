---
schema_version: 2
id: tx-order-insert-trail-before-update
kind: pitfall
risk: mid
status: verified
keywords: [fin_strategy_transition, sweepStaleEphemeral, 事务序, SELECT谓词见新态, 留痕恒零, 27183, tx原子化, drill_ephemeral_sweep, audit RED]
scope: [internal/trading/simdrill, fin_strategy, fin_strategy_transition, scripts/audit/asserts]
summary: 留痕 INSERT 的 SELECT 谓词若与迁移 UPDATE 在同一事务内，且 UPDATE 先行，SELECT 读的是「迁移后新态」→ 留痕谓词恒空、行恒零（且零报错——Exec 成功返回）；凡「audit trail/迁移留痕」两拍写法，序必须 INSERT 先于 UPDATE（读旧态），或改用 UPDATE…RETURNING 取迁移行 id 集再落痕
first_seen: 2026-11-25
last_seen: 2026-11-25
evidence_refs:
  - type: ticket
    ref: 升件#40873（审计/生命周期 audit_lifecycle_sweep_trail_order——RED=scripts/audit/asserts/sweep-trail-order.sh）
  - type: commit
    ref: fd9339991（#27183 tx 原子化 refactor——病灶引入点：两拍裸 Exec 收同 tx 时把原序 INSERT→UPDATE 翻成 UPDATE→INSERT）
  - type: ticket
    ref: fix/40873-sweep-trail（修复支——复原序 INSERT 先于 UPDATE，原子性保持）
repro_or_symptom: "sweepStaleEphemeral 自 #27183 上线（11-21）后产线零 drill_ephemeral_sweep transition 行——清扫在跑（degraded 行确有退役）但留痕全丢；原序版本 7 行实证在案（2026-11-10）。零报错：INSERT…SELECT 谓词恒空=0 行写入，Exec 正常返回，静默无痕。"
root_cause: "PostgreSQL 事务内可见性=语句级快照按命令序：同一 tx 内 UPDATE 之后的 SELECT 看得到本次 UPDATE 的新态。两拍写法「SELECT WHERE state='degraded' 落痕 → UPDATE state='degraded'→'retired'」的原序被 refactor 收进同一 tx 时翻转，INSERT 的 SELECT 只见到已 'retired' 的行→谓词恒空。原子化修的是「两拍裸 Exec 幽灵窗」，但收同 tx 不查序语义=把对的序改成错的序。"
fix_and_prevention: "①修=复原序：留痕 INSERT（SELECT 读旧态谓词）先于迁移 UPDATE，原子性由同 tx 保持——retireEphemeral 单件版（UPDATE+RowsAffected→INSERT 按 id）之所以不受害，因其 INSERT 按已知 id 写不依赖谓词读旧态。②通用律——凡「迁移+留痕」收同 tx：留痕若靠谓词重 SELECT，必须先于 UPDATE；更稳写法=UPDATE…RETURNING id 集/CTE（WITH r AS (UPDATE…RETURNING id) INSERT…SELECT FROM r）一单语句序义自明不踩序坑。③序断言入 scripts/audit/asserts/sweep-trail-order.sh——源码行序断言（UPDATE 行号<INSERT 行号=RED），防未来 refactor 再翻。④判别式自查：tx 内凡有「SELECT WHERE <将被改的条件>」的语句，问它想见的是旧态还是新态。"
notes: "与 eval-insertonly-fingerprint-gate 同族=写面设计先问「这行什么时候该存在」——本件问的是「这行该读到什么态」。refactor 收 tx 时逐语句核「读态语义」是本次唯一可迁移教训：原子化是对的，翻转序是无声的。"
---

## 同事务 SELECT 读新态：留痕 INSERT 先于 UPDATE

- **病灶**：`sweepStaleEphemeral` 两拍「SELECT 谓词落痕→UPDATE 迁移」
  在 #27183 收同 tx 时被翻成 UPDATE→INSERT。PG 事务内后续语句
  可见本事务已做的修改——INSERT 的 `WHERE state='degraded'` 只
  见到已被改成 'retired' 的行→恒零行、零报错、静默无痕。
- **修法**：复原序（INSERT 先 UPDATE 后，同 tx）；更稳替代=
  `UPDATE…RETURNING`/CTE 单语句，序义自明。
- **通用律**：凡 refactor 把「读旧态谓词」语句收进与「改态语句」
  同一事务，逐句核读态语义——收 tx 对，翻序错。
