---
schema_version: 2
id: statetable-crosschannel-latestframe-phantom
kind: pitfall
risk: medium
status: verified
keywords: [reconview, md_account_events, balance_and_position, 状态表, 跨频道最新行, 假差, trading 漏算]
scope: [tools/reconview, md_account_events 读面, 对账升件面]
summary: "状态表(T-0123 键=venue/channel/ccy/inst_id 各存一最新行)下「跨频道 ORDER BY ts DESC」拿到的是最常被写的那类帧不是目标帧。reconview trading 读 channel IN (account,balance_and_position) 取最新行——b&p 帧成交后高频刷新占头而其 data 无 details[] 键→trading 整量漏算→假差=trading 全额(升件#13297 实读 11279.26 vs 真 11306.18 差恰 $26.92)。修=频道锚定 WHERE channel='account'。"
first_seen: 2026-11-13
last_verified: 2026-11-13
skills: []
---

# 状态表跨频道采最新帧=透镜假差（升件#13297 根因之一）

**实案**：reconview 读 okx/USDT venue 现值 =11279.26 而升件
自报/手工四分量分解=funding216+savings11013.26+trading26.92
+dcd50=11306.18——差恰=$26.92=trading 户全额。同一表不同时
刻拍读数振荡：account 行最新时读对，b&p 行最新时 trading=0。

**根因**：`md_account_events` T-0123 状态表化——每
(venue,channel,ccy,inst_id) 键存一最新行。查询
`channel IN ('account','balance_and_position') ORDER BY ts DESC`
拿到的是「最常被写的频道」：b&p 帧随成交刷新频率远高于
account 快照帧。而 b&p 帧 data 形状={balData,posData,...}无
details[] 键——解析零行,trading 分量静默归零。无 error、无
WARN——水面以下假差。

**修法**：读口径按帧形状锚定频道——`WHERE channel='account'`
（快照帧带 details[]）。普律：对状态表写「最新帧」查询前先问
目标 schema 存在哪些频道行；字段回退（eqUsd‖eq）救不了整键
缺席。侦测器 r.bal 走 REST 活读径（T-0048）不受此害——本坑仅
诊断透镜面。
