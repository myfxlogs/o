---
schema_version: 2
id: tmux-sendkeys-enter-vs-cm-tui-submit
kind: pitfall
risk: medium
status: verified
keywords: ["tmux send-keys", "Enter", "C-m", "TUI 输入框", "devin pane", "踢径", "交互式提交", "滞留输入行"]
scope: [report.sh, 踢径, tmux 注入, devin TUI]
summary: tmux send-keys 往交互式 TUI（devin/claude pane）注入消息时，`Enter` 键名只输入不提交——文本滞留输入行；须用 `C-m` 才真正提交。两步骤缺一不可：`-l "<文本>"` 后 `sleep 1 && send-keys C-m`。
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0323-裁毕踢交互席-report-KICK四通道.md（修法钦定形 C-m 在案）
  - type: server_fact
    ref: devin O-DM pane 双侧实证 2026-11-04（Enter 滞留输入行 vs C-m 真提交）
---

# tmux send-keys 注入 TUI：Enter ≠ 提交，C-m 才是

## 病灶

`tmux send-keys -t <pane> -l "text"` + `tmux send-keys -t <pane> Enter`
——文本进输入行但 **Enter 键名在 devin TUI 不注册**，消息滞留输入框
不提交。踢径形同虚设：对端 pane 看见字、进程收不到消息。

## 实证（2026-11-04 双侧）

- `send-keys -l "..."` + `send-keys Enter` → 文本滞留输入行（业主亲眼）
- `send-keys -l "..."` + `send-keys C-m` → 真提交（消息作为输入送达）

## 正形

```bash
tmux send-keys -t O-DM -l "消息全文" && sleep 1 && tmux send-keys -t O-DM C-m
```

- `-l` 保文本字面（防特殊字符被当键名解释）
- `sleep 1` 让 TUI 消化文本再收提交键
- **`C-m`（回车码）才是 TUI 认的提交**；Enter 键名是终端序列语义非 TUI 语义

凡往交互式 pane 注消息（派件/踢席/自动回报）一律此形——send-keys 的
键名集是终端语义，交互 TUI 的提交键是另一套，不默认等价。
