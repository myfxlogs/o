---
schema_version: 2
id: capgate-idle-refire-hold-verdict
kind: pitfall
risk: medium
status: verified
keywords: [idle_capital, 资金闸, 同值复燃, 沿再发, 持闸销, 定性销, 底层命题判词, capalloc, 敞口闸]
scope: [hunger 资金闸探针, ops_event 升件处置, 唤知会话判词面]
summary: idle_capital 升件=状态沿再发语义——条件真时每拍重火，定性销非清偿；同值（净值/需额/闸值/利用率近不变）=滑窗同态非新病，判词复用上件「闸门非瓶颈+持闸恰配」即可，数值跳变才重判。勿为压噪声改探针——会埋真 cap_hit。
first_seen: 2026-11-11
last_verified: 2026-11-12
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT id,ts,note FROM ops_event WHERE esc_scope='idle_capital' AND event='升件' AND ts>now()-interval '24 hours'
    observed_at: 2026-11-12T00:30:00Z
  - type: doc
    ref: internal/portfolio/hunger（资金闸探针 escEdgeU 沿再发面）
---

## 事实/症状

资金闸探针 `idle_capital`（需额<25%净值∧live在册>0）12h 窗内连升 6+ 件
（#10534/#10569/#10754/#10809/#10824/#11058/#11092），note 数值近同：
净值 16,076–16,094、需额 1,700–1,850、敞口闸 6,000、单笔闸 1,500、
利用率 28–31%。每轮唤知会话销一件，下一拍探针重火——销件速率追平
升件速率，信号面被同型复燃占格。

## 根因或策略解释

探针语义=状态沿再发（escEdgeU：销而未修=条件仍真即重发，#9122→#9141
先例钦定为设计内）——**定性销是判词层动作不是清偿层动作**（同能力债
档「定性销案≠清偿」同构）。条件（部署率<25%）持续为真→每拍重火属
探针正确行为，不是探针病灶也不是新升件病灶。复燃件与首件的差只在
时间戳与滑窗数值尾漂。

## 处置

- **同值复燃判词**（note 数值与上件近同，±滑窗漂移内）：销+note 引上件
  判词+「底层命题判词:真（闲置部署如实）——非病灶：<一句>」。判词现配
  =闸门非瓶颈（开仓拒因看 fin_opp_eval 拒因分布，breakeven/费率水位
  主导）+商业产出 30d 净负→无加闸证据锚→持闸恰配（闲置=保守判词）。
- **数值跳变重判**：需额≥90%闸（升级 cap_hit 路径）/闸值被改/利用率
  趋势显著移/live 族转净正窗口→capalloc 窄门三态重判（加/减/持）。
- **禁面**：勿改探针压噪声（同值重火=设计内信号形态，压了连真
  cap_hit 一起埋）；勿以「市场不好/闲置正常」空判词销——必附命题判词。

## 证据

- check: `SELECT id,note FROM ops_event WHERE esc_scope='idle_capital'
  AND event='升件' ORDER BY id DESC LIMIT 6`——连拍 note 数值对拍
- result: 6 件数值域重叠（净值±0.1%/需额±8%/闸不变）=同态复燃实证
- ref: 升件#10534→#11092 处置行批档（v28/v29/v31 同判词序列）

## 反证条件

- check: 升件 note 数值结构突变（需额跳升超闸/闸值变化/live 数变）
  而处置仍引旧判词——复核是否按同值误销
- falsifies_when: 探针改真边沿触发（只发条件翻真沿）后同值不再重火
  →本条退（refire 形态本身消失）；或 cap_hit 同式复燃证明本档
  判词面应并档而非分件。

## 适用边界

仅适用 `esc_class=资金闸 scope=idle_capital` 的同值沿再发件；
cap_hit（需额≥90%）方向相反、idle 但 note 数值结构变化（新族上册/
闸值调整/NAV 大动）的件不适用同值判词——须重走 capalloc 三态。
商业产出 commercial_stall 同型复燃共用本判词骨架但拒因分解口径
不同，见 资金闸/商业产出 两档分裁。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：判词面判例非运行阈值指标——验收锚=同窗
  升件 note 数值对拍+处置行判词连续性，非量表可测面）
