---
schema_version: 2
id: receipt-lexeme-announcement-vs-citation
kind: pitfall
risk: high
status: verified
keywords: [land_queue, review_ok, aseat, 契词, 回执判词, 宣告位, 引述位, ORDER BY id ASC, 分席证明]
scope: [scripts/land_queue.sh, ops_event 簿记面, 一切「词面判回执」闸]
summary: 回执判词的契词匹配必须落「宣告位」（行首窗/回执事件族）而非「引述位」（行中任意位置）——诊断票文/会话叙事引契词+支名会被当作者回执夺 aseat=分席证明失真（fail-open 向）；取行序=ORDER BY id ASC 首报工语义，DESC 最新行会被后续补行改写作者面
first_seen: 2026-11-24
last_verified: 2026-11-24
evidence_refs:
  - type: ticket
    ref: O-38132（fix/37450 land_queue 永跳诊断——实证断因与修法裁定）
  - type: server_fact
    ref: ops_event#38132（处置席/工单——票文引「候审回执|施工收口」契词+fix/37450 支名）、#38135/#38213（唤知/会话收口叙事引契词）——旧谓词 note~契词 ANYWHERE 全命中
  - type: server_fact
    ref: O-37450 真回执 #38000 用「交付@sha」非契词——rc3 fail-closed 判词本来就正确（契词行实不在案，spec 前提「#37730 契词行在案」系误植——该 id 实为 O-37448 回执）
repro_or_symptom: ops_event 任意行 note 中出现契词字样+支名/sha 即被 review_ok 当完工回执取作者席——写诊断票、会话收口叙事、复盘行时只要引述契词表就污染 aseat；ORDER BY id DESC 取最新行时后续叙事行优先于真回执
root_cause: ①契词判词用 note ~ '(完工回执|完工推支候审|候审回执|施工收口)' 子串全位匹配——不分宣告/引述②取行 ORDER BY id DESC=最新行胜——回执语义应为首次报工，补行/叙事不应改写作者面③event ~ '完工' 只认一族漏 施工收口/候审回执/工单完工/回执 事件族
fix_and_prevention: ①回执判词=宣告位：event ~ '完工|候审|施工收口|工单完工|回执'（回执事件族自报）OR note ~ '^.{0,40}(契词族)'（行首 40 字符窗——实测真回执契词位 max=21 字符，元行引述位 min=58，窗 40 两侧余量）②ORDER BY id ASC LIMIT 1=最早契词行=首报工语义③落成自动销 EXISTS 同判词同步收（同机制洞不同消费面，一并收紧方向皆 fail-closed 归人工）④写诊断/叙事行时引契词注意：行首窗内出现契词+支名仍会被误认——深引述写「契词表」字样替代裸词可避（本条目自身即活例：票文首行含支名+「施工收口」于 >40 位故安全）
notes: 契词表一行未动（候审回执|施工收口|完工推支候审|完工回执 钦定面不扩）；rc 分因 2/3/4 语义不动；非契词异形（交付@sha/落成@sha）依旧不参选=协议硬闸保分席可证不软化
---

## 断因实录（O-38132）

spec 前提「契词行 #37730 在案仍 rc3」经生产对拍**不成立**：#37730 实为 O-37448 回执（[施工收口:O-37448]），O-37450 唯一交付行 #38000 用「交付@239692f4d」非契词——**rc3 判词本来就正确**（契词行实不在案，fail-closed 按设计工作；后由外席代行合 #38235 解锁）。

真洞在另一面：旧谓词子串全位匹配使**元行**（诊断票 #38132 票文引契词表、会话收口叙事 #38135/#38213）充当作者回执——ORDER BY id DESC 取最新行时叙事行夺 aseat=唤知/处置席。后果方向=fail-open：复审席与真作者同席时分席证明失效。

## 修法

`review_ok`+落成自动销 EXISTS 同判词：回执=事件族自报（event~完工|候审|施工收口|工单完工|回执）∨行首窗契词（note~'^.{0,40}（契词族）'）+ORDER BY id ASC 首报工。桩测四锚：元行引契词不充回执/元行不夺真回执/取最早契词行/交付@sha 异形不参选——61 案全绿。
