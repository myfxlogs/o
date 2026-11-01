# SG 角色授权册（T-0232 四角色收窄——D-0036 §12.2 裁4 乙案）

> 授权清单入档锚：收窄前基线=T-0223 册面终态（role_table_grants 实查
> 2026-11-01 实拍）；收窄后差集=fin_0095 迁移落地态。复查径：
> `SELECT grantee,table_name,privilege_type FROM information_schema.role_table_grants WHERE grantee LIKE '%_r_';`

## 收窄前基线（实拍 2026-11-01）

- `research_rw`：INSERT 23 表/UPDATE 21 表/SELECT 67 表+meta 读+
  task_transition EXECUTE+多序列 USAGE——36 表写宽权（T-0169 病灶源）
- `collector_rw`：md_* + registry/hostmetrics + app_config（SG 面）
- `projector_rw`：八桥 upsert+fin_position/strategy 读+registry/config（不扩不收）

## 收窄后四角色（钦定面）

| 角色 | 授权 | 消费径 |
|---|---|---|
| `research_ro` | SELECT fin_position/fin_strategy/fin_cashflow/fin_venue_untradeable/md_earn_rate/app_config | 研究面 fin 读径（按需） |
| `review_rw` | INSERT,SELECT fin_review/fin_review_verdict+序列 USAGE；SELECT fin_position/fin_strategy/fin_proposal/fin_opp_eval/fin_execution/fin_cashflow | OCI o-research FIN_PG_DSN——判决事实追加+取证读 |
| `roster_rw` | I/U/S fin_employee_registry,ops_headless,fin_host_metrics；I/S ops_event,ops_ticket_usage+三 seq；SELECT ops_seat_latest/ops_ticket_state/app_config/app_secrets；meta USAGE+SELECT meta_task/task_phase_*/EXECUTE task_transition | 全臂 beat/report.sh/headlesspump/headless-usage/mbus/feishu-notify 自报族 |
| `projector_rw` | 八桥镜像 upsert+fin_position/strategy 读+registry/config（原面不动） | OCI o-projector FIN_PG_DSN |
| `collector_rw` | **SG 面全撤+NOLOGIN**（只留 md 面=OCI PG-md 侧） | — |
| `research_rw` | **全撤+NOLOGIN**（36 表写归零；职能拆入 research_ro/review_rw/roster_rw） | — |

## 差集说明

- research_rw 自报/读取职能全承于 roster_rw（ops_event/ops_headless/
  ops_ticket_usage/registry/hostmetrics/meta 读/app_config/app_secrets）——
  避免误伤 report/earn/飞书推送/mbus 派单链（D-0036 §9 钦定避伤面）；
  纯读径归 research_ro；判决写径归 review_rw。
- app_secrets SELECT 保留于 roster_rw=feishu-notify 解密径存续（与收窄前
  research_rw 同曝面零扩）。
- 口令不落册：三新角色 SCRAM 哈希自 research_rw 同值复制（fin_0095 运行
  时取 pg_authid）；env 面只换用户名不动密文。
