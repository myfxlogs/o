# T-0237 funding-hedge live 经济性——费率倒挂修至净利转正

> 状态：已批（11-01 清账定性留队·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**已批**——在飞支t/0237-fee-econ +3未并——funding-hedge live经济性调研产出未收口（D-0037双优先仍在）
> 优先级：**最高（D-0037 执行端双优先②）**
> 立项：项目负责人 @ 2026-10-29

## 病灶

17 条 live funding-hedge 真实账：利息 +1.0783＋funding_settle +0.0020−费 3.0461
＝**净 −1.97**。1114 张执行单的摩擦成本吃穿全部毛收益——量级/频率不足以摊费。

## 钦定方向（施工调研，三选一或组合）

1. **提单笔量级**：减少笔数×加大 notional，同敞口摊薄 per-trade fee；
2. **降频**：提 eval 阈值（research.funding.min_net_apr 现值查证）滤掉薄利单；
3. **费率面**：核对 taker/maker 路径与返佣——执行单是否全走了贵腿。

## 验收锚

- 调研报告：近 30d 逐单 fee/毛收益分布、阈值现值、费率路径实证。
- 方案经负责人审批落地后：**月度净利转正**（D-0037 KPI），先以周为单位跟踪。
- 任何参数变更：票面留变更前后对照+变更令出处。

## 范围外

- 不动发现面（不调 md_signal 产出）；不碰 basis/dcd 族参数。

## 参数落地录（决策席执 2026-10-29 01:43Z）

- **批准**：负责人批复——业主 2026-10-29 明确同意 `min_net_apr 0.05→0.20`
  （出处=本票调研报告段）。
- **落库实证**：SG `UPDATE app_config SET value='0.20'`——generation 360，
  config_changed 通知发出；OCI-md 副本（md_1068 投影径）<15s 同步实证。
- **生效径**：`cfg.Watch` 缺位发现——o-research 仅 boot `cfg.Load` 快照
  （D-0005「禁 boot 一次性捕获」契违，坑录移交档盯梢）——`systemctl
  --user restart o-research` 载新快照生效，active+评估拍实证。
- **待验锚**：下轮 funding 提案净算须 net_apr≥0.20——逐日盯 fin_proposal
  /fin_opp_eval funding-hedge 行阈下移实证；maker 径=T-0238 已立。
