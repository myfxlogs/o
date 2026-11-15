---
id: mt5-shared-session-crosshost
title: MT5共享会话跨机复挂——owner/consumer角色制
tags: [mt5, session, crosshost, venue, grpc]
源: 升件:venue:mt5_shared_session_crosshost（eng#52）/ merge 7948db3e 投产
---

# MT5共享会话跨机复挂——owner/consumer角色制

**病灶**：mt5-session-conflict-churn 条目钉过的「并发腿账」结构余账——
两腿（汇市巡 app_secrets mt5.* + OCI桥腿 app_config bridge.mt5.*）共用
同账户 3001157，各自 login 互踢会话=churn 温床。复挂+冷却只减频率，
跨腿撞是结构问题。

**解法**（7948db3e 投产）：账户会话单 owner——汇市巡独占
ConnectEx/login，token 落 `app_config mt5.session_token{addr,token,ts}`；
OCI 桥腿降为纯消费者，走 sharedAttach 复挂既有 token，**永不 Connect**。
会话所有权从「谁掉谁登」改为「一主一客」。

**判据**：
- 跨机撞会话连败消失=修成立（churn 升件窗清零）
- session_token 陈化（ts 超时窗）=桥腿 attach 拒→回 owner 重登后写新
  token，consumer 侧按 token 龄重试不抢登
- owner 侧 CheckConnect 复挂+冷却退避照旧（同 churn 条目防点）

**推广**：凡「多客户端共享同账户会话」的 venue 协议——先定
owner/consumer 角色制，token 注册表单写者；「断线即新 login」
在跨机场景必变互踢。防点=会话所有权单写者，非重连逻辑。

**互见**：mt5-session-conflict-churn（churn 病灶+复挂冷却件）
