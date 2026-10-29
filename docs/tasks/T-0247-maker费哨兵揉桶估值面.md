# T-0247 maker 费哨兵揉桶——配置存在性与取值语义分层（钱径估值面）

> 状态：**待派** | 领取：虚位 | 优先级：**高（钱径估值面失真——潜伏非急诊：
> 产上键未配今零暴露，配上 0/负档即静默错价；修面 bounded 三处）**
> 执行径：**无头**
> 立项：决策席 @ 2026-10-29（管理令 20261029-183339-MANAGE-34——无头审计回放
> T-0238 驳回理由经核实属实：maker_fee_bps≤0 被静默当缺键退回双 taker 估）

## 病灶（码面三处同型）

- `internal/research/funding/run.go params()`：`e==nil` 才赋 `p.MakerFeeBps`
  ——缺键与显式置 0 在参数层不可分。
- `internal/research/funding/eval.go`：`if p.MakerFeeBps > 0` 才走 maker 复合
  估——置 0（免 maker 费档，实存）/负值（maker 返佣档，实存）退回
  `2*taker`，返佣营收静默丢、估值面失真（evidence 记真值而算用另值）。
- `internal/portfolio/lifecycle/opener.go breakevenOK`：`maker > 0` 同型+
  `_` 吞读错——负值/0 值同退回双 taker。

误差方向=保守向（高估摩擦→拒开/迟开）——机会成本病灶非亏损矢量；
产上 `research.funding.maker_fee_bps` 未配（app_config 实证）今零误价。

## 钦定界

- **存在性与取值分层**：配置在册（err==nil，含 0/负）→复合估按真值算；
  键缺→双 taker 保守回退（软键语义保留）。
  - eval 面：Params 加在册标志（如 `MakerFeeSet bool` 或指针/哨兵式），
    `feeBps = taker + maker` 按真值——负值=返佣计入 cost 负项如实；
  - opener 面：`merr==nil` 判在册，`frictionBps=2*(maker+slip)+2*(taker+slip)`
    按真值；读错与键缺同回退（fail-closed 向保留，读错情形计
    missing_param 显形更优——裁量内）。
- 桩测四分支：置 0→复合非双 taker；置负→复合含返佣项；键缺→双 taker
  回退；正值→复合（回归）。
- 不碰判据阈闸值/不动 maker 执行面（T-0238 旗开前置锚照旧）。

## 范围外

费用档位自动发现（venue 费率拉取另案）；exec.maker_enabled 启用流程。

## 验收锚

1. 四分支桩测全过（0/负/缺/正）；
2. evidence 面 maker_fee_bps 记值与计值同源（不再记真值算另值）；
3. CI 绿。
