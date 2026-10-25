---
schema_version: 2
id: ws-subscribe-frame-flood
kind: pitfall
risk: high
status: verified
keywords: [ws, subscribe, close 1009, message too big, 分帧, 宇宙扩张, okxws]
---

# WS subscribe 单帧爆量——close 1009 斩连拖死行情

## 现象

订阅参数集一大（T-0122 实证：507 标的×3 频道=1521 args 单帧），
OKX WS 服务端 `close 1009 (message too big)` 直接斩连——断连环里
tick 停流，依赖最新价的 spot()/估值/对账全吃陈值。

## 修法/防法

- subscribe args **必须分帧**（实操惯例 ≤100 args/帧，或按帧字节封顶）；
- 宇宙面/订阅面扩张票（任何批量订阅）先分帧再上量；
- 适用面：okx-market-ws 及其他批量订阅 venue 同族。
