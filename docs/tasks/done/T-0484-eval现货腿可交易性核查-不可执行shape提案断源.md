# T-0484 eval 面现货腿可交易性核查——不可执行 shape 提案断源（T-0481 修法件②a）

- 状态：已批（CTO 批 2026-11-08——T-0481 修法件②选径 a=eval/提案面
  加可交易性核查排除；径 b（BTC-USD→BTC-USDT 对冲口径变更）=经济案
  级归业主线明确不裁、径 c（模板限 USDT 计价族）由本件落地后视情）
- 领取：无头施工席常径（swe-2-flash-high）
- 优先级：P2（armed 提案积压断源件——quarter_basis_long eval 产
  不可执行提案，armed 积压已 13 张首堵点域）
- 收口移交归：决策席
- 立项：T-0481 真径面 corollary——evalBasis 以 BTC-USD index-tickers
  指数价充现货腿价产提案（basis.go:75-79），指数≠可交易现货，
  评估/执行口径错位→不可执行 shape 进提案链

## 范围

1. `internal/research/` eval 面（evalBasis/提案产面——双腿腿件
   「venue 实有可交易性」核查：基键现货腿 inst 不在 venue
   instruments 实有面→该 shape 提案不产/直接判不可执行）；
2. 可交易性判据源：venue instruments 注册表/快照（与
   open.go loadMeta 同谱形判据——评估面只读消费，禁另造口径）；
3. 桩测断言：BTC-USD 系（现货不在册）shape 提案零产+
   在册 shape 照常产（修前红+修后绿）；
4. 禁动 execution/真径下单面（评估面断源，执行面 fail-closed
   兜底不动）。

## 验收

- 桩测断言绿+check_all 对口；
- eval 对不可执行 shape 提案=零产（产侧复跑后
  fin_opp_eval quarter_basis_long 类行不再新增——部署后自验锚）；
- 判据源与 loadMeta 同口径（口径错位不复发）。

## 红线

- 不动真径执行/下单/钱径码位；
- 不动 fin_opp_eval 存量行；
- 先红桩律适用。
