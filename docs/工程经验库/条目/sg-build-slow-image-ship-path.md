---
schema_version: 2
id: sg-build-slow-image-ship-path
kind: ops
risk: medium
status: observed
keywords: ["SG构建慢", "docker save", "镜像直达", "deploy_image", "golang module", "compose build", "192.229.85.35"]
scope: [deploy_image.sh, SG宿主, docker构建链]
summary: SG 宿主拉 golang module 构建超 15min 即换镜像直达径——本机 build→docker save|gzip→ssh load→compose up（不重 build）；source 仍走 deploy_sync 保持跟踪面一致。
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: []
evidence_refs:
  - type: doc
    ref: scripts/deploy_image.sh
  - type: doc
    ref: docs/数据流.md
---

## 事实/症状

SG（192.229.85.35）compose build 拉 golang module 50min+ 未完——宿主网络到 proxy 慢，源码构建链在 SG 侧不可用。

## 根因或策略解释

SG 出口带宽/代理质量不足以支撑 Go module 全量拉取+编译；镜像是已编译产物，传输体积远小于构建时长。

## 处置

镜像直达径（现行 deploy_image.sh 唯一合法径）：本机 `docker build` → `docker save img | gzip -1 | ssh SG "gunzip|docker load"` → SG `compose up`（无 --build）。SG 侧 /opt/o 源树仍由 deploy_sync 同步保鲜（SG cron backup_pg.sh 等脚本面依赖）。

## 证据

- check: `head -20 scripts/deploy_image.sh` + SG `crontab -l` 实测 `/opt/o/scripts/backup_pg.sh` 日拍
- result: deploy_image.sh 注释明载「SG 5Mbps 构建链慢，镜像传输取代远端编译」业主令 2026-10-24
- ref: scripts/deploy_image.sh:1-15

## 反证条件

- falsifies_when: SG 网络/代理改善使源码构建稳定 <15min，或业主另令改径

## 适用边界

- 仅 SG 构建场景；OCI/本机构建不适用
- go.mod 变更（新依赖）时镜像径优势更大；纯脚本/配置改动 deploy_sync 已够

## 伴生量表（T-0332）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免：部署径判据件非运行面病灶——判据=SG build 时长人工观察， gauges.tsv 无对应表）
