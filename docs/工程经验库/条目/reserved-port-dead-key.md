---
schema_version: 2
id: reserved-port-dead-key
kind: pitfall
risk: medium
status: verified
keywords: [预留口, 拼键, 死面, 键域, 假活面, 接线, 端到端实证]
scope: [internal/research, 接口预留/接线面, 键域]
summary: 预留口（AttachSignals 类挂点）交付时若拼键域未经端到端命中实证=假活面——读闸桩测全绿但键永不命中（okx-USDT-SWAP 非 instId）；预留口必须同票端到端接线或钉死域映射契约，桩测须含真实键域命中断言（T-0139→T-0140 返工实案）
first_seen: 2026-10-26
last_verified: 2026-10-26
skills: [策略评估与仓位]
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0140（面1——T-0139 AttachSignals 拼键 okx-USDT-SWAP 非合法 instId 永不命中，读闸桩测绿=假活面）
  - type: code_fact
    ref: internal/research/sigdist SurfaceSignals 语义映射+TestSurfaceSignalsMapping 真实键域命中锚
---

# 预留口拼键死面=桩测全绿的假活面

## 病灶形

T-0139 交付 AttachSignals 读闸+桩测（stub 直证附据形全绿）——但装配侧键
`venue+"-USDT-SWAP"`=`okx-USDT-SWAP` 非合法 instId（真形 `BTC-USDT-SWAP`），
且接线也未做——**双死面被绿桩测掩盖**。桩测用 stub 自证自，键域对不对
桩测不管——「读闸语义对」≠「端到端命中」。

## 判据挂接

- 预留口（可选依赖挂点）交付时二选一：同票端到端接线，或票面明钉「接线
  另票+键域契约」（映射函数归属/键域形）。
- 桩测须含**真实键域命中断言**（非 stub 自证）：键空间从生产数据形采样。
- 接线票的键域修正锚：映射函数单测（surface→kind/inst 域）+端到端绿。

## 处置

T-0140：SignalReader 签名改 surface 语义映射（键域知识归 sigdist 适配器）+
cmd/core 接线+TestSurfaceSignalsMapping 真实键锚。
