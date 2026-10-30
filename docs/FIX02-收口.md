# FIX-02 收口报告：mt5_swap/carry_diff 生产「断供」急诊

> 施工席（外部提示词单，管理席拟 2026-10-31）｜仓 `/opt/o`｜worktree `/opt/o-wt/fix-mt5-swap-outage`
> 分支 `fix/mt5-swap-outage`｜commit `[角色:施工] FIX-02 …`
> 探针只读面零触 mutating；本报告=票面要件五项：诊断实证 / diff 摘 / 消哑路径 / 测试绿戳 / 部署后终验锚。

## 0. 一句话结论

**桥无罪，票面前提被诊断否决**：mt5_swap/carry_diff 采集臂全程健康（PG-md 真源 4h 拍正常落行、tick 鲜活、探针三符号全绿）；SG 面冻结的真病灶=**o-projector（OCI→SG 八桥镜像）2026-10-30 02:56:57 UTC 重启后静默僵死 13.5h+，零日志零告警**。票面三假设（①mtapi 形态变 ②符号方言变 ③SymbolGroup nil）全部实测否决。码面按票面无条件项交付**消哑**（B2）；「根治 SymbolParams 解析层」（B1）**无病灶可修**——如实记录不虚造。

## 1. 诊断实证（swapsurvey/探针输出摘录→病灶定位）

### 1.1 dev→mtapi 实测（tools/swapsurvey 同径定向探针，2026-10-30 16:12Z）

BTCUSDm/ETHUSDm/SOLUSDm 逐 SymbolParams——**三符号全健康**：

```
=== BTCUSDm ===  transport_err=<nil> inband_error=<nil>
  result_nil=false symbol_info_nil=false symbol_group_nil=false
  sg: swap_type=SwapType_InPoints swap_long=-1778.7000 swap_short=0.0000 group="Standard\Crypto\BTCUSDm"
  si: desc="Bitcoin vs US Dollar" digits=2 points=0.01000 cs=1.0
=== ETHUSDm ===  （同构全绿，swap_long=-51.6）
=== SOLUSDm ===  （同构全绿，swap_type=SwapType_SwapNone）
```

与 docs/能力账/mt5-swap-survey-2026-10-25.md 档案逐值一致——**假设①②③全否**（无形态变、无改名/下市、无 sg=nil）。

### 1.2 生产径复刻实测（OCI 本机，env 凭据同桥径，16:24Z）

复刻桥调用链 Dial→Connect(env)→SymbolParams×3，含会话龄化 20s×3 轮——**全绿**（age 0→63s 全 OK）。排除：出口语/凭据径/25s 互踩（SG 核 09:31:59Z 起每 25s OpenSession 同账号并行在场）/网关址差异（两端同 `mt5grpc3.mtapi.io:443`）。

### 1.3 真源 vs 镜像对拍（病灶定位的决定性一步）

| 面 | mt5_swap | carry_diff | SOLUSDm tick | basis_ann |
|---|---|---|---|---|
| **PG-md（OCI 127.0.0.1:5435，o-collecto 真写目标）** | **13:11:50Z 当日** | **13:11:50Z 当日** | **16:28:25Z 鲜** | 16:27:59Z 鲜 |
| SG（镜像面，票面断供事实来源） | 01:13:03Z（冻结） | 01:13:03Z（冻结） | 02:56:55Z（冻结） | 02:55:59Z（冻结） |

**o-collecto 的 SG_PG_DSN 实值=`postgres://…@127.0.0.1:5435/o`（本机 PG-md）**——SG 上一切 md_* 行皆经 o-projector 镜像。真源健康+镜像冻结=**断供发生在镜像层**。

### 1.4 病灶实锤（o-projector）

```
Oct 30 02:56:57 systemd: Stopping o-projector.service…  /  Stopped.  /  Started.
Oct 30 02:56:58 [projector] 投影臂起（mode=serve）——八桥镜像+proposal 双向+config 第九桥副本
Oct 30 02:57:00 [projector] 对账毕——活跃集 115 inst、dcd 活策略 0 品
（此后 13.5h+ 零日志——重启前对账：173 inst / dcd 46 品）
```

- 重启时刻 02:56:57 与 SG 侧冻结窗（01:13–03:23 逐 kind 冻结线）吻合；重启后对账面异变（173→115 inst、dcd 46→0 品）为旁证异常，**留观不扩查（超票面）**。
- 桥臂 [mt5bridge] 日志侧写：会话1（10-29 17:13 起）polls 三拍全落；10-30 05:11:43 流断 goaway→05:11:49 会话2 起至今——**mt5_swap 13:11:50 拍=会话2 期内正常落行**，桥自愈证据链完整。
- SG 核 25s OpenSession 轮转（09:31:59Z 重启后 144/h 恒频、同账号 login 3001157 哈希比对实锤两端同户）为**新增互踩负载**，实测不阻断新会话 RPC（1.2 节），但长驻会话+恒频互踩的叠加风险已在 nav_live 注释自认（「并发会话互踩返零才是真实风险」）——**建议另票评估**，本票未动（票面边界：不动 OpenSession 会话管理）。

## 2. diff 摘（消哑 B2——票面无条件项）

`internal/mt5/bridge.go`+`internal/mt5/swapdialect.go`（后者=换算件自 bridge.go 拆出——行规模门禁 D-077/D-588 棘轮 300 行新超标阻断，纯码搬家零语义变更；与 swapdialect_test.go 同族对位）：

