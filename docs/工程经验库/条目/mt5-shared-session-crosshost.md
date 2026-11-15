---
id: mt5-shared-session-crosshost
title: MT5共享会话跨机复挂——owner/consumer角色制
kind: pit
date: 2026-11-16
source: 升件:venue:mt5_shared_session_crosshost（eng#52 canon 提炼）/
  merge 7948db3e 投产 + mt5tokprobe 跨机实证（SG acquire+OCI use 同
  token 并发读全绿、单边断连不连坐）
keywords: [mt5, session, crosshost, venue, grpc, owner-consumer]
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

**共享期三坑**（比单腿期更毒，防面=码面带锁非口头约）：
- **消费端 Disconnect=连坐杀会话**——OpenSession「用毕 Disconnect」
  遗留契约撞共享=自毁；修法=`Client.shared` 标置使 Disconnect 幂等
  空转，共享会话不归任何消费端登出（僵尸界=券商 TTL 自然收尸）
- **会话死多端齐 ConnectEx=双 login 竞速**——owner 独占 login 权，
  消费端只退避等共享行（腿星化可观测非静默）
- **cfg.Get snap 滞读易变 token**——共享行读写直走 pool，绕 Watch
  刷新窗口

**观测约定**：
- `OpenSession 复挂共享会话（零新 login）`=正常（25s 周期读面全切
  此径=churn 源绝迹的产线实证签名）
- 消费腿 `共享会话未立/已死（等汇市巡…）`=**正常等待非故障**；
  长星化=查汇市巡 owner 活性
- `WARN [mt5] 共享会话行写败`=owner 落行败（消费腿晚拍拿新值，
  会话本体不炸）

**判据**：
- 跨机撞会话连败消失=修成立（churn 升件窗清零）
- session_token 陈化（ts 超时窗）=桥腿 attach 拒→回 owner 重登后写新
  token，consumer 侧按 token 龄重试不抢登
- owner 侧 CheckConnect 复挂+冷却退避照旧（同 churn 条目防点）

**边界/余账**：
- `mt5.session_token` 键名含 token=credentialRE 自动 owner 硬拒域——
  免册零闸改；app_config 明文（bridge.mt5.password 先例同构）
- OpenSession 兜底径（共享行缺时 legacy ConnectEx）保留——owner 未
  就位期可用性面；与 owner 重建竞速窗=conflict-churn 冷却退避兜
- `bridge.mt5.*` 凭据键成历史遗留（消费腿不再读——app_config 行留存
  无害，清理另议）

**推广**：凡「多客户端共享同账户会话」的 venue 协议——先定
owner/consumer 角色制，token 注册表单写者；「断线即新 login」
在跨机场景必变互踢。防点=会话所有权单写者，非重连逻辑。

**互见**：mt5-session-conflict-churn（churn 病灶+复挂冷却件）
