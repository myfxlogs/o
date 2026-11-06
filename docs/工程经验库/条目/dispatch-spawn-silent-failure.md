---
schema_version: 2
id: dispatch-spawn-silent-failure
kind: pitfall
risk: high
status: observed
keywords: [派单拉起未生, systemd-run, o-worker, 领单缺席, 调度静默, dispatch]
scope: [meta调度面, ops_event派单行, systemd-run spawn, 施工席拉起径]
summary: 派单 ops_event 行落账≠工人拉起——spawn 径开火即忘无活性回执，拉起静默失败时派单行成「谎报在飞」：票挂在飞队列无限等，唯人眼对拍 systemd/relay 才显形。2026-11-06 同日晚 T-0403 首派未拉起（DM 自报补派）、T-0405 派单后零领单零单元（~1h 后人工对拍逮住）。
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: [决策席, 施工席]
evidence_refs:
  - type: server_fact
    source: postgres
    query: relay/ops_event 派单行↔领单行对拍（T-0405 派单在册而领单缺席）
    observed_at: 2026-11-06T11:50:00Z
---

## 事实/症状

- T-0403：DM 首派后工人未拉起，DM 自报「补派（前轮派单未拉起）」——
  首见即实锤 spawn 径可静默败；
- T-0404：8 秒内连派两次（19:35:46/19:35:54）——疑似同款未拉起
  重派；
- T-0405：18:59 派单行在册，其后 relay 零 `领单`、systemd 零
  `o-worker-T-0405` 单元——**在飞假象挂 ~1h**，业主人工
  对拍清单才发现；
- 对照组 T-0409：派单→拉起正常（systemd 单元+领单行双实证），
  证 spawn 径非恒败是间歇败；
- 检测面边界实证（同日 20:57）：T-0404 worker **拉起后静默死**
  （unit 消失+无收口）被 DM 自检出并重派——证 DM 检测件覆盖
  「起了后死」形，**不覆盖「从未拉起」形**——后者正是本坑，
  缺口=派单↔领单 SLA 对账（起后死有 unit 曾存在可查，
  未生连参照物都没有，只能事件对账）。

## 根因或策略解释

`scripts/meta/dispatch_worker.sh`（或 DM 内嵌派单径）=派单行写账
+systemd-run spawn 两步分离：第一步 ops_event 落「派单」即视为
在飞，第二步 spawn 成败无回执——**开火即忘**。与在库
`absence-detector-missing-report-rows` 同族同形：「有信号才动」
的机制件看不见信号缺席（收口段缺席→派单段也缺席）——缺席发现
器的覆盖面没罩到调度起手式。

## 处置

- 探针形：ops_event 派单行↔领单行对账——派单后 SLA 窗（建议
  5min，施工拉起 bootstrap 在窗内）无领单行=升件「拉起未生」
  附派单证据行；
- 或 spawn 径加活性回执：systemd-run --wait/拉起后心跳首行
  反哺派单行状态；
- 本条目随下个调度可靠性票消——票号不另立（业主令
  2026-11-06「记进去不浪费票号」）。

## 证据

- check: `systemctl --user list-units | grep o-worker-T-0405`+
  `grep 领单.*T-0405 ~/devin-relay.log`
- result: 单元缺席+领单行缺席，派单行在册（双缺一在=静默在飞）
- ref: relay 2026-11-06 18:59 派单行/19:36 0403 领单（补派后生）

## 反证条件

- check: 重读 o-worker spawn 日志/journal 找 T-0405 spawn 尝试
- falsifies_when: 证明 spawn 其实起了但领单写败——则病灶挪到
  领单写径非 spawn 径（同形不同位，条目照改）

## 适用边界

- 面界=DM 常径机派（systemd-run o-worker-<tid>）——交互席手动
  拉起/外派径（如 T-0409 外派刀）不在此坑面；
- 间歇败非恒败——可能 spawn 依赖（devin CLI 可用性/模型配额/
  wt 竞态）条件触发，根因未坐实只证「静默」成立。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a
- 豁免由：meta 调度面坑非产线数据面——候选量表=ops_event
  派单↔领单行对账差（派单行 count−领单行 count>0 且超 SLA
  窗=升件），待入表时随消坑票补建。
