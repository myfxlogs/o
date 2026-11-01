---
schema_version: 2
id: dbp2-cutover-file-ledger-fireq
kind: pitfall
risk: high
status: verified
keywords: [dlmd, deadline, fireq, meta_obligation, calendar, OnCalendar, 双账切换, 误报]
scope: [meta, mbus.sh, mbusd/dlmd, systemd timer, EDR 义务账]
summary: DBP2 全切只更 DB 行致 .fireq 文件账死径（timer fired 空转零命中）；日历账 FireAt=now+period 双段锚漂移（建账即点火+续期自锁偏移位）。
first_seen: 2026-11-01
last_verified: 2026-11-01
skills: []
evidence_refs:
  - type: commit
    ref: 1bab8cb
  - type: doc
    ref: /opt/meta/mbusd/cmd/dlmd/main.go
  - type: server_fact
    source: postgres
    query: SELECT ref,state FROM meta.meta_obligation WHERE kind='deadline'
    observed_at: 2026-11-01T15:52:00Z
---

## 病灶

1. **DBP2 全切误伤文件账**：`mbus.sh deadline fire` 全切后只 UPDATE meta_obligation 行——`.fireq` 标退役但 transient/edr-cal timer 仍调此径→全类死径（「fired」回显零命中不报错更毒）；文件账实靠 dlmd sweepDue 事件顺带点火。
2. **日历账锚漂移**：建账 `FireAt=now`（armed 即逾期秒点火→排期锚=dlmd 启动时刻）+续期 `FireAt=now+period_sec`（锚=上次点火时刻自我锁死+非定周期 spec 平 30d 错）。
3. **文案误报**：enforceFire 通用模板把日历周期提醒渲成「EDR 升级 L3」噪声喂升级链。
4. **esc 断链**：nid 直拼 subject 原文——全角/空格件建账『非法 id』静默吞（L2→L3 安全网失效）。

## 铁律

- 双账并存径（DB 行面+文件 .ob 面）切换必须保双侧写桥——只切一半=死径静默。
- 时点账锚=固定排期源（OnCalendar），非运行时戳；锚错=逐日漂移无人发现直到误报。
- 伴生坑：systemd-analyze 出行带 weekday+TZ 尾缀须剥离按本地时区解；env 剥 locale 时 sed 全角字节类匹配=字节截断（同族 T-0292 enum_base_of 案）。

## 回改点

`mbus.sh` cmd_deadline fire 段 .fireq 桥接（.ob 存在才落标）；`mbusd/cmd/dlmd/main.go`：nextCalElapse（systemd-analyze 钦定径）+建账首臂对正+armed 自愈 CALENDAR_ALIGN+onFireReq EARLY_FIREQ 早火闸+日历账不建 dlm-cal transient+enforceFire 日历分径+escalate idSlug 净化。
