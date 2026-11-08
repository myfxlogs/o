---
schema_version: 2
id: eval-mirror-proposal-gated-blind-spot
kind: pitfall
risk: medium
status: verified
keywords: [观测盲, 族沉默探针, fin_opp_eval, projector serve, proposal-gated, rejected-only, xvenue-carry, eval镜像谓词, D-0030, T-0479]
scope: [projector eval镜像, hunger ProbeFamilies, gauges eval_zero量表, rejected-only族]
summary: SG fin_opp_eval 镜像=仅被 fin_proposal 引用行（D-0030 写权矩阵+PG-md ~93upd/s 体量闸）——全 rejected 族评估在正本活跃却对 SG 侧探针/量表结构性恒零，族沉默探针与 eval_zero 量表对 rejected-only 族日频误报；修域=测量面/活动桥/接受噪声三择呈裁量，非评估链 bug。
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: []
evidence_refs:
  - type: task
    ref: T-0479（观测盲:xvenue-carry——挂起诊断 await=裁量）
  - type: code
    ref: internal/projector/mirror.go qEvalFresh（EXISTS fin_proposal 谓词）+ops.go projectEvals
  - type: server_fact
    source: postgres@PG-md(OCI:5435)
    query: SELECT count(*) FROM fin_opp_eval WHERE template='xvenue-carry' AND ts>now()-interval '7 days'
    observed_at: 2026-11-08T10:30:00Z
    server_revision: ""
    response_hash: ""
  - type: server_fact
    source: postgres@SG(o-postgres)
    query: SELECT max(ts) FROM fin_opp_eval WHERE template='xvenue-carry'
    observed_at: 2026-11-08T10:30:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

T-0479 升件 `观测盲:xvenue-carry`（ops_event #3226）：carry_diff 源信号
7d 鲜行=2 而 fin_opp_eval 该族 7d=0。生产三通道取证反转票面前提：

- **PG-md 正本（OCI 5435）**：xvenue-carry 评估行持续产出且新鲜
  （10:24 rejected/net_ann_economics——swap 成本腿吃光 funding，判词正确）；
  o-research journal 同步实证评估臂逐拍在跑。
- **SG 镜像（o-postgres）**：同族末行 2026-10-30——此后零更新。

## 根因或策略解释

projector `-mode=serve` 在役：`qEvalFresh`（mirror.go）eval 镜像谓词=
**仅被 fin_proposal 引用的 eval_id 行**（D-0030 §六写权矩阵钦定——
lifecycle admit join 需要 ~20 行）。PG-md fin_opp_eval 实测 ~93
upd/sec（n_tup_upd 60s 增量 5561）——全量镜像物理不可行，谓词是承重墙。

rejected-only 族（评估全 rejected→零提案）对 SG 结构性恒零：
族沉默探针（hunger ProbeFamilies，o-core@SG 读 SG fin_opp_eval）与
gauges.tsv `xvenue_carry_eval_zero`/`stake_carry_eval_zero` 同病灶日频
误报（ops_event #3226/#2450、#3446/#3445 在案）。探针族覆盖本身即化石
依赖——SG 模板集缺 stake-carry（PG-md 有），无化石行的族不入探针域。

## 修法/判定规则

- **先分域再判病**：「信号活+SG 评估零」先核 eval 正本（PG-md）有无
  行——正本有=观测域错位非评估链死；正本无才追消费链。
- eval 镜像谓词不可放宽（体量+canon 双闸）；rejected eval 的 SG 可见
  性只能走新活动面（脉冲表/测量面改判）——架构案级呈裁量。
- 探针族覆盖化石依赖是已知盲形：新族未产生提案前不入探针域。
