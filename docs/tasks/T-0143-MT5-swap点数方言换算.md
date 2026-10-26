# T-0143 MT5 swap 点数方言换算件（carry_diff 解锁闸）

## 态：待派
## 域：桥面蒸馏 / MT5 能力面
## 源出：T-0141 验收（carry_diff 缺席根因——venue swap_type=1 点数方言）

## 背景
T-0141 验收实证：BTCUSDm/ETHUSDm `SwapType=1`（点数族）非 PercCurPrice——
`swap_long=-1786.5`/`swap_short=0` raw 已落 mt5_swap.meta，年化换算件缺=
val=NULL→carry_diff 永久缺席。点数→币值换算需 SymbolParams 的
TradeTickValue/TradeTickSize（或 ContractSize）径——mtapi 载荷字段先按
协议面取证纪律实读全 message，再定换算式。

## 实锤在案（防重取证）
- meta raw：`{"swap_long":-1786.5,"swap_type":1,"swap_short":0}`（2026-10-26）
- 空腿零成本事实：swap_short=0→短腿对冲免费（负 funding 径净值全归 funding 端）
- SwapType 枚举谱：skills/mt5-quotes-history + gen proto `SwapType_*` 全谱实读

## 施工要点
1. mt5/bridge.go swapLoop 扩：SwapType=Points 径——tick_value/tick_size 实读
   （SymbolParams 载荷全字段先 dump 核证，禁臆断字段名）→年化 bps 换算落 val
2. 同 commit 修 T-0141 在案两小缺陷：swapLoop 会话域 ctx（断连止拍）+
   `symbols_read_failed` 拒因预注册
3. 测试：点数方言换算数值桩+非 Points 仍 val=NULL 桩
4. 解锁判据：carry_diff 行鲜≤2×拍且 val 非 NULL

## 验收锚
- BTCUSDm/ETHUSDm mt5_swap.val 非 NULL（bps 量级合理——点数换算链全携 meta）
- carry_diff 2 对落行且 ts 鲜
- 断连重连后 swapLoop 零累积（进程 goroutine 数稳定）
