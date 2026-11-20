# o-core 换件部署窗（deploy_watch）——工单#25760

| 字段 | 值 |
|---|---|
| 状态 | 生效（支 `fix/25760-ocore-window`） |
| 票 | 工单#25760 `ocore-deploy-window` |
| 落件 | `scripts/deploy_watch.sh` + `migrations/fin_0217_ocore_window.sql`（配置种）+ `scripts/config-write-domain.tsv`/`dmwrite.go`（dm 域）+ 本档 + `scripts/deploy_watch_test.sh` |
| 窗值 | `19:00-21:00 UTC`（2h）；`max_hold=26h`；票面理由锚 `ops_event#25794` |

## 一、判决

`deploy_watch` 谱系判命中 `cmd/core` 依赖闭包 = o-core 镜像必换 + SG 钱径重启。
原行为「随 merge 随到随换」把重启散布在全天；本件收窄为 **UTC 低活动窗集中
放件**：

- **窗外**：入窗队列让位——本拍不换 o-core，落一条 `ocore-window 入窗` 事实；
- **窗内**：照走原有锁/败避让/部署全段（`deploy_image.sh` 不变）；
- **越窗未放**：最老待放 commit 龄 > `max_hold` → 升件 `deploy_ocore_window_overdue`；
- **窗配置非法**：fail-closed 不换钱径 + 升件 `deploy_ocore_window_config`。

本件只改 `deploy_watch` 的自动径。会话紧急径（`deploy_at_sha.sh`/直接
`deploy_image.sh`）不入窗——应急不受窗押。

## 二、窗配置

`app_config`（fin 正本→PG-md 只读副本）：

| 键 | 值 | 类 | 语义 |
|---|---|---|---|
| `deploy.ocore_window_utc` | `"19:00-21:00"` | dm | UTC 窗 `HH:MM-HH:MM`；左闭右开；start>end=跨午夜 |
| `deploy.ocore_window_max_hold_h` | `"26"` | dm | 最老待放 commit 龄超此值→升件（小时） |

取值理由（业主未钦，按低活动自裁，票面已载）：

- `fin_execution` 90d 按 UTC 时：19-21 点仅 4/1232（0.3%），13-17 点 1081
  （88%），04 点 108（9%）——避开两个执行峰；
- `ops_event` 14d 按 UTC 时：19-21=645-790，日峰 2318（02 点）——属 18-23
  点低频带；
- 2h 窗 = 15min 拍约 8 次重试余量；26h = 24h 日窗 + 2h 跨度 + 漏放一次
  可检出面。

缺键/DSN 缺 = 脚本内建同值缺省（19:00-21:00 / 26）——迁移未落也不改判据。
DM 调窗走 `DMSet` 三拍（自动落 `参数调整` 行）；写域分册
`scripts/config-write-domain.tsv` 的 `deploy.*` 行与 `dmwrite.go` 同集。

## 三、判定序（deploy_watch 内）

```
HEAD != DEPLOYED
  → o-core-clean 水位 == HEAD？         → 跳部署（既有，不回归）
  → image_face_clean(DEPLOYED)？        → 落水位、零重启（既有，不回归）
  → DEPLOYED 空（首部署/灾备）？        → 立即换（不入窗）
  → 窗配置非法？                        → fail-closed + 升件 config
  → 窗外？                              → 入窗队列 + 越窗检查；不换 o-core
  → 窗内 → 部署锁/同 sha 败避让/换件    → 既有全段
```

窗外让位后**继续走 OCI 臂面**（collecto/projector/research 的 `deploy_oci`
不重启钱径，仍按原滞差自动换）——窗只押 o-core，不押臂面鲜度。

## 四、账面（ops_event）

| 行 | 时机 | note 锚 |
|---|---|---|
| `ocore-window 入窗` | 每个待放尖首次窗外让位（7d 去重） | `sha=<尖12> base=<部署> win=<窗> UTC` |
| `deploy <sha> rc=…` | 实际换件（既有 deploy_image 行） | 释放事实=该行 |
| 升件 `deploy_ocore_window_overdue` | 最老待放龄 > max_hold | 待放/部署/最老待放时刻/龄/窗 |
| 升件 `deploy_ocore_window_config` | 窗值形非法 | 原值+fail-closed 判词 |

升件去重=未销 24h 窗（`esc_pending` 同源）；窗配置坏时每 24h 提醒一次，
直到配置修好或换件完成。

## 五、测试

```bash
bash scripts/deploy_watch_test.sh
```

桩 psql/ssh/go + 本地 bare origin+clone，真脚本拷入跑；七态：

1. 窗外谱系证内 → 入窗队列、不换；
2. 窗内 → 照换、不排队；
3. 谱系证空 → `o-core-clean` 水位、零重启（不回归）；
4. 水位在案 → 跳部署（不回归）；
5. 越窗未放 → 升件；
6. 跨午夜窗（23:00-01:00）→ 窗内照换；
7. 窗配置非法 → fail-closed 不换 + 升件。

时钟缝 `O_DEPLOY_NOW_EPOCH`（秒）可重放窗内/窗外/越窗（同 `O_MIRROR_NOW`
先例）；生产不设=真时钟。

## 六、边界与自伤面

- **队列卡死**：窗外只让位不阻塞——窗一到即放；窗永远不匹配（配置坏）→
  config 升件；窗到了仍不换（锁/败/脚本停）→ overdue 升件。两级兜底均落账，
  不静默。
- **重复重启**：窗内每个 15min 拍对最新 HEAD 换件——同窗多 merge 会多换；
  窗只保证「重启不出窗」，不保证「每窗只换一次」。紧急径可随时直调。
- **首次部署**：`DEPLOYED` 空=启机/灾备，不入窗（不能把系统按在窗外）。
- **窗值自裁**：低活动证据变化时 DM 可改 `deploy.ocore_window_utc`，改值
  必落 `参数调整` 行；改窗不改 `max_hold` 的推导（26h 覆盖 24h+2h）。
- **键名联动**：deploy 键域已同册登记 hunger ④族读面册
  （`depConfigReaders["deploy"]`=部署管窗）与面基线（
  `face:dep_pair:config:deploy→部署管窗`/`face:gate_domain:deploy`）——
  改键名前缀或删种键须同步登记，否则 ④族把 deploy 域误标 orphan 升件。
- **迁移号**：`fin_0217`（`fix/gridfit` 占 0215、`fix/regime-mapper` 占 0216
  在途——同链登记顺序以 land_queue 落地时重基为准）。

## 七、变更史

| 日期 | 变更 |
|---|---|
| 2026-11-20 | 初版：工单#25760；谱系证内换件收 19:00-21:00 UTC 窗；越窗/配置坏升件；七态桩测 |
