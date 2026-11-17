---
schema_version: 2
id: guard-no-second-exit-attempts-exhausted
kind: pitfall
risk: high
status: verified
keywords: [末路平仓, emergency_exit, attempts耗尽, reduce-only, 止血令, scanOnce, 仓#86, 敞口归零]
scope: [internal/guard, internal/trading/execution, internal/trading/execwiring, cmd/core]
summary: 守卫正径 plan/sizer 失效后 attempts 耗尽仓永出 scanOnce 批=敞口无第二出口；末路=guard 内置 reduce-only 出口（不共 plan/sizer，venue 实读定量，auto+order 双触发同径）
first_seen: 2026-11-17
related: [close-spot-externally-settled-deadlock, close-path-gate-denial-unreachable, order-leg-sz-unit-mismatch-naked-bag]
evidence:
  - type: ticket
    ref: 工单#22247（董事会点破件，账#22225）——仓#86 守卫自 20:35 起 5 试同被 sizer 卡死，attempts 达上限退出扫描集，永续腿裸空 4h30m 无人能归零
  - type: code
    ref: internal/guard/emergency.go + emergency_exit.go + internal/trading/execution/emergency.go + execwiring.EmergencyExec
repro_or_symptom: "对冲/方向仓触发平仓后正径持续败（sizer 读败/venue 拒/闸拒）→attempts 涨到 maxAtt→scanOnce 谓词 COALESCE(attempts,0)<maxAtt 永久滤出→仓卡 closing/close_failed 无人再碰；期间 venue 裸敞口（如对侧腿已外结的幸存合约腿）持续流血，tools 无应急平仓件（okxunwind/rescuespot 皆单案件）"
root_cause: 平仓重试预算耗尽=扫描集除名，但「除名」语义被当成了「已了结」——没有任何径区分「重试完应验收」与「重试完仍有敞口」。仓#86 实证链：residual_clear 外扫 spot 腿→sizer 对齐零整单拒（#21745 已修该子病）→attempts 耗尽除名→SWAP 裸腿无人能平。病根=安全件只有一条执行径，一条径死=全死。
fix_and_prevention: "①末路出口与正径分界：不共用 plan（gate.Check/令牌/live 闸/对冲带/Earn 赎回）与 sizer（sizedCloseLegs/alignCloseSz）——腿量=venue 实读残量（ContractPos/SpotAvailBal）对齐 lot 网格直发。②reduce-only 不可增敞口：合约腿 Close=true→buildOrderParams 置 reduceOnly；现货腿恒 sell availBal=只减向；合约腿方向=venue 符号位实读（signedPosSizer 可选面）优先、腿账侧兜底。③两触发同径：auto=attempts>=maxAtt 且非对冲态（裸敞口|全清收殓；双腿俱在对冲态不自动动）+order=ops_event「止血/仓#N」令行（自我席令票/处置席令行同通道，窗内 emg.seen 去重）。④幂等：跨次重跑键=venue 实读残量≈0→不发单；单次内传输败 Resolve 同 clOrdId（emg<exec>x<no>）定真态；单仓冷却 guard.emergency_retry_seconds。⑤venue 实读/sizer/lot 读败=整仓让位 fail-closed 升件不发单。⑥落账：fin_execution decision_id/token_id=NULL 无令牌位+fin_order_leg 腿行+bareBags 残袋对拍+markClosed 决策行（trigger=emergency_exit）+ops_event「末路平仓」行。⑦markClosed guard 行改 upsert——closed 终态必立且 trigger_kind 同步真值（受令可及无 guard 行仓）。⑧凡「重试预算耗尽除名」设计必配第二出口或显式终态收殓——除名≠了结。"
notes: "与 close-spot-externally-settled-deadlock 互补：那条修「对齐零腿死锁」，本条修「正径整体失效后无径可走」——#21745 是末路要处理的病灶之一。对冲态双腿俱在的耗尽仓 auto 不动（受令可达）——票面钦定 auto 限非对冲态。mt5 腿经 RouterExecutor 同径（OrderClose 携 ticket 天然只减）。"
---
