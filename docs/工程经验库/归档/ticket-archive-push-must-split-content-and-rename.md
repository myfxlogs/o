---
schema_version: 2
id: ticket-archive-push-must-split-content-and-rename
kind: pitfall
risk: medium
status: retired
keywords: [T-0219, check_ticket_numbers, done归档, pre-push, R100, head_archive_move, 票翻完成, git mv]
scope: [scripts/check_ticket_numbers.sh --remote, docs/tasks/done/, 决策席裁决推送径]
summary: 完成票「翻状态内容改+归档 git mv」塞同一推送区间必撞 T-0219 远端对账闸——head_archive_move 豁免只吃「远端顶层 blob 与 done/ 件逐字同文」的 R100 改名；内容有改即被判「票号远端已被他票占用」。须两推分离：先推翻后顶层件使远端同文，再推纯归档。
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: []
evidence_refs:
  - type: commit
    ref: T-0375/T-0376 裁决推送（两推分离实蹈：052ecc43→f56db1de、d17ad579→d104622a）
  - type: doc
    ref: scripts/check_ticket_numbers.sh（head_archive_move/staged_archive_move 豁免径）
symptom: |
  决策席裁完合 master，同本地链上含「票面翻完成改行」+「git mv 归
  done/」，一次 git push 即红：
  `FAIL(T-0219): 票号 T-XXXX 远端已被他票占用（双席取号竞态——让号重
  编）: docs/tasks/T-XXXX-....md`——报文只指远端占用，不提示「改名不
  纯」根因，按撞号处置方向是死路。
root_cause: |
  --remote 闸 new_paths=推送区间（merge-base..HEAD）ACR 面；done/ 路径
  命中后走 head_archive_move 豁免——`git diff -M base..HEAD` 查出 R 后
  比对 `base:旧径 blob == HEAD:done/ blob`。票面在支上被施工回填改过+
  席翻完成再改 → 双侧 blob 永异于远端旧版 → 豁免败 → 落 key 占用判红。
  R96%（同 commit 内改+改名）与同区间改+改名同病——豁免只认逐字同文。
fix: |
  两推分离（决策席完成票标准径）：
  ① merge+票翻完成（顶层径 M 不入 ACR）→check_all 戳→push——远端顶层
    件先到终态；
  ② git mv 归 done/（零改动 R100）→check_all 戳→push——mb..HEAD 纯改名
    同文，head_archive_move 豁免 NOTE 放行。
  本地须先 reset 到翻状态件推第一推（O_ALLOW_NONFF=1 闸内豁免），再
  ff 回归档件推第二推——闸锚本地 HEAD 扫区间，`push <sha>:master`
  逃不掉（HEAD 含归档件照判）。
prevention: |
  归档 commit 永不与「同票内容改」同推送区间；判红先想「改名不纯」
  非真撞号——`git diff --name-status -M origin/master HEAD` 亲查 R 后
  blob 对拍即确诊。完成票三件套顺序：翻状态件→纯改名件→（可选）
  handoff/他件，各自独立推送各持对 SHA 绿戳。
---
