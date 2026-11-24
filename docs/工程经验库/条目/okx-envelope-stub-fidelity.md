---
schema_version: 2
id: okx-envelope-stub-fidelity
kind: pitfall
risk: high
status: verified
keywords: [okxexec剥壳, 桩测失真, unmarshal信封, PublicGet返形, md_open_interest, Get, 信封, 返形契约]
scope: [internal/trading/okxexec, internal/data/okxws, 工单#36961]
summary: okxexec.Get/PublicGet 内部 do() 已剥 {"code","msg","data"} 信封返 data 数组本体——调用方再按 {Data []T} 解包=生产恒败；桩测喂信封形=绿测掩盖恒败（md_open_interest 零行数周）；修法=直解 []T+桩测喂真返形
first_seen: 2026-11-24
last_verified: 2026-11-24
evidence_refs:
  - type: ticket
    ref: 工单#36961（md_open_interest 生产恒零行取证——o-collecto 每 5min OI 解析败 unmarshal array→struct）
---

# 信封剥壳后的桩测失真（OIPoller 实证）

**坑形**：`okxexec.Client.Get/PublicGet` 内部 `do()` 已把 OKX 信封
`{"code","msg","data"}` 剥壳，**返文=data 数组本体**。调用方若再按
`struct{Data []T}` 解包=生产恒 `cannot unmarshal array into struct`。
更毒的是桩测：`liq_oi_test.go` 的 get 桩返回完整信封 JSON——与真
PublicGet 返形不符，测试全绿而生产每拍失败数周，`md_open_interest`
零行无人察觉（md_liquidation 同迁移族有 8k 行，表"在"掩盖了写径死）。

**判别**：①WARN 日志周期性复现同一 unmarshal 型错；②目标表存在且
迁移在册但行数恒零/冻结；③同族姊妹表（liquidation）有数——表在≠写径活。
先对 `Get` 实现核返形（`internal/trading/okxexec/client.go` do()剥壳），
再对照桩测喂的字节形——桩≠真形时绿测无证明力。

**修法**：解析端按真返形直解 `[]T`（同 scan.go parseScanTickers 先例）；
桩测改喂剥壳后形态（裸数组），让测试真覆盖解析径。fail-closed 保留：
坏行跳过不造零值。

**推广**：凡「客户端封装已归一化返形」的 SDK/REST 壳，新增消费端
**先读 do()/Get 的返形契约再写 unmarshal**；桩测数据必须逐字节模拟
真返形（含剥壳与否），信封形桩是本族已知失真源——同型排查对象：
凡 okxws/okxexec 下 PublicGet 消费点。
