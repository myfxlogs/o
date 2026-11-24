---
schema_version: 2
id: baseline-arm-late-cashflow-double-count
kind: pitfall
risk: high
status: verified
keywords: [native_baseline, 迟簿, 双计, arm时窗, 补偿腿, lateSrc, scanner回填, fin_risk_decision]
scope: [internal/ledger, 对账基线凭据, 一切快照式归因凭据]
summary: 快照式基线凭据（native_baseline 首拍 resid 建账）与迟到账现金流存在结构性竞争——occurred_at<arm_ts 且 ts>arm_ts 的 custody 现金流使基线已含的 venue 效应再入 net=双计；根修=armed 键逐迟到账行补反号负腿（lateSrc=cfid 幂等），reconcile 拍内自愈不待人工迁移
first_seen: 2026-11-24
last_verified: 2026-11-24
evidence_refs:
  - type: ticket
    ref: O-37868（NFT 双簿：基线#35360+迟簿deposit #1808 同笔入金双计→账本 103.58M=venue 51.79M×2）
  - type: server_fact
    ref: fin_cashflow #1808 ts=2026-11-24 05:02（arm 2026-11-23 02:44 之后落账，occurred_at=2025-08-25 史前）——O-36422 币形闸放开后 scanner 首扫回填史前入金批（NFT/SHIB/BTT 同批；SHIB/BTT 靠 withdraw 对冲巧合幸免）
  - type: decision
    ref: fin_0250 迁移负腿冲销+豁免#24/#26作废+机制闸判词=要闸
repro_or_symptom: venue 余额含某笔历史资金但账本未落→首拍对账武装 resid=全额建基线→迟后该笔 cashflow 落账（occurred_at 早于武装、ts 晚于武装）→ledger=venue+该笔=精确 2 倍；USD 残差随价漂复火循环
root_cause: armNativeBaselines 的 cf 求和只取「武装时已存在行」——武装后落账但 occurred_at 早于武装的现金流，其 venue 效应已含在 resid 内、cashflow 又进 net，一份钱计两次。scanner 基线截止谓词（scanner-baseline-cutoff 条目）只挡 occurred_at≤账本最早行的史前件，挡不住「账本最早行之后、武装之前」的迟簿窗
fix_and_prevention: ①armed 键迟到账自愈闸 compensateLateBaselineBooks——fin_cashflow JOIN native_baseline MIN(ts) 键集，occurred_at<arm_ts∧ts>arm_ts∧kind∉(earn_in,earn_out) 逐行补 amt=-amount 负腿，lateSrc=cfid 幂等（手工迁移腿与机腿同键域不双补）②快照式凭据凡「一次性吸收未解释量」皆有此竞争面——新设基线/豁免机制时先问迟簿窗③检测面已存（diff 告警即发现径），缺的是自动归正——闸放 reconcile 拍内不依赖 venue 现读/在途窗
notes: 反向同构——withdraw 迟到账补 +|amount| 负腿（amt=-amount 符号自洽）；earn_in/out 与 arm/reconcile cf 口径同排除；白名单键无基线可双计故自愈只及已武装键
---

## 现象

O-37868：NFT venue 恒 51.79M，账本净 103.58M 精 2 倍。基线凭据 #35360（11-23 武装时 cf 无 NFT 行→resid=全额建）与迟簿 deposit #1808（bill 源 12701247，occurred 2025-08-25，ts 11-24 05:02 落账）同笔入金双计。USD 残差随 NFT 价漂（5.79e-8/5.79e-7 十倍换算缺陷特征叠加，O-38064 另票），豁免 #24/#26 栈叠治标复火循环。

## 根因

`armNativeBaselines` 的 `sum(amount)` 无 ts 窗——武装时刻「已存在」的现金流全进 net_then，resid 吸收其余全部（含未落账的 venue 侧历史资金）。迟后落账的现金流 occurred_at 早于武装=其 venue 效应已在 resid 内，cashflow 再入 net=双计。**结构性存在**：scanner 史前回填、迁移补账、币形闸放开首扫皆可复发——非一次性事故。

## 修法

`compensateLateBaselineBooks`（internal/ledger/native_base.go）：armed 键上 `occurred_at<MIN(baseline.ts) ∧ ts>arm_ts ∧ kind∉(earn_in,earn_out)` 的现金流逐行补 `native_baseline` 负腿 `amt=-amount`、`lateSrc=cfid` 幂等锚。置 reconcile tol 后 inflight 前——纯账本内修正，不依赖 venue 现读与在途窗。fin_0250 迁移手工腿与机腿同 lateSrc 键域（'1808'）天然不双补。
