---
schema_version: 2
id: close-spot-externally-settled-deadlock
kind: pitfall
risk: high
status: verified
keywords: [close sizer, externally_settled, 对齐零, residual_clear, bare_bag, 单腿径, ExecuteDir, venue实读, 平仓死锁]
scope: [internal/guard, internal/trading/execwiring, internal/trading/execution, 平仓路径]
summary: 平仓腿 venue 实读对齐零=外部已结非读败——整单 fail-closed 死锁；正解=该腿不构平腿+残量挂账，幸存腿前置归并走单腿径
first_seen: 2026-11-16
last_seen: 2026-11-16
evidence_refs:
  - type: ticket
    ref: 工单#21745（升件#21741）——仓#86 spot 腿被 residual_clear 外扫后 avail≈0.0012，sizedCloseLegs 对齐零整单拒，SWAP 空腿挂 4 拍决策 allowed 零执行
  - type: commit
    ref: fix/close-sizer-externally-settled（待审）
repro_or_symptom: "双腿对冲仓 spot 腿被外部清场（residual_clear 杂币清扫/外部移动）→venue 实读≈0→alignCloseSz 对齐零→sizedCloseLegs 整单 false→tend 每拍 fail-closed 不发任何腿→幸存 SWAP 腿永远平不掉，仓挂 closing 空转、策略开仓权限被停"
root_cause: sizedCloseLegs 把「腿 venue 实读对齐零」与「读败」同归 fail-closed——但对齐零有两种语义：读面坏（该拒）与标的已不在场（venue-flat=外部已结，该跳腿不该锁仓）。仓#86 实证：residual_clear 在腿 filled→fin_position commit 窗内把 VVV 当杂币扫掉（TOCTOU 根修见 fundccy-legcommit-toctou-residual-sweep），spot 腿从此 venue-flat，平仓径无路可走。
fix_and_prevention: "①对齐零腿出列为 settledLeg（记 venue/inst/avail/want）+对应槽清空，不构平腿不发残量单；读败/sizer 缺/lotSz 败仍整单 fail-closed。②幸存腿前置归并——execwiring 契约 Legs[1].InstID==\"\"→ExecuteDir 单腿径，双腿幸存保持原序走 Execute。③全腿外结=venue 已平→markClosed(0) 直迁 closed（决策行 close_exec_id=0 显式记零执行）。④残量挂账 noteSettledLeg：残量=min(avail,want)（avail 含杂币不全归本仓），spot 腿 bare_bag 三列+fin_holding spot_dust recon 行+「裸袋」事件行；swap/mt5 腿残仓>0=敞口升件（bareBagSwap 同律）。⑤trig.facts['externally_settled'] 进决策载荷留痕。⑥禁用「先回补现货再平」方案——回补币会被 residual_clear 再扫=依赖环。⑦凡 venue 实读消费点（sizer/对账/平仓量）区分「读败」与「读到零」两语义：零是事实陈述，是否致命由调用侧按腿集余量判。"
notes: "与 close-path-gate-denial-unreachable 互补：那条记 close 径闸拒不可达，本条记 sizer 计划期败的对齐零子类已从「整单拒」改为「逐腿外结」。bareBags 对拍环须跳空槽腿（外结腿无平腿可对）。"
---

## 平仓腿 venue 实读对齐零：外部已结≠读败——逐腿出列不锁仓

- **实证**：仓#86（VVV-USDT spot buy + VVV-USDT-SWAP sell 对冲）——
  residual_clear 在仓行 commit 前把 spot 币扫成现金，venue 现货余额
  剩 ≈0.0012 尘。此后每拍 close 触发（决策#35286–#35289 allowed）都在
  `sizedCloseLegs` 对齐零处整单拒：spot 腿算不出正量→`return cls,false`
  →连健在的 SWAP 空腿也不发→仓永远 closing，策略#286902 开仓权限被停。
- **教训**：「读不出」与「读到零」是两种事实——前者该 fail-closed 拒发，
  后者说明标的已不在场，锁死整单反而把幸存腿陪葬。逐腿判而非整单判：
  坏腿出列记账，活腿照走。
- **契约位**：execwiring `CloseExec.ExecuteClose` 以 `Legs[1].InstID==""`
  分派 ExecuteDir 单腿径——外结腿清空槽+幸存腿前置归并即复用现成契约，
  不需要新执行通道。
- **账实一致**：外结腿不产生平腿成交，不能假装卖出；残量（venue 实读
  余量封顶腿账求偿量）走 bare_bag/fin_holding/事件行挂账，决策载荷
  `externally_settled` 留名——簿实差由显形件吸收非由静默锁仓掩盖。
