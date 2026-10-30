---
schema_version: 2
id: headless-dual-dispatch-path-no-mutex
kind: pitfall
risk: high
status: observed
keywords: [无头双发, devin-headless-run, 单票互斥, flock, 双派发径, mbus-dispatch, devin-relay, spawn, 让位, T-0255, T-0256]
scope: [meta调度面, /opt/meta/devin-headless-run.sh, /opt/meta/mbus-dispatch.sh, /opt/meta/devin-relay.sh]
summary: 同票两派发径并行（总线 directive + relay〔派单〕）各 spawn 一无头席且 wrapper 契约层零互斥——T-0255 双席同活 40s 实证，靠 agent 自辨让位才免 fin_* 钱径表双写；根修=wrapper 单票 flock+锁前生代 pgrep 兼容闸+让位协议成文（T-0256）
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: doc
    ref: /opt/meta/devin-headless-run.sh（flock+锁前生代兼容闸 @T-0256）
  - type: doc
    ref: /opt/meta/devin-relay.sh:185-189（〔派单〕spawn 径）
  - type: doc
    ref: /opt/meta/mbus-dispatch.sh:69,92-94（directive spawn 径+在飞闸）
  - type: doc
    ref: ~/devin-headless-T-0255.log（02:32:29/02:33:08 双 spawn 行+副本让位声明）
  - type: commit
    ref: 7cc45f4（meta 仓：互斥闸+让位/双轨协议成文）
---

## 事实/症状

2026-10-30 02:32:29 与 02:33:08，`devin-headless-run.sh T-0255` 被 spawn 两次——dispatch log 单 spawn 行 vs wrapper 共享 log 双 spawn 行实锤；两 devin -p 同票同活 40s，共用同一 prompt/log、零互斥。副本凭 agent 自辨（兄弟已建 worktree+领单 commit 1ee3018）写让位声明退 rc=0，方免 fin_known_residual/fin_ledger 双写。同源痕迹：Dev-97（T-0250 收口 report 重复件）。

## 根因或策略解释

双派发径并行无同票去重：①mbus-dispatch.sh directive 径（MAIL_NEW→route_directive→spawn_run）与②devin-relay.sh v5〔派单〕标记径（窗格标记→nohup wrapper）各自独立 spawn 同票副本；dispatcher 有 in_flight pgrep 闸但 relay 径无等价闸，且闸与 spawn 之间无原子性（TOCTOU）。wrapper 契约层任何径 spawn 直通 devin -p，无互斥原语——副本兜底全靠 agent 自觉（自觉是行为不是机制，漏一次=钱径双写）。

## 处置

T-0256 根修（钦定不碰 dispatcher/relay 结构去重——T-0254 界）：

- wrapper 起首单票 flock（/tmp/devin-headless-<TID>.lock，fd8）：持不到=副本让位（共享 log 留行+MANAGE 报+rc=0），锁随进程死自释——停电殁/崩溃重 spawn/驳回返修全不受阻；
- 锁前生代兼容闸：pgrep 锚 argv[0]/argv[2] 实锤（devin -p …-<TID>-prompt / 同票 wrapper 进程），覆盖无锁旧代在飞——实证=对 T-0256 自身在飞 devin 起同票副本秒让位；
- devin/hook 调用点 8>&-——子进程树不承锁 fd，防孙辈攥锁过本器亡（relay fd9 坑同款）；
- 让位协议+双轨纪律成文 /opt/meta/devin-headless-role.md（注入全无头 prompt）：三相核实（进程/席位/远端支）→让位声明+report.sh 异常+rc=0，副本零写 fin_*，驳回返修豁免。

## 证据

- check: `HEADLESS_LOGDIR=/tmp/dhr-test-logs DISPATCH_DRYRUN=1 DISPATCH_HOOKDIR=/tmp/dhr-test MBUS_BIN=/bin/true` 同 TID（T-9999）双 spawn——先 spawn+8s hook 持锁，2s 后副本
- result: 一活一让——log 行「09:59:36 spawn+DRYRUN 桩」/「09:59:38 让位：同票在飞（flock …被持）副本 rc=0」/「09:59:44 进程退出 rc=0」在案
- ref: /tmp/dhr-test-logs/devin-headless-T-9999.log
- check: 对 T-0256 本席在飞 devin（旧代 wrapper spawn、无锁）起同票副本 `bash /opt/meta/devin-headless-run.sh T-0256 <票面>`
- result: 副本秒让位「锁前生代在飞（devin -p …T-0256-prompt 实证）」rc=0；同票 devin -p 进程数仍=1（零新调用）；MANAGE 箱收让位件 20261030-095955-Dev-130
- ref: ~/devin-headless-T-0256.log 09:59:55 让位行；meta 仓 commit 7cc45f4

## 反证条件

- check: 持锁者进程死后同票重 spawn
- falsifies_when: 仍让位=锁误持（stale fd/孙辈攥锁——8>&- 未兜住或 flock 语义破）
- check: 无同票在飞实证时 spawn
- falsifies_when: 仍让位=兼容闸误判（pgrep 模式过宽匹配祖先/无关进程）

## 适用边界

互斥只保同机同票——/tmp 锁不跨机共享（现无跨机 spawn 径）；audit wrapper 不在闸面（审计席无 fin_* 写面，钦定界未含）；驳回返修成立前提=旧实例全退后才重 spawn（dispatcher in_flight 闸+票面驳回态路由保）；pgrep 兼容闸以 argv 锚定防 bash -c/手派祖先命令行内嵌同款串误配让位（桩测实证）。
