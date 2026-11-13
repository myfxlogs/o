---
schema_version: 2
id: bridge-domain-writer-side
kind: pitfall
risk: medium
status: verified
keywords: [bridge, bridge-reverse, domain.tsv, 镜像, 投影, 写件驻侧, 空副本, projector, md_klines]
scope: [migrations/domain.tsv, internal/projector, 全域桥表声明面]
summary: 桥表域分凭表名直觉（md_*→bridge）而非写件驻侧——正本/副本方向倒挂，桥审只查双链 DDL 存在不查数据流向，副本侧永空集无告警
first_seen: 2026-11-13
last_verified: 2026-11-13
skills: []
evidence_refs:
  - type: server_fact
    ref: SG md_klines 19 inst 形成行在库，PG-md 同表 0 行（domain=bridge 正向谱下副本方向倒挂）
  - type: code
    ref: internal/projector/reverse.go reverseKlines（md_earn_rate 先例同形）
  - type: test
    ref: internal/projector/projector_test.go TestReverseKlines
---

# 桥表域分看写件驻侧不看表名——md_ 名≠md 正本（工单#13065 切片④）

## 病灶

`domain.tsv` 把 `md_klines` 登为 `bridge`（正向桥=正本在 PG-md、projector 投影→SG）。
实际写件是 SG 侧 `o-core/okxws`（fin 链直接 INSERT）——正本在 SG，数据流向应
SG→PG-md 即 `bridge-reverse`。结果：SG 19 个 inst 形成行在库、PG-md 副本 0 行，
假想机/评估面读 PG-md 侧=永空集。

桥审只断言双链 DDL 存在+授权，不查数据流向——域分错挂无告警，靠消费侧空集
才浮（本例=eval 消费径勘查时实证）。

## 修法

1. **域分判据=写件驻侧**：`bridge`=正本 PG-md（投影→SG）；`bridge-reverse`=正本
   SG fin 链（反向镜像→PG-md 只读副本）。`md_*` 表名不定方向——`md_earn_rate`
   （T-0214）即 md_ 名而 SG 正本的先例。
2. **反向臂同形件**：全表 diff（低频小表期界对账兜底足够）+ upsert 键=正本 PK；
   `projector.New` 反向册对表（fail-closed）+ `Reconcile` 调用 + 表名入
   `domain.tsv bridge-reverse` 行。
3. **不造新 NOTIFY 范式**：无 SG 触发帧的反向表随 Reconcile 期界收敛（日频表
   10min 期界绰绰）。

## 通用律

- 新表入册先问「谁写」再问「谁读」——写件驻侧定域分，读侧定授权/镜像向。
- 双链表零行副本=域分倒挂签名（非「还没采到」）——先查 writer 驻侧再查镜像臂。
- 域分翻转纯元数据（domain.tsv 行词），不碰已落 migration（append-only）。
