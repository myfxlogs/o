---
schema_version: 2
id: lesson-producer-coverage-gap
kind: pitfall
risk: high
status: verified
keywords: [fin_econ_lesson, 教训产线, 覆盖面审计, 经验蒸发, 退役验尸, 举一反三]
scope: [lifecycle, fin_econ_lesson, retireminer, 经验管]
summary: 教训产线按对象粒度建（仓位级三件）漏掉父级事件面（策略级退役）——蒸发口大小与产线覆盖度无关，最大量对象（2175 退役）恰是零产线那件；新产线先审「哪个对象的什么事件量最大且无产线」再补
first_seen: 2026-11-13
last_verified: 2026-11-13
evidence_refs:
  - type: commit
    ref: fin_0172
  - type: doc
    ref: internal/portfolio/lifecycle/retireminer.go
---

## 经验产线覆盖面审计律：最大蒸发口往往零产线

- **实证**：fin_econ_lesson 26 案全仓位级（trade_postmortem×15/
  rootcause_diag×9/semantic_caliber×2）——产线三件都在跑；而
  fin_strategy_transition 退役行 2175 条**一条教训没产**。仓位结案
  有验尸、策略死亡（比仓位更高一级的经验单元）零产线——举一反三
  器官有矿无料，2175 次死亡经验全蒸发。
- **根因**：产线按「先想到的对象」建（仓位结案→postmortem），没人
  审「事件面全集 vs 产线覆盖集」差集——越大的经验口越容易漏
  （日常不可见、死完即走无人问）。
- **修法形**：扫「事实账已留痕而无教训行」的事件集补产线
  （本件=retireminer 扫 transition 未配套行）；留痕面在案=补产
  幂等自愈（UNIQUE(entity_ref) 天然去重）。
- **审计式**（新教训产线/矿面立项先跑）：
  `事件面全集（事实账行）− 产线覆盖集（kind 集）= 蒸发差集`
  ——差集按量排序，最大那件先做。
- **节流律**：矿面补产必带水位闸（pending<N 停喂）+扫窗+单拍
  限量——对齐裁定消化节奏，反淹矿面=另一种蒸发（产了没人裁）。
