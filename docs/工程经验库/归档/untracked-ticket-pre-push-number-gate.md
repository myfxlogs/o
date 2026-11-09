---
schema_version: 2
id: untracked-ticket-pre-push-number-gate
kind: pitfall
risk: medium
status: retired
keywords: [票未入库, pre-push 票号闸, 领单翻单, 业主令直批, 票号 ≤ 远端最大号, 票面回填]
scope: [流程机器, docs/tasks 队列, pre-push T-0219 对账闸, 施工领单径]
summary: 业主令直批/机派票常未 commit 入 master（T-0419/0420/0422/0425/0426 在案）——施工把票件 add 上分支 push 即被 T-0219 闸拦「号≤远端最大号」；正形=票件留 /opt/o 未跟踪面，锁单靠分支名对上领取行，状态翻/结果回填直接写 /opt/o 票件本体
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [施工席]
evidence_refs:
  - type: commit
    ref: T-0420
  - type: doc
    ref: scripts/check_ticket_numbers.sh
  - type: doc
    ref: docs/collab.md §3
  - type: server_fact
    ref: 11-07 亲证——T-0420 票 add 分支 push 被拒「0420 ≤ 远端最大号 T-0424」；git ls-files 证 0419/0420/0422/0425/0426 五票 untracked 唯 /opt/o 工作树在案；T-0419 票结果回填段已落 /opt/o 票件本体（先例同径）
repro_or_symptom: 施工按 collab §3「分支上翻单 commit+push 锁单」把 /opt/o 里的未入库票 add 进 t/NNNN-* 分支→pre-push 票号对账闸 FAIL「新立票号 ≤ 远端最大号——本地视野陈，重取 next_ticket.sh」
root_cause: 闸语义假设「docs/tasks/ 新文件=新立票须撞号检查」，但业主令直批/COS 手写票绕开 next_ticket.sh commit 步——票号合法而文件未入库；施工无权重取号也无权改票本体，add 即撞闸
fix_and_prevention: ①票件不上分支——撤 add，分支只载实现；②锁单语义降级为「远端分支名=票面领取行」（领取行机派时已填即锁）；③状态翻（已批→施工中→待验）与结果回填直接写 /opt/o 票件本体（未跟踪件非部署树 D-651 台账面——T-0419 先例在案）；④收口 report.sh 收口+移交照走
notes: 与 ticket-number-arbiter-already-exists 条目同源异面——彼=立机制前不查存量；本=票面生命周期文件态缺口（闸与流程假设错位）。根治归 DM 机械轨（dm_ledger 票翻时顺手 commit 票件）非施工域
---

## 现象

`/opt/o` 工作树 `git status` 见 docs/tasks/T-04NN-*.md `??` 未跟踪
（业主令直批/他席手写票未经 next_ticket.sh 提交）。施工领单后把票
add 上施工分支推送——pre-push 票号闸以「新立票号 ≤ 远端最大号」
硬拒。

## 根因或策略解释

T-0219 闸的撞号语义假设「新增票文件=新立票」，对未入库存量票误判
为陈号。施工面三禁（不改票本体）+无权替票取号——票文件 commit 径
对施工物理关闭。

## 处置

- 分支不载票件：实现面 commit 独立推（票号闸「无新立票=免对账」放行）。
- 锁单=远端分支存在性对上票面领取行（机派已预填分支名即锁在案）。
- 票面状态翻+结果回填写 `/opt/o/docs/tasks/T-NNNN-*.md` 本体——未跟踪
  件不属部署树台账禁写面（T-0419 回填先例在案，收口三步同拍照走）。

## 证据

- check: `git push` 载票提交 vs 撤票裸分支
- result: 前者 FAIL(T-0219) 拒推；后者推送放行
- check: `git ls-files docs/tasks/ | grep -E "0419|0420|0422|0425|0426"`
- result: 零命中——五票唯 /opt/o 工作树在案；T-0419 票本体含完整结果
  回填段（先例实证同径可行）
- ref: T-0420 领单实证

## 反证条件

- check: DM 机械轨（dm_ledger 票翻）开始把票件 commit 入 master
- falsifies_when: 新票常态入库后施工按 collab §3 分支翻单 push 不被闸
  拦——彼时本坑失效，条目可退

## 适用边界

票件 untracked 且号 ≤ 远端最大号时适用；票已入库的常径领单不适用
（分支翻单照旧）。票号重取/改票本体仍禁——坑在文件态不在号权。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：流程机器面坑非产线数据面——无适配
  gauges.tsv 表；复发显形=pre-push T-0219 拒流水+`git status` ??
  票件人工抽阅）
