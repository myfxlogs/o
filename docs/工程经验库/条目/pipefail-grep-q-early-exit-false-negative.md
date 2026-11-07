---
schema_version: 2
id: pipefail-grep-q-early-exit-false-negative
kind: pitfall
risk: medium
status: verified
keywords: [pipefail, grep -q, SIGPIPE, 假阴性, bash 闸脚本, 早退管, pre-push]
scope: [scripts/*.sh, bash 门禁/闸脚本]
summary: set -o pipefail 下 `produce | grep -q` 是假阴性地雷——grep -q 首个命中即退，生产者遭 SIGPIPE(141)，pipefail 传非零 → if 判假：豁免永不触发（本文实证）/ 拦截判可被绕过（反向更险）
first_seen: 2026-11-07
---

## 病灶

`set -o pipefail` 的脚本里写 `cmd | grep -qE "pat"`：
- `grep -q` 首个命中**立即退出**，不再读输入；
- 上游 `cmd` 后续写入撞 SIGPIPE → 退出码 141；
- pipefail 取管道内**最后非零** → 整个 if 判假。

危害双向都有：
- **豁免判**假阴性→该放行的被拦（本条目实证：T-0219 补档豁免写好仍 FAIL）；
- **拦截判**假阴性→该拦的可能被放行（比前者更危险——闸变成概率件）。

## 修法（三选一，按场景）

1. `grep -q pat <<< "$VAR"`（herestring）——无管道无 SIGPIPE，首选；
2. `cmd | grep pat >/dev/null`（去 q）——grep 吞全输入，上游善终；
3. `if cmd | grep -c pat | grep -qv '^0$'`——数匹配。

注：`grep -c`/无 q 形天然安全——它必须读完全部输入。

## 实证

2026-11-07 T-0219 闸修案：`git log | grep -qE "T-0*0219"` 手测命中、
脚本内恒空——bash -x 追到 grep 行后直接跳 on_branch 段；
最小复现 `set -o pipefail; printf %s "$big" | grep -q x` → miss。
同文件另三处存量 `| grep -qx`/`| grep -Fxq` 同型雷同 commit 三清。

## 回改点

写 `| grep -q` 前先想 pipefail：上游会遭 SIGPIPE 吗？会→换 herestring 或去 q。
审阅 bash 闸脚本见到 `| grep -q` 一律按嫌疑件看。
