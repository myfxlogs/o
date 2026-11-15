---
id: mt5-shared-session-crosshost
title: MT5 共享会话——token 跨机复挂实证与 owner/consumer 角色制
kind: pit
date: 2026-11-16
source: 跨机实证 mt5tokprobe（SG acquire + OCI 149.118.58.98 use 同 token
  并发读 equity=3247.31 全绿、单边断连不连坐）+ 双腿共用 3001157 跨腿
  互撞结构余账收口（mt5-session-conflict-churn 共生面）
---

# MT5 共享会话（跨机 token 复挂实证）

**机制**：`ConnectEx` result=**后端 MT5 会话句柄**——不绑 gRPC 连接、
不绑出口 IP、不绑主机。两台主机各自 gRPC 连接递同一 token 并发
CheckConnect/AccountSummary/OnQuote 全绿；单边断连不连坐。

**坑形（共享期新增三坑，比单腿期更毒）**：

1. **消费端误登杀会话**——任一共享端 `Disconnect`=把后端会话登出，
   **其他共享端全部陪葬**（OpenSession 短调用面的「用毕 Disconnect」
   遗留契约撞共享=自毁）。修法=shared 标置：消费径注凭走
   `SetSharedToken`（置 `Client.shared`），`Disconnect` 幂等空转——
   共享会话不归任何消费端登出，僵尸界=券商 TTL 自然收尸。

2. **双 login 竞速**——共享会话真死时，多端各自 `ConnectEx` 重建=
   回到跨腿互撞老坑（socket-closed/paused 指纹）。修法=**owner 独占
   login 权**：汇市巡（持 app_secrets 凭据）唯一 ConnectEx 面，会话
   建立后落 `app_config mt5.session_token` 共享行
   `{addr,token,ts}`；OCI 桥腿=纯消费腿（`sharedAttach`：读行→
   SetSharedToken→CheckAlive），共享行缺/死=退避等 owner 重建，
   **永不 ConnectEx**（腿星化可观测：connect_failed+会话终行）。

3. **共享行快照滞读**——`config.Client.Get` 走 snap（Watch 刷新有
   窗口），token 是易变会话态非配置态。修法=共享行读写直走 pool
   （`sharedSessionRead/Write` 直 SQL），绕 snap 径。

**观测约定**：
- owner 正常：`OpenSession 复挂共享会话（零新 login）`（短调用面）/
  connect 起缺席（腿复挂静默径——同 conflict-churn 条目约定）
- 消费腿等 owner：`共享会话未立（等汇市巡建）`/`共享会话已死（等汇市巡
  重建）`——**正常等待非故障**；长时间星化=查汇市巡 owner 活性
- `WARN [mt5] 共享会话行写败`——owner 落行败（消费腿晚拍拿新值，会话
  本体不炸）

**边界/余账**：
- `mt5.session_token` 键名含 token=credentialRE 自动 owner 硬拒域——
  免册零闸改；app_config 明文（bridge.mt5.password 先例同构，会话
  句柄按凭证形管）
- OpenSession 兜底径（共享行缺时 legacy ConnectEx）保留——owner 未
  就位期的可用性面；与 owner 重建竞速窗=conflict-churn 冷却退避兜
- `bridge.mt5.*` 凭据键成历史遗留（消费腿不再读——app_config 行留存
  无害，清理另议）
- 部署序：镜像面（cmd/core owner+OpenSession）先上产共享行→OCI
  collecto 后上消费径；中间窗 OCI 旧桥腿仍走自连凭据径（旧码兼容，
  部署完成即切共享）

**判据**：凡「多进程/多机共用一会话句柄」设计——先问谁建谁登出谁
重建；三角不齐=连坐或竞速坑，本条目三角=owner 建/无人登出/owner
重建。
