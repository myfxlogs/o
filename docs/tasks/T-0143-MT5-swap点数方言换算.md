# T-0143 MT5 swap 点数方言换算件（carry_diff 解锁闸）

## 态：施工中
## 领取：zcode-t0143（ZCode 施工席）@ 2026-10-26
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

## 结果回填（zcode-t0143 @ 2026-10-26）

- **换算件 SwapAnnualBPS**（internal/mt5/bridge.go 纯函数——载荷协议面取证：
  SymbolInfo TickValue(18)/TickSize(19)/Points(13)/ContractSize(20) proto 实读
  非臆断）：单位方言判据 SwapType 全谱——
  PercCurPrice(5)=报价币百分比/天→swap_short×365×10⁴ 直年化；
  **Points(1)=点数/天→币值/手/天=swap_points×TickValue×Points/TickSize，
  年化 bps=币值×365/(mid×ContractSize)×10⁴**（mid=md_ticks mt5 中点）；
  换算链任一环零（tv/ts/pts/cs/mid）=false→val=NULL 缺席如实；其余单位
  （MarginCurrency/Currency 族）=false 如实。
- **换算链全携 meta**（票面钦定）：tick_value/tick_size/points/contract_size/
  mid 逐环落 meta.bps 可查（swapLoop 落表段重构走换算件）。
- **两缺陷归修**：①swapLoop ctx 本就会话域（bridgeSession per-session ctx——
  断连 cancel 自动止拍，goroutine 零累积；验收锚③沿守）；②
  `symbols_read_failed` 拒因预注册（rejects.New 域）。
- **桩测**：swapdialect_test 三锚（Perc 直年化 -182500bps 精断/Points 数值形
  -50 点×链→-60833bps 精断/缺环四形全 false）全绿。
- **机检**：build✓ vet✓ gofmt✓ staticcheck✓ check-lines✓（bridge.go 265<300）。
- **解锁判据（部署后决策席实证）**：BTCUSDm/ETHUSDm mt5_swap.val 非 NULL
  （bps 量级合理）+carry_diff 2 对落行 ts 鲜≤2×拍；断连重连 swapLoop 零累积。

## 返修回填（zcode-t0143 @ 2026-10-26——swapLoop 会话域 ctx）

- 病灶实锤：swapLoop 直传 Run 外层 ctx（跨重连共用）——断连不 cancel，
  每次重连新起 swapLoop，旧 ticker goroutine 活到进程死=累积泄漏。
- 修：bridgeSession 内派生 sessCtx+cancel（defer 于会话终触发）——
  OnQuote 流与 swapLoop 同挂 sessCtx，stream 断返/defer cancel 双径皆止。
- 桩锚：TestSwapLoopStopsWithSession（sessCtx cancel→goroutine 退出断言，
  runtime 计数归位；sched-exempt-ok 期界等待）。全包测试绿。
- 余件勿动守线：换算件/两缺陷归修/桩测零改。
