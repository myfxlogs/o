# 分析查询只读径（fin_ro）——工单#25758

| 字段 | 值 |
|---|---|
| 状态 | 生效（支 `fix/25758-adhoc-ro`） |
| 票 | 工单#25758 `adhoc-query-mirror-route` |
| 落件 | `scripts/fin_ro.sh` + 本档 + 三份 cognition runbook 入口注记 + `ops_event` 口径行 |
| 数据窗 | 2026-11-20 04:06Z；OCI PG-md 37 表 / SG 主库 39 表（`o_fin_ro` 可见 35，`information_schema` 对拍） |

## 一、判决

**分析/排查类 SQL 的默认径改为 OCI PG-md 镜像读面；SG 主库只供钱径、采集、
守卫常驻件。** 临时/探测/排查查询不再用 `psql "$FIN_DSN"` 直压 SG
`o-postgres`；确需 SG-only 表时，显式 `--sg` 走 `FIN_PG_RO_DSN` 窄读角色，
并限时窗/LIMIT。

病灶实证（票面）：排查会话全程直打 SG o-postgres——分析负载与钱径主库
同机同库，重查询会挤兑钱径/采集/守卫。本件封的是「习惯径」，不是新建
复制链；桥域已有 OCI 镜像面，够用。

## 二、三径

| 用途 | 径 | 命令 |
|---|---|---|
| 分析/排查（默认） | OCI PG-md 镜像读面：SSH + `docker exec o-postgres-md psql` | `scripts/fin_ro.sh -c "SELECT ..."` |
| SG-only 表（显式） | SG 主库窄读角色 `FIN_PG_RO_DSN`（`o_fin_ro`，SELECT-only 白名单） | `scripts/fin_ro.sh --sg -c "SELECT ..."` |
| 钱径/采集/守卫常驻件 | 各件自有 DSN（`FIN_PG_DSN`/`MD_PG_DSN`/`SG_PG_DSN`/…） | 常驻件代码，不经本件 |

`fin_ro.sh` 默认镜像；`--sg` 必须显式。**不自动回落 SG**——静默回落等于
把「禁压主库」又绕回去。

## 三、表册边界

镜像面（OCI PG-md）= 37 表，含：

- **md_\* 市场数据正本**：`md_ticks`/`md_signal`/`md_funding`/`md_klines`/
  `md_mark_index`/`md_regime_fact`/`md_regime_memo`/`md_daily_snapshot`/
  `md_dcd_product`/`md_earn_rate`/`md_macro_event`/`md_repair_log`/
  `md_tick_scan`/`md_venue_funding`/`md_ctx_fact`；
- **研究/供给面**：`fin_opp_template`/`fin_proposal`/`fin_capability_surface`/
  `fin_surface_gap`/`fin_template_proposal`/`fin_sub_request`/
  `fin_venue_untradeable`/`fin_forward_check`/`fin_data_retention`/
  `fin_trx_account`/`fin_trx_txs`/`fin_trx_vote_apr`/`fin_cmc_history`/
  `fin_cmc_positions`；
- **两侧共有**：`app_config`/`fin_opp_eval`/`fin_opp_hypothesis`/
  `fin_strategy`/`fin_venue_candidate`；
- `arch_eval`/`arch_signal`/`schema_migrations`。

实测表册（随桥演进）：`scripts/fin_ro.sh --tables`。

SG-only（镜像无；`fin_ro.sh` 的 `SG_ONLY` 常量）= 34 表/视图：

- **钱径/账本**：`fin_cashflow`/`fin_execution`/`fin_risk_decision`/
  `fin_condition_gap`/`fin_guard_close_status`/`fin_known_residual`/
  `fin_host_metrics`/`fin_exec_drill`/`fin_employee_registry`；
- **教训/验尸**：`fin_eng_lesson`/`fin_econ_lesson`/`fin_econ_rule`/
  `fin_eval_reject`/`fin_ticket_draft`；
- **NAV**：`md_equity_curve`；
- **ops 事实账/票面/状态**：`ops_event`/`ops_event_act`/`ops_esc_pending`/
  `ops_esc_valence`/`ops_headless`/`ops_decision`/`ops_precedent`/
  `ops_ticket_desk`/`ops_ticket_phase`/`ops_ticket_state`/`ops_ticket_usage`/
  `ops_state`/`ops_state_row`/`ops_seat_card`/`ops_seat_latest`/
  `ops_face_snapshot`/`ops_event_sem_matrix`/`ops_chain_residual`；
- `app_secrets`（凭据域，**任何查询径都不收**）。

`o_fin_ro` 是白名单角色，不是「SG 全表可读」：

- 已授（示例）：`ops_event`/`ops_event` 族部分/`fin_cashflow`/
  `fin_condition_gap`/`fin_econ_rule`/`fin_eng_lesson`/`fin_eval_reject`/
  `fin_execution`/`fin_risk_decision`/`fin_ticket_draft`/`ops_decision`/
  `ops_esc_pending`/`ops_headless`/`ops_precedent`；
