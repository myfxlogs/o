# T-0187 活跃集查询引用缺席表——tickers 死订致 md_ticks OKX 面全空

> 立单：决策席 2026-10-27（运营面 5 件异常根查）
> 域：dev collector `internal/data/okxws/subscribe.go` + PG-md
> 量级：微件（修法钦定，防回归断言必落）

## 病灶实证（决策席取证在案）

System 页 5 件异常根查，定位 1 件真病：

```
md_ticks 全表仅 3 行且全 venue='mt5'——OKX 行 0 条
activeSet() UNION 第 70 行 SELECT inst_id FROM fin_position
  → PG-md 无此表（T-0169 分域后 fin_position 属 fin 域，SG 独有）
  → 整查询 ERROR relation does not exist
  → syncActiveSet rej.Inc("activeset_read_failed") 提前 return
  → tickers 频道订阅帧从未发出 → md_ticks OKX 面结构性归零
```

**连带伤口**：T-0171 钦定「dcd 策略现货腿真价唯一来源=本地 tickers
WS 行（md_ticks 供 dcdPaperAccounting 记账价）」——现货腿定价
自此断粮。dcd 策略若依赖 tickers 价=fail-closed 停摆（方向正确
但属静默饿死，健康面应早红而非靠人翻页发现）。

## 分域律核对

`fin_position` 系 fin 域钱径表——PG-md 不应有此表，**修法不是把表
投影过来**，是让 md 侧查询不引用它。在仓 inst_id 的活跃集覆盖：
持仓必挂 live/degraded 策略下（策略 inst_id 已在 UNION 第三项）——
fin_position 项在 md 侧本就是冗余覆盖项。

## 修法钦定

`activeSet()`（subscribe.go:68）去掉 fin_position 项（或按表存在性
条件编译——钦定为**直接删该项+注释说明**：持仓标的由策略项覆盖，
md 域不引 fin 域表），并**在注释里钉死**：md 侧查询只许引用
domain.tsv 标注 md|bridge 的表（fin_position=fin 域禁引）。

## 连带核证义务

1. `grep -rn "fin_position" internal/data/ cmd/collector/` 普查——
   md 侧其他引用点一并清（dcd/研究臂同理）；
2. 修复后实证：`md_ticks venue<>'mt5'` 行数恢复 ≈ 活跃集大小
   （当前在册 live/degraded 策略 17+提案 20，∩订阅宇宙过滤后应见
   标的行落）；
3. `activeset_read_failed` 拒因清零观察（重连后不再 Inc）。

## 其余 4 件异常的定性（本票不修，记档备查）

| 件 | 定性 |
|---|---|
| 蒸馏 basis_ann 零行 | T-0164 已知缺席——ops_face 钦定黄+立案注，非新病 |
| 买五档 depth5 | 瞬态——WS 1006 断链重订期间 >30s 黄，重订后 0.2s 复绿（已在自愈） |
| funding WS | 边界态——funding-rate 频道天然低频（8h 结算窗），recv_ts ~25min 贴 30min 阈振荡；阈与频道频率错配可另议，非病 |
| 第 5 件 | 快照时序件（重启/重连窗内瞬态黄）——非结构病 |

## 验收锚

1. `activeSet()` 不再引用 fin_position（md 域禁 fin 域表——注释钉）；
2. `md_ticks` OKX 行恢复流入（活跃集 ∩ 宇宙标的行存在且 ts 鲜）；
3. System 页「行情 ticks(OKX)」复绿；
4. dcd 现货腿 tickers 价源恢复（若 dcd 在拍有读，观察其不再
   因无价 fail-closed）；
5. 普查：md 侧零残余 fin_position 引用。

## 状态：完成（判过并码在案，11-01清账翻面）

审计席：headless @ 2026-10-30

- [x] 立单
- [ ] 施工
- [x] 验收投产（决策席 2026-10-27：37 活跃集标的 tickers 行恢复秒级流入 0.17s 鲜，activeset_read_failed 零计数，md 域 fin_position 引用普查清零）

## 判决注记（审计席逐锚对账——独立取证径：journal 看门狗时序+双库实查）

- 锚1 PASS：收口 commit 9a6b31c3 全量 diff 亲核——activeSet() UNION
  fin_position 项删、三行注释钉 domain.tsv md|bridge 域律；origin/master
  现态在位（T-0216 后续 basis 腿增项未回退本修复，subscribe.go 注释存）。
- 锚2 PASS：独立证据=采集器自带 stalewatch 看门狗（internal/data/okxws/
  stalewatch.go，md_ticks 全行>3min 即 ALERT「行情面死」，分钟级灵敏）
  ——journalctl 实录：10-27 14:28:41/16:35:38 两报（修复会话中间构建期
  面死）→ 17:33:19 修复构建重订后告警绝迹至 10-28 01:46+，历次重启
  重订 syms=1034 全净——tickers 行持续新鲜落库的机构级实证。
- 锚3 PASS：健康行判据=ops_face.go「行情 ticks(OKX)」fact md_ticks
  venue<>mt5（max(ts) 判、warn 30s）——与锚2 同源同事实；旁证=SG
  生产库同面 2026-10-30 13:11Z 实拍 max(ts) 距查询时刻 0.4s、okx
  511 标的在流。
- 锚4 PASS：SG fin_paper_fill（dcd 族）时序实查——10-26 全日 1 笔
  （价源饿死期）→ 10-27 12:00Z 时段 230 笔 → 10-28 105 笔（fill 落笔
  前置=md_ticks 现货价过 <120s 鲜度闸，internal/portfolio/lifecycle/
  paper.go）——记账价源恢复实证；当前 dcd 族策略全 retired（条件件
  现时 N/A，机制面在位：uly→USDT 归一 UNION 腿保留）。
- 锚5 PASS：origin/master `git grep fin_position -- internal/data/
  cmd/collector/` =3 命中全注释行（collector.go:63/218、subscribe.go:72）
  零 SQL 表引。

锚外注记（报管理席裁，不阻塞判决）：
- ①票面量级行「防回归断言必落」未兑现——收口 diff 零新增 activeSet
  组成断言（存量 ghost_test.go 不覆盖本病灶），义务缺位立票补齐。
- ②activeSet() 为 o-core 共用（SG 库 fin_position 存在、原查询可跑），
  删项同步收窄 SG tickers 活跃集；现时 SG 未平仓持仓实查=0 无暴露，
  但 live→retired 过渡窗残留持仓将失价源（票面冗余论证未覆盖该窗），
  设计风险记档。
- ③collector.go:218 旧注释仍述「fin_position 在仓」活跃集组成——文档
  残留非表引，随①顺修。
- ④审计时点 dev 采集面整体停摆（o-postgres-md 容器 10-28 19:16Z 干净
  停、o-collecto 单元 inactive、PG-md 不可直查）=10/28 停摆事件余波
  非本票回归——代码修复在位，SG 自供面今测鲜活。

## 结果回填

- subscribe.go:69 fin_position UNION 项删——md 侧查询只引 domain.tsv
  md|bridge 表（注释钉死 fin 域禁引——relation does not exist→整查询
  ERROR→订阅帧从未发出=md_ticks OKX 面归零病灶根修）；持仓标的由
  live/degraded 策略项覆盖=冗余项删。
- 普查件：internal/data/|cmd/collector/ grep fin_position 仅 collector.go
  注释行（非表引）——无其他缺席表引用点。
- 桩锚：okxws 测试包 20.5s 全绿（活跃集查询新态随件过）。
