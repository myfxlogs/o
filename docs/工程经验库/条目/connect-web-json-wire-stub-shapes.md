---
schema_version: 2
id: connect-web-json-wire-stub-shapes
kind: pitfall
risk: medium
status: verified
keywords: [connect-es, connect-web, unsupported content type application/proto, protobuf.Timestamp from JSON, ConnectError, stub, 前端真渲验证, protojson]
scope: [web, dashboardv1, 截图台架, 前端验证]
summary: Connect-ES 前端真渲验证 stub 双坑——transport 实走 JSON wire（回 application/proto 即 unsupported content type 拒收）；Timestamp JSON 形须 RFC3339 字符串非 {seconds,nanos} 对象；流面信封=application/connect+json 且 END 块恒 JSON
first_seen: 2026-10-28
last_verified: 2026-10-28
skills: []
evidence_refs:
  - type: ticket
    ref: T-0221
  - type: commit
    ref: e741ee5
  - type: doc
    ref: web/src/react/api.ts
---

## Connect-ES 前端真渲验证 stub 的 wire 形状双坑

- **症状**：给 dashboard SPA 配本地 stub 后端截图验证时——
  ① stub 按「Connect 默认二进制」假设回 `content-type: application/proto`，
  前端卡片全落 `读件缺——[internal] unsupported content type application/proto`；
  ② 改 JSON 后投票引擎行报
  `ConnectError: [unknown] cannot decode message google.protobuf.Timestamp from JSON: object`
  （nextWindow 给了 `{seconds,nanos}` 对象形）。
- **根因**：
  ① `createConnectTransport({ baseUrl })`（api.ts 无 useJsonFormat 显式项）在
  本仓 connect-web 版本实际走 **JSON wire**——unary 应答须 `application/json`
  + protojson 体；凭版本直觉猜二进制即翻车。
  ② protojson 的 WKT 形状律：`google.protobuf.Timestamp` JSON 形=RFC3339
  **字符串**；int64=**字符串**数字。Go 侧 struct 直 marshal 出的
  `{seconds,nanos}` 对象形只有 protojson marshaler（UseProtoNames/
  protojson.Marshal）会折成 RFC3339——手写 stub 直塞对象形必炸。
- **律**：前端真渲验证 stub 三件套——
  1. unary 回 `application/json`+protojson（驼峰键、WKT 用 JSON 形）；
  2. server-stream 回 `application/connect+json`，信封=1 字节 flag+4 字节
     BE 长度+JSON 文本，END（flag=2）恒 JSON（发首事件后持流不关=live 稳态）；
  3. 报文形状用生成 `*Schema` + `fromJson/toBinary`（@bufbuild/protobuf）
     交叉验证，不手拼。台架配置（vite transformIndexHtml 主题注入等）放
     /tmp 不入仓；`originalUrl` 在 transformIndexHtml 的 **ctx 本体**上，
     非 ctx.server 上。
- **复用**：T-0221 截图台架以此四态真渲 System/TRX 页成功（亮/暗截图入
  docs/design/）；下个前端验证票直接按本条搭 stub 免重踩。
