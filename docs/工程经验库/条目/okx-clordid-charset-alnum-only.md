---
schema_version: 2
id: okx-clordid-charset-alnum-only
kind: pitfall
risk: high
status: verified
keywords: [clOrdId, OKX, 51000, 字符集, 幂等键, venue 方言, 下单拒]
scope: [internal/trading/execution/flow.go, 全域 venue 幂等键生成面]
summary: OKX clOrdId 只收大小写英数（≤32）——连字符等特殊符=51000 Parameter error；Place 与 Resolve 同串双拒即 ambiguous，fail-closed 兜住但单开不出（真单实勘 exec_id=1）
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [okx-exec]
evidence_refs:
  - type: server_fact
    ref: 真单实测 exec_id=1：Place 拒→Resolve 同串亦拒 51000（T-0038 票面）
  - type: code
    ref: internal/trading/execution/flow.go（ph<exec>x<leg>x<nano> 生成式）
  - type: test
    ref: internal/trading/execution/earn_test.go TestClOrdIDCharset（^[A-Za-z0-9]{1,32}$ 锚）
---

# OKX clOrdId 字符集=大小写英数 only（T-0038）

## 病灶

幂等键生成用了人读友好格式 `ph-1-2-456789`（连字符分隔）。OKX REST 契约：clOrdId
只收**大小写字母+数字，≤32 位**——连字符即 `51000 Parameter clOrdId error`。更险的
是歧态势链：Place 因字符集拒（order 未建）→ 定态径 Resolve 携**同一非法串**查询亦拒
→ 三分支落在 unknown/ambiguous → fail-closed 不开仓（钱零损，但单永远开不出且留痕
面全是噪音）。

## 修法与通用律

- 生成式改字母位分隔：`ph<exec>x<leg>x<nano>` / `ph<exec>rb<nano>`——英数自检锚
  `^[A-Za-z0-9]{1,32}$` 常绿（生成面漂移即红）。
- **venue 方言字段先验字符集再上生产**：幂等键/ClientOrderId/标签类自由字符串面，
  每个 venue 契约不同（OKX 英数 32、他 venue 或宽松）——适配层生成式集中一处+字符集
  正则锚，禁散落 Sprintf。
- 病灶暴露面=真单：模拟户/单测皆不拦（fake 不校验字符集）——**首单前对自由字符串面
  做契约 grep 实拍**是验证梯必修步。
