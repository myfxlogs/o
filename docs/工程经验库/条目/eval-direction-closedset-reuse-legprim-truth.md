---
schema_version: 2
id: eval-direction-closedset-reuse-legprim-truth
kind: pitfall
risk: medium
status: verified
keywords: [fin_opp_eval direction CHECK, 闭集词表, cfd_long cfd_short, leg原语, mt5_dir_mom, direction复用, 腿形真相面]
scope: [internal/research 评估臂, internal/platform/legset 构型闸, fin_opp_eval/fin_proposal direction 词表]
summary: fin_opp_eval/fin_proposal.direction 是 DB CHECK 闭集词表（md_1100 末版 19 词）——新方向族扩词=迁移撞在途同约束票（#35487 perp_trend 同改）；正解=族位 direction 复用已在册词（long_perp/short_perp 位语义=仓侧非 venue 形），构型真相面归 legs 原语列（cfd_long/cfd_short 须随族定义入 LegPrim 闭集）。
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: []
evidence_refs:
  - type: code
    source: migrations/md_1100_gridfit.sql direction CHECK 闭集
  - type: ticket
    source: O-35163 / O-35487（同约束在途迁移）
---

# eval.direction 闭集复用 + 腿原语载构型真相

- **坑**：工单#35163（cfd-directional-family）。给 mt5_dir_mom
  CFD 族起 direction 名时差点新造 `cfd_long`/`cfd_short`——
  fin_opp_eval/fin_proposal.direction 是 CHECK 闭集（md_1100 末版
  19 词），新词=迁移；而 #35487 正在途扩同一约束加 perp_trend——
  两迁移撞约束=合并冲突+词表漂移双病。
- **律**：direction 词=**仓侧族位标签**非 venue/腿形——方向族
  （dir_perp_mom/perp-trend-daily/dirmom 各臂）统一
  long_perp/short_perp 在册词复用零迁移；腿原语才是构型真相面
  ——eval.legs JSON 须按臂发真腿名（mt5 CFD 臂=cfd_long/
  cfd_short，模板 params.legs 契源），LegPrim 闭集新增随族定义
  同 commit 入库（grammar.go 先例）。
- **配套**：fin_strategy.venue 有 DEFAULT 'okx'——提案开户径
  须从 fin_opp_template join 带 venue（mt5 模板→venue='mt5'），
  否则 CFD 策略行 venue 误标 okx 串仓键/摩擦 venue 径全错；
  legset.InstFitOK 对 venue 官名族用结构闸（拒 OKX inst 形+伪键，
  formOther 放行）——官名方言禁枚举白名单。
