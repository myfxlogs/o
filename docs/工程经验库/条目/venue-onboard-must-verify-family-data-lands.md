---
schema_version: 2
id: venue-onboard-must-verify-family-data-lands
kind: method
risk: medium
status: verified
keywords: [venue接入, 接市验收, 逐族验, scan表, 目录落表, md_tick_scan, 水位闸, scanwm, 新CEX, T-0440, 连接通不等于数据落表]
scope: [internal/venue/onboard, internal/platform/scanwm, md_tick_scan, 接市向导]
summary: 新 CEX/broker 接入验收不能止于「连接通」——必须逐族验「目录进得了 scan 表」（md_tick_scan 各族行数>0 实证）才算接市完成；族缺席类病（目录超帽截断/端点方言/权限差）连接全绿零症状，只有落表实证能暴露。通用兜底=scanwm 族谱水位闸（注册面派生期望族集，零鲜落 ≥10min 升件）
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: []
evidence_refs:
  - type: server_fact
    ref: 2026-11-07 T-0440 实证——okx 连接/私读/公开读全绿，SPOT 族目录 1.31MB 被 1MB 帽截断致 spot=0（md_tick_scan swap=528 futures=6 spot=0），业主前端搜 XRP 无现货才发现；病存续期间全系统无告警
  - type: code_ref
    ref: internal/platform/scanwm（族谱水位闸通用形制——期望集=注册面派生零 venue 硬编码）；internal/data/okxws/scan.go scanFams 注册面正本
---

# 接市验收逐族验「目录进得了 scan 表」——连接通≠数据落表

## 方法

新 CEX/broker 接入（onboard 四拍/向导彩排）的验收判据必须到**数据落表**
层，逐族过：

1. **连接通**（现有向导面）——凭据/网关/REST 私读通；
2. **逐族目录落表**（本条补）——对每个应扫族（venue 注册面声明的族集，
   okx 当前=swap/spot/futures）实证 `md_tick_scan WHERE venue=X AND
   inst_type=族` 行数 >0，且 recv_ts 随拍前进；
3. **水位闸兜底常设**——接市时注册面挂入 scanwm 闸域（T-0440 通用形制：
   期望集=注册面派生，族零鲜落持续 ≥10min → ops_event 谱域缺行升件），
   新 venue 注册即自动入闸，闸码零 venue 硬编码。

## 为什么

okx SPOT 案（T-0440）：连接/私读/公开读全绿数月，SPOT 目录 1.31MB 被
1MB 帽静默截断→族级 fail-closed 每拍让位→spot=0。病存续期全系统零告警
（部分族让位仅 docker 日志一行），发现靠业主人工对拍。**「连接通」层
验收结构性看不见族级数据缺席**——缺席无感族第 N 条实证，对治=落表实证
+水位闸机械兜底，双保险缺一不可。

## 判据

- 接市完成态=「各族 scan 表行数>0 + recv_ts 鲜度前进」实证行落 ops_event
  /票面，非「跑起来了」口说；
- 水位闸升件（谱域缺行/谱域不可判）=接市后族级病灶第一信号，处置走
  常票（闸只报不修）。
