---
schema_version: 2
id: expect-prod-coldstart-false-fire
kind: pitfall
risk: medium
status: verified
keywords: [expect_, 预期产出, 冷启动, 部署, 假升件, self_aware, 死窗]
scope: [cmd/core, expect_esc, 探测, 部署]
summary: 期望产出探针部署零窗即火——新册面首产拍未满必先假升后自愈，须按末次部署龄冷启宽免
first_seen: 2026-11-13
last_verified: 2026-11-13
evidence_refs:
  - type: ticket
    ref: 13830
  - type: commit
    ref: 16dbc3e73
  - type: doc
    ref: cmd/core/expect_esc.go
---

## expectProdEscCheck：部署冷启零窗=确定性假升件

- **实证**：self_aware 器官 2026-11-13 随部署上线——升件#13763
  `expect_self_aware` 零行即火，#13767 仅 1m39s 后恢复自闭。新册面
  「产出窗内无行」在部署瞬间是必然态（首拍 6h，生产行要等首个 beat
  才写），探针把「还没轮到产」判成「断供」=每发新器官必噪一件。
- **教训**：判定「应产未产」前先问「给了它没有首产时间」——窗语义
  （6h 拍 9h 窗）只管稳态断供，不管冷启。修法=查末次权威部署行
  （`seat='DEPLOY' event='部署' note LIKE '%o-core:%' rc=0`），部署龄
  <宽免阈（10min）整册让位；读径查询失败照常评不借宽免暗灭，真断供
  宽免期后下一拍照升。
- **关联病灶**：任何「注册表+窗计数」型探针加新面时都要想到冷启——
  部署即火不是器官坏了，是探针假设了稳态初值。
