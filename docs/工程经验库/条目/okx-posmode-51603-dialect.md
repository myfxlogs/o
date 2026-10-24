---
schema_version: 2
id: okx-posmode-51603-dialect
kind: pitfall
risk: high
status: verified
keywords: [okx, posMode, long_short_mode, posSide, 开仓腿, 51603, Order does not exist, absent, 定态, 拒码留痕]
scope: [internal/trading/okxexec, internal/trading/execution, venue 方言装配面]
summary: OKX 双向户（long_short_mode）SWAP 开仓腿 posSide 必填——只给平仓腿补是半修；51603=确定单未建（absent 非查询败）；业务拒码 sCode≠0 必须留痕
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: []
evidence_refs:
  - type: server_fact
    ref: 真单 exec_id=2 实拍拒（posSide 缺位开仓腿拒；/account/config uid=…48 posMode=long_short_mode）
  - type: doc
    ref: internal/trading/execution/okx.go buildOrderParams
---

# OKX posMode 方言三坑（T-0040 真单二拒实勘）

## 病灶

- **posSide 半修**：T-0036 只给平仓（Close）腿补 posSide/reduceOnly，开仓腿空着
  ——双向户 SWAP 开仓同样必填 posSide，真单 exec_id=2 二拒实锤。
- **51603 误判**：`{"code":"51603","msg":"Order does not exist"}` 是**确定的单未建**
  （absent 语义，同 clOrdId 可重试幂等键），曾被当查询败→unknown→歧义禁补。
- **拒因吞没**：Place 业务拒（sCode≠0）只归一为 absent，真拒码/拒因不落库
  ——排查面缺口。

## 裁决

- posMode **实读** `/account/config`（一次缓存），禁硬编假设；读不回=fail-closed 拒单。
- `long_short_mode`+SWAP（tdMode=cross）：开仓 sell→short/buy→long，平仓 buy→short/
  sell→long；net_mode 与 spot（cash）免带。
- envelope code=51603 → StateAbsent（类型化 APIError 按码定态，禁字符串匹配）。
- sCode≠0 必带码+msg 进 fin_order_leg.venue_err_code/msg 留痕。

## 判据

- venue 方言装配改纯函数（buildOrderParams 式）可表驱动测：posMode×close×side×instType
  四维正交断言。
- 「查询不到单」先分清：业务码 51603（确定 absent）vs 传输/HTTP 败（unknown 禁补单）——
  两态处置相反，混同=补单风险或死锁二选一错。
