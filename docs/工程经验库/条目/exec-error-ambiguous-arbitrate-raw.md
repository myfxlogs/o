---
schema_version: 2
id: exec-error-ambiguous-arbitrate-raw
kind: pitfall
risk: critical
status: verified
keywords: [歧态势, REQUEST_ERROR, Request processing error, OrderClose, OrderSend, OpenedOrders, 假平仓, len(nil)=0, venue 仲裁]
scope: [一切 venue 执行类 RPC, mt5, okx, 钱径]
summary: 执行类调用返回错误后仓位/订单状态不定——报错既不证「未执行」也不证「已执行」，唯一仲裁=venue 权威现读；吞 err 用空结果当「已平/零仓」是假平仓读法
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [mt5-trading-mutating, okx-exec]
evidence_refs:
  - type: doc
    ref: tools/mt5ordstream/main.go
  - type: doc
    ref: tools/mt5close/main.go
  - type: doc
    ref: skills/mt5-trading-mutating/SKILL.md
---

## 事实/症状

真户实测两例同族：① OrderSend 连返「Request processing error」（栈卡 MarketOpenWaiterAsync）第 3 次退避后成交——报错期间若有仓则歧义；② OrderClose 报同款错后 OpenedOrders 实读证仓仍在（ticket=2567601594 残留），探针收尾却因 `o, _ :=` 吞 err 打印 `len(nil)=0` 误报「0 仓」。

## 根因或策略解释

venue 执行 RPC 的传输/网关错发生在「请求已入场、结果未回」灰区：撮合可能已发生（报错误报）或确未发生（报错实报）。错误文本本身不携执行态信息——MarketOpenWaiter/MarketCloseWaiter 超时只证「等待侧超时」，不证撮合侧结局。

## 处置

- 执行类调用 err/apierr 后**强制仲裁步**：venue 权威现读（OpenedOrders/OrderHistory/订单查询）定仓态，再决重试/补单/放弃——歧态势禁直接补单（okx-exec 同律）
- 收尾/审计读**禁吞 err**：`resp, _ :=` 后 `len(GetResult())=0` 把「查询失败」读成「零持仓」=假平仓——err≠nil 时必须显式 UNKNOWN 且 fail-closed
- 重试带完整参数（MT5 平必带 Lots）；重试前仍先仲裁，防重复执行已成交请求

## 证据

- check: mt5ordstream 实单平→RError→OpenedOrders 实读
- result: 报错时仓仍在（profit 浮动着）→mt5close att1 带 Lots 平成→再读 0 仓；同 probe 未修前曾打印假「0 仓」
- ref: skills/mt5-trading-mutating 真户谱 2026-10-24 两行

## 反证条件

- check: 连续 N 次 RError 后 OpenedOrders 均显示已执行（或均显示未执行且可复现区分）
- falsifies_when: venue 错误码升级为确定性语义（如含 executed flag）——届时条目收窄到无该 flag 的旧网关

## 适用边界

适用一切「调用→灰区→状态查询」三段式 venue（MT5/OKX/一切交易所）；纯幂等查询无歧态势问题。与平仓事件仲裁坑（部分平/push 不携余量）互补：那条管 push 侧假信号，本条管 call 侧假报错。
