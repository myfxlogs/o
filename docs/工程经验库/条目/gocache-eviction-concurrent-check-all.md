---
schema_version: 2
id: gocache-eviction-concurrent-check-all
kind: pitfall
risk: medium
status: verified
keywords: [gocache, go-build, check_all, 并发, could not import, 环境故障, 多席]
scope: [testing, ci, 环境]
summary: 多席并发 check_all 共享 ~/.cache/go-build——构建项中段被逐→全包「could not import」非码面假红；枝面净树跑为权威戳
first_seen: 2026-11-08
last_seen: 2026-11-08
evidence_refs:
  - type: ticket
    ref: T-0458
---

## GOCACHE 逐出：多席并发 check_all 共享 ~/.cache/go-build 中段假红

- **实证**：T-0458 外审合并树 check_all 复跑（本机同期 ≥3 席 check_all 并发）——
  go test 段中盘起全包面炸 `could not import o/internal/...` / `cannot open
  file ~/.cache/go-build/<hash>-d: no such file or directory`；构建缓存项
  在跑程中被逐（盘余 75G 非压，GOCACHE 仅余 524M=刚遭批量清理）。
- **定性**：环境故障非码面红——错形=「import 图内任一依赖缓存缺」级联全包，
  与真 vet/编译红形不同（真红=单包点错，假红=跨包同形 could not import）。
- **处置**：①判读先辨真假——跨包同形 could not import+缓存文件缺=环境性，
  重跑即复（缓存重填）；②戳的权威面=**worktree 净树跑**（各席 wt 独立树
  不受合并写入扰动，戳绑枝 SHA 即交付件实证）；部署树 master 复跑在
  多席并发期=树被中途合并换+缓存被逐双扰，非必跑项（合并冲突解仅
  docs 件时零码差，枝戳罩合并树）。
- **防重踩**：check_all 跑前 `pgrep -f check_all` 看并发；GOCACHE 可
  `GOCACHE=<wt 私径>` 隔离跑（私有缓存逐出免疫，代价=冷编译首轮）。
