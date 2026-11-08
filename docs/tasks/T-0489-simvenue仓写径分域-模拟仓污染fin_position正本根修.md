# T-0489 simvenue 仓写径分域——模拟仓污染 fin_position 正本根修

- 状态：呈批（COS 立票呈 CTO 裁定——钱径域数据卫生件，范围/修法归 CTO 裁域）
- 归口：钱径域·账本写径（simdrill/simvenue 侧）

## 病灶（2026-11-08 19:00+ 实证）

simdrill 矩阵产跑的 simvenue 仓**直接写 `fin_position`/`fin_execution` 正本表，与真户仓零域区分**：

实证行（fin_position #35-39，2026-11-08 09:35:53~54 一秒连爆 5 仓，
strategy_ref 287285-287291 连号，生命周期 8~112ms，全 `fast_close`）：
- #36 MUU-USDT-SWAP settled=-0.3346（业主在 OKX 找不到订单——
  正确，单从未出模拟器）
- venue 面标记 `@okx` 只描述「模拟 OKX 形态」，成交=模拟器本地合成。

## 实害（三链污染）

1. **诊断额度空烧**：hold<5min→fast_close→postmortem pending——
   5 个 sim 仓正被当真案尸检（D1 闸对 sim 不设防）。
2. **业主对账失真**：`[经营事件]` 推送不写 venue，模拟仓按真仓
   播报——业主去 OKX 对账扑空（本案实发）。
3. **T-0145 判据腐蚀**：「残仓清零」验证读 fin_position——sim
   残渣会触发误报/掩盖真残；仓巡滞留哨兵同病。

## 修法方向（裁域——供 CTO 裁词择径）

- 径 A（荐）：simvenue 写径分域——仓行落 `sim_` 命名域表
  （sim_position/sim_execution），fin_position 只纳真户；
  正本表查询面零改动（天然隔离），下游各自选读。
- 径 B：fin_position 加 `venue_domain` 列（live|sim 强制非空），
  全下游谓词补域滤——改面大，漏改即污染。
- 配套（无论何径）：`[经营事件]` 推送面 venue/sim 标记必带；
  postmortem/D1 闸对 sim 域天然不入（径 A 免费得）。

## 判据

- sim 产跑后 fin_position 真户面零新增（径 A）或全行
  venue_domain='sim'（径 B）；
- [经营事件] 推送含 venue 标记，业主肉眼可分 sim/live；
- postmortem 队列对 sim 仓零立案；
- 判据不倚「记得加滤」——结构性分域为优（防漏改再污染）。

## 范围（供裁）

- internal/trading/simvenue + simdrill 写径；
- fin_position/fin_execution 写面；
- [经营事件] 推送投影面；
- 桩测新件（sim 仓不入真户面断言）。

## 禁线

- 禁动真户历史行；禁改 postmortem 判词语义；
- 禁把 sim 仓标成 live 交付对账。
