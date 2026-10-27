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

## 状态

- [x] 立单
- [ ] 施工
- [ ] 验收投产
