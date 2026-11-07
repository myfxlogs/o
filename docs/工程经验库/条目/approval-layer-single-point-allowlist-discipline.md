---
schema_version: 2
id: approval-layer-single-point-allowlist-discipline
kind: pitfall
risk: high
status: verified
keywords: [claude-code, 审批层, allowlist, glm 1301, Bash 全冻, rtk hook, 宽配整改, deny 优先, PreToolUse, 精确前缀]
scope: [/opt/o/.claude/settings.local.json, CEO 终端审批链, rtk hook claude, 权限规则语义]
summary: 交互席 Bash 审批链有 glm 依赖点（1301/抖动→Bash 全冻）——解=allowlist 热径全表面化成纯前缀规则（机械 glob 匹配不经模型）+危险面留弃权径；实测 deny 规则先于 rtk hook-allow 生效（force push 硬闸与 hook 并存）；宽配（中置 * / 裸 Bash(rtk *)）=任执行包装洞须收
first_seen: 2026-10-31
last_verified: 2026-10-31
skills: [生产取证]
evidence_refs:
  - type: server_fact
    ref: 隔离实证台（独立 HOME+repo+claude -p 2.1.270）：热径 20/20 规则机械放行真执行（FETCH_HEAD mtime/journal restart 事件=地面真值）；危险 6/6 拦（force push×2=deny 规则、rm -rf=工作区卫兵、sudo/curl/未知二进制=弃权径）；rtk hook 在场时 force push 仍拒=deny>hook-allow
  - type: doc
    ref: code.claude.com/docs/en/permissions——"deny, then ask, then allow. An allow rule can't carve an exception out of a deny rule"；"rules are enforced by Claude Code itself, not by the model"
  - type: doc
    ref: /opt/o/.claude/settings.local.json（整改后全表——备份 /tmp/settings.local.json.bak-1793405182）
---

# 审批层单点：allowlist 纪律（热径纯前缀化+危险面留弃权+deny 硬闸）

## 症状

CEO 终端 Bash 审批在 glm 1301/抖动时全冻（多次实证）——热径命令卡在审批链，管理动作停摆。

## 机理（实测钉死）

1. **规则匹配=机械 glob**（官方档引文如上）——纯前缀规则（`Bash(cmd *)`/`Bash(cmd)`）命中即放行，不经任何模型。
2. **rtk hook（PreToolUse）=第二层**：known 命令 10-20ms 自允+改写（git/gh/ls 族，连 `git push --force` 都自允!）；未知名**弃权**（空输出→落规则面）。冻结暴露面=弃权集（ssh/systemctl/mbus/feishu/curl/…）。
3. **deny > hook-allow**（隔离台实证）：rtk 在场 force push 仍被 deny 规则硬拦——deny 是压得住 hook 的最硬闸。

## 纪律（复用）

- **热径全表面化**：管理高频面逐条入 allow，形=纯字面前缀+尾 `*`（禁中置 `*`——"matches everything before the first *"，吞注入）；ssh 类=**内命令形精确**（`ssh host "docker exec o-postgres psql *"`），根@host 全开放形禁入表。
- **写面精确到命令形**（commit/push/`systemctl --user restart <具名件>`）；**危险面零入表**（rm -rf/sudo/凭据写/curl/未见二进制——弃权径=慢危险好过快危险）；force push 用 **deny 硬闸**（hook-allow 压不住它）。
- **裸 `Bash(rtk *)` 类规则=任执行包装洞**（`rtk err <任意命令>`）——禁入表（hook 层自足，规则层收紧不碍热径）。
- 整改后验证=隔离台跑矩阵（独立 HOME+repo+`claude -p`：allow=真执行有地面真值；deny/弃权=被拒），禁拿真会话试闸。

## 关联

- /opt/o/.claude/settings.local.json 整改 diff（2026-10-31，宽配 ssh 形×2+rtk 裸形撤除，deny 三键落）——业主令「CEO 终端审批层去 glm 单点」交付件。
- 复合命令语义：`&&`/`;`/管道=逐段子命令全 match 才免审（段一缺=整条入审批）——热径链式命令各段都要在表。
