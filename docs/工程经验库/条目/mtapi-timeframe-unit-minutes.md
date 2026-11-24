---
schema_version: 2
id: mtapi-timeframe-unit-minutes-not-mql5-enum
kind: pitfall
risk: high
status: verified
keywords: [PriceHistoryExMany, timeFrame, MQL5 ENUM_TIMEFRAMES, 周期错配, 周K伪日K, md_klines venue=mt5, confirm 前沿, 16408, 1440]
scope: [internal/mt5 klines 面, md_klines venue='mt5' 数据面, mtapi 网关参数方言]
summary: mtapi PriceHistory* 族 timeFrame 域单位=分钟数非 MQL5 ENUM_TIMEFRAMES——生产 klines.go 误搬 PERIOD_D1=16408（分钟语义≈11.4d）被网关吸附 W1，全簿 67011 行周日戳周K伪装 1D 自 2021 存续；末棒新鲜+recv_ts 秒鲜使陈窗探测盲区，confirm 前沿周级翻旗=下游日棒消费方饿停表象。
first_seen: 2026-11-24
last_verified: 2026-11-24
skills: [mt5-quotes-history]
evidence_refs:
  - type: rpc_probe
    source: mt5grpc2.mtapi.io:443 共享会话消费腿实测（tools/mt5tfprobe——tf∈{1,60,240,1440,10080,16408,32769} 回棒间距实测图）
  - type: server_fact
    source: postgres md_klines venue='mt5'（67011 行 extract(dow)=0 唯一值）
---

# mtapi timeFrame=分钟单位——MQL5 枚举值误搬=周K伪日K全簿

- **坑**：工单O-38639。`internal/mt5/klines.go` 自 #25563 落地起
  `mt5FrameD1=16408`——系照 MQL5 `ENUM_TIMEFRAMES PERIOD_D1=16408`
  抄值，但 mtapi `PriceHistoryExManyRequest.timeFrame` 域单位是
  **分钟**（proto 例 `"timeFrame":"240"`=H4 即锚）。16408 分钟
  ≈11.39 天，网关吸附最近支持周期 W1——请求「日K」实收**周K**。
  实测方言图（EURUSDm tf∈ 值→末三棒间距）：1→分钟、60→小时、
  240→4h、**1440→真日K（11-22/23/24 连续日戳）**、10080→周K、
  16408→周K（同 10080）、32769→周K。
- **签名**：md_klines venue='mt5' bar='1D' 全簿 ts **100% 周日戳**
  （`extract(dow)=0` 唯一值，67011 行 2021-05→今）；末棒=形成中
  周K confirm=0 挂账到下周日，confirm=1 前沿=上上周日棒——表象
  「收线旗永不翻/消费方饿停」，实为周期错配非断供。recv_ts 秒级
  鲜（扫拍每 4h 全量覆写）+末棒 ts 新鲜（本周日）双重伪装。
- **律**：①跨系统搬常量先核单位域——「枚举值」直觉=最大坑源，
  proto 例值+实测间距图是唯一裁判（tools/mt5tfprobe 可复跑）。
  ②周期类数据面落库须验 ts 间距分布非只验末棒新鲜——周K伪1D
  在「末棒新鲜+recv_ts 鲜」下对陈窗阈（>14d）天然免疫；已补
  `klineAvgGapSuspect` 间距>3d WARN 闸。③伪周期存量清册走
  md_1107 DELETE+扫拍自回填（NumBars=200≈10 月真日棒）。
