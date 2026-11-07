---
schema_version: 2
id: git-commit-dashdash-before-flags
kind: pitfall
risk: high
status: verified
keywords: ["git commit", "pathspec", "参数序", "ticketdraft", "物料化零成功", "静默炸"]
scope: ["tools/ticketdraft", "exec.Command 拼 git 调用"]
summary: "git commit -o -- file -m msg——-- 终结选项解析,其后 -m/消息体全被当 pathspec 报 did-not-match；ticketdraft 物料化自 T-0393 落地起每拍必炸零成功,fin_ticket_draft pending 积压 20 件无人察觉——工具链跑过≠跑通,静默 rc!=0 须接升件面实测"
first_seen: 2026-11-07
---

## 病灶

`exec.Command("git","commit","-o","--",rel,"-m",msg)`：`-o`(--only) 收
pathspec；`--` 之后选项解析终结，`rel`/`-m`/msg 三参数全成 pathspec——
git 报 `pathspec '-m' did not match any file(s)`。

## 遮蔽链

1. dm_tick 每拍尾窗调 ticketdraft → draft#1 即炸 exit 1 → dm_esc 该升
   但 pending 依旧堆——无人盯「物料化产出=0」水位；
2. 20 件 pending 攒了 1.5 天，探针还在持续产草案——生产面看似「候选
   管道在工作」实际物料化从来是死路；
3. 撞坑实证：人工跑同令复炸（写败@LIFECYCLE-HUNGER T-0443 半成品）。

## 对治

- 参数序钦定：`git commit -o -m msg -- rel`——选项全在 `--` 前；
- exec 拼命令一律「flags… -- paths…」序，pathspec 永居 `--` 后；
- 同族：工具「运行无错」断言须看产出事实（pending 数单调降），非
  仅 rc——升件面已有径但未接水位判。
