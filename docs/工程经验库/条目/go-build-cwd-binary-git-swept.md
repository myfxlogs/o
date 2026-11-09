---
schema_version: 2
id: go-build-cwd-binary-git-swept
kind: pitfall
risk: medium
status: observed
keywords: ["go build", "cwd二进制", "git add -A", "Bin文件", "commit stat", "fillsprobe", "core二进制"]
scope: [go build, git提交卫生, worktree]
summary: go build ./pkg/cmd（无 -o）把二进制写 cwd——git add -A 扫入 commit 两犯在案（16MB/28MB）；cmd 构建一律 -o /tmp 或 add 前查 git status。
first_seen: 2026-10-2x
last_verified: 2026-10-2x
skills: []
evidence_refs:
  - type: doc
    ref: docs/工程经验库/待归纳.md（staging 原档——纪元3归纲转正）
---

## 事实/症状

`go build ./pkg/cmd`（无 -o）把可执行二进制写在当前工作目录——`git add -A` 把它扫进 commit：T-0085 fillsprobe 16MB、T-0086 core 28MB 两犯（后者 amend 急救）。

## 根因或策略解释

go build 默认输出=包名二进制落 cwd；与源码改动面无关的大文件混入 diff，commit stat 是唯一可见体征。

## 处置

- cmd 构建一律 `go build -o /tmp/x ./cmd/x`（或显式 OUT 路径）
- build 后 `git status --short` 核无意外新文件再 add
- commit 前核 diff stat——出现与改动面无关的 Bin/大文件=违规体征

## 证据

- check: staging 原档两犯实录（T-0085/T-0086）
- result: commit stat 出现与改动面无关的 Bin 文件
- ref: 待归纳 staging 原文（本条目转正件）

## 反证条件

- falsifies_when: 仓规强制 go build 一律 -o（lint/gate 拦截）或 .gitignore 全覆盖二进制名

## 适用边界

- 本机/CI 任意 go build 场景；go run ./tools 不落二进制不受影响

## 伴生量表（T-0332）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免：提交卫生件——判据=diff stat 人工核）
