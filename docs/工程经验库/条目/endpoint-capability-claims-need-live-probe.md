---
schema_version: 2
id: endpoint-capability-claims-need-live-probe
kind: pitfall
risk: high
status: verified
keywords: [API方言, 立票前提, 真户探针, from/to, withdrawal-history, 择源]
scope: [integration, okx, design]
summary: 立票钦定面里「某端点带某域」的能力断言（如 /asset/bills 带 from/to、withdrawal-history 漏内部转出）未真户实测即写死——施工择源时被探针逐条推翻（前提全反）；端点能力断言=可实测命题，落票前必真户探针钉死
first_seen: 2026-11-01
last_seen: 2026-11-01
evidence_refs:
  - type: ticket
    ref: T-0274
  - type: file
    ref: skills/okx-asset-funding/SKILL.md
---

## 立票端点能力断言未探针钉死→择源前提全反

T-0274 立票钦定两候选前提：①`/asset/bills` 带 from/to 户号域（分辨内外向）；
②`/asset/withdrawal-history` 只盖链上提币、OKX 内部转出漏网。施工真户只读
探针（SG 真钥 GET）逐条推翻：①实响应无 from/to/subType（仅 notes 人读标）；
②内部户转出（wdId 440849407，chain/txId 空、to=手机号）**在册且 state=2**。
两前提全反→择源反转（②为常设主径），若施工照票面直写①=分辨域空转+外向
漏记恒差复发。

**纪律**：端点「带什么域/盖什么形态」=可实测命题非文档记忆——立票引用前
真户探针（只读 GET 成本≈零）；docs-v5 字段表与真户响应存在出入（精简响应/
版本漂移），以探针为准并回填 skill（okx-asset-funding T-0274 修订在案）。
施工侧遇票面方言断言与实测冲突：实测为准+票面偏离说明，禁照猜施工。
