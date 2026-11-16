---
schema_version: 3
id: dcd-exit-venue-capability-immature
kind: case
risk: low
status: verified
keywords: [dcd_call, 退场终裁, venue物理缺, 能力面不成熟, MANAGE-151]
summary: DCD全族退场终裁35案=venue物理缺类设计退非教训——能力面不成熟恒拒，防点=收养前能力面核成闸。
first_seen: 2026-10-30
last_verified: 2026-11-16
evidence_refs:
  - {type: table_row, id: "fin_econ_lesson#229-263(dcd_call×35)"}
  - {type: ops_event, id: "139/141(MANAGE-151部署收口)"}
  - {type: ops_event, id: "15867/18464(终裁清偿销件)"}
  - {type: ops_event, id: "18466(工单canon-dcd-retire-consol)"}
---

# DCD全族退场终裁簇=venue物理缺类设计退（判词口径定案）

**情境**：fin_econ_lesson retire_postmortem 簇中 reason=「DCD全族退场
终裁——能力面不成熟恒拒（MANAGE-151裁二/DM执）」共 35 案
（#229-263），全 direction=dcd_call、venue=okx、template_ref=dcd_call、
from_state=paper、positions=0、execs=0、realized_usd=0、
notional_usd=50、age_days=3.60，died_at 全同刻
2026-10-30T02:56:06.0575Z——批量退场非逐案死亡。
**实数**：inst 键=ETH-USDT-261027/261028/261029 系与 BETH-USDT 系
OKX 期权合约（-C 看涨），paper_net_usd 微正 ~0.004-0.009（#229-244
段）后段归零；proposal_ref 各异——族提案面广撒网同批清退。
**根因**：DCD 产品面在 OKX venue 物理缺类=能力面不成熟恒拒——
非策略死于行情/执行/仓位管理，是 venue 根本不供该产品类的设计态
退场。MANAGE-151 裁二定案、DM 执行部署收口（ops_event #139/#141），
终裁已清偿。
**判据挂接**：可防一问=venue物理缺——防点不在策略内部而在收养前
能力面核成闸：venue 产品类/能力面未核成前不立族。
**回改点**：semantic:retire_postmortem判词口径——DCD全族退场终裁簇
verdict_ref 引 `canon:dcd-exit-venue-capability-immature`，免每案
重推同判词（#229-263 同批判例即本条前身；verdict_ref 锚定 UPDATE
归复审侧，本条只交条目+索引）。
**证据**：fin_econ_lesson #229-263 numbers/verdict_ref 实拍（35 案
同因同批同刻）；ops_event #139/#141 MANAGE-151 部署收口、#15867
dcd_call calib 失真升件由终裁清偿销（预测对象物理缺不可兑现）、
#18464 dcd_put 评估断=同因设计态+残仓续管注记、#18466 工单锚。
**适配界**：①**分簇界**——同族 dcd_call 另有 594 案「产品提案被
模板盲 promotion 误建成 funding-hedge paper——出范围污染清扫」
（reason 异、死法=管线误建非 venue 缺类、防点=模板门闸 D-0018
令），判词口径不同**不在本 canon 锚域**，勿误引；②残仓不强平——
退场=族提案面清退，存量仓位归 lifecycle/guard 续管至到期（#18464
残仓 287249 BTC-261204-6600P 实证）；③dcd_put 评估断 3d=同因
设计态非病灶（#18464）；④本 canon 只罩 retire_postmortem
reason=「DCD全族退场终裁%」簇——eval proposed 面 dcd_call×529
僵批清退归方向票 eval-dcdcall-purge（ops_event #19020）另案，
非本条判词域；⑤本簇全零仓零执行，positions>0 实仓案不在此列。
**处置**：判词口径结晶本条；轮裁弃形不变（逐案 verdict_ref 行
必落），变的是引用源（canon 锚替判例链）。
