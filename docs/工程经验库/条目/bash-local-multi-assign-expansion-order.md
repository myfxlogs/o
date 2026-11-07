---
schema_version: 2
id: bash-local-multi-assign-expansion-order
kind: pitfall
risk: high
status: verified
keywords: [bash, local, 单行多赋值, 展开序, 词展开, 动态域, flock, 锁文件名, 互斥失效]
scope: [bash 函数, scripts/*.sh, 锁/租约脚本]
summary: bash 函数里 `local a="$1" b="...$a..."` 单行多赋值时 b 的词展开先于 a 赋值生效——$a 沿动态域取调用方同名变量（未置=空）而非本函数新值；用于拼锁文件名=互斥静默失效（两进程锁到同一错名文件各拿各锁）
first_seen: 2026-11-07
---

## 病灶

```bash
_testdb_lease_try() {
  local port="$1" lock="$DIR/o-testdb-lease-$port.lock"   # ←雷
  exec 8>>"$lock"; flock -n 8 ...
}
```

`local` 是 builtin：**先把全部实参词展开完，再执行赋值**。`lock` 词里的
`$port` 展开发生在 `port="$1"` 生效前——沿 bash 动态域向上找同名变量：
调用方局部 `port` 已置=拿到旧值（外径碰巧对，更险）；未置=空
（锁名丢端口，本例 `/tmp/o-testdb-lease-.lock`）。

危害形态：**互斥静默失效**——两进程各自 flock 同一错名文件，各拿各锁
全成功，谁也没拦谁；无报错无异常，只在并发争用时暴露（单跑桩测全绿）。
T-0435 实证：池内径两跑同拿 5434/5435（桩测 L1 红）、排队径 7ms 穿透
holder（L2 红）；外部 DSN 径因调用方 port 已置而幸存——同件同型雷只炸
一半路径，恰是最难排查的形。

## 修法

local 声明**一行一个**（后行引用前行已生效的赋值）：

```bash
local port="$1"
local lock="$DIR/o-testdb-lease-$port.lock"   # 此时 $port 已是本函数值
```

## 实证

2026-11-07 T-0435 租约件桩测：L1 两跑同刻争池 PA=PB=5435（应互异）、
L2 holder 持锁期主跑 6ms 穿透；`bash -x` 追到
`local port=5434 lock=/tmp/lprobe/o-testdb-lease-.lock`（arg=5434 而
lock 名空端口）实锤。拆行后三轮全绿。最小复现：
`f() { local a="$1" b="[$a]"; }; f x` → b=`[]`（非 `[x]`）。

## 回改点

- 函数内 `local` 行出现「后变量引用前变量」即拆行——一行一个赋值。
- 锁文件/互斥键路径拼装处必查：拼错名=静默失效无报错，桩测必须含
  **并发争用场景**（单跑序列测不触发本雷）。
- 同型雷：`readonly`/`declare`/`export` 单行多赋值同序（词先展开后执行）。
