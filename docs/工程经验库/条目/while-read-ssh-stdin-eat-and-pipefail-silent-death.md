---
schema_version: 2
id: while-read-ssh-stdin-eat-pipefail-silent-death
title: "while read 循环内裸 ssh 吃光 stdin 清单 + pipefail 下 grep 无匹配静默斩脚本"
severity: high
status: active
domain: ops-gate
tags: [shell, ssh, pipefail, gate]
created: "2026-10-29"
last_verified: "2026-10-29"
related: [T-0223]
summary: "闸件内 while read <清单; do ssh ...; done 裸 ssh 继承 stdin 吞剩余行→循环首行即终=对拍静默失效；set -euo pipefail 下 grep 无匹配返 1 经管道传染→set -e 无声斩脚本——T-0223 闸双坑同件实证。"
evidence_refs:
  - type: commit
    ref: t/0223-bridge-audit
---
---

# while read 循环内裸 ssh 吃 stdin + pipefail 静默斩

## 实证锚（T-0223 bridge-audit 闸 2026-10-29）

- `sg_q() { ssh ... "docker exec psql ..." }` 包在 `while read ... done < "$TRG_LIST"` 内：ssh 默认继承 stdin，首轮把 TRG_LIST 剩余 17 行当远端输入吃光——循环只跑了第 1 条即终，存在性列「18 声明实查齐」绿字是纯谎报。
- `for m in migrations/*.sql; do grep -oE ... "$m" | sed ...` 下 `set -euo pipefail`：首个无匹配迁移件 grep 返 1 → pipefail 把 1 传出 → set -e 静默斩整闸，无任何红字。
- 桩测三红两列全绿假象定位根因后才曝此二坑——闸类件必须桩测红径，绿不能信。

## 修法钦定

- while-read 循环内一切 ssh 必带 `-n`（stdin←/dev/null）或 `< /dev/null` 显式喂。
- `grep | sed` 等「可合法零匹配」管道必尾 `|| true`。
- 闸件声明清单抽取后断言非空（空=抽取败 fail-closed 红）——防「清单零项=全绿」第三型静默。
