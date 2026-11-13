---
schema_version: 2
id: wsmsg-data-dual-shape
kind: pitfall
risk: medium
status: verified
keywords: [wsMsg, data 载荷, candle, 数组载荷, 对象载荷, WS 解析, okxws]
scope: [internal/data/okxws, WS 采集, parse, handle]
summary: OKX WS data[] 载荷频道异形——tickers/funding/mark=对象而 candle*=数组，统一 map 解码会把 K 线帧解析成全批 parse_fail
first_seen: 2026-11-13
last_verified: 2026-11-13
evidence_refs:
  - type: ticket
    ref: 13065
  - type: commit
    ref: 173653490
  - type: doc
    ref: internal/data/okxws/parse.go
---

## wsMsg.Data 频道异形：新频道接帧先验载荷行形

- **实证**：md_klines 面落地时发现 `wsMsg.Data` 原为
  `[]map[string]json.RawMessage`——对象形单假设。candle 频道载荷
  是数组 `["ts","o","h","l","c","vol","volCcy","volCcyQuote","confirm"]`，
  若直接接帧：整帧 Unmarshal 败→parse_fail 计 rejects，日K零行
  而探面只剩「解析败」症状，根因埋在帧头层。
- **教训**：OKX v5 `data[]` 元素行形按频道分两类——对象
  （tickers/funding-rate/mark-price/index-tickers/books5）与数组
  （candle* 全族/将来 trades 类）。接入新频道前先查该频道载荷
  形态（docs-v5 参数字典面），Data 只载 `[]json.RawMessage`，
  由频道解析面自解（obj() 壳保对象面旧径）。数组位序是契约：
  位 0=ts 毫秒、位 8=confirm（"1"=收线）——缺位/型错收 nil。
- **关联病灶**：收线帧 confirm=1 是「瞬间终态」——节流闸若统
  跳会丢收线边界终值；规则=形成中帧可节流，confirm=1 帧必落。
