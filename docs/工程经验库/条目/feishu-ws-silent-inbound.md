---
schema_version: 2
id: feishu-ws-silent-inbound
kind: pitfall
risk: medium
status: verified
keywords: [feishubridge, lark ws, 长连接, 事件订阅, im.message.receive_v1, card.action.trigger, 静默门, 观测面]
scope: [cmd/feishubridge, 飞书开发者后台, 入站事件径]
summary: 「订阅开着」≠「事件在投递」——WS 握手成功+ping/pong 活只证传输层；控制台三坑各自哑火：①事件开在 webhook 模式不长连接模式（两清单互不通用）②权限域缺（p2p 消息须 im:message.p2p_msg；群聊非@天然不投递）③改动在草稿态未发版本。且处理器窄门静默丢=「没投递」与「被门拦」不可分——入站处理器必须入口日志
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: server_fact
    source: feishubridge debug 日志
    query: SDK LogLevelDebug 下用户发 签 999——ping/pong 外零帧到达；控制台改后同路径帧到达+全链通
    observed_at: 2026-11-11T16:05:00Z
    server_revision: ""
    response_hash: ""
  - type: doc
    ref: cmd/feishubridge/main.go:onMessage
---

## 事实/症状

业主在飞书后台报「已开」两事件订阅，发 `签 999` 零回应。
桥件日志（Warn 级）零异常、socket ESTABLISHED 20 分钟——
表面像通了，实证是飞书侧根本没投递：SDK Debug 级侦测版显示
发消息时刻**连一帧都没到达**（只有 ping/pong）。
控制台修正后同路径重发——帧到、门过、结案、回执全链通。

## 根因

飞书事件订阅有三层独立的「开」，每层哑火形都相同：

1. **订阅方式**：「使用长连接接收事件」vs「发送至开发者服务器」
   是两份独立事件清单——webhook 模式下加的事件 WS 收不到，
   切模式后须重加。WS 握手成功与此层无关（通道是应用级的）。
2. **权限域**：`im.message.receive_v1` 按消息域分权限——
   单聊须 `im:message.p2p_msg`，群聊须 `im:message.group_at_msg`
   且**群聊非@消息天然不投递**（不是配置问题，是产品设计）。
3. **版本发布**：权限/事件改动草稿态不生效，须创建版本发布。

## 处置律

- **判「通」唯一标准=真事件到达**——订阅开关/socket 状态/
  握手日志全是传输层弱证，一个真消息/真钮才是终证。
- **入站处理器入口必须日志**（收消息/收卡钮各一行，记
  chat/sender/type）——窄门静默丢使「没投递」与「被门拦」
  不可分，今天就是靠入口日志定位到「连门都没到」。
- **SDK 帧级日志**（`larkws.WithLogLevel(LogLevelDebug)`）
  只用于侦测版——生产 Warn，Debug 每 90s 刷 ping/pong 噪。
- **侦测径**：worktree 出 Debug 构建→停 unit→手跑同默认参→
  验毕杀侦测件起 unit——/opt/o 部署树一行不动。
