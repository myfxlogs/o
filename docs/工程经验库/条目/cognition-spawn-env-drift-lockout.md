---
schema_version: 2
id: cognition-spawn-env-drift-lockout
kind: pitfall
risk: high
status: verified
keywords: [唤知机, dispatcher, systemd-run, 临时unit, env漂移, O_WORKER_DEVIN_BIN, 跨机锁, advisory lock, 唤起败, 锁饿死, RuntimeMaxSec, 单飞闸, cognition_spawn_fail, 处置遮蔽]
scope: [唤知环 spawn 面, scripts/cognition/dispatch.sh, scripts/cognition/run.sh, 跨机双活 dispatcher]
summary: 唤知环全环死案——四个机制叠加成「认知环死而全世界以为它活着」：①systemd-run 临时 unit 不继承 drop-in Environment（O_WORKER_DEVIN_BIN 漂）,run.sh devin 硬路径写死别机→discovery 败→同秒瞬退；②跨机单飞锁=ops_event 唤起行在案即占,释锁径只有「systemd-run spawn 落败」一处——run.sh 早退不写败迹→坏机占满 30min TTL 连败×4,好机每拍读到「跨机在飞」退让=饿死；③「唤知面死」探针判据=24h 无处置行——交互会话手写处置把它遮蔽（探针看产出不看「自主产出」）；④会话零消息节点零 transcript——死得无痕。修法=run.sh devin 自寻路(command -v 优先,$HOME/.local/bin 兜底,不写死主机径)+EXIT trap 统一写唤起败释锁(OK=RC=0∧transcript 非空;SIGKILL 残余由 30min TTL 兜底)+dispatch 传 COG_UNIT/RuntimeMaxSec=2h(挂死→TERM→trap 释锁,防占单飞闸永双会话)+cognition_spawn_fail 探针(唤起败 2h≥2=结构性升件)。
first_seen: 2026-11-10
last_verified: 2026-11-10
skills: []
evidence_refs:
  - type: server_fact
    source: SG(ant) journalctl -u o-cognition-* + ops_event 唤起行
    detail: 04:33/05:05/05:36/06:08 UTC 四场唤起,run.sh 同秒退
      「session discovery failed; refusing cold restart」;
      sessions.db 该会话 04:10 后零 message_nodes;
      ops_headless COG-20261110 state=stalled 戳=唤起行同秒
  - type: test_fact
    source: bash run.sh with O_WORKER_DEVIN_BIN=/nonexistent
    detail: trap 写 唤起败 unit=o-cognition-t-trap reason=session_discovery 落测库
---

# 唤知环全环死案——spawn env 漂移×锁不释×探针遮蔽×零痕

## 坑①：临时 unit env 不继承——硬路径写死别机=瞬退

`systemd-run --user` 起的临时 unit 是**全新环境**，不继承
`o-cognition-dispatch.service.d/*.conf` 里的 Environment（drop-in
只管 dispatcher 自身）。run.sh 原写死
`DEVIN_BIN=/home/mluser/.local/bin/devin`——SG 机无此径
（devin 在 `/root/.local/bin/devin`）→ `devin list` 败 →
`o_worker_session_state` 返 2 → 「session discovery failed;
refusing cold restart」同秒退。**env 传参只治标——根因是路径
不该写死**。

修法：run.sh 自寻路——
`O_WORKER_DEVIN_BIN` 覆写优先 → `command -v devin`(PATH) →
`$HOME/.local/bin/devin` 兜底。主机无关，dispatch 不再传 devin 径。

## 坑②：占锁不释——早退型坏机饿死好机

跨机单飞锁语义=「ops_event 唤起行在案(30min 窗)且无同 unit 唤起败」。
释锁径原来只有 dispatch 自己写「唤起败」一处,且只在
`systemd-run` **调用本身**落败时——run.sh 起来后任何早退
（quota/discovery/坏 runbook/devin 瞬退）都不写败迹。
败机每拍照常赢锁(抢锁先于跑码)→同秒退→锁占满 30min→下拍再赢。
好机每拍读「跨机在飞」退让——**连败次数=好机饿死时长/30min**。

修法：run.sh `trap EXIT` 统一释锁——`OK=1` 条件=RC=0∧transcript
非空（devin exit 0 但零消息的静默败也算败）,否则写
`唤起败 unit=$COG_UNIT reason=<因>`。COG_UNIT 由 dispatch
`--setenv` 传入。SIGKILL/宕机写不了=残余黑洞,由锁 30min TTL 兜底
——饿死上限从无限降为一拍级。

## 坑③：产出型健康探针被「谁的产出都行」遮蔽

`cognitionEscCheck`(唤知面死)判据=未决升件在案+24h 零处置行。
它不区分处置行出自**自主会话**还是**交互会话**——环死 2h+
期间交互处置照写,探针全程静默。**「环活着」≠「自主环活着」**。

修法：新探针 `cognition_spawn_fail`——`ops_event` 唤起败 2h 窗
≥2=结构性升件(单败=瞬态让位不扰)。败迹有了②的 trap 恒产,
探针消费同事实源,零新写径。

## 坑④：会话挂死=单飞闸永占+跨机双会话

无 RuntimeMaxSec 时 devin 挂死：本机单飞闸见 active 永退
（占死）,跨机 30min 锁过期后他机另起一场→**双会话双处置双账**
（正是跨机闸要防的形态,挂死从另一个入口进来）。

修法：`systemd-run -p RuntimeMaxSec=7200`——超时 SIGTERM→
EXIT trap 释锁(唤起败 reason=session_incomplete)→探针看得见。
2h 上限对「收口退」常态是宽裕上限非约束。

## 证据

- check: SG `journalctl --user -u o-cognition-1794290920`——
  `run.sh[296300]: session discovery failed` 与唤起行同秒；
  dev 机 `~/.local/state/devin/cli/sessions.db` message_nodes
  自 04:10 UTC 后零行直至人工复测
- result: 修后 run.sh trap 实测 `唤起败 reason=session_discovery`
  落账；`TestCognitionSpawnFailEsc` 单败不火/连败火/销后重火/
  复绿自闭全谱绿
- ref: `scripts/cognition/run.sh`(DEVIN_BIN 寻径+release_lock trap);
  `scripts/cognition/dispatch.sh`(COG_UNIT/RuntimeMaxSec);
  `cmd/core/meta_esc.go` cognitionSpawnFailEscCheck

## 反证条件

- check: systemd-run 临时 unit 改为继承 spawn 者 env(行为变更)——
  则坑①机理不复现但自寻路仍正解
- falsifies_when: ops_event 唤起/唤起败 语义重构(锁面换载)——
  则②③的释锁/探针契约须随同迁
