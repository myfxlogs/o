---
schema_version: 2
id: merge-conflict-markers-committed
kind: pitfall
risk: medium
status: canon
keywords: [git, merge, 冲突标记, 流水线, 提交]
scope: [engineering, workflow]
summary: merge 冲突后脚本化 add+commit 会把 <<<<<<< 标记原文提交——add≠解冲突，提交前必 grep 标记清零
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: []
---

# 合并冲突标记被一并提交

## 现象
`git merge` 票档冲突 → 脚本化编辑+`git add`+`git commit` 流水线未先解冲突——`<<<<<<</=======/>>>>>>>` 标记原文进合并提交（O 实证 b3e4445，T-0113 票档；同 session T-0114 合入差点重演）。

## 根因
`git add` 冲突件=标记「已解」（git 不查内容）；合并态下 commit 直接生成含标记的合并提交。脚本流水线把「编辑文件」当「冲突已解」——两回事。

## 防法
- 合并冲突后、add 前必跑 `grep -n '<<<<<<<\|>>>>>>>' <file>`（空=可提）。
- 机械口径：**merge 非 ff 失败 → 下一步永远是人工解冲突，不是跑写文件脚本**。
- 验收级严重度：中——标记进档不炸构建，但票档=审计面，脏档要追加修复提交压平（历史不可改）。

实证：T-0113/T-0114 验收合入（2026-10-25）。
