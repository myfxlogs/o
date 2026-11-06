---
schema_version: 2
id: shell-pipe-rc-head-always-zero
kind: pitfall
risk: medium
status: verified
keywords: [shell, 管道, rc, head, compgen, 存在性判定, dm_ledger, 台账写面]
scope: [scripts/meta, ops_event]
summary: "cmd|head 管道 rc 取末段恒 0——存在性判定假成立；台账写面须捕获输出验串非空，键格式+终态词负表纵深防御"
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [施工席]
---

# shell 管道 rc 恒真坑——`compgen|head` 存在性判定失效事故（T-0414）

## 病灶

`df=$(compgen -G "$dir/$pat" | head -1); [ -n "$df" ]` 才是存在性正判；
`compgen -G ... | head -1 >/dev/null || continue` 是假判——管道 rc 取
末段 `head`，head 对空输入仍 exit 0，恒真=存在性检查失效。T-0414
dm_ledger B 面因此越过 done/ 铁证，给无档键/活票误落 35 完成行。

## 判据

- 存在性：捕获输出验串非空 `[ -n "$(compgen -G ...)" ]`，或 compgen
  直接 `>/dev/null` 不接管道（compgen 自身 rc 即存在性）。
- 同理坑：一切 `cmd | head/tail` 当 rc 哨兵的写法。
- 防御纵深：台账写面前白名单键格式（T-NNNN）+终态词负表（作废/
  废弃/取消不翻）。

## 关联

- T-0414（事故全披露在票面）；ops_event id 2887-2924 段作废声明行。
