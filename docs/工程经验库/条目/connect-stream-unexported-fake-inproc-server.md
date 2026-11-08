---
schema_version: 2
id: connect-stream-unexported-fake-inproc-server
kind: pitfall
risk: low
status: verified
keywords: [connectrpc, connect, 伪流, 桩测, eventbus, watch, httptest, 客户端流]
scope: [testing, eventbus, 桩测]
summary: connect 客户端流 concrete struct 字段不导出——伪流不可外构，桩测走进程内 connect 服务端 httptest 喂帧
first_seen: 2026-11-08
last_seen: 2026-11-08
evidence_refs:
  - type: ticket
    ref: T-0458
---

## connect 客户端流桩测：concrete struct 不可外构，进程内服务端是真缝

- **实证**：T-0458 锚 e（dm 帧→Watch→重渲断言）首版想仿
  `EventBusClient` 接口返回伪流——`eventv1connect.EventBusWatchClientStream`
  是 **concrete struct** 且字段 `stream connect.ClientStream` 不导出，
  包外构造直接 vet 炸（`cannot refer to unexported field stream`）。
- **教训**：connect 生成码的客户端流=concrete struct 包 interface——
  桩测不要试图从包外拼 struct；**真缝=进程内 connect 服务端**：
  `connect.NewServer()`+`RegisterEventBusHandler(server, 伪svc)`+
  `connecthttp.Mount`+`httptest.NewServer`，客户端照产线径
  `NewEventBusClient(NewClient(NewTransport, srv.URL))`——全真协议面，
  帧序喂完在 handler 里 `<-ctx.Done()` 挂起对齐 real bus 语义，测试
  cancel 收敛。范本=`cmd/seatcard/main_test.go`。
