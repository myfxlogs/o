---
schema_version: 2
id: eval-insertonly-fingerprint-gate
kind: pitfall
risk: high
status: verified
keywords: [fin_opp_eval, insert-only, 版本史, 指纹闸, evalgate, 盘烧, md_1097, 同指纹重复写, upsert时代, 写径闸]
scope: [internal/research, internal/research/evalgate, migrations/md_1097, migrations/fin_0197]
summary: 版本化 insert-only 落地（md_1097 撤 UNIQUE 改纯 INSERT 版本史）后，eval 环同 (inst,template,direction) 指纹未变仍每拍 INSERT——md 侧 89 行/s≈7.7M 行/日≈2-4GB/日直烧盘；修法=写径指纹闸 evalgate.SameFingerprint（最新行 verdict+net_apr 双同→跳 INSERT 复用行 id 续提案径），判词/净额变迁仍产新版史
first_seen: 2026-11-17
last_seen: 2026-11-17
evidence_refs:
  - type: ticket
    ref: 工单#22904（P1——md fin_opp_eval 89行/s持续、OCI 盘余 12G→3-5 日撑爆；修法①写径指纹闸首选）
  - type: commit
    ref: fix/eval-fingerprint-gate（evalgate 单点件+8 写面接线）
  - type: ticket
    ref: 工单#19778（insert-only 版本史立案——本闸为其落地后续治理，非回退）
repro_or_symptom: "md_1097 上线后 fin_opp_eval 行增速从 upsert 时代单行/键 变为每评估一版——eval 环拍频×候选数=89 行/s 恒速写入；funding-hedge 同 inst 同 verdict=rejected 10min×98 次零信息增量仍逐行落账"
root_cause: insert-only 版本史的设计前提=「每次评估都是一个新信念版本」——但 eval 环多数拍的评估结论与上拍完全相同（行情未变→verdict/net_apr 未变），这些「伪版本」携带零信息增量却按全量行写。版本化解决了 upsert 灭史（旧行被覆写拿不到入场信念），却引入了对偶病灶：无变化判据的无限版本膨胀。写径缺「信念是否迁移」的判定闸。
fix_and_prevention: "①单点闸件 internal/research/evalgate.SameFingerprint——(inst,template,direction) 最新行（id DESC，fin_opp_eval_key_latest 索引同键）verdict+net_apr 双同→返 (行id,true) 跳 INSERT、复用 id 续提案/呈批径；异或无历史→(0,false) 走原 INSERT。net_apr 比对走库内 IS NOT DISTINCT FROM ::numeric（NULL↔NULL 同指纹——gapscan 面 NULL 族同闸；float64→numeric 显式转=精确域判）。②8 写面全接线（funding/dirtrend/dirmom tx 径+gapscan/dcd/hypothesis×3/grammar pool 径）——写面分散时闸必须单点防判据漂移；projector mirror.md→SG 复制径不闸（逐行搬运含 id，闸了=复制失真）。③funding persist 的 spotLegOK 起草闸须前置到指纹闸之前——终判词先定（proposed→rejected 直落），否则 proposed↔rejected 摆动每拍产新版=闸失效；原「插后 UPDATE」两行写径并一为终态直落。④下游 eval_id 消费面（fin_proposal.eval_id 幂等查/condreq eval_id 呈证）复用旧行 id=upsert 时代同义——行 id 稳定反而让 eval_id 去重查复稳（post-1097 每拍新 id 曾使该查恒零=次生 bug 顺带根修）。⑤凡「版本化/历史化」表设计先问：什么是「新版本」的判据？无判据的版本化=用历史表当日志表烧盘——判据=指纹字段集，写径先查指纹再决定是否产新版。⑥旧验收锚随语义修订同步改（TestEvalInsertOnlyVersions 同指纹期望 2 版→1 版+变迁期 2 版）——锚钉的是语义不是行为，语义裁定变了锚必须跟着改否则假红。"
notes: "与 simdrill-ledger-venue-domain-not-substituted 同族教训：写面设计先问「这行什么时候该存在」——存在判据缺位则写径恒产。retention 面：存量洪泛行（数百万级）清理非本闸范围，归唤知/业主定保留政策。验收余项：md 行增速<10k/h 为部署后实测判据（本闸保证同指纹零写，指纹真变迁的写量=信念史本义）。"
---

## 版本化 insert-only 缺指纹闸=用历史表当日志表烧盘

- **实证**：md_1097（工单#19778）把 fin_opp_eval 从 UNIQUE upsert
  改纯 INSERT 版本史——入场信念成史可查的立案本意成立，但 eval 环
  多数拍结论与上拍全同（行情未变→verdict/net_apr 未变），伪版本
  零信息增量仍逐行落账：md 侧实测 89 行/s≈7.7M 行/日≈2-4GB/日，
  OCI 盘余 12G→3-5 日撑爆（工单#22904 P1）。
- **病灶**：「每次评估=新版本」的隐含前提在多数拍不成立——版本化
  表的写径必须先回答「本拍信念是否迁移」，迁移判据=指纹字段集
  （本案=verdict+net_apr），无判据的版本化=日志表烧盘。
- **修法**：`evalgate.SameFingerprint` 单点闸——同键最新行指纹双同
  →跳 INSERT 复用行 id；异→产新版。8 写面同消费（分散写面单点
  闸防判据漂移）；projector 复制径不闸。
- **连带次生修**：①funding spotLegOK 起草闸前置（终判词先于指纹
  判定，否则 verdict 摆动绕闸）；②eval_id 幂等查复稳（post-1097
  每拍新 eval_id 使 `WHERE eval_id=` 查恒零——指纹闸复用旧 id
  顺带根修该断链）。
- **不可做**：给版本表加「每 N 分钟最多一版」时间闸——信念真实
  迁移被时间窗吞=史失真；判据必须是内容指纹非时间。
- **验收锚教训**：旧锚 TestEvalInsertOnlyVersions 钉「重评=2 版」
  ——语义被 #22904 裁定修订后锚同步改「同指纹=1 版+变迁=2 版」，
  锚钉语义不钉行为。

## 关联

- 工单#22904——本案立案（P1 盘烧）
- 工单#19778——insert-only 版本史立案（本闸=其落地后续治理）
- `drill-cleanup-misses-guard-close`——写面覆盖漏面姊妹判例
- `simdrill-ledger-venue-domain-not-substituted`——「写面存在判据
  缺位」同族教训

## 附录：闸 SQL 形

```sql
SELECT id, (verdict=$4 AND net_apr IS NOT DISTINCT FROM $5::numeric) AS same
FROM fin_opp_eval
WHERE inst_id=$1 AND template=$2 AND direction=$3
ORDER BY id DESC LIMIT 1
-- 走 fin_opp_eval_key_latest(inst_id,template,direction,id DESC) 索引
```

---
*首见：工单#22904 施工席（2026-11-17）；复核：8 写面逐一接线+funding
起草闸前置序+evalgate 桩测（同指纹/变迁/NULL 族/键隔离四断言）。*
