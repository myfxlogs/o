---
id: protocol-version-as-category-taxonomy
title: 协议版本当类目建模——MT5 当 venue 类目，MT4 接入即轮子重造
kind: pit
date: 2026-11-26
source: 工单O-39440（业主令「MT5就是错的，如果以后要接入MT4呢？他们应该统归CFD才对」「历史的标记就是错误的」）
---

# 协议版本≠类目——taxonomy 三层钉死（O-39440① 根修）

**坑形**：`fin_venue_roster.proto='mt5'` 把**协议版本**当**资产类目**——
Exness/XM 两个经纪商被标成「MT5 类」，venue token 'mt5' 又把 Exness
户号挂到协议名下（`mt5:3001157` 读作「MT5 的 3001157」实为「Exness 的
3001157」）。MT4 经纪商接入时 proto 无处安放——要么开 'mt4' 平行类目
（读径/对账/执行全族复制=重造轮子），要么硬塞 'mt5'（语义说谎）。

**taxonomy 三层**（业主钦定）：

| 层 | 字段 | 取值 | 语义 |
|---|---|---|---|
| 类目 | roster.proto | cex / dex / **cfd** | 资产/连接大类——MT4/MT5/cTrader 统归 cfd |
| 平台 | roster.platform | mt4 / mt5 / … | cfd 类内连接协议——读件按 platform 分派客户端 |
| 户主 | roster.venue | exness / xmglobal / fxpro… | 对手方经纪商名=数据面 venue 键 |

account_id=`<venue>:<定位符>`——cfd 类=login 号（exness:3001157），
cex 类=子户型（okx:trading）。

**判据**：凡「X 协议的 Y 户」句式=类目/平台/户主混层——先问这词是
协议（mt5）、经纪商（exness）还是资产类（cfd），再决定它落哪列。
新对手方接入=roster+fin_account 插行零 schema 改动；MT4 户入册
platform='mt4' 后 MT5 会话面自动缺席（platform 过滤）不误报。
