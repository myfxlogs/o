---
schema_version: 2
id: ticket-scope-suppress-key-mismatch
kind: pitfall
risk: low
status: verified
keywords: ["票抑键", "esc_scope", "升件簇", "抑键粒度", "销记负担", "MT5簇"]
scope: [工单, 探针抑制, ops_esc_pending, 处置席]
summary: 票抑键=升件 esc_scope 精确单键——同根断供簇跨多 scope 时一票只压同名 scope，余簇每拍续火，处置席逐拍销记成常态负担；想压整簇=票面 scope 须写源探针键或探针认票关联。
first_seen: 2026-11-20
last_verified: 2026-11-21
skills: []
evidence_refs:
  - type: server_fact
    ref: 工单/mt5-old-version-vendor 立案 scope=mt5（11-20）；蒸馏 xvenue_spread/carry_diff/mt5_swap + MT5 桥行情 四 scope 升件自 05:35Z 死窗起每 ~3min dedup 窗续火（#26640-43→#26707-10→#26979-82…），唤知会话逐拍销记
  - type: code
    ref: 票抑键=源升件 esc_scope（runbook 工单段「同 scope 探针静默让位」）；hunger 探针抑契「同 scope 开工单在案即抑」（internal/portfolio/hunger/closeoutprobe.go:24、unitdrift.go:20-21）——抑键=单 scope 精确匹配，无族配/关联票
---

# 票抑键是 scope 精确单键——一票压不住跨 scope 同根簇

## 坑形

断供根因是一处（MT5 vendor 拒连），但症状落在**多个探针 scope**：
蒸馏 xvenue_spread/carry_diff/mt5_swap 三个 sigdist 面 + MT5 桥行情
面，各自独立死阈独立升件。立案 `工单/mt5-old-version-vendor`
scope=mt5，只压住 scope 名恰为 `mt5` 的探针（若有）——四个实际
续火 scope 一个都不匹配，于是每拍 dedup 窗过即重火，唤知会话
每轮开场先销四行同簇信号，销记成流水线噪音。

## 机理

票抑契设计前提=「票接管同 scope 升件所问的问题」——键就是
esc_scope 字面。它没有错：它假设一票一 scope。跨 scope 同根簇
（单点断供→多面症状）打破该假设——票在案、根因在办、信号照火。
处置席逐拍销记是对的（信号为真），但成本是每轮固定四行噪音。

## 判词/修法候选（本拍裁定素材，非施工）

- 票面 scope 直接写续火探针键（如 `蒸馏 carry_diff` 等）——
  只压一键，簇大仍需多票；
- 票面载簇签（如 `suppress_scopes=[...]`）+探针抑契读票簇签——
  要码修，机制票三栏备齐再裁；
- 维持逐拍销记（现状）——零码成本，销词模板化；
- 判：机制缺口在抑键粒度，非处置错、非探针错；簇火期间逐拍
  销词载根因票号即合规闭环。
