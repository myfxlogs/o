# T-0275 sweeper OCI 建制恢复（10-28 cutover 漏件急诊）

| 字段 | 值 |
|---|---|
| 状态 | 施工中 |
| 立项 | `[角色:决策]` @ 2026-10-31（CEO 裁件 MANAGE-1497 转办，源=MANAGE-1392 业主授权；MANAGE-3043 再催——验收锚转写入下） |
| 领取 | devin-施工席 @ 2026-10-31（DM-3055 派单） |
| 优先级 | 高（急诊——「清道夫-md」beat 自 2026-10-28 21:14Z 断，驾驶舱红灯如实红） |
| 涉及面 | OCI 盒（`~/o/sweeper` 二进制、`~/.config/systemd/user/o-sweeper.service`、复用 `~/o-projector.env`）；repo：`deploy/o-sweeper.service`（新件）、`scripts/oci_bootstrap.sh`（units+verify 两段补件）、`docs/运维/OCI盒档案.md`（服务清单+凭据面注记）、dev 机遗件清理 |

## 背景

10-28 SG→OCI 采集层 cutover 漏件：`cmd/sweeper`（PG-md 清道夫，席位名「清道夫-md」）在 dev 是**用户级 systemd unit 但无册面无容器**，迁移清单按「看得见的 unit/容器」盘点漏了它——红灯 2 天+被误挂账为显示问题（T-0241 v4③，已翻案）。病灶全录 `docs/工程经验库/条目/cutover-orphan-service-checklist-gap.md`。

CEO 裁（MANAGE-1497）：①即修=建制内恢复（非新设非红线）；②部署口 CTO 定，倾向与 SG 同机制；③sweeper 纳入 cutover checklist 防再漏。

## CTO 部署口裁（本票执行依据，勿另议）

**用户级 systemd unit `o-sweeper.service`**——与 OCI 全部 Go 臂（o-collecto/o-projector/o-research）同机制，且逐字复刻 dev 原设计（`~/.config/systemd/user/o-sweeper.service` + `EnvironmentFile=%h/o-projector.env` 复用投影 env）。

- 淘汰项=compose：OCI docker 面只载 `o-postgres-md` 一件，Go 二进制全走 `~/o/`+用户级 unit；为 sweeper 单开 compose=同盒双机制分裂。且漏件根因正是「无 unit 孤儿」——入 systemd 册=防再漏本体，恰合「建制内恢复」。
- **零新密件零新授权**：`~/o-projector.env` 已含 `MD_PG_DSN`（本机 5435）+`SG_PG_DSN`（隧道 15433，projector_rw）；SG 实查 `projector_rw` 对 `fin_employee_registry` 持 INSERT/UPDATE/SELECT（Register+Beat 所需全集，本席亲证 information_schema.table_privileges 2026-10-31）。
- **MANAGE-3043「加回compose」措辞处置**：部署口已授权 CTO 定（MANAGE-1497②），本裁 systemd 用户级同构为终审——勿翻案再议 compose。

## 目标

1. OCI 侧 sweeper 以用户级 unit 建制内复跑，beat 恢复，首拍清道收尾；
2. sweeper 补入三层册（deploy/ 模板、oci_bootstrap 复立径、盒档案清单），迁移/重灌不再漏；
3. dev 侧遗件清除，防幽灵复活再造假红。

## 验收标准

