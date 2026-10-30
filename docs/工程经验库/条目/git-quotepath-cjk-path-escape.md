---
schema_version: 2
id: git-quotepath-cjk-path-escape
kind: pitfall
risk: medium
status: verified
keywords: [git ls-tree, quotepath, 中文文件名, CJK路径, git show 读票, 收割脚本, T-0253]
scope: [bash, git-plumbing, docs/tasks, meta-eval]
summary: git ls-tree 默认 core.quotepath=true 把非 ASCII 路径八进制转义（"docs/tasks/T-0027-SG\346..md"）——拿转义串再 git show 静默查空，票面类中文名全灭；加 -c core.quotepath=false 出原生 UTF-8 名
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: ticket
    ref: T-0253
  - type: doc
    ref: /opt/meta/eval/bin/verdict_resolve.sh:38
---

## 事实/症状

收割链按票号回查正本票面：`git ls-tree --name-only origin/master docs/tasks/`
拿路径再 `git show <ref>:<path>`——ASCII 票全中、中文名票全空。无报错无
异常，静默零行。

## 根因或策略解释

`ls-tree`/`status`/`diff` 族默认 `core.quotepath=true`：非 ASCII 字节按
八进制转义并加引号（`"docs/tasks/T-0027-SG\346\255\243..md"`）。转义串
不是文件系统路径——`git show ref:"..."` 按字面串查 tree 对象，查无此名
返回空。判词归一第一版正此因全落副本兜底。

## 处置

- `git -c core.quotepath=false ls-tree ...`——一行级开关，出原生 UTF-8
  路径名，下游 `git show ref:$path` 直接可用；
- 或 `-z` 输出 NUL 分隔配合 `read -d ''`（含空格/特殊名更稳）。

## 证据

- check: `git ls-tree --name-only origin/master docs/tasks/ | grep T-0027`
- result: 输出 `"docs/tasks/T-0027-SG\346\255\243..md"`（转义带引号）；
  加 `-c core.quotepath=false` 后出 `docs/tasks/T-0027-SG正式营业位切流.md`，
  `git show` 命中票面字段。
- ref: T-0253 verdict_resolve.sh 正本查找径（f5287d4 前全落副本实证）

## 反证条件

- check: 设 `git config core.quotepath false` 于仓级
- falsifies_when: 仓级配置已关则无需行级开关（仍建议行级——脚本不可依赖
  调用环境配置）

## 适用边界

一切用 `ls-tree`/`ls-files`/`status --porcelain`/`diff --name-only` 产物再
回填 git/文件路径的脚本；O 仓 docs/tasks 票面全中文名=必中面。
