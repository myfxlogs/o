---
schema_version: 2
id: connect-web-rig-stub-traps
kind: pitfall
risk: low
status: verified
keywords: [playwright, route, glob, connect-web, timestamp, protojson, 台架, 夹具, RFC3339]
scope: [前端台架, playwright, connect-web, 夹具]
summary: Connect 台架夹具两 trap——playwright route glob `**/x` 不匹配 localhost 根相对 RPC 路径（须全 URL）；buf 2.x fromJson 的 Timestamp JSON 严格 RFC3339 串（{seconds} 对象=ConnectError cannot decode）
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: doc
    ref: meta仓docs/施工任务书-驾驶舱体验v2.md（结果段·机检行）
  - type: provenance
    ref: /tmp/ckrig/rig.py（t1030-cockpit-v2 台架九断言）
---

## Connect+Playwright 台架夹具两 trap——glob 不中×Timestamp 拒解

- **实证**（驾驶舱 v2 台架 2026-10-30）：夹具注入后页面恒载中，两组真因——
  ①`pg.route("**/o.dashboard.v1.CoreService/GetCockpit", …)` 零命中：请求
  URL=`http://localhost:5199/o.dashboard.v1.CoreService/GetCockpit`（根相对
  RPC 路径），playwright glob `**/` 前缀段不匹配——三种 pattern 同页实测唯
  全 URL 命中；②夹具时间戳写 `"asOf":{"seconds":"…"}` 对象形→客户端
  `ConnectError: cannot decode message google.protobuf.Timestamp from
  JSON: object`——buf 2.x 严格 protojson，well-known Timestamp 仅认
  RFC3339 串。
- **教训**：台架 route 一律全 URL
  （`http://localhost:<port>/o.dashboard.v1.CoreService/<Method>`）；夹具
  时间戳用 `datetime.fromtimestamp(...).strftime("%Y-%m-%dT%H:%M:%SZ")`。
  调试径=页内 `import('/src/react/api.ts')` 直调 client 拿真 ConnectError
  （组件 `.catch(()=>{})` 吞错不显，恒载中≠无响应）。
- **先例对照**：int64 字段（armed.id/lastSeq）JSON 串形照常过——唯独
  Timestamp well-known 型最严；T-0261 首建台架径未踩①②因其夹具走
  route 注入前 stub 层（本轮复建台架即撞）。
