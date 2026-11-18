---
schema_version: 2
id: promotion-inst-form-blind-gate
kind: pitfall
risk: high
status: verified
keywords: [promotion, inst构型, 期权inst, funding-hedge, BETH-USDT-*-C, 模板门闸, AcceptsInst]
scope: [internal/portfolio/lifecycle, internal/research/grammar]
summary: promotion 径模板门闸只查经济判据不核 inst 构型——DCD 期权合约
 inst（BASE-QUOTE-YYMMDD-STRIKE-C|P 形）被盲 promotion 误建进 funding-hedge
 paper 占位死。修法=grammar.Family.AcceptsInst 构型核（期权形须含锁仓
 腿/远期形须含季度期货腿），promote 两段（paper→probe∨probe→live）同
 拦 inst_form_mismatch
first_seen: 2026-11-13
last_seen: 2026-11-18
evidence_refs:
  - type: ticket
    ref: 工单#15943——econ 矿#273/#274 实证，live 占位死
  - type: table
    ref: fin_strategy——funding-hedge|ETH-RLUSD-261106-2770-C/2780-C 两行
      retired 误建尸在案（期权形 inst 进 carry 族）
  - type: commit
    ref: fix/probe-batch（待审）
repro_or_symptom: "promotion 候选评估按 (inst,模板) 直配——期权形 inst
 BETH-USDT-261027-*-C 落 funding-hedge 族评估/晋升，paper 占位占 TTL 不产
 真单（期权 inst 在 carry 执行面无腿径）"
root_cause: "promote() 判据链（时长/净回报/回撤→confGate→entryCondition→
 execDrillPass）全查经济/执行凭据——inst 构型 vs 族腿构型无一问。族定义
 有 Legs 原语（grammar.Families 闭集），但 promotion 径从不消费它；inst
 形态语法（期权=BASE-QUOTE-YYMMDD-STRIKE-C|P、远期=BASE-QUOTE-YYMMDD）
 亦无判件——形态不合的配对一路绿灯到 paper"
fix_and_prevention: "①grammar.Family 加 AcceptsInst：期权形 inst 须族含
 LockedCcyCall/LockedUsdtPut 腿，远期形须含 QFutLong/QFutShort 腿；spot/
 swap/非 inst 形不拦（构型核不越界管计价域——D-0018 USDT 计价另闸）。
 ②LookupFamily 未注册模板让位不拦（裁定族 dir_perp_mom 等非文法族不误
 伤）。③promote() 两段同闸：paper→probe 拒建+probe→live 纵深再拦存量
 误建。④凡「配对型门闸」先问：配的两端各自构型核了吗？只核单边=另一边
 形态不合混进。"
notes: "与 promotion 链其他闸正交：本闸=静态构型配（无市场态依赖）；
 confGate/entryCondition/execDrillPass=动态判据。存量误建行不追销——
 retired 尸留案，闸只防新配。"
---
