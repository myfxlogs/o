---
schema_version: 2
id: shouko-report-before-branch-push
kind: pitfall
risk: medium
status: verified
keywords: [收口, 移交, report.sh, git push, 半交付, 未推送, 施工席, 决策席, 待裁闸]
scope: [report.sh 里程碑自报, t/* 施工分支推送径, ops_event 收口/移交行, 决策席裁决面]
summary: 施工席先发收口/移交自报行（ops_event 落账=入待裁闸）再推分支——甚至整
  run 退出仍未推：裁决面被唤醒时远端支仍是旧 SHA，收口 note 声明的 tip 只存于
  worktree 本地 ref；collab §8「未推送=未交付」=半交付窗口非交付。
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: doc
    ref: docs/collab.md §8（分支已推送=已交付；未推送=未交付）
  - type: ops_event
    ref: "T-0353：收口 auto@b030bef2(02:19Z)→收口 tip 23214f54(02:25Z)→移交(02:25Z)，unit 退出后 origin/t/0353-middle-band-dcd 仍 2fd073b3"
---

## 事实/症状

T-0353 返修①施工 run（o-worker-T-0353，02:08–02:26Z）依次落 `收口
auto@b030bef2`、`收口 tip 23214f54`、`移交` 三行后 unit 退出
（Result=success），但 `git ls-remote origin t/0353-middle-band-dcd`
仍为旧收口支 `2fd073b3`——返修三记 commits（39b2396b/b030bef2/9ddf7631/
f973cd67/23214f54）全在 worktree 本地 ref 未上远端。待裁闸按 ops_event 水印
放票，决策席被唤醒面对的「收口支」在远端不存在。

## 根因或策略解释

`report.sh 收口/移交` 落 ops_event 行与分支推送是两个无顺序约束的动作——
施工协议把「推支」放在收口前但未机内化，run 在事件落账后即可体面退出。
ops_event 是唯一真源/待裁闸水印：收口行一旦落账票即入裁集，远端支状态不
参与闸判定。本次未断链纯靠 worktree 共享 .git 对象仓（本地 ref 可读可裁）
+决策席把未推支连同 merge 一并推远端兜回。

## 处置

- 施工席纪律（提示词层）：收口自报前 `git push origin
  HEAD:refs/heads/t/NNNN-slug` 先行——推送成功才发收口/移交行，顺序写进
  收口三步同拍。
- 裁决面纪律（本席已按此裁）：收口 note 声明 SHA 先 `git ls-remote` 对远端；
  未推≠拒因——对象在共享仓可裁可合（merge 即公示），但作为**偏离项**记入
  裁词留痕，同坑再现两票以上可升级立根修票。
- 根治候选（归决策裁）：dev_run.sh/worker_runtime.sh 退出闸加「收口行已落
  但远端支 ≠ 本地 tip 则拒退/补推」断言；或 report.sh 收口事件内嵌
  `git rev-parse @{u}` 对拍提示。

## 证据

- check: `git ls-remote origin t/0353-middle-band-dcd` → `2fd073b3`（移交后）；
  `git log --oneline origin/master..t/0353-middle-band-dcd` → 5 记本地未推。
- result: 决策席 merge 后 `git push origin t/0353-middle-band-dcd` 由裁决方代推
  → `23214f54` 上远端，交付闭环。
- ref: ops_event id 1951/1953/1954/1957/1958；merge commit 1156bad7。
