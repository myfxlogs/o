---
schema_version: 2
id: promotion-inst-form-blind-gate
kind: pitfall
risk: high
status: observed
keywords: [promotion, 模板门闸, inst 构型, DCD 期权, funding-hedge, 盲 promotion, legset, InstFitOK, execfam, fin_proposal, fin_strategy]
scope: [internal/portfolio/lifecycle, internal/platform/legset, internal/platform/execfam, fin_proposal→fin_strategy promotion 径]
summary: execfam.Has 只核「有执行径」不核「inst 构型合族腿」——DCD 期权合约被盲 promotion 误建 funding-hedge paper 748 行；修法=legset.InstFitOK 族→构型映射闸
first_seen: 2026-11-14
last_verified: 2026-11-16
skills: []
evidence_refs:
  - type: doc
    ref: internal/platform/legset/legset.go
  - type: doc
    ref: internal/portfolio/lifecycle/run.go
  - type: server_fact
    source: postgres
    query: SELECT template_ref,count(*) FROM fin_strategy WHERE inst_id ~ '-[CP]$' GROUP BY 1
    observed_at: 2026-11-16T05:30:00Z
    server_revision: ""
    response_hash: ""
  - type: server_fact
    source: postgres
    query: SELECT id,inst_id,template_ref,state,direction FROM fin_strategy WHERE template_ref='funding-hedge' AND inst_id ~ '-[CP]$' LIMIT 20
    observed_at: 2026-11-16T05:30:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

promotion 径（fin_proposal→fin_strategy paper 开户）模板门闸只查 `execfam.Has(template)`——模板在执行族册即放行，不核 inst_id 构型是否合该族腿构型。生产实证：DCD 期权合约 `BETH-USDT-261xxx-<strike>-C`（期权产品形）配 `template_ref='funding-hedge'`、`direction='dcd_call'` 被批量误建 funding-hedge paper 策略（748 行 `-C` 尾 +若干，全部 retired；econ 矿 #273/#274 立案，工单#15943）。同型反向亦可能：SWAP/远期形 inst 配 dcd/basis 模板——腿派生臂（legset.*SpotArm）对错配 inst 静默失明或产垃圾键。

## 根因或策略解释

族册（execfam）语义=「模板有执行径」，与「inst 构型合族腿构型」是两个正交判据；门闸只实现前者。inst 构型知识此前散在 legset 派生臂的隐含假设里（FundingHedgeSpotArm 剥 `-SWAP`、BasisSpotArm 剥 `-\d{6}`+`-USD`、DCDSpotArm JOIN md_dcd_product.product_id），无任何一处把「inst 形↔族」写成显式契约——错配提案过闸后由下游记账/派生臂各自静默吞掉（dcd 族 JOIN 不中=无现货腿价源 fail-closed 让位；funding-hedge 误配=占位死仓）。

## 处置

`legset.InstFitOK(template, instID)`=族→允许 inst 构型唯一判据（inst 构型五类：swapPerp `-SWAP$`/datedFut `-\d{6}$`（DatedFutRe 同源）/dcdCall/dcdPut/dcdStrangle——DCD 期权形正则 `-[0-9]{6}-[0-9]+(\.[0-9]+)?-[CP]`，QUOTE 域 USDT/RLUSD/USDC/USDG 不收窄，strangle `+` 复合形双腿类型齐判）。映射：funding-hedge/xvenue-carry/dir_perp_mom→USDT 本位永续（叠 IsUSDTInst=D-0018 计价闸）；quarter_basis_long/perp_quarter_spread→datedFut；dcd_call/vol_sell_single_call→dcdCall；dcd_put→dcdPut；vol_sell_strangle→dcdStrangle；未注册族=false fail-closed。`admitProposals` 在 execfam.Has 后消费，不符→`blocked[tmpl+"/inst形不符"]` 提案留 fin_proposed（与模板门闸同「看见了做不了」语义）。覆盖断言 `TestInstFitOKCoversExecFam` 钉死 execfam.All() 全族有契约——扩族漏登构型=测试红。

## 证据

- check: `go test ./internal/platform/legset/` → ok（TestInstFitOKFamilyContract/TestInstFitOKCoversExecFam）
- check: `go test ./internal/portfolio/lifecycle/` → ok 232.9s（TestInstFormGateBlocksMismatch 拒例+存量 admit 测试全绿）
- ref: fin_strategy 生产实证——funding-hedge 模板下 748 行 `-C` 尾 inst（BETH-USDT-*-C）+ `okx/earn-flex` 伪键行，均 retired

## 反证条件

- falsifies_when: 发现合法提案被 InstFitOK 误拒（某族真实 inst 构型不在五类内——如新 venue 品种形态）→ 构型分类漏类需扩。
- falsifies_when: eval 侧另有 promotion 径绕开 admitProposals 直插 fin_strategy——本闸只管提案径，直插径不在覆盖面。

## 适用边界

- 只管 fin_proposal→fin_strategy promotion 径（admitProposals）；simdrill/excell 直插策略行、opener live 径不在本闸域。
- 构型分类按 OKX inst_id 形态归纳；新 venue/新产品族进册时须同步 legset.InstFitOK（测试钉死）。
- 评估侧前置拒（exec_family_missing）未同步加 inst 核——错配提案仍会产 fin_opp_eval/fin_proposal 行留 proposed 队列，由本闸拦建策略；评估侧前置拒属另案 scope。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a
- 豁免由：构型错配率无现成 gauges 表适配——错配提案滞留 fin_proposed 态可由 desk/滞留探间接观测；误建策略行=0 由本闸+测试断言承载。
