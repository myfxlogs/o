---
id: venue-bucket-collapses-account-dim
title: venue 桶聚合键=账户维盲——子户/多户余额互吞，对拍面结构性失明
kind: pit
date: 2026-11-26
source: 工单O-39440（业主令「以账户来分」「直接拉真实数据」——OKX 四子户聚合+MT5 双户一户实读零）
---

# venue 桶聚合键吞账户维（O-39440 根修实证）

**坑形**：对账/实读/簿记三面共用 `venue:ccy` 二元域键——venue 内
多账户实体（OKX trading/funding/earn/dcd 四子户、MT5 Exness 3001157
+XM 110660304 双登录）被折叠成单桶：
- OKX `/asset/balances`(资金户)/`/finance/savings`(Earn)/DCD 在持
  全部并入 `out[{"okx",ccy}]`——子户间转移在对拍面不可见；
- MT5 采集只认 `venue="mt5"` 单会话（roster 里 xmglobal 户
  state=registered 但实读径从未遍历它）——第二户 ~$1000 NAV
  结构性漏计；
- `fin_cashflow` 无账户列——deposit/withdraw 落 `venue=okx`
  无法区分资金户/交易户归属，豁免行同样 venue 级无法按户精确扣。

**判别**：venue 实读余额与台账净额长期「对得上」但 NAV 与
venue 面板总和对不上——差值不是 bug 是**聚合键吞了维度**。
查 `fin_account`/roster 有无未被实读径遍历的 active 户；查
`fin_cashflow` 写入点有没有 account 语义被丢弃。

**修法**（O-39440 落地形）：账户注册表 `fin_account` 为身份真源
（account_id=venue:sub 不可变）；实读径按注册表逐户/逐子户发射
`{account_id,ccy}` 键；`fin_cashflow.account_id` NOT NULL+触发器
(kind→账户 显式映射兜底全部写入点)；对拍遍历 active 账户行、
venue 级台账组件经 compAccount 归户；凭据缺席=cred_gap 升件
不猜数。**教训：域键设计时账户维必须是一等公民——事后补维
=全链键面迁移（读/写/豁免/告警/索引六面同改）。**
