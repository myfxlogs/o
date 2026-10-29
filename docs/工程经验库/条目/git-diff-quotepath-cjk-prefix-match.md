---
schema_version: 2
id: git-diff-quotepath-cjk-prefix-match
kind: pitfall
risk: medium
status: verified
keywords: [quotepath, 中文文件名, git diff, C-quote, 路径匹配, 白名单前缀, CI 路由]
scope: [CI workflow, git 路径枚举, 文件类判定, 本仓全谱中文名 docs]
summary: git diff/status --name-only 默认对非 ASCII 路径 C-quote（"docs/...\345..."）——拿输出做白名单前缀/路径匹配全灭；凡枚举 git 路径做判定必须 -c core.quotepath=false（或 -z）
first_seen: 2026-10-29
last_verified: 2026-10-29
skills: []
evidence_refs:
  - type: code_fact
    ref: T-0225 ci.yml route 步；scratch repo 实测 git diff --name-only 输出 "docs/tasks/T-9999-\344\270\255\346\226\207\345\220\215\347\245\250.md"（带引号前缀），加 -c core.quotepath=false 后裸名 docs/tasks/T-9999-中文名票.md 匹配 docs/*
  - type: code_fact
    ref: 既有 codify：scripts/hooks/pre-commit 全谱 `git -c core.quotepath=off`（D-077 gofmt 闸注释「中文路径 quoted 形态破匹配」）
---

# git 路径枚举遇中文名=C-quote 破匹配（quotepath 坑·新面复发）

## 病灶

- `git diff --name-only`/`git status` 默认 `core.quotepath=true`：非 ASCII
  字节路径整体 C-quote 成 `"docs/tasks/T-9999-\345\217\210....md"`——
  字面以 `"` 开头，任何 `docs/*`、`*.md` 式前缀/后缀匹配全灭。
- 本仓 docs/任务单/经验库几乎全中文名——T-0225 CI 快径路由若漏此参，
  docs-only diff 永远判成越界=快径形同虚设（fail-safe 方向，不炸但
  功能死透、无报错可察——最毒形态：「跑了，只是永远不生效」）。
- 同坑已在 pre-commit 各闸 codify（`core.quotepath=off`），属已知坑
  在新面（CI workflow、以及将来任何 shell 里枚举 git 路径做判定的
  地方）的复发——钩子注释里的局部知识不等于可检索条目。

## 裁决

- **凡 shell 里 `git diff/status --name-only` 类输出接路径判定**（前缀
  白名单、后缀过滤、grep 文件名），一律 `git -c core.quotepath=false`
  ——写在取数那一步，不靠下游兼容。
- 需要免疫文件名内换行/引号的极端面用 `-z` NUL 分隔（注意 bash
  `$()` 吞 NUL，须 `< <(git diff -z)` 进程替换接 `read -d ''`）。

## 判据

- 写任何「git 枚举→模式匹配」管线时强制自问：中文名文件经此管线
  会判成什么？本仓中文名是常态非边角——默认就是破匹配。
- 测试夹具必须含一个中文路径用例（T-0225 实测夹具：`docs/tasks/
  T-9999-中文名票.md` → 白名单命中），否则坑测不出来。
