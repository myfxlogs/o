---
schema_version: 2
id: dcd-product-position-invisible-in-positions-api
kind: pitfall
risk: high
status: verified
keywords: [幻影仓判据, positions API, sfp, dcd_live, 期权仓, BTC-USDT-YYMMDD-CCCP, /account/positions 覆盖面, order-history, venue账单, 零腿仓, legs.one]
scope: [internal/guard, internal/trading/execution, external okx dcd/sfp]
summary: OKX /account/positions 只覆盖 SWAP/普通产品面——sfp/dcd 产品仓（如 BTC-USDT-261204-66000-P 单产品期权仓）在该面恒空，positions=[] 判「幻影仓」前提错；真仓判据=fin_position 行+dcd_live 指纹+venue order-history LIVE 态+venue 账单三证交叉，缺一不可
first_seen: 2026-11-17
last_seen: 2026-11-17
evidence_refs:
  - type: ticket
    ref: 工单#22845——票面称仓#42（exec 1134，BTC-USDT-261204-66000-P，50 USDT 名义）为幻影仓需定账；venue /account/positions=[] 被当作「零敞口」实证
  - type: commit
    ref: fix/guard-leg-anomaly-throttle@c16968404（legs.one 腿形态异常降频定性——零腿=dcd 单产品仓合法腿形态非病灶）
repro_or_symptom: "工单立「仓#42 幻影」前提=venue positions=[]；实查 fin_position id=42 closed_at NULL、execution_id=1134 state=open、inst_id=BTC-USDT-261204-66000-P、notional=50 USDT，venue 侧该 inst 订单 LIVE 态、dcd_live 指纹在册、venue 账单有 -50 USDT 成交扣款——真仓 50 USDT 在场"
root_cause: 「venue positions API 空=无敞口」是 SWAP 域直觉的错置。OKX sfp/dcd 产品（单产品期权/dated futures 类）的持仓不进 /account/positions 返回面——该面只枚举永续/期货标准仓位族；dcd 产品的敞口唯一可查面是 sfp/dcd 域 order-history + bills。判「幻影」时读了错误的读面，positions=[] 是覆盖面缺失非敞口为零。
fix_and_prevention: "判任何 venue 仓是否幻影前先核 inst 的产品族：①SWAP/永续→/account/positions+order-history 双证；②sfp/dcd/期权族→positions API 结构性空集，改走 sfp/dcd order-history（LIVE/部分成交态）+venue bills（名义扣款行）+dcd_live 指纹表三证。零腿仓≠畸形：dcd 单产品仓本就无腿可挂（fin_order_leg 行数=0 合法），guard legs.one 分支对它恒 WARN 属噪音病灶非仓病灶——降频定性走 fin_guard_close qualifier 行（attempts 不动不进末路 auto 径）。「定账为幻影」=fabricate 假账，真仓误定账会双向污染 NAV 与对账差。"
notes: "与 datedfut-leg-phantom-cashflow 互补：那条记 datedFUT 腿被 instID 后缀域判据漏排致落账错域；本条记判据读面错——positions API 的产品覆盖面本身就是判据前提，判据适用域=它看得见的域。另：dcd_live 族无结算消费件是已知真缺口——到期结算后仓位仍 open 态无消费者收敛，属另案立票面。"
---

## DCD/sfp 产品仓不在 /account/positions 面——幻影判据先核产品族读面覆盖

- **实证**：工单#22845 立案前提「仓#42 幻影=venue positions=[]」。
  实查 `fin_position id=42`：closed_at NULL、execution_id=1134、
  strategy_ref=287249、inst_id=`BTC-USDT-261204-66000-P`（sfp 族
  dated 期权单产品）、notional 50 USDT。venue 侧该 inst order-history
  LIVE 态、dcd_live 指纹在册、账单有成交扣款——真仓在场，票面前提错。
- **病灶读面**：`GET /account/positions` 返回面只枚举 SWAP/永续标准
  仓位族；sfp/dcd 产品（单产品期权、dated futures 类）的敞口在该面
  **结构性恒空**——positions=[] 是 API 覆盖面缺失，不是敞口为零。
  把它当「零敞口实证」=读了错面得出假阴性。
- **判据族谱**：
  - SWAP/永续仓：`/account/positions`（posSide/net 模式）+
    order-history 双证即可。
  - sfp/dcd/期权族：positions API 恒空，改走 sfp/dcd 域
    order-history（LIVE/部分成交态）+ venue bills（名义扣款行）+
    dcd_live 指纹表——三证交叉缺一不可。
- **连带**：dcd 单产品仓 `fin_order_leg` 零行=合法腿形态（单产品
  无对腿可挂），guard `legs.one` 分支对它反向平径恒不可达、每拍
  WARN 刷噪音——腿形态异常已降频定性（`fix/guard-leg-anomaly-
  throttle`，qualifier 行入 `fin_guard_close`，attempts 不动）。
- **不可做**：对真仓「定账为幻影」=fabricate 假账——NAV 虚减、
  对账差虚增、且 venue 侧真实敞口失去账本锚。反证在案：判据面
  错置时挂起还票+报反证，不替票面前提圆场。
- **同族缺口**：dcd_live 族无结算消费件——到期结算后仓位仍 open
  态无消费者收敛，是真缺口非本案病灶，已报唤知立案面。

## 关联

- 工单#22845——仓#42「幻影」票面前提错实证+守卫腿形态异常降频
- `datedfut-leg-phantom-cashflow`——instID 后缀域判据漏 datedFUT
  的姊妹案（判据域边界同族）
- `guard-no-second-exit-attempts-exhausted`——guard 末路径判例
  （腿形态异常定性行刻意不动 attempts 即为避此径）
- `drill-cleanup-misses-guard-close`——带外关账径漏 guard_close
  的姊妹判例（本案 qualifier 行正是 guard_close 面）

## 附录：三证取证命令形

```sql
-- ① fin 侧仓行
SELECT id, execution_id, strategy_ref, inst_id, notional, closed_at
FROM fin_position WHERE id = 42;

-- ② exec 状态+腿数（dcd 单产品仓零腿合法）
SELECT id, state FROM fin_execution WHERE id = 1134;
SELECT count(*) FROM fin_order_leg WHERE execution_id = 1134;

-- ③ venue 侧：sfp/dcd 域 order-history LIVE 态 + bills 名义扣款
--    （/account/positions 对 sfp/dcd 恒空——勿以它判幻影）
```

---
*首见：工单#22845 施工席取证（2026-11-17）；复核：判据域边界与
simdrill/datedfut 两案互证。*
