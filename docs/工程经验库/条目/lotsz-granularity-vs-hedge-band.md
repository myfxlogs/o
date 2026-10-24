---
schema_version: 2
id: lotsz-granularity-vs-hedge-band
kind: pitfall
risk: high
status: verified
keywords: [lotSz, 粒度, hedge_band, delta_drift, 名义额, 量纲失配, 小额仓, 网格取整]
scope: [internal/trading/execution/sizing.go, internal/guard 判据面, 首仓序]
summary: 对冲带（20bps）远小于小额仓 lotSz 网格粒度（$24 腿×0.01 张=$2.37=9800bps）——floor 网格取整后结构必触发 delta_drift 真平仓
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-exec]
evidence_refs:
  - type: server_fact
    ref: T-0056 exec34 双腿 filled：perp 0.1 张($23.66) vs spot 0.010539($24.93)=漂移 520bps>带 20bps→guard 6 分钟内真触发平仓（exec35 双腿 filled 回卷）
---

# lotSz 网格粒度 vs 对冲带量纲失配（T-0056 实证）

## 病灶
- BuildLegs 双腿各自 floor 到乐器 lotSz：perp 0.01 张粒度×ctVal0.1×px≈$2.37；spot 1e-6 粒度≈$0.0024。$50 名义（腿 $25）下 perp 可调步长=腿名义的 ~10%——**取整后漂移 520bps 是结构性的**，hedge_band 20bps 在此名义额尺度物理不可满足。
- 判据与可执行性脱节：band 是风控期望，lotSz 是物理约束——判据紧过物理约束=开仓即触发看护平仓（首仓序两度被杀的第二形态，第一形态=T-0046 量纲雷）。

## 裁决
- 开仓预检必含「粒度可行性」：`floor 后双腿名义漂移 ≤ band` 才许开——不可满足=名义额太小，拒开或升名义额（对 perp 粒度：名义 ≥ lotSz×ctVal×px/band_bps×1e4，ETH 例≈$2.37/0.002≈$1185——$50 物理不可对冲）。
- 或带分级：小额仓 band≥粒度上界（判据随物理约束放宽须显式决策，禁静默）。
- 备选乐器降粒度：选 lotSz×ctVal×px 小的标的（ETH $2.37 已是候选表最小档——$50 名义在 OKX USDT 本位 SWAP 全谱不可行，须 $1200+ 或提带）。

## 判据
- 「开仓成功即被 drift 平仓」复盘先查粒度失配（legs sz 非整 lot 倍数×漂移恰=取整差）——非行情漂移非 bug。
