---
schema_version: 2
id: okx-ws-flap-triage
kind: caliber
risk: low
status: verified
keywords: [okx, websocket, 断流, depth5, books5, okxws, flap, 复绿, dead阈]
scope: [o-collecto, 探测席断流探针, 处置席分诊]
summary: OKX WS 面断流升件先看同窗「恢复」行——死窗 1-6min 内复绿
  =瞬断自愈不处置不销件等它销；簇发(≥3件/小时,depth5+okxws 双面同簇)
  =重订阅节奏/阈值质量问题归码修票,不逐 flap 唤醒。死窗超时无恢复行
  =真断供照断流档深查。
first_seen: 2026-11-25
last_verified: 2026-11-25
skills: []
evidence_refs:
  - type: server_fact
    source: ops_event 升件/恢复行对
    query: "SELECT id,note FROM ops_event WHERE event IN ('升件','恢复')
      AND note LIKE '%断流%' AND ts>now()-interval '1 hour' ORDER BY id"
    observed_at: 2026-11-25T03:56:00Z
    server_revision: ""
    response_hash: ""
  - type: lesson
    ref: fin_eng_lesson#103 signature=升件:断流:okxws WS 行情 cluster_n=3
---

## 事实/症状

断流探针对 feed 面打 dead 阈（如 depth5 阈 2m0s），超时即
`升件/断流` + 恢复即 `恢复` 行成对落账。2026-11-25 03:18-03:56
一窗实证：depth5 面 5 件（陈 2m19s-2m46s）+ okxws WS 行情面
3 件，每件死窗 58s-6m0s 内全部复绿——无一真断供。

## 判读口径（处置席分诊尺）

1. **单发瞬断**：升件落地后同窗有「复绿」行（note 引 `升件#id`
   +死窗时长）→ 定性=WS 瞬断自愈，销件载复绿行号+死窗。
   死窗内未复绿前不抢处置——等复绿行或死窗超时。
2. **簇发 flap**：同面∨同 venue 多面 ≥3 件/时（本簇 8 件/38min
   双面）→ 瞬断定性仍逐件销，但**簇本身另立码修票**——
   复发形态三候选：o-collecto 订阅断连重订阅节奏过频/
   dead 阈值相对 OKX WS ping 节奏过紧/上游网络抖动。
   逐 flap 唤醒是拍成本浪费，簇归票消根。
3. **真断供**：死窗超时无恢复行（本簇零案）→ 照断流弹目
   深查 OCI o-collecto 订阅段，不等自愈。

## 防点

- 探针升件 note 自带「断连重订阅会自愈」提示——处置席先查
  `event='恢复'` 同窗配对再定性，别把瞬断当真断修。
- lesson 矿工簇签名 `升件:断流:okxws WS 行情`（fin_eng_lesson#103,
  evidence #39780/#39803/#39839）本条目即其 canon 终态。

## 相关

- 簇发根查票：工单/okxws-flap-cadence（同拍立案）
- 断流弹目：scripts/cognition/runbook.d/playbook-esc.md
