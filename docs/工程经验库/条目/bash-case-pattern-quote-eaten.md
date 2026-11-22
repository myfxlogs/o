---
schema_version: 2
id: bash-case-pattern-quote-eaten
kind: pitfall
risk: medium
status: verified
keywords: [bash, case, glob, 单引号, 模式匹配, 桩件, 静默不匹配, deploy_watch_test]
scope: [scripts/**/*_test.sh psql 桩, case "$sql" in 模式分派]
summary: bash case 模式内嵌单引号被当引号剥除——event='心跳' 实配 event=心跳，模式静默永不匹配含字面引号的主体，桩分支哑火症状像产品 bug
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: []
evidence_refs:
  - type: code
    ref: scripts/deploy_watch_test.sh psql 桩 split_part 心跳末行查询分派支（工单O-30514 修）
  - type: experiment
    ref: "case \"xx event='心跳' yy\" in *event='心跳'*) → NOMATCH；*\"event='心跳'\"* → MATCH"
---

# bash case 模式内嵌单引号被剥除致静默永不匹配（O-30514 桩测实证）

## 现象

`case "$sql" in *event='心跳'*) ...` 写法直觉上应匹配含字面
`event='心跳'` 的 SQL 串——实际**永不匹配**。bash 在解析 case
模式时把模式内的单引号当**引号字符**处理：`'心跳'` 被剥成
`心跳`，模式实际是 `*event=心跳*`（无引号），而主体含
`event='心跳'`（有引号）→ NOMATCH。

## 判据（钉死）

- 模式内裸写 `'` = 引号语法不是字面量；要匹配字面单引号须
  `\'` 转义或把含引号段双引号包裹：`*event=\'心跳\'*` /
  `*"event='心跳'"*`。
- 症状链：桩模式哑火→应答支落空→fixture 返回空→下游断言红——
  红的样子像产品码 bug（本案曾误判为 hb_write 查询形不匹配，
  逐层排查才定位到模式引号）。
- 对照：同文件其它支早用 `*event=\'升件\'*` 转义写法=正确先例，
  新增支漏转义即埋雷。

## 处置

- 修：桩模式改单引号转义 `event=\'心跳\'`（heredoc 'EOF' 内
  字面保留 `\'`，bash 运行时解析为字面 `'`）。
- 预防：case 模式要匹配含引号 SQL 片段时，写完先做单行 echo
  复现验证（`case "xx event='心跳' yy" in <pattern>)`），
  勿凭直觉假设引号是字面量。
