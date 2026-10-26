---
schema_version: 2
id: connect-server-stream-no-headers-before-first-write
kind: pitfall
risk: high
status: verified
keywords: [connect rpc, server-streaming, watch, 响应头, 悬挂, 就绪帧, watch_open]
scope: [rpc, frontend, realtime]
summary: connect 服务端流首写前不发响应头——客户端 Watch()/for-await 悬挂至首个事件；服务端流必须 LISTEN 注册后立即发初始就绪帧（watch_open 惯例）
first_seen: 2026-10-26
last_seen: 2026-10-26
evidence_refs:
  - type: provenance
    ref: T-0156（cmd/core Watch e2e 桩测三轮实证+processor 埋点定位）
---

## connect 服务端流零首写=客户端悬挂

- **实证**：`CoreService.Watch`（LISTEN ev_* 脊）挂流后阻塞 `WaitForNotification`、不发任何帧——connect-go/connect-web 客户端 `Watch()` 调用与浏览器 `for await` 迭代均等响应头才推进，响应头在服务端**首次写出**才发——结果客户端悬挂至首个数据事件（无事件=无限悬挂）。桩测连挂：`cl.Watch(wctx)` 直接 `deadline_exceeded`，事件从未有机会被拍。该服务自 T-0067 起从未被任何前端真用过（前端走 SSE /api/watch），属从未踩到的潜伏雷。
- **规则**：①服务端流 handler 必须在阻塞等待前**立即发一帧**——LISTEN 注册完成后发 `watch_open` 初始帧（非 ev_* 通道，数据事件语义不混；前端借此帧做连接即回填+断流重连快照恢复）；②服务端流桩测锚=先收就绪帧再拍事件（无就绪帧的流测试全靠运气窗口）；③服务端流前端消费件必须处理重连（stream drop 后 3s 重连=传输维护非数据轮询）。