- [ ] OCI：`systemctl --user is-active o-sweeper`=active 且 `is-enabled`=enabled；`journalctl --user -u o-sweeper` 见 `[sweeper] PG-md 清道夫起` 且无 Fatal 循环
- [ ] SG：`SELECT now()-last_beat FROM fin_employee_registry WHERE name='清道夫-md'` < 2min（30s 拍恢复）
- [ ] PG-md（盒 5435）：`fin_data_retention` 审计行 `last_swept_at` 非 NULL（boot 首拍即时跑，不等 24h tick）
- [ ] **（MANAGE-3043 ①锚转写）md_ticks 旧数据被清**：PG-md `SELECT count(*) FROM md_ticks WHERE ts<now()-interval '7 days'`=0（10-31 实查现态已 0 行，验收复核保持）——注：`md_ticks` 为 state 表（产品键 upsert 末帧覆盖非追加），`deadTailSweep` 在 PG-md **结构性不删**（`fin_active_set` 缺席→fail-safe abort 钦定行，防半读谓词清空病灶 T-0197④）；死尾积累有界（universe 行数级），真需清尾须先立「PG-md active_set 供源」票，非本票面
- [ ] **（MANAGE-3043 ①锚转写）ops_event 有 sweeper_md 记录**：`SELECT * FROM ops_event WHERE ticket='T-0275' AND (note ILIKE '%sweeper%' OR note ILIKE '%清道夫%')` 非空——`ops_event`=席自报事件簿（T-0239，SG fin 库），收口 `report.sh 收口 T-0275 "清道夫-md…"` 落行即满足；sweeper 本体不写 ops_event（PG-md 无该表）
- [ ] PG-md 三张未入账表先入账再首拍（防报缺刷屏）：`arch_eval`/`arch_signal`→class `permanent`，`md_daily_snapshot`→class `state`（`INSERT INTO fin_data_retention(table_name,class) VALUES …`，10-31 实查仅此三件缺）
- [ ] `deploy/o-sweeper.service` 模板入册（文件头注装配法+env 复用说明）
- [ ] `scripts/oci_bootstrap.sh`：units 段 o-sweeper 入用户 unit 安置环+臂体校验列（`~/o/sweeper`）+env 注记（复用 o-projector.env 无需 escrow 新件）；verify 段尾查清单加一条「谁在 beat=谁在跑」roster 反向核对（SG `fin_employee_registry` active 席 beat 新鲜度逐席对拍——kb 判据落机）
- [ ] `docs/运维/OCI盒档案.md`：服务清单加 `o-sweeper.service` 行（用户级·PG-md 清道），env 件行注 `o-sweeper` 复用 `o-projector.env`
- [ ] dev：`systemctl --user disable --now o-sweeper`（若仍 enable）+删 `~/.config/systemd/user/o-sweeper.service`、`/opt/o-projector/o-sweeper` 遗件，daemon-reload（10-31 实查：unit 已 disabled+dead，余删件动作）

## 执行要点（施工照做，不再勘查）

```bash
# dev 交叉编 + 船运（OCI=arm64，同 collector/projector 径）
cd /opt/o && GOARCH=arm64 GOOS=linux go build -o /tmp/o-sweeper ./cmd/sweeper
scp -i ~/.ssh/id_ed25519 /tmp/o-sweeper ubuntu@149.118.58.98:~/o/sweeper

# 盒上（ubuntu 会话）：unit 内容如下——After 只钉 network-online（用户级件
# 不可依赖系统级 o-sg-tunnel；进程对 DSN 失败 fail-closed log.Fatal，
# Restart=always 覆盖隧道暖机竞态，同 projector 形）
[Unit]
Description=o-sweeper PG-md 清道夫（T-0169 分库账日拍）
After=network-online.target
Wants=network-online.target
[Service]
EnvironmentFile=%h/o-projector.env
ExecStart=%h/o/sweeper
Restart=always
RestartSec=30
[Install]
WantedBy=default.target

# ~/.config/systemd/user/o-sweeper.service → daemon-reload → enable --now
# linger 已在位（盒档案），无需再开
```

## 范围外

- 新 DB 角色/新凭据/escrow 新件（复用 projector_rw+o-projector.env——授权已亲证够）
- sweeper docker 化、SG 侧任何改动、`cmd/sweeper` 码面改动（本票纯建制恢复）
- dev 旧 PG-md 面退役残面（若有）另立单

## 参考

- `cmd/sweeper/main.go`（DSN 契约+席位名）；`internal/platform/roster/roster.go`
- `deploy/o-projector.service`/`o-projector.env.example`（同形模板）；`scripts/oci_bootstrap.sh` stage_units/stage_verify
- `docs/工程经验库/条目/cutover-orphan-service-checklist-gap.md`（病灶+判据）；`docs/运维/OCI盒档案.md`

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
