---
schema_version: 2
id: rate-criterion-denominator-floor
kind: pitfall
risk: high
status: verified
keywords: [年化判据, 小分母放大, 部署本金, deploy_floor, NAV回退, death_factor假达标, 判据失真, annualJudgment, 分母下限]
scope: [cmd/core 年化判据面, 判据消费面单源, app_config 判据参数, 率值指标设计]
summary: 率值判据的分母过小时判据数学失真——部署本金年化在部署$76/净$20.68 下读 330.61% 假达标（真 NAV 口径≈2.5%），已被 death_factor 判词与饿判据消费；同本 3min 后滑窗读 -67% 锯齿。判据分母必须带下限：低于 floor 回退底线口径（NAV=总资产真相），「分母不足」语义=证据不足回退判，非达标非不达标。
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    desc: "工单O-34963/值守裁定#34962：部署$76/30d净$20.68→部署口径年化330.61%被 death_factor 判词（0.5→1，ops_event#34937）与饿判据消费为超额达标；同本3min后滑窗-67%（#34938/#34942）"
  - type: doc
    ref: "cmd/core/annualized.go annualJudgment——判据输入单源出口：dep≥floor→部署口径，否则 NAV 回退；floor=app_config annualized.deploy_floor_usd 缺省500 fail-closed"
related: []
---

# 率值判据分母下限（小分母放大失真）

**坑形**：率值型判据（年化/收益率/比率）分母=「被评价对象的投入基数」
时，基数趋零→率值趋无穷大，判据在小样本/低部署下恒「达标」或恒
「不达标」——方向取决于分子符号，且逐日滑窗放大锯齿（实证同本
3min 330%↔-67%）。判据消费面（升件/梯度/自观）把失真值当真判据
喂下游，假达标=该响的信号被喂哑。

**判别**：凡「判据=净额÷基数×年化系数」结构，先问基数下限在哪。
分母可为零/可趋零的口径（部署本金/NAV分项/样本量）必须配 floor；
floor 以下不是「判不达标」也不是「判达标」，是**证据不足→回退
更粗但更真的底线口径**（本案=NAV 总资产口径——部署不足时账户
整体收益率才是业主要的判据）。

**修法**：判据输入单源出口（annualJudgment）两态——分母≥floor
走精口径、<floor 或读败回退底线口径；floor=app_config 热配键
（缺省 fail-closed）；读面签随实际消费口径同源（显什么判什么）。
桩测必含边界四案：零/低于/高于/恰等 floor。

**推广**：同构面=任何「率÷可趋零分母」判据——资金费率年化、
策略胜率、换手率、单笔期望。设计新判据时 floor 与口径回退径
应与算式同案给出，不是补丁期补。
