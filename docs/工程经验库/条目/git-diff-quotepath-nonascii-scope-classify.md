---
schema_version: 2
id: git-diff-quotepath-nonascii-scope-classify
kind: pitfall
risk: medium
status: verified
keywords: [git, quotepath, 非ascii, check_all, scope, ci, clone]
scope: [ci, check_all, git]
summary: 新 clone 默认 core.quotepath=true——git diff --name-only 把中文路径转义成引号八进制串，变更面分类器认不出 docs/* 前缀而误判 full；CI clone 必须 core.quotepath=off
first_seen: 2026-11-09
last_seen: 2026-11-09
evidence_refs:
  - type: commit
    ref: bfb041e66
---

## OCI check_all 甩离线：新 clone 的非 ASCII 路径转义会把定向谱误判成 full

- **实证**：提案「check_all 甩 OCI」第二次 e2e（sha=bfb041e66）OCI runner
  预判 `surface-full` 直接不跑（exit=75 note=surface-full）。变更面实际只有
  `docs/工程经验库/**` + `scripts/**`（全在定向谱允许集），但 OCI 工作
  clone 是 `git clone` 新建，未继承本仓 `core.quotepath=off`——`git diff
  --name-only` 输出 `"docs/\345\267\245.../x.md"`（带引号+八进制转义），
  `case` 前缀匹配 `docs/*` 全部落空→`HAS_OTHER=1`→full。
- **教训**：路径分类/白名单类判据依赖 git 输出**未转义**字面 path——
  凡是新建 clone/CI 容器/裸仓工作树，先 `git config core.quotepath off`
  或用 `git -c core.quotepath=off diff`；否则中文文件名（本仓 docs/
  工程经验库、skills 中文目录）会被当成「清单外件」。
- **处置**：provision 建工作 clone 时写 `core.quotepath off`；runner 预判
  面另用 `git -c core.quotepath=off` 作双保险；check_all 自身读同一 config，
  与部署树（已设 off）同径。桩测覆盖 D 面外不涉及路径转义——e2e 报告
  面已留 sha 证据。
