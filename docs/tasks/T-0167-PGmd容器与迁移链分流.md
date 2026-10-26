# T-0167 PG-md 容器+迁移链分流（D-0030 §六/§八——分域施工①）

## 态：候派（等 T-0166 设计验证过+T-0165 落地）
## 立项：决策席 @ 2026-10-26
## 决策档：D-0030

## 施工
1. 开发机 `docker-compose.dev.yml` 起 PG-md（timescale 同版），数据卷与施工树
   目录隔离；systemd `o-prod.slice`（collector/research/projector/PG-md 归入）。
2. `migrations/` 分链：`md_NNNN_*`/`fin_NNNN_*`；桥表镜像 DDL 同源 include。
3. `check_migrations.sh` 扩三断言（两链各单调/桥表同源/域表禁串链）。
4. `pgstore.Migrate(-domain=md|fin)` 选链；SG 走 fin、开发机走 md。

## 桩锚
- 两链各自 Migrate 全过；串链表→check 红；桥表 DDL 漂移→check 红。
## 验收锚
- 开发机 PG-md 空库 Migrate md 链全过；SG fin 链 Migrate 幂等零变更。
