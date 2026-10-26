# T-0167 PG-md 容器+迁移链分流（D-0030 §六/§八——分域施工①）

## 态：候派（等 T-0166 设计验证过+T-0165 落地）
## 立项：决策席 @ 2026-10-26
## 决策档：D-0030

## 施工
0. **隧道钥匙收窄（D-0030 §八 r4 改判——Tailscale 非必要）**：SG sshd
   `PermitRootLogin prohibit-password`+fail2ban；隧道走专用低权用户 `tunnel@`
   （`Match User tunnel`→`ForceCommand /bin/false`+`PermitOpen 127.0.0.1:15433`）
   或钥匙行加 `restrict,port-forwarding,permitopen="127.0.0.1:15433"`；root 钥匙
   从隧道退役；`alert-bridge-tunnel` 同收窄。验证：以隧道钥匙 `ssh SG id`→拒；
   PG 隧道通；SG 24h 暴力认证计数降。
1. 开发机 `docker-compose.dev.yml` 起 PG-md（timescale 同版），数据卷与施工树
   目录隔离；systemd `o-prod.slice`（collector/research/projector/PG-md 归入）。
2. `migrations/` 分链：`md_NNNN_*`/`fin_NNNN_*`；桥表镜像 DDL **归一化断言**
   （D-0030 §六——弃「同源 include」措辞，D-0031 六-⑥）。
3. `check_migrations.sh` 扩三断言（两链各单调/桥表 DDL 归一化同形/域表禁串链
   ——域表判定用显式表清单，不按 md_/fin_ 前缀）。
4. `pgstore.Migrate(-domain=md|fin)` 选链；SG 走 fin、开发机走 md。

## 桩锚
- 两链各自 Migrate 全过；串链表→check 红；桥表 DDL 漂移→check 红。
## 验收锚
- 开发机 PG-md 空库 Migrate md 链全过；SG fin 链 Migrate 幂等零变更。

## 复审注记（2026-10-26→D-0032）

- 候派前置补一条：T-0165 返修（report 留 SG）须验收合入 master——现 8037071
  仅在 `t/0165-research-node`，master 票态仍「已批」，report 留 SG 未定。
- 派单时票面须并入 md_macro_event 跨机读面裁法：D-0032 裁默认=投影第八桥表
  （§三/§四/§六「七桥表」改八）；若复议改选「report 读 PG-md」，须补第二
  DSN+隧道失败静默语义，并回 D 档撤回锚③结论。
- 本票「桥表 DDL 同形」实现须比列定义全项+NOT NULL/DEFAULT/CHECK/UNIQUE/FK/
  索引集（D-0032 §三-5）——只比列名/类型/PK 不满足回滚「DDL 相同」前提。