- 未授（示例）：`md_equity_curve`/`ops_event_act`/`ops_ticket_desk`/
  `fin_employee_registry`/`fin_exec_drill`/`fin_guard_close_status`/
  `fin_host_metrics`/`fin_known_residual`/`fin_econ_lesson`。

未授表 = `permission denied`，**不要用 `FIN_PG_DSN` 绕**。需要原始表时
另立只读授权票；先用报表面/派生视图顶上。当前 `md_equity_curve` 未授，
NAV 分析走 `ops_event` 的 `profit_daily`/`books_daily`/`owner_ledger` 量表
行（`--sg` 可读）或盈利报告面。

## 四、用法

```bash
# 默认：OCI 镜像（只读事务 + 30s 语句超时）
scripts/fin_ro.sh -c "SELECT count(*) FROM fin_opp_eval
  WHERE ts > now()-interval '7 days'"

# 元组输出（脚本友好）
scripts/fin_ro.sh -t -c "SELECT venue, count(*) FROM fin_opp_eval GROUP BY 1"

# 文件 / stdin
scripts/fin_ro.sh -f /tmp/q.sql
cat /tmp/q.sql | scripts/fin_ro.sh

# SG-only 表：显式 --sg（o_fin_ro 白名单，限时窗/LIMIT）
scripts/fin_ro.sh --sg -c "SELECT id, ts, event, left(note,120)
  FROM ops_event WHERE ts > now()-interval '1 day'
  ORDER BY id DESC LIMIT 50"

# 连通性 / 表册
scripts/fin_ro.sh --check
scripts/fin_ro.sh --tables
```

可选：`--timeout <秒>`（默认 30）、`--csv`、`-q/--quiet`。
覆写：`O_FIN_RO_MIRROR_HOST`/`O_FIN_RO_MIRROR_CT`/`O_FIN_RO_TIMEOUT`；
SSH 路由复用 `scripts/oci_ssh_route.sh`（直连死→SG 跳板自愈），
镜像容器=`o-postgres-md`，与 `check_bridge_registry.sh` 的 `md_q()` 同径，
**不新造凭据**。

## 五、纪律

1. **分析/排查禁直压 SG**：`psql "$FIN_DSN"` 只用于钱径/采集/守卫常驻件和
   写面/票面；临时查询默认 `fin_ro.sh`。
2. **只读**：客户端预检拒 `INSERT/UPDATE/DELETE/DDL/COPY/CALL/SET/事务控制/
   危险函数`；会话 `default_transaction_read_only=on`；SG 径角色本身
   SELECT-only。预检是纪律件不是安全边界——镜像面是副本，仍禁写。
3. **限时窗/LIMIT**：`statement_timeout=30s` 是兜底不是许可；大表扫仍须
   时间窗/索引/聚合。SG 径尤其。
4. **凭据域禁读**：`app_secrets` 任何径都不收；镜像面没有它。
5. **不静默回落**：镜像缺表→报错并给 `--sg` 提示；`--sg` 未授→
   `permission denied`；都不自动切 `FIN_PG_DSN`。

## 六、新鲜度与权威

- OCI 镜像对 `md_*` 市场数据、OCI 原生研究表=正本；
- `fin_strategy`/`fin_venue_untradeable`/`fin_trx_*`/`fin_cmc_*` 等=
  反向桥副本，可能滞后；`fin_opp_eval`/`fin_opp_hypothesis` 两侧都有，
  方向按数据流正本读；
- **钱径决策以 SG/venue/报表为准**；镜像用于分析/排查，不用于下单/风控
  判据。需要时先对拍：
  ```bash
  scripts/fin_ro.sh     -c "SELECT max(ts) FROM fin_opp_eval"
  scripts/fin_ro.sh --sg -c "SELECT max(ts) FROM fin_cashflow"
  ```

## 七、例外

- `o-core`/collector/projector/research 等常驻件：自有 DSN，不走本件；
- `check_all`/migrations/部署管道：不改；
- 写面/票面/收口：照旧 `FIN_PG_DSN`；
- 新增桥接表：只改 `SG_ONLY` 册 + 本档表册，不另造径。

## 八、验收与账

- `scripts/fin_ro.sh --check`：镜像 `o/o` 37 表，绿；
- `scripts/fin_ro.sh --sg --check`：`o_fin_ro` 35 表可见，绿；
- 镜像查询 `fin_opp_eval` 绿；`ops_event` 无 `--sg` → SG-only 提示（rc=3）；
- 写语句 `INSERT` → 只读预检拒（rc=2）；
- `ops_event` 口径行：`ticket='fin_ro'`，锚 `工单#25758`；
- 票面回执：`工单#25758 完工回执——fix/25758-adhoc-ro@<sha>:全链绿戳在案`。

## 九、变更史

| 日期 | 变更 |
|---|---|
| 2026-11-20 | 初版：工单#25758；默认 OCI 镜像径 + `--sg` 显式 SG 窄读；三份 cognition runbook 入口注记 |
