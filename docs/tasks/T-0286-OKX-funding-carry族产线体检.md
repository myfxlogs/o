# T-0286 OKX funding-carry 族产线体检（T-0118 拆解·族票①）

| 字段 | 值 |
|---|---|
| 状态 | 收口待审 |
| 立项 | `[角色:决策]` @ 2026-11-01（T-0118 母票按机会族拆解首批） |
| 领取 | devin-施工席 @ 2026-11-01 |
| 优先级 | 高 |
| 涉及面 | `fin_opp_template(funding-hedge)`、`fin_opp_eval`、`fin_proposal`、`fin_strategy`（读径体检为主；修复另立子票） |
| 执行径 | 无头/交互皆可 |

## 背景

T-0118 母票「机会发现扩张」按机会族拆票——本票=**唯一 enabled 模板族**
funding-hedge（OKX funding 套息，7d eval 产出 814 件）。T-0200 二期定性
在案：armed→strategy 晋升领取器停摆是全链第一断（armed_ttl_expired
7d 347 件）——本族是断链下唯一仍产出 armed 的族，体检=定性本族在
断链中的实际水位+晋升梗阻实据。

## 钦定界

- 体检产出=**族产线台账**：eval 产率（7d 分日）→proposed→armed→
  TTL 过期/晋升去向逐层计数与滞留点。
- armed 提案滞留面：本族 armed 行 TTL 存活期/退役因分布实证。
- strategy 侧：本族 live/paper/retired 策略计数与入场判据实录。
- 判据口径：本族 eval 打分口径在案（APR 阈值/资金费率源）——
  产率塌/细流属族内口径问题还是链断后果，明判。
- **前置断点照抄不重复定位**：armed→strategy 晋升停摆=T-0200 在案
  第一断（本票只证本族受影响度，不修）。

## 验收标准

- [ ] 本族 eval→armed→strategy 逐层计数台账贴票面（产侧 SQL 实证）
- [ ] armed_ttl_expired 本族占比/存活期中位数在案
- [ ] 本族判据口径（APR 阈/费率源）明列+产率定性（如实细流 vs 口径死）
- [ ] 修复建议落票面（口径调/晋升复起依赖/模板退役——三选一定性）

## 范围外/禁线

- 不修码不部署不动策略开关；armed 提案不手工 disarm/promote
- 晋升领取件修复另行立案（本票只出体检与建议）

## 结果回填（devin-施工席 2026-11-01）

**族产线台账（产侧 SQL 实拍 2026-11-01）**
- eval→proposed：7d 产出 816 件——rejected 809/proposed 5；分日：10-30 rejected 541、10-31 rejected 10+78、proposed 仅 10-27~29 三笔。proposed 净 APR 全=0.05285（阈值边界值≈5.3%）。
- proposed→armed：现役 armed=0（全数 `armed_ttl_expired` 退役——318 件中位存活 4h00m12s；eval_rejected 519/legacy 242）。
- strategy 侧：funding-hedge live 23/paper 83/retired 1710——live 挂册在营但 armed 断粮中。

**判据口径**：proposed 命中线净 APR≈5.3%（0.05285 全员同值=阈边界）；费率源=OKX funding 行情面（funding_view md 517 行/7d 鲜活）。

**定性**：如实细流而非口径死——rejected 主力=费率 regime 低（客观无 >5.3% 机会），非判据恒不满足；链断后果=armed 层断粮（产了 5 件 proposed 也 4h 过期烧光）。修复建议=**晋升复起依赖型**（口径不动：低 regime 下拒判健康；armed→strategy 断点归 T-0200 在案件）。

**范围外确认**：零码改零部署；armed 提案零手工 disarm/promote；晋升领取件修复另行立案（注：t/0286-armed-promote-claim-revive 支已在册——本票不越）。
