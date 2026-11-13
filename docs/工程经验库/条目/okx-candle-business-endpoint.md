---
schema_version: 2
id: okx-candle-business-endpoint
kind: pitfall
risk: high
status: verified
keywords: [candle1D, business 端点, 60018, channel doesn't exist, OKX WS, kline]
scope: [okx-market-ws, internal/data/okxws, klinesub]
summary: OKX candle* 频道全族只在 /ws/v5/business 端点——/public 全拒 60018，K 线订阅必须独立连接
first_seen: 2026-11-13
last_verified: 2026-11-13
evidence_refs:
  - type: ticket
    ref: 13065
  - type: commit
    ref: 8411fa91f
  - type: server_fact
    source: cex
    query: WS subscribe candle1D BTC-USDT-SWAP @/public vs @/business
    observed_at: 2026-11-13T13:30:00Z
---

## OKX WS 端点分工：candle* 全族归 /business 不归 /public

- **实证**：日K面初版把 `candle1D` 随活跃集 tickers 同帧挂在
  `wss://ws.okx.com/ws/v5/public`——服务端 60018 全拒「channel:
  candle1D,instId:X doesn't exist」（BTC/ETH/SWAP/SPOT 一律）。
  直探实测：candle1D/1Dutc/1W/1H 于 /public 全拒、于
  `wss://ws.okx.com/ws/v5/business` 立返真帧。错误文案写
  「instId doesn't exist」有误导性——真因是端点错非乐器错。
- **教训**：OKX WS 端点分工面——/public=tickers/funding/mark/
  index/books 等行情面；/business=candle* K 线族（含标记价/
  指数 K 线变体）。接新频道先直探端点归属（一帧订实测），
  勿按「公开行情=public」直觉随挂。K 线臂独立连接（klineLoop）
  生命周期与行情连接解耦——面死连不拖。
- **关联病灶**：60018 拒收集按 inst 拉黑在双臂共表合理——
  inst 不存在双臂同真；但「频道错」也走同键，拒集重连清表
  是防「错频道黑名单错杀乐器」的唯一回收径（修频道后
  重连即复）。
