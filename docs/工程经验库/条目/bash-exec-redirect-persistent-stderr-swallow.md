---
schema_version: 2
id: bash-exec-redirect-persistent-stderr-swallow
kind: pitfall
risk: high
status: verified
keywords: [bash, exec, 重定向持久化, stderr 吞没, 静默死, "2>/dev/null", flock 锁函数]
scope: [bash 函数, scripts/*.sh, 锁/租约脚本, 长跑闸脚本]
summary: exec 8>>file 2>/dev/null 的全部重定向都持久落 shell——本意压开文件报错，实际把调用方 stderr 永久重定向 /dev/null，之后整跑的 warn/fail/测试输出全哑（红也静默 rc1 无任何痕）；exec 行禁搭任何 2>/dev/null
first_seen: 2026-11-07
---

## 病灶

```bash
_testdb_lease_try() {
  if ! exec 8>>"$lock" 2>/dev/null; then return 2; fi   # ←雷
  ...
}
```

`exec` 纯重定向=**落本 shell 永久生效**（fd8 持锁是想要的，fd2 也一并被
永久指到 /dev/null——不是命令级临时重定向）。之后：
- 脚本所有 stderr 输出（warn/❌/step 行）静默消失；
- `fail` 仍 exit 1 但**零报错痕**=静默死；
- `2>&1` 管道收方（tee→stderr）收到的也是 /dev/null 重定向后的空流；
- `bash -x` trace 也走 fd2 → xtrace 在 exec 行断尾，死亡点后移难追。

T-0435 实证：check_all 租约点着后整跑哑（🔑 行后零输出），go test 红的
❌ 被吞=静默 rc1 无戳无报错，三轮复跑同症；`bash -x` 追到 `+ exec` 断尾。

同型雷：`exec 8>&- 2>/dev/null`（关 fd 永不败，但 2>/dev/null 照样持久
吞 stderr）——本票同件两处同清。

## 修法

- exec 重定向行**禁搭任何 2>/dev/null/2>file**——开文件败让 bash 自报
  （fail-closed 可观测），成功路径零副作用；
- 需要压错的 child 命令（flock/mktemp 等）随便搭——那才是命令级重定向；
- 真 要临时换 stderr 用子壳 `( exec ... ) ` 或先存后还
  `{ exec 2>&3-; } 3>&2` 显式存取。

## 实证

2026-11-07 T-0435：check_all 三连跑静默 rc1；xtrace 断 `+ exec`；
去 2>/dev/null 后 🔑 行、go test 行、✅ 行全数复现可观测。

## 回改点

- 见到 `exec [fd]>…` 行尾带 `2>` 一律雷：重定向持久化不随命令结束。
- 静默死排查首查：脚本输出中断点之前的最后一条 exec 有无搭 2>。
- 关联：pipefail-grep-q-early-exit-false-negative（同「闸脚本静默化」族）。
