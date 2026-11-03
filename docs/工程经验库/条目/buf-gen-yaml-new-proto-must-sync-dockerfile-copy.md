---
schema_version: 2
id: buf-gen-yaml-new-proto-must-sync-dockerfile-copy
kind: pitfall
risk: medium
status: verified
keywords: [Dockerfile, buf.gen.yaml, proto, COPY, web-build, scoutv1, npm build exit2]
scope: [Dockerfile.core, buf.gen.web.yaml, proto/, 镜像构建]
summary: buf.gen.web.yaml inputs 扩新 proto 包而 Dockerfile web-build 段选择性 COPY 未跟——镜像内 buf regen 缺源目录，生成件缺席致 npm build exit2；新增 proto 包须同步 COPY 清单
first_seen: 2026-11-03
last_verified: 2026-11-03
skills: []
evidence_refs:
  - type: server_fact
    source: docker build
    query: deploy_image.sh 构建日志——RUN cd web && npm run build exit 2
    observed_at: 2026-11-03T07:10:00Z
    server_revision: "o-core:621cf5f3ddad 构建失败"
---

## 事实/症状

T-0305 前端引 scoutv1.Board——`buf.gen.web.yaml` inputs 加 `proto/scoutv1`，
本地 `npm run build` 绿（worktree 内 gen 已生）。`deploy_image.sh` 镜像构建挂
`npm run build` exit 2：镜像 web-build 段只 `COPY proto/dashboardv1`，
镜像内 buf regen 时 proto/scoutv1 源缺席→scout_pb.ts 不生→tsc 引用断裂。

## 根因或策略解释

web-build 段为层缓存走**选择性 COPY**（非 `COPY . .`）——proto 路径白名单
与 buf.gen.yaml inputs 是两张清单，扩一边漏另一边即崩。本机绿因为 gen 件
是本地 buf 生的，镜像内从零 regen 才暴露清单差。

## 自检

改 `buf.gen.*.yaml` inputs 时：`grep "proto/" Dockerfile.core` 对账——
每个 inputs path 须有对应 COPY 行（Go build 段 `COPY . .` 不受影响，只卡
web-build 选择性段）。同则适用其它选择性 COPY 阶段。

## 处置

补 `COPY proto/scoutv1 proto/scoutv1`（T-0305 本 commit 已落）。自检规则钉
此——下次新增 proto 包先过此查。
