---
schema_version: 2
id: systemctl-is-active-inactive-substring-match
kind: pitfall
risk: high
status: verified
keywords: ["systemctl is-active", "inactive", "grep 子串误配", "漏拉闸", "o-worker", "T-0324"]
scope: [dm_tick.sh, systemd-run, o-worker-*, 清扫闸件]
summary: systemctl is-active 对不存在/已死 unit 输出 `inactive`（rc≠0）——字面含 `active` 子串，`grep 'active'` 恒匹配把死 unit 判活。闸件活性判定必须按行精确比对 `grep -qx 'active\|activating'` 或直出 `=~ ^(active|activating)$`，禁子串匹配。
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: server_fact
    ref: 2026-11-04 决策席实测 is-active o-worker-T-9999/t0324（不存在/已杀）→stdout "inactive" rc=4；`echo inactive|grep -q 'active\|activating'` 命中
  - type: doc
    ref: docs/prompts/T-0324-返修①-漏拉闸误配与陈旧补拉守卫.md
---

# systemctl is-active 的 "inactive" 含 "active" 子串——grep 活性判定恒误判

## 坑形

闸件写 `systemctl --user is-active "u.service" | grep -q 'active\|activating'`：
is-active 对**不存在/已死** unit 一律输出 `inactive`（rc=4/3）——该字面
第 3 字符起即 `active` 子串，grep 命中→`! grep` 为假→恒判「活」→补拉/
保活类闸永不触发，死锁静默化（比无闸更坏——看着有兜底实则死信）。

## 修法

活性判定按**整行精确**比对：

- `systemctl --user is-active "u" 2>/dev/null | grep -qx 'active\|activating'`，或
- bash 直出 `[[ "$(systemctl --user is-active u 2>/dev/null)" =~ ^(active|activating)$ ]]`。

凡 systemd 状态词做分支判据一律精确匹配（active/inactive/failed/activating
词面互含）。同类坑：grep 状态列、jq 字段子串比较。

## 案发

T-0324 收口件漏拉清扫闸（dm_tick 尾段）——决策席 2026-11-04 实测后判驳回。
