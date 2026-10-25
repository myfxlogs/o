# T-0085 历史卖腿 venue 凭据补录（仓#1/#2 现货卖腿缺席）

| 字段 | 值 |
|---|---|
| 状态 | 完成 |
| 立项 | `[角色:决策]` 会话指派令 2026-10-25（本票面由施工席自指派令转档——决策席未及入库，验收席核） |
| 领取 | zcode-glm @ 2026-10-25（worktree /opt/phase-wt/t0085，支 t/0085-sell-leg-backfill） |
| 涉及面 | fin_order_leg/fin_cashflow 历史行补录（生产数据面）、tools/fillsprobe（只读取证件） |
| 范围外 | 费用行补录（pos2 卖腿 fee 缺席另行报告）、trade_pnl 读面、告警面 |

## 病灶

仓#1（BTC，exec4）与仓#2（ETH，exec10）的平仓现货卖腿系工具径执行
（pos1=exec5 回滚腿 venue 实成但腿行缺 px 无现金行；pos2=okxunwind 应急径全缺）
——fin_order_leg/fin_cashflow 双缺，两仓现金故事不完整。

## 验收标准

- [ ] 两笔卖腿从 OKX fills/bills 档案取真实成交补录（occurred_at=venue 成交时）
- [ ] 补后两仓 net=Σcashflow 归粉尘域（delta 中性往返=费用级）
- [ ] 幂等（重跑零重复）；venue 凭据（tradeId/ordId/fillSz/fillPx/fee/ts）在案可溯

## 结果回填

**venue 凭据（tools/fillsprobe 实拉，fills-history+bills 双源对拍）**：
- pos1 卖腿：tradeId=1074076920 ordId=4037181268784566272 sell 0.00030866 BTC @80746.5 fee=−0.02492321 ts=2026-10-24 05:29:37.709Z（与 pos1.closed_at 精确吻合；腿行已存=exec5 leg15，clOrdId phresc819777=rescuespot 径——补 px+ts）
- pos2 卖腿：tradeId=886828213 ordId=4037690546712727552 sell 0.010578 ETH @2361.55 fee=−0.02498048 ts=2026-10-24 09:42:35.374Z（ordId 与 T-0050 okxunwind 日志吻合；腿行全缺——补录挂 exec14（其既定 close exec）leg_no=2，cl_ord_id=venue ordId 凭据）

**补录动作（生产事务，幂等形）**：leg15 UPDATE px/ts（venue 真值）+两条 spot_sell 毛额流入行（24.92321469 归 pos1、24.98047590 归 pos2，occurred_at=venue 成交时，source_ref=腿行 id）——INSERT...WHERE NOT EXISTS 幂等，重跑零变化实证。

**净额归零口径核验**：pos1 net=−0.113819、pos2 net=−0.043101——费用+滑点粉尘域（delta 中性往返口径闭合）✓。账本 okx/USDT 净额 2132.80→2209.83（+49.90 本票两行+他席并行行流）。
**范围外发现**：pos2 卖腿 venue fee=−0.02498048 未入账（fee 行缺席；pos1 同源 fee 已在账）——fee 补录超本票范围，报决策裁（同款径可 5 分钟落）。
