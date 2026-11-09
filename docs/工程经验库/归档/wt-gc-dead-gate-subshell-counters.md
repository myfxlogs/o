---
schema_version: 2
id: wt-gc-dead-gate-subshell-counters
kind: pitfall
risk: high
status: retired
keywords: [wt_gc, systemctl is-active, o-worker unit 名, 闸恒死, 管道子壳, bash while read 计数器, worktree prune]
scope: [scripts/meta, bash, systemd --user, worktree GC]
summary: 安全闸写成死件双坑——①unit 名方言错位（basename 小写无连字符 vs o-worker-T-NNNN）is-active 恒 inactive 闸形同虚设；②管道 `|while read` 子壳吞计数器变量永远 0。修法：list-units 拍在飞 TID 集+归一 T-%04d 比对；进程替换 < <(...) 主壳计数
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: [施工席]
evidence_refs:
  - type: doc
    ref: scripts/meta/wt_gc.sh
  - type: server_fact
    ref: 11-04 亲证——is-active o-worker-t0330=inactive 而 o-worker-T-0330=active；干跑 SKIP 活run 命中 T-0331/T-0335
  - type: review_signature
    ref: T-0331 返修①驳回（决策席无头拍 2026-11-04）
repro_or_symptom: 钦定「unit active 禁碰活 run 工树」双闸实为死件——tid=basename|tr 小写得 `t0330`，真实 unit=`o-worker-T-0330`（大小写+连字符双错位），is-active 恒 inactive，在飞 run 工树可被 --force 误摘；计数器 rm_tree/keep/skip 声明后恒 0（管道子壳变量不回流主壳）
root_cause: ①systemd unit 名 `o-worker-T-NNNN.service` 与工树目录名方言（t0330/audit-t-0001/t-0226/t0079-v2）无双向归一即比对；②`producer | while read` 中 while 体在子壳执行，变量自增不回流父壳——bash 经典坑
fix_and_prevention: ①闸必实查在飞集：`systemctl --user list-units 'o-worker-*' --state=active,activating --plain --no-legend` 剥前缀/后缀得 T-NNNN 集，工树名取首个数字段归一 `T-%04d` 后 grep -Fxq 精确比对；②消费端循环用 `while ... done < <(producer)` 进程替换保主壳计数；③删败行显式 echo（branch -d 挂树拒删非漏记）；④写「会动的闸」后必须亲跑实证 SKIP 行——不实证=默认死件嫌疑
notes: 同族案 T-0329 漏拉闸双盲根修；「闸活着」是运行时断言非静态读码可证——恒 false 的闸比没闸更毒（给人已护错觉）
---

## 现象

`wt_gc.sh` 钦定双闸之 unit-active 闸恒死：`is-active o-worker-t0330` 永远 inactive（真实 unit=`o-worker-T-0330`），在飞 run 工树裸奔于 `--force` 摘除面；分类计数变量 `rm_tree/rm_branch/keep/skip` 全恒 0，GC 行 note 零计数。

## 根因或策略解释

- **名方言双错位**：工树目录方言 `t0330` 经 `basename|tr A-Z a-z` 不变即拼 `o-worker-$tid`——unit 实名 `o-worker-T-0330`（大写 T+连字符），双错位致恒查无此 unit。
- **管道子壳**：`git worktree list | while read` 的 while 体运行于子壳，`x=$((x+1))` 全部丢在子壳，父壳读回仍是初值 0。

## 处置

①在飞集快照比对：`list-units 'o-worker-*' --state=active,activating` 剥 `o-worker-`/`.service` 得 LIVE TID 集，工树名 `tid_norm`（首个数字段→`T-%04d`）后 `grep -Fxq`；②`while ... done < <(...)` 进程替换；③实跑删除成败分别计数+败行显式 echo；④闸类改动收口必附亲跑 SKIP 实证行。

## 证据

- check: `systemctl --user is-active o-worker-t0330.service` vs `o-worker-T-0330.service`
- result: 前者恒 inactive，后者 active（同机同时刻）
- check: `wt_gc.sh --dry-run` 于 o-worker-T-0331/T-0335 在飞时
- result: `SKIP 活run: /opt/o-wt/t0331 (unit o-worker-T-0331)` 等 2 行，ops_event id 1696 note 分类计数齐
- ref: T-0331 返修①收口 commit（t/0331-wt-gc）

## 反证条件

- check: 比对口径换模糊匹配（子串/正则）是否等价
- falsifies_when: `man3310`/`msig1030` 等含数字但非 TID 语义的树名若被证必活配，则精确匹配口径需收紧（当前宁 conservative 误跳不误摘）

## 适用边界

- systemd --user unit 命名约定 `o-worker-T-NNNN.service`（T-0324 瞬态 run 建制）；
- 工树名方言集合以 /opt/o-wt 实测为准，新方言（无数字段/多数字段）出现需复核归一规则；
- LIVE 集为起器时快照——扫行中新起的 run 不在集内（窗口钦定可接受）。
