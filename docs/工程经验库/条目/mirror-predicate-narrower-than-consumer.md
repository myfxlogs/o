---
schema_version: 2
id: mirror-predicate-narrower-than-consumer
kind: pitfall
risk: high
status: verified
keywords: [projector, 镜像谓词, fin_opp_eval, dirEntryGate, entry_cond_defer, paper→probe, 活跃集, ActiveSetUnion, md_ticks, md_mark_index, 投影盲]
scope: [数据面, PG-md→SG 桥, 生命周期晋升链, 采集订阅集]
summary: 投影/订阅谓词按「写入时点角色」收窄而消费面按「在册角色」读取=结构性盲——fin_opp_eval serve 谓词仅投 fin_proposal 引用行，策略立项后续评永不达 SG 致 dirEntryGate 永判评估行陈；paper 态 inst 不在 ActiveSetUnion 致 md_ticks 断供 pnl 摊提饿死 confGate 恒拦。修法=谓词对齐消费域（在册策略自然键/记账消费态），非放宽闸阈值
first_seen: 2026-11-24
last_verified: 2026-11-24
skills: [施工席]
evidence_refs:
  - type: doc
    ref: internal/projector/mirror.go qEvalFresh（原谓词 EXISTS fin_proposal.eval_id——O-36961 扩在册策略自然键臂）
  - type: doc
    ref: internal/platform/legset/legset.go ActiveSetUnionSansSubReq（策略臂仅 live/degraded——paper/probe 单腿 perp 无臂）
  - type: server_fact
    ref: 生产 PG-md fin_opp_eval dir_perp_mom 最新行 11-23 16:41 在写 vs SG 镜像冻结 11-22 17:46（提案引用行止）；md_open_interest 恒零=oi.go 信封误解（PublicGet 返 data 数组本体）
  - type: server_fact
    ref: fin_paper_fill paper 259 策略 kind 分布 funding 4918/pnl 263(均值1.0)/fee 259/open 258——pnl 饿死实证；md_ticks paper inst 龄 44-98h vs md_mark_index 全量新鲜
---

## 事实/症状

O-36961 paper→probe 晋升链全断（probe=0/live=0 全史）：

1. **confGate 恒拦**：202/202 判据过线策略 LB≤0——confGate 采样
   `kind IN (funding,swap,pnl)`，dir 族 pnl 摊提笔设计即为收入流观测
   粒度（dirpaper.go 头注自证），但生产 pnl 均值=1.0 行/策略——
   `dirPaperAccounting` 价源 `md_ticks` 对 paper inst 恒陈
   （tick_max_age_s=120s 闸→dir_mark_stale 跳过）。根因：提案立项即
   fin_retired→inst 出 ProposalArm；fin_strategy 臂仅 live/degraded；
   BasisSpotArm 只覆 basis 现货腿——**paper/probe 单腿 perp 无任何臂**。
2. **entry_cond_defer 潜在拦 89%**：dirEntryGate 读 SG fin_opp_eval
   最新行须 proposed 且 <72h；serve 谓词 qEvalFresh 只投
   `EXISTS fin_proposal.eval_id` 行——策略在册期 eval 臂续评产新 id
   行无提案引用（open>0 闸拒新提案）→**结构性永不投影**→SG 行冻结
   于提案时点→>72h 全判「评估行陈」。
3. **md_open_interest 恒零**：oi.go 把 `PublicGet` 返值当
   `{data:[...]}` 信封解——okxexec.doOnce 实返 `env.Data` 数组本体
   （InstrumentMeta/parseScanTickers 同律直解 []struct），每拍
   unmarshal array→struct 败 WARN 让位。

## 根因或策略解释

**谓词域错配律**：桥/订阅面的 SELECT 谓词按「行产生时的角色」写
（提案引用/活跃集 live 态），下游闸按「行消费时的角色」读
（在册策略/记账中策略）——中间生命周期态（paper/probe、已立项
未退役）掉出所有谓词=三面同断（tickers 订阅、kline 回填优先级、
eval 镜像）。资金流不断的假象掩盖：funding-rate 宇宙级订阅+funding
记账臂独立于活跃集，confGate 序列退化为 funding 噪声方差——
pnl 真 alpha 观测粒度饿死而序列仍"有数"，闸如实拦但拦的不是策略
质量是数据断供。

## 处置/修法

- eval 镜像：qEvalFresh 加 `OR EXISTS fin_strategy(state<>'retired'
  ∧ inst/template/direction 自然键命中)`——md 侧只读副本（md_1070）
  行集小廉价；孤儿键/退役键仍不投（谓词收窄不外扩，桩测三态锚）。
- 价源（并行席件）：dirpaper 对 -SWAP inst 改读 md_mark_index
  mark-price 桥面行（全量订阅本已覆——basis.go 期货腿同径先例），
  非 SWAP（mt5 CFD）照旧 md_ticks——免扩订阅集 259 腿。
- OI：oi.go 改直解 []struct（信封已被 okxexec 剥壳）。

## 验证/防复发

- 桩测：projector serve 谓词三态（在册键投/孤儿键拒/退役键拒+续评
  覆写）；dirpaper mark 源分流案；oi parse 真形案。
- 复发探针面：paper 策略 inst 的 md_ticks/md_mark_index 龄、
  fin_opp_eval 按 (inst,template,direction) 最新 ts 与策略 entered_at
  差——「在册角色谓词覆盖」立项自查清单：凡消费面按 state 过滤，
  上游谓词须含同 state 集。
