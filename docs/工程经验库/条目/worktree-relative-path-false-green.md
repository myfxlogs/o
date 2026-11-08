---
schema_version: 2
id: worktree-relative-path-false-green
kind: pitfall
risk: medium
status: verified
keywords: ["worktree", "相对径", "假绿", "cwd重置", "部署树", "施工树", "测试跑错树", "GW-001"]
scope: [多 worktree 并行施工, 桩测, shell 工作目录]
summary: 多 worktree 并行下 shell 会话 cwd 被重置回部署树（/opt/o）后跑相对径测试命令，静默跑的是部署树旧版测试+旧版被测件——改动未生效也全绿（假绿）。施工测试一律绝对径或先 pwd 自证在自身 worktree。
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: []
evidence_refs:
  - type: doc
    ref: docs/infra/GW-001-内循环机械轨-o-mech.md（段C 施工实证：新增⑨场景后跑测试无⑨输出——cwd 已被重置回 /opt/o，跑的是部署树无改动副本）
---

# 相对径测试命令在重置后的 cwd 上跑出假绿——worktree 施工必须自证树

## 病灶

施工 session 在 `/opt/o-wt/<票>` 干活，但 shell 每次调用间 cwd 会被重置回
启动目录（部署树 `/opt/o`）。此时 `bash scripts/meta/tests/x_test.sh`
这类**相对径命令**跑的是部署树副本：被测件是旧版、测试是旧版——刚加的
新场景（如 GW-001 段C mech_tick_test ⑨）不在其中，输出「全绿」纯属假绿，
改动的正确性零被证。

## 正形

- 测试/构建命令一律**绝对径**（`bash /opt/o-wt/<票>/scripts/...`），或
- 命令首行 `cd /opt/o-wt/<票> &&` 显式落树，或
- 跑前 `pwd`/`git branch --show-current` 自证在自己的树（§9 检出纪律自查同款习惯）。

## 反形

```bash
# cwd 已被重置回 /opt/o——下句静默测的是部署树：
bash scripts/meta/tests/mech_tick_test.sh   # 假绿
```

## 证据

- 2026-11-08 GW-001 段C 施工实证：worktree 内新增 mech_tick_test ⑨ 场景
  后跑测试无 ⑨ 输出且全 PASS——cwd 重置致跑 /opt/o 旧副本；绝对径重跑
  （worktree 副本）⑨ 正常出现并通过。
