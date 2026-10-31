# T-0085 历史卖腿 venue 凭据补录（仓#1/#2 现货卖腿缺席）

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 立项 | `[角色:决策]` 会话指派令 2026-10-25（本票面由施工席自指派令转档——决策席未及入库，验收席核） |
| 领取 | zcode-glm @ 2026-10-25（worktree /opt/phase-wt/t0085，支 t/0085-sell-leg-backfill） |
| 涉及面 | fin_order_leg/fin_cashflow 历史行补录（生产数据面）、tools/fillsprobe（只读取证件） |
| 范围外 | 费用行补录（pos2 卖腿 fee 缺席另行报告）、trade_pnl 读面、告警面 |

审计席：headless @ 2026-10-30

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

## 判决注记（审计席逐锚对账——SG 生产库只读实拍 + 收口 commit bcfb938/b27b71e diff 双径取证）

- 锚① venue 凭据补录（occurred_at=venue 成交时）：SG `fin_order_leg` 实拍——pos1 卖腿 id=15（exec5 leg_no=3，cl_ord_id=phresc819777）px=80746.5/ts=2026-10-24 05:29:37.709Z 已落 venue 真值，与 `fin_position` id=1 closed_at 精确吻合；pos2 卖腿 id=67（exec14 leg_no=2，cl_ord_id=venue ordId 4037690546712727552）sell 0.010578 ETH-USDT @2361.55 ts=09:42:35.374Z 全行在案。`fin_cashflow` 两行 spot_sell：id=131 amount=24.92321469 position_ref=1 source_ref=15、id=132 amount=24.98047590 position_ref=2 source_ref=67，occurred_at 双双=venue 成交时（非入账时）。算术自洽：0.00030866×80746.5≈24.9232、0.010578×2361.55≈24.9805 与毛额行吻合。
- 锚② net 归粉尘域：SG 实算 Σcashflow per position——pos1 net=−0.11381925、pos2 net=−0.0431009（$25 名义 delta 中性往返的费用+滑点级残留，粉尘域成立，与票面声明值一致）。
- 锚③ 幂等+凭据可溯：source_ref=15/67 各仅一条 spot_sell 行（15 另挂 fee 行 id=12——pos1 卖腿费 venue 时已在账，非本票产物）；tradeId/ordId/fillSz/fillPx/fee/ts 六件凭据票面在案，ordId 持久化于 leg67 cl_ord_id。取证件 `tools/fillsprobe/main.go` 在库（只读 fills-history+bills 双源拉取，零写径）。
- 注记（非锚不阻）：pos2 卖腿 venue fee=−0.02498048 未入账——票面「范围外」明载另行报告，与实态一致（source_ref=67 无 fee 行），费用行补录留待决策席立票；施工 session 曾误提交 16MB 编译产物 fillsprobe 二进制，b27b71e 同票剔除，终态 tools/ 只收源码。
