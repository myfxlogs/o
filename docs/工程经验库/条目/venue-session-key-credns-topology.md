---
id: venue-session-key-credns-topology
title: 跨分库消费腿不可解 cred_ns——共享会话键 venue 寻址裁定
kind: pit
date: 2026-11-24
source: 工单O-37448（venue-roster-sessions——design D2 钦定 <cred_ns>.session_token vs 桥腿 PG-md 拓扑冲突裁定）
---

# 会话键 cred_ns vs venue 寻址（O-37448 裁定）

**坑形**：design SSOT D2 与 spec ③ 钦定共享会话键=`<cred_ns>.session_token`
（XM=`xmglobal.mt5.session_token`）。施工面发现**硬拓扑冲突**：
cmd/collector OCI 桥腿跑 PG-md 池——`fin_venue_roster` 在 fin 域
（SG PG），PG-md 角色/库两层皆不可达；消费腿按 venue 参数化声明
服务哪户（D2 原文「桥腿声明它服务哪户」），venue→cred_ns 解析在
桥腿物理不可行。cred_ns 键形要成立，须给桥腿另造
`bridge.<venue>.cred_ns` 配置镜像（roster cred_ns 虽不可变仍是
第二真源+手误面）或 owner 写指针行——两案皆加机制伺候键形偏好。

**裁定**：共享行键=`<venue>.session_token`（'mt5'→存量键不动=
Exness 零迁移窗；'xmglobal'→'xmglobal.session_token'）。
venue=账户身份 token 一身三任（D1 钦定），会话行归账户不归凭据
前缀——语义上 venue 寻址更正；消费腿唯一可声明身份=venue 名，
键与声明参数同源零解析零配置镜像。

**判别**：spec 字面键形 vs 消费腿物理可达性冲突时，先查消费腿
的库/角色拓扑再裁键形——「键名好看」不敌「消费腿解不出键」。
cred_ns 不可变≠可裸复制进第二配置面（手误仍在）。

**修法/验收**：`VenueAcct.SessionKey()`/`BridgeCollector.sessionKey()`
单点派生；桩测双户各持 `<venue>.session_token` 互不覆写+
败户无行+Exness 键形回归三锚在案（roster_test.go）。

**推广**：凡「名册行字段 X → 派生键/路由」设计，先枚举全部消费腿
的读面边界（哪个池/哪个角色/有没有名册读权）——派生链上任何一
跳物理不可达，键形就必须落在消费腿可自证的身份上（venue token
本身即此形）。
