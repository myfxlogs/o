# T-0196 funding_view 血统闸违规修——T-0164 族唯一自盖章件

> 状态：作废（superseded——11-01 清账定性·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**作废**——superseded：funding_view血统闸修投产销号（dd11c244三票验收）
> 域：信号口径/蒸馏 · 决策席直派

## 病灶实证（决策席审计 2026-10-27）

`internal/research/sigdist/run.go:136`——funding_view 写行 `ts=now()`
自盖章（写拍时间），违 T-0164 血统闸「派生信号 ts=最旧上游观测」。
全族对照：basis_ann/xvenue_spread/carry_diff/mt5_swap 皆 leastTs 正形，
此件孤例——上游 md_funding 断流陈腐时，信号仍披鲜皮（陈≠不存在的
语义被本件单独掏空）。

## 施工面

- funding_view 行 ts 改=上游观测龄——注意 `md_funding.ts`=结算拍（可
  未来值，事件时非观测时），血统正解取源行 **recv_ts**（观测戳），
  多源行取最旧（leastTs 同构）。
- 定参：读面 query 里把参与行的 max/适用 recv_ts 聚合出血统戳；
  若无可用源行（val 全缺）照现语义不产行。
- 单测：构造 funding 行 recv_ts 陈旧 → 信号行 ts 随陈（非 now）。

## 验收判据

- funding_view 行 ts=参与源行 recv_ts 语义（非写拍 now）
- 测试覆盖「上游陈→行 ts 陈」断言
- ops 面 funding_view 鲜度仍可读（recv_ts 口径下鲜度=帧流活性真值）

## 结果回填

- funding_view 行 ts=源行 recv_ts（观测戳血统闸——md_funding.ts=结算
  拍可未来值非观测时；f.recv 零值=ts 兜底）；最新参与行观测龄即血统
  戳——全族与 basis_ann/xvenue_spread 同构（孤例自盖章病灶根治）。
- 桩锚 TestFundingViewTsLineage 绿——recv_ts 陈 4h 行→signal ts 随陈
  （非写拍 now 实证）。

## 决策席验收（2026-10-27 夜）——投产销号

- 改法照钦定：ts=源行 recv_ts（观测戳），recv 零值 ts 兜底；DISTINCT ON
  最新参与行观测龄。
- 库面实证：funding_view 行 ts 现为 recv_ts 口径（行龄 17s=观测龄真实值，
  原 now() 恒 0 假鲜根治）。
- TestFundingViewTsLineage 绿（recv_ts 陈 4h→行 ts 随陈）。
