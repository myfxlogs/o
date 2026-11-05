---
schema_version: 2
id: buf-default-template-plugin-fail-kills-all
kind: pitfall
risk: medium
status: verified
keywords: [buf, generate, protoc-gen-es, node_modules, worktree, gen, context canceled, bootstrap]
scope: [buf.gen.yaml, buf.gen.mtapi.yaml, web/node_modules, worktree gen 再生]
summary: buf 默认模板整拍炸——protoc-gen-es（web/node_modules 依赖）缺席不只 ES 面败，Go 插件一并 context canceled rc=1 零产出；mtapi 模板纯 Go 插件裸 wt 即产。裸 wt gen 再生须按模板分拍。
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: commit
    ref: T-0375
  - type: doc
    ref: scripts/bootstrap_worktree.sh（分拍正径+降级径）
symptom: |
  裸 worktree 跑 `buf generate`（默认模板）：Failure: plugin
  web/node_modules/.bin/protoc-gen-es: no such file or directory，且
  protoc-gen-go/protoc-gen-connect-go 报 context canceled——rc=1，
  internal/gen 零产出（Go 面也拿不到，不只 ES 面缺）。
root_cause: |
  buf v2 一次 generate 多插件并发跑，任一插件起不来即整体 context
  cancel——成功插件不落盘。默认模板的 protoc-gen-es 挂
  web/node_modules/.bin 相对路径（gitignored 件），新 wt 无 node_modules
  即炸全拍。mtapi 模板（buf.gen.mtapi.yaml）纯 Go 插件（PATH 上），
  无此依赖——裸 wt 实测 rc=0 产 internal/gen/mtapi/mt5。
fix: |
  gen 再生按模板分拍：mtapi 先行（裸 wt 可行面）；全谱默认模板仅在
  web/node_modules 在场时试（要全正径先 npm ci）；败面降级径=部署树
  rsync 叠加补缺（不 --delete——保住 buf 鲜产面），明示非静默。
  见 scripts/bootstrap_worktree.sh（T-0375 件②）。
prevention: |
  新 wt 先 bootstrap（AGENTS.md/施工席职卡钉行）；勿假设 buf generate
  在裸 wt 全谱可跑；fixture/CI 裸态跑 gen 前先查插件依赖面。
---
