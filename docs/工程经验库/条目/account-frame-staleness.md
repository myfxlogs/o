---
schema_version: 2
id: account-frame-staleness
kind: pitfall
risk: medium
status: verified
keywords: [md_account_events, 帧滞后, 对账差, venue 滞后, tradingFundEq, 划转, 处置后虚差]
scope: [cmd/core/restlive.go, md_account_events, ledger 对账, 运维处置判效]
summary: venue 侧权益读面无条件取 md_account_events 最新帧（无鲜度闸）——账户间划转不即时触 balance_and_position 推送，处置生效后 venue 值滞后 ~3-4min，期间对账虚差照刷；判处置生效须等帧落地，勿见差即补行
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [生产取证]
evidence_refs:
  - type: server_fact
    ref: SG 处置划转 transId 3242707203 完成 ~07:57，md_account_events 首帧含 ETH 于 08:00:51 落（滞后 ~3.5min），末条对账差 08:00:02 后静默
  - type: code_ref
    ref: cmd/core/restlive.go tradingFundEq:141-154——SELECT data FROM md_account_events ORDER BY ts DESC LIMIT 1（无 ts 鲜度条件），帧有 details 即永不吃 REST 兜底
---

# 账户事件帧滞后——venue 侧读面非实时

## 坑形

`tradingFundEq`/`okxCoinsFor` 均以 `SELECT data FROM md_account_events ORDER BY ts
DESC LIMIT 1` 取 venue 交易户快照，**无 ts 鲜度闸**：帧只要带 details 即被采信，
REST /account/balance 兜底仅在帧缺/details 空时触发。OKX `balance_and_position`
推送对**账户间划转**（funding↔trading、savings 赎回）不即时落帧——实测滞后
~3-4min。

后果：钱径处置（划转/赎回/归位）完成后，venue 侧权益读面仍呈处置前值，
对账虚差继续刷——若误判「处置无效」而追加补偿行/重复划转=**真差**。

## 判据

- 处置类运维后对账差不随第一拍消失≠处置失败——先查
  `SELECT channel,ts FROM md_account_events ORDER BY ts DESC LIMIT 1` 帧时点，
  再看帧 details 是否含处置后的币行。
- 帧每 ~5s 一条（account channel 常态推）——但内容反映 venue 推送事件，
  非轮询现值。

## 处置律

1. 钱径处置后等 ≥1 个帧落地周期（实测 ≤4min）再判对账静默与否；
2. 验证处置生效用** venue 侧实读**（REST /account/balance、poschk 类
   运维件），不用对账告警消退作证——告警消=帧追上了，非钱到位；
3. 滞后窗内**禁**因虚差续刷而写 recon_adjust/重发划转。

## 防再发（候选项——T-0104 审计随行裁决）

- 帧鲜度闸：tradingFundEq 取帧加 `ts > now()-interval`（帧陈→REST 兜底）；
- 或 mutating 面完成事件后强制 REST 刷新一次 venue 读面。
