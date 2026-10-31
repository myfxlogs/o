---
schema_version: 2
id: bash-exit-trap-early-coverage-setu
kind: pitfall
risk: medium
status: verified
keywords: [bash, EXIT-trap, set -u, set -e, 早期die, 释锁, 完工投件, 退出码保真, deploy_image]
scope: [带 die/闸的运维脚本加 EXIT trap 义务（释锁/事件投件/清理）, 任何 trap 必须覆盖早期失败径的脚本改造]
summary: 给带「闸+die」的脚本加 EXIT trap 义务时，trap 若设在闸后，闸拒径（最早 die 点）零覆盖；trap 提前到闸前则连锁三个雷——①set -u 下 trap 内引用未初始化变量（如 LOCK_HELD/TOKEN）即 unbound 错，退出码可能被玷污；②被 trap 调的函数若定义在 die 点之后，闸拒时函数未定义=command not found；③trap 内二次捕获 rc（`local rc=$?`）会被 trap 内前序命令污染。修法=trap 设在第一个 die 之前；函数体内变量全用 `${VAR:-}`/前置初始化防御；rc 由 trap 行 `$?` 展开成参数传入（`trap 'fn $?; cleanup' EXIT`），不赌 bash 退出码保真语义；义务函数配隔离 stub 四拍自证（成功/闸拒/投件败静默/锁撞）。
first_seen: 2026-10-31
last_verified: 2026-10-31
repeat: 1
source: DEPLOYNF deploy_image.sh 完工投件——trap 需覆盖出处闸拒径（闸 die 在 SHA/TOKEN/trap 原位之前），stub 自证四拍全绿后固化
---

## 现象

`scripts/deploy_image.sh` 原结构：出处闸（die 最早点）在 21 行，`SHA`/`TOKEN`/`LOCK_HELD`/`release_lock`/`trap release_lock EXIT` 全在闸后。加完工投件时若沿用「trap 原位扩一行」，出处闸拒（脏树/未并码）路径 trap 根本未设——被闸也零投件，违反业主令「事事有应答」。

## 雷点（trap 提前后连锁暴露）

1. **set -u**：闸拒时 trap 触发，`release_lock` 读 `LOCK_HELD`/`TOKEN` 未初始化 → unbound variable 错。
2. **函数定义时序**：bash 函数按行序生效；die 在函数定义行前 → 闸拒时 trap 调未定义函数。
3. **rc 捕获污染**：trap 函数内 `local rc=$?` 必须是首语句，任何前移即捕到别的东西；改传参 `$?` 在 trap 行展开最稳。
4. **退出码保真**：trap 内命令失败是否会改脚本最终 rc 属 bash 语义细节——实测 die exit 1 + trap 内投件/释锁后 rc 仍 1（拍2 实证），但设计上不应赌，义务函数全路径自返 0（`|| true`/`|| echo` 收尾）。

## 修法（DEPLOYNF 落地形态）

- `release_lock`/`deploy_event` 定义 + `LOCK_HELD=0` 初始化 + `trap 'deploy_event $?; release_lock' EXIT` 全部提到第一个 die 之前（⓪ 退场义务段）。
- `deploy_event` 内 sha 惰性回填 `${SHA:-$(git rev-parse …|| echo unknown)}`——闸拒时 SHA 未赋值也能报对 commit。
- mbus 投件 `timeout 15`+`|| echo 投件败` 收尾：回执非命门，部署 rc 永不受投件影响。

## 判据

隔离 stub（fake docker/ssh + EDR_TEST 隔离 mbus 根）四拍：成功 rc=0+事件 rc=0 / 闸拒 rc=1+事件 rc=1（exit 码不被 trap 玷污）/ 投件败部署仍 rc=0 / 锁撞 rc=1+事件 rc=1。缺任一拍=改造不完整。
