---
schema_version: 2
id: kline-subscribe-write-pred-mismatch
kind: pitfall
risk: high
status: verified
keywords: [md_klines, candle1D, 订而不写, LocalWritePred, ActiveSetUnion, klinesub, lastKline看门狗, KLINE-DAILY-RECV, 断流探针]
scope: [okxws business段, 日K采集, fin_sub_request, 写谓词对齐]
summary: 订阅集扩大（SubRequestArm 进 ActiveSetUnion）而落库谓词未随（LocalWritePred 只认仓/活策略）——帧到而被静默丢,看门狗在打点前不判死,「探针报断流≠venue断流」要先查写谓词。
first_seen: 2026-11-17
last_verified: 2026-11-18
skills: [事实核验, KB维护]
evidence_refs:
  - type: escalation
    ref: 升件#24361
  - type: ticket
    ref: 工单/kline-localwrite-activeset
  - type: doc
    ref: internal/data/okxws/klinesub.go
  - type: doc
    ref: cmd/core/localwrite.go
  - type: doc
    ref: internal/data/okxws/handle.go
---

## 事实/症状

升件#24361：KLINE-DAILY-RECV 断流探针报「日K candle 收帧面断更」——
`md_klines` max(recv_ts) 冻结于 2026-11-17 20:02Z（龄 14h>阈 30min）。
venue 侧 OKX business 端点实测 candle1D 订阅在活、帧在推。

## 根因或策略解释

**订而不写**：T-0401 把 `fin_sub_request` 开放行（SubRequestArm/提案臂订阅面）
并入 `ActiveSetUnion`（klinesub.go）——active set 当时仅 TRX-USDT-SWAP
（`requester=user_intent, reason=quote_watchlist` 开放行）；但 T-0209 落的
写谓词 `LocalWritePred`（cmd/core/localwrite.go）只承认
`inst_id ∈ (开放仓 ∪ 非退役策略)`——TRX-USDT-SWAP 只剩一条已退役
funding-hedge 策略行 → candle 帧到 handle.go:113 处被静默丢弃。

**看门狗次序陷阱**：`lastKline` 看门狗在打点前（handle.go:92-94）先盖戳——
帧到即续期，因此「业务帧静默」告警不触发（实查 ~10:11Z 无此行），
断流探针（30min recv_ts 冻结）与看门狗各说一半真话：venue 侧帧是活的，
死的是写谓词准入集。md_ticks 不受影响（TRX tick 走 OCI collector 桥接，
非 SG-local 产线）——**只有 SG 自产面被写谓词拦**。

## 可防一问

订集与写谓词是同一不变量的两半：扩订阅集（ActiveSetUnion 加源）时必须
同拍审写谓词准入域——否则「订而不写」静默丢帧且看门狗不报警。
修法=写谓词对齐 ActiveSetUnion 全键域（含 sub_request 提名），
验收=挂测断言 sub_request 提名 inst 的 candle 落 md_klines。

## 排查口诀

断流升件先分「venue 真断 ∨ 写谓词丢」：看门狗/活性臂未告警而 recv_ts 冻结
=帧到被丢——查写谓词准入集 vs 订阅集差分，非先查 venue。
