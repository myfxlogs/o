---
id: probe-predicate-null-anchor-seat-scope
title: 探针判据三坑形——NULL 空态越界/活性锚混他席/引证面半截机读
date: 2026-11-10
tags: [probe, ops_event, esc_class, audit]
related: [cognition-spawn-env-drift-lockout, runbook-write-path-needs-grant]
---
# 探针判据三坑形（升件#9572-#9577 同批根修）

探测薄壳「只述事实」面的三个反复坑形，皆「探针活着但语义说谎」：

## ① NULL 只在「臂未装」时是合法空态
`max(ts) IS NULL` 读作「从未产出」=臂未装的合法态；一旦**装臂证**
（唤起行/注册行）在案，同一 NULL 语义翻转为「装臂后从未产出」=死。
audit_stall 原判据 `last!=nil && age>48h` 对装臂臂永绿=审计臂死而
信号面无感知。修法=NULL 分支回退查装臂证，装臂证陈超阈即死。

## ② 活性锚必须是本环自主产出（seat 限定）
「24h 无处置行=认知环死」的锚若不限 seat，他席（复审/CEO/交互会话）
处置行重置死钟=唤知全死而他席活=永不报——同
cognition-spawn-env-drift-lockout 坑③「看产出不看自主产出」死法。
修法=活性锚加 `AND seat='处置席'`（认知环处置座）。

## ③ 引证面半截机读=批量销案漏销
「未销才抑」谓词只认 `升件#<id>` 单 id 形——区间引证「升件#a-b」
只销首件，批内余件滞留未决重报重处置（#9275→#9285 产线实证）。
修法=谓词扩认 `升件#(\d+)-(\d+)` 区间（`AS m(mm)` 别名取数组元），
消费面四处同步（meta_esc/feed_esc×2/ledger run）+runbook 扫描式
同形——同一语义谓词的每一处消费点都要改，漏改=双口径漂移。

## 判词
探针代码评审三查：NULL 空态语义域是否随装臂态翻转？活性锚是否
限自主产出 seat？引证/枚举面是否机读全覆盖（含区间/注释枚举/册档）。
