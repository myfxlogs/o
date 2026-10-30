# T-0143 MT5 swap 点数方言换算件（carry_diff 解锁闸）

## 态：驳回：锚② ts 鲜不达——mt5_swap 生产停摆~13h（2026-10-30 01:13Z 后零写，tick 侧秒鲜存活），carry_diff 鲜门断供
## 审计席：headless @ 2026-10-30
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

## 二返修回填（zcode-t0143 @ 2026-10-26——InPoints 正典式）

- venue 实证（防重取证）：BTCUSDm/ETHUSDm tick_value/tick_size=0 **不填**、
  points=0.01/contract_size=1 实填——一返修 tick 链（×TickValue/TickSize）在
  真实 venue 形必 false=换算永缺席。
- 正典式：点值=points×contract_size（报价币/点/手）；币值/手/天=swap×points×cs；
  年化 bps=swap×points×cs×365/(mid×cs)×10⁴（cs 约掉量纲自洽）。
  tick 双字段降候选佐证非必需环（0=不阻断，meta 照携）。
- 桩锚改真实 venue 形：ETHUSDm 空腿免费形（tick 双 0/swap_short=0→ann=0
  合法产出非缺席）+swap_short=-1786.5 代入 mid=79842→≈-816.8bps/yr 量级断；
  缺环桩改真缺环形（points=0）。
- 验收判据不变：mt5_swap.val 非 NULL+carry_diff 落行（部署后实证）。

---

## 验收档（决策席 2026-10-26）✅（三轮）

- **合入**：`dc6dd67`→master（build✓ test 7✓——含会话域泄漏桩 TestSwapLoopStopsWithSession）
- **返修一**：sessCtx+cancel 会话域派生落（bridgeSession defer 止拍——goroutine 零泄漏）
- **返修二（根因）**：venue 实证 tick_value/tick_size=0 不填——换算式归正典点值径
  `点值=points×contract_size`（tick 双字段降候选佐证非必需环）
- **生产实拍**：mt5_swap val=0.00 真实零（swap_short=0 空腿免费）；
  **carry_diff 投产** BTC +621/ETH +532bps 56s 鲜；xvenue_spread 同鲜
- **消费语义钉**：carry_diff 大**负**值=亮牌（funding 砸负→长 perp 收+空 CFD 零成本组合）；
  正值=funding 正常态非机会读数。下游消费面禁反读。
- **坑沉淀**：session-scoped-goroutine-ctx 条目入工程经验库（worker 顺带归档）

---

## 判决注记（无头审计 @ 2026-10-30，收口 diff 5d9ca42+4fe1bf6+dc6dd67 逐锚对账）

- **锚① mt5_swap.val 非 NULL+链全携 meta：过**——SG fin.md_signal 实查：
  BTCUSDm/ETHUSDm val=0 真实零（meta ann_ok=true、swap_short=0 空腿免费合法
  产出），meta 全携 points=0.01/contract_size=1/tick 双字段/mid/swap raw；
  SOLUSDm swap_type=0→val=NULL ann_ok=false 非 Points 方言 fail-closed 正确。
  码面 SwapAnnualBPS InPoints 正典式（points×cs 点值链）与票面一致，
  -1786.5 点×0.01×1×365/(79842×1)×10⁴≈-816.7bps 量级断与桩断相符。
- **锚② carry_diff 落行且 ts 鲜：不达（生产实态）**——2 行在（BTC-USDT-SWAP
  +253.0/ETH-USDT-SWAP +330.2bps），但 ts=2026-10-30 01:13:03Z 实测陈 ~9.9h。
  根因链实锤：上游 mt5_swap 同刻 01:13Z 后零写，sigdist 8h 鲜门+血缘 ts
  （xvenue.go）断供=设计内 fail-closed；同 OCI 采集进程 OnQuote 侧 md_ticks
  venue=mt5 <1s 鲜、xvenue_spread 8.7s 鲜、岗位 beat 4s——进程与会话存活，
  仅 swapLoop 产出径停摆，跨 ops_face 死阈值（mt5_swap warn5h/dead9h）。
  停摆因未定（rejects 内存态 OCI 不可达：SymbolParams 拉取无每调 deadline
  悬挂形与凭据/token 失效形皆可能——ops 注记钦定查 OCI 桥 swapLoop 凭据/
  gateway），按「不确定→驳回避险」落驳回，重飞/运维先实查 OCI collector。
- **锚③ 断连重连 swapLoop 零累积：过（码面+桩锚）**——bridgeSession sessCtx
  派生+defer cancel、OnQuote/swapLoop 同挂（bridge.go:104-133），桩
  TestSwapLoopStopsWithSession 断言退出+goroutine 计数归位；运行态计数
  远端不可达，无反证。
- **附查（报管理席——本票面外事证）**：同刻面 basis_ann 陈 8.2h、
  funding_view 陈 7.7h——多生产源同窗停摆疑面更大，宜并案排查。
- **其他对账**：symbols_read_failed 注册+挂载一致（bridge.go:75/174）；
  bridge.go 295 行≤300；KB 条目 session-scoped-goroutine-ctx 在库；
  commits 含 T-0144 后续叠加（SwapAnnualBoth/meta 扩），对账以票面锚为准。

---

## 判决注记·复核（无头审计二跑 @ 2026-10-30，收口 diff 5d9ca42/4fe1bf6/dc6dd67 独立逐锚对账）

- **锚① mt5_swap.val 非 NULL+链全携 meta：过（写时达标）**——生产库（隧道 127.0.0.1:5544）实查：BTCUSDm/ETHUSDm val=0 非 NULL（swap_short=0 空腿免费合法零非缺席），meta ann_ok=true 全携 points=0.01/contract_size=1/tick_value=tick_size=0/mid/swap raw，swap_long_ann_bps≈-842/-848bps 量级合理；SOLUSDm swap_type=0→ann_ok=false/val=NULL 非 Points 方言 fail-closed 正确。码面 SwapAnnualBPS InPoints 正典式与二返修 diff 一致，symbols_read_failed 预注册、sessCtx 会话域、KB 条目 session-scoped-goroutine-ctx 均实存。
- **锚② carry_diff 2 对落行且 ts 鲜：不达**——2 对落行在（BTC-USDT-SWAP +253.0/ETH-USDT-SWAP +330.2bps），但 ts=2026-10-30 01:13:03Z，实测陈 13h10m（票内解锁判据 ≤2×拍=8h 断供）。上游 mt5_swap 同刻停摆；同时刻 md_ticks venue=mt5 BTCUSDm/ETHUSDm 秒鲜、xvenue_spread/depth5 秒鲜——采集进程与 OnQuote 会话存活，独 swapLoop 产出径死。停摆根因本席取证径不可定（rejects 内存态不可达），按「不确定→驳回避险」。
- **锚③ 断连重连 swapLoop 零累积：码面过/运行态未证**——sessCtx 派生+defer cancel+OnQuote/swapLoop 同挂（bridge.go），桩 TestSwapLoopStopsWithSession 断言在位；但生产 swapLoop 独停~13h 本身即稳定性疑点，无正面实证。
- **结论**：驳回。重飞/运维先查 OCI 盒 collector swapLoop 停因（SymbolParams 拉取悬挂/凭据失效形皆候选），复拍后锚②自回绿。