1. **rej 分键**：`symbol_group_nil` 注册入 Door（NewBridgeCollector）——原 177 行 `sg == nil → continue` 静默径（票面点名的第二哑径）获独立键。
2. **裁定重构**：poll 闭包拆 `pollSwap`（单拍逐 sym，返落行数+首败样本）+`fetchSwap`（单 sym 读→裁定→换算→落行）。裁定四分支：transport err / in-band err（`r.GetError()`——原码与 swapsurvey 不同，in-band 错误体此前整被吞）/ Result nil → `symbols_read_failed`；SG nil → `symbol_group_nil`；写败 → `tick_write_err`（既有键）。换算/meta/落行 SQL 逐行未动（换算三件迁 swapdialect.go）。
3. **全灭拍 WARN**：`len(syms)>0 && 落行数==0` → `log.Printf("WARN [mt5bridge] swap 拍全灭——%d sym 零落行（面上盲拍，样本: %s）")`，样本携首败 sym+败因串（诊断种子）。单 sym 败仍只计数（票面既有语义不动）。
4. **测试注入缝**：`symFetch` 字段（同 `dial` 惯例注释「测试注入面」）——失败形态 fixture 可注入。

## 3. 消哑路径说明

- **链径选择**：票面钦定「ops_event/日志 WARN（沿用既有告警链径，不自创通知件）」。实测 PG-md **无 ops_event 表**（fin 域件仅 SG）且 o-collecto 无其写权——collector 侧既有链只剩日志。故取 **journalctl 日志 WARN** 分支（`internal/data/okxws/stalewatch.go` 面死告警同构语义）；未新造任何告警基础设施。
- **告警阈值语义**：4h 拍语境下「全 sym 连续失败」判据取单拍全灭即警（拍间 4h、一天最坏 6 行，无刷屏面；dashboard「蒸馏 mt5_swap」行 warn=5h/dead=9h 阈对齐）。
- **边界遵守**：tick 臂零触、OpenSession 会话管理零触、4h 节拍值未动（诊断证明节拍非病）、in-band 错误体捕获属消哑裁定分支非解析改造。

## 4. 测试/绿戳

- `swapdialect_test.go` 扩三锚（失败形态 fixture 入测）：
  - `TestSwapPollFailShapesCounters`——四失败形（transport err/in-band err/Result nil/SG nil）分键计数+败因样本非空；
  - `TestSwapPollAllFailBlind`——全灭拍零落行判定+首败样本；
  - `TestSwapPollHealthyShapeLands`——实拍健康形（InPoints L=-1778.7/points=0.01/cs=1/mid=80000）照常落行+meta 年化链随行（消哑零伤正道）。
- check_all 全链绿戳：见 §8 机检行（build/vet/import/gofmt/env/rejects/migrations/bridge-audit/test 全谱）。

## 5. 部署后终验锚（诚实版——前提已修正）

原锚「部署后首拍 mt5_swap 落行+carry_diff 衍生」**在 PG-md 真源上已常态满足**（13:11:50Z 实证）。SG 可见面终验锚改为：

> **o-projector 修复/重启后（管理/DM 另行排），SG `md_signal` kind=mt5_swap/carry_diff ts 应在下一镜像拍内跟平 PG-md（mt5_swap 下拍 ≈17:11Z）；SG `md_ticks` venue=mt5 SOLUSDm 行复活。** 此为「生产验证段」终验判据——消哑码随 o-collecto 下个部署批生效后，若 swap 臂再盲，journalctl 必现 `WARN [mt5bridge] swap 拍全灭` 行。

## 6. 移交决策/管理席的处置件（施工不越权，仅列）

1. **o-projector 僵死修复**（生产运维=管理席全权面）：重启前建议先取僵死现场（goroutine dump/`systemctl --user status`）定位 02:57:00 后主循环卡点；重启后核对对账异变（115 inst/0 dcd）是否为真源实态。
2. **SG 面健康闸盲区**：SG 行冻结 13.5h 期间 dashboard「蒸馏 mt5_swap」行按 warn=5h/dead=9h 应早已红——无人/无件把它推向 Feishu。建议另票补「healthComps 红行→ops_event/Feishu」推送径（现=被动面板）。
3. **25s OpenSession 轮转互踩**（SG 核 09:31:59Z 重启后新增恒频负载）：与桥长驻会话同账号叠加，nav_live 自认风险——建议另票评估轮转缓存的调用方/节拍。
4. **basis_ann/funding_view 在 SG 的「死」同为镜像冻结伪象**（真源鲜）——本票未动，随 projector 修复自愈。

## 7. 施工偏离声明（collab.md §7 自检 H/结果回填要件）

- 票面 B1「按诊断实果修 SymbolParams 解析/调用层」：**诊断实果=解析/调用层无病**（三假设全否+生产径探针全绿+真源落行正常）。按票面「先诊断后修」结构，B1 的正确交付=如实的否证记录，非虚造码改。若决策席裁定需补「会话龄化自愈」类加固（如 swap 全灭拍触发会话重建），属新票面（涉 OpenSession 会话管理边界）。
- 票面假设序 ①②③ 全部否决——诊断超出票面预案，但全程遵守「禁先改/探针只读面/mutating 零触」。
- 触碰生产面=**只读取证**（journalctl/psql SELECT/只读 RPC）；零重启零写入零 mutating。
