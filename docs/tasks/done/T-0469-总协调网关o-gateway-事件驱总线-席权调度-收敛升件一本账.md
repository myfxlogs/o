# T-0469 总协调网关 o-gateway——事件驱总线 + 席权调度 + 收敛/升件一本账

- 状态：已施工待外审（实现+桩测八组全绿+check_all 全链绿——外派收口移交外审席裁决；勿 merge 勿部署勿启用 service）
- 归口：外审席裁决径
- **前置义务已结**：/opt/meta 弃养死因查毕——结论见文末「施工回填」节

## 病灶（2026-11-08 全日实证）

1. **有探测无扳机**：deploy_window_due 三态机完备，「窗清→开火」无承接
   ——OCI 面 0441 merge 后数小时无人发，CTO 三班处置全对仍断链。
2. **调度碎裂 9 壳**：派单（dm_tick 班扫+派单行+systemd-run）、候水
   （dispatch_water）、水位闸（dispatch_worker）、依赖闸（dm_depgate）、
   退班账（dm_shiftend）、交接哨兵（dm_handoff）、升件径（escalate/
   ops_esc）、巡检（gauges_tick）、收敛（mirror_push/deploy_window_due）
   ——同一套「谁起谁退」职能碎成九份，各席位注意力轮流接盘=今日断链根。
3. **席位注意力依赖**：所有收敛/生命周期动作的最终承接都是「某席恰好
   在班恰好看到」——席退/水位满/换班即断链，今日实证三处。

## 定位

**o-gateway = 公司总协调面唯一执行网关**（meta v2）。三本账归一：

- **席权账**：谁起谁退——spawn/retire/补拉/水位/死席清扫全权；
- **路由账**：谁做什么——已批件机派（派单行+拉起同拍）、交接对账
  （交必有接，断链即分派认领）、候水账；
- **收敛账**：推已绿 sha→发 image/oci 面→验活→部署行；升件认领
  路由（升件→归口席→未认领再升/冻结）。

**边界红线（永不逾越）**：gateway 只做**机械调度与钦定执行**——
判断永归席位：票优先级=CTO 裁域、升件定性=归口席裁域、部署该不该
=deploy_window_due 判据件裁。gateway 读判定结果去执行，不自立判据。

## 架构

### 形态

常驻 systemd user service `o-gateway.service`（Type=simple 常驻，
Restart=always，OnFailure 升件）——事件驱非轮询：

| 事件源 | 机制 |
|---|---|
| ops_event 新行 | PG LISTEN/NOTIFY（无则短轮询降级） |
| `.git/check-green-*` 落戳 | inotify 文件事件 |
| 票队/派单态变化 | ops_event 行触再评 |
| systemd 席态 | 周期活性采集（退化为拍,非主驱） |
| 时钟兜底 | 低频拍（30s-60s）防事件漏缝 |

### 一本账状态机

```
IDLE → SENSE(事件+采集) → CLASSIFY(席权|路由|收敛|升件) →
  席权:  已批未领+水位有余 → spawn拉起;rc75 → 候水账;
         席死/孤儿 → 清扫+补拉判;
  路由:  交行无接超阈 → 归口席分派+升件;派单行无拉 → 补拉;
  收敛:  净发窗+绿戳 → push→deploy面→验活;
  升件:  未认领超龄 → 再升一级(归口→CTO→CEO→业主);
→ LEDGER(ops_event seat=gateway 全链行) → IDLE
```

- **幂等键**全动作钉死：spawn=票号单实例、push=sha、deploy=sha:face、
  升件=esc id——重事件零重复动作；
- **败即升件+面冻结+不自愈重试**（处置权归归口席——防循环炸）；
- **崩溃安全**：全状态可从 ops_event 账重放重建，Restart 后接续；
  ledger 先行后动（写账→执行，崩在中途账上有痕）。

### 降级安全（单脑险对冲）

- gateway 死 → **席位自扫描兜底层不灭**：dm_tick/ceo_tick/席位班扫
  保持原样（不动一字）——公司降回「席位注意力制」慢转不瘫；
- gateway 活性自身被监：heartbeat 行每拍落 ops_event，
  gauges_tick 添 `gateway_hb` 行（停鸣>阈=升件——谁看门人=巡检闸）；
- kill switch=stop+disable unit 即回现行制，零迁移成本。

## 判据

- 事件驱实证：戳落文件→≤事件延迟内 PLAN（非等拍）；
- 席权实证：已批票→spawn 拉起同拍派单行；rc75→候水账态正确；
- 交接实证：人造断链→归口席分派行落账；
- 收敛实证：净发窗+绿戳→push→deploy→VERIFY 全链序正确；
- 冻结实证：人造升件未决→收敛面全面零动作；
- 幂等实证：同事件重放零重复动作；
- 降级实证：gateway stop→席位班扫照旧接手（兜底不瘫）；
- 崩溃实证：kill -9→Restart 后状态接续无重复执行。

## 范围

- `scripts/meta/gatewayd.sh`（常驻主循环：事件采/分类/状态机/调钦定件）；
- `deploy/o-gateway.service`（常驻单元，Restart=always）；
- `scripts/meta/gateway/*.sh`（四类动作子件：席权/路由/收敛/升件）；
- gauges.tsv `gateway_hb` 行；
- `scripts/meta/tests/gateway_test.sh`（桩测：事件/席权/路由/收敛/
  冻结/幂等/降级/崩溃八组）；
- mirror_push.timer 退役注记（功能并入）；
- meta 弃养死因查+结论落票；
- 本票回填。

## 禁线

- 禁动判据件/席位职卡/既有闸/dm_*/deploy_* 本体——gateway 只调用；
- 禁自立判断语义（优先级/定性/该不该——永归席位）；
- 禁自愈重试/禁冻结期强发/禁无戳部署/禁推未绿 sha；
- 禁碰钱径；禁用无头席位施工/裁决（业主令：全件走外派外审）；
- 外派收口移交外审席——勿 merge 勿部署勿启用 service。

## CEO 裁决空位

- [ ] 一步到位 V2 全谱放行（事件驱常驻+席权调度+收敛+升件归一）
- [ ] 边界红线认可（机械调度归网关/判断永归席位——派单=已批后机械拉起，
      不裁优先级；升件=路由归口不裁定性）
- [ ] 降级安全认可（gateway 死=席位自扫描兜底不瘫，双轨共存期认可）
- [ ] mirror_push.timer 功能并入 gateway 认可
- [x] meta 弃养死因前置查为施工首步认可
- [x] 外派施工+外审裁决径遵令确认

## 施工回填（T-0469 施工席 2026-11-09）

### /opt/meta 弃养死因结论（前置义务）

**非崩溃死=组织换代弃养**。实证：

- `meta-gateway.service`/`meta-caster.service` 均 disabled+inactive；
  gateway.log 末行 2026-11-01 20:37「meta: 收信号 terminated」=干净
  SIGTERM 人为停，非崩溃——此后从未重启、单元留 disabled。
- 2026-11-02 06:25 留 `.meta-gateway.migrated` 标记+`meta-pre-surgery.db`
  快照=迁移手术当日执行，旧栈状态已搬走。
- 仓非空：meta.db WAL 仍活跃（11-06）、meta.sock 在、run/ 工件新——
  废弃的是**服务面**非仓本体；git 有未提交 T55 半成品
  （ceo.go/dispatch.go/gateway.textproto+t_t55_test.go）=施工中止在案。
- 死因定性：席位体系从 meta socket 席换代为 /opt/o 无头席
  （ops_event+o-worker+dm_* 壳群），旧网关随换代退役未启用新岗。
- **约束条款随案走（防同死法）**：本票 o-gateway.service 同样交付态
  =未 enable——启用归外审/部署径；disabled 单元文件残留属正常退役
  态。残留风险登记=meta 仓 T55 半成品未提交（范围外不动，见交付备注）。

### 实现面（全部新增，禁动件零改动）

- `scripts/meta/gatewayd.sh`：常驻主循环 IDLE→SENSE→CLASSIFY→ACT→IDLE。
  LISTEN ev_pipeline 持久 psql coproc（piggyback keepalive 冲刷 NOTIFY——
  实测非 tty 下通知随下一命令落地）；ops_event.id 水位+脏位域标记+
  `.git/check-green-*` 戳签名三通道驱动；域全量拍兜底（60/60/300/120s）；
  事件拉近最小间隔（10/10/45/30s）防抖；psql 会话断=自动重建+降级轮询；
  advisory lock 42540356 单例（拒锁 sleep 重试=failover 等待）；ops_headless
  GATEWAY 心跳+ops_event 低频心跳行。
- `scripts/meta/gateway/common.sh`：DSN 解析链（GW_ENV→DM_ENV→
  REPORT_ENV 同 ops_esc 式）、gw_q/gw_sql/gw_ledger/gw_esc/gw_state_*。
- `gateway/seatauth.sh`：dm_depgate 直调（已批快派）+dm_queue 队扫+
  死席道龄（ops_headless last_activity）→systemctl start 唤起
  o-cto/o-ceo-seat.service（600s 冷却幂等）+败自报升件。
- `gateway/route.sh`：dm_queue dispatched 补拉——dispatch_order 保序+
  人工径滤+票文件存查+活单元跳+模型回取（最新派单行）→dispatch_worker
  拉起（rc75→dispatch_water 候水）+dm_handoff 对账；零动作不落汇总账。
- `gateway/converge.sh`：deploy_window_due 判据直调（不自裁）→应发则
  mirror_push→deploy_image/deploy_oci 按 faces 发射；升件未决冻结
  （gateway 自写升件路由行排除防自冻）+面级 DEPLOY-FIRE-FAIL 冻 24h+
  部署行 sha:face 幂等查重+VERIFY 部署行验活；败升件不自愈重试。
- `gateway/escalate.sh`：未认领超龄升件扫描（认领判据量表同式+
  gateway 自行排除）→升件#id 级幂等+级冷却阶梯 L1归口→L2 CTO→
  L3 CEO→L4 业主（COS上行+feishu 尽力推）。
- `deploy/o-gateway.service`：Type=simple+Restart=always+OnFailure=
  o-gateway-fail.service（ops_esc 升件告警 oneshot）。
- `deploy/o-gateway-fail.service`：OnFailure 升件告警 oneshot。
- `scripts/meta/gauges.tsv`：+`gateway_hb`（ops_headless GATEWAY 停鸣
  >15min=1，断供档）。
- `deploy/o-mirror-push.timer`：退役注记（功能并入收敛账本，双轨
  兜底期保留，停用归部署/验收径施工席不动）。
- `scripts/meta/tests/gateway_test.sh`：八组桩测全绿（见下）。

### 测试实证

`bash scripts/meta/tests/gateway_test.sh` → **gateway scenarios PASS**：
①事件 notify→脏位→域即评/水位续位零重脏/绿戳 stamp 档即评；
②席权 depgate 调用+死席唤醒+冷却幂等+depgate 败升件；
③路由 补拉/活单元跳/rc75 候水/零动作不落账；
④⑤收敛 应发全链 PLAN→FIRE→VERIFY+冻结全面零动作+面冻按面隔离；
⑥幂等 部署行查重+升件路由级幂等/冷却/阶梯 L1→L4 COS上行；
⑦降级 无 DSN 域件自含退出+daemon 轮询兜底存活；
⑧崩溃 kill -9→重启水位/域拍续位零重复动作。

`scripts/check_all.sh` mode=full 全链绿（build/vet/import/gofmt/env/
rejects/migrations/ops-chain/seat-writes/ticket-seq/go test 全谱），
绿戳落 check-green-<sha>。

### 交付备注（范围外发现，不动只记）

- `/opt/meta` 仓有未提交 T55 半成品（internal/gateway/ceo.go、
  dispatch.go、config/gateway.textproto、t_t55_test.go）——前任施工
  中止遗留，归 meta 域处置席，本票不越权清理。
- `meta-gateway.service`/`meta-caster.service` 单元文件仍 disabled
  残留于 ~/.config/systemd/user——退役常态，无动作。
- psql LISTEN/NOTIFY 非 tty 管道下为 piggyback 语义（通知随下一条
  SQL 命令冲刷）——gatewayd 以 keepalive SELECT 1 周期冲刷兜底，
  真库联调时若有延迟观测可调 O_GATEWAY_KEEPALIVE_SEC。
- gateway 域子件 env 覆写面（O_GW_*）为桩测面，生产默认值即票面钉值。

### 外审裁段（外审席 2026-11-09）

`[外审裁:T-0469] 过 @merge=626140363 | 断言对拍 8/8 | 戳=check-green-626140363(独立复跑 mode=full 全绿) | 部署态=滞留待戳`

- **读票**：判据八条全列——事件驱/席权/交接/收敛/冻结/幂等/降级/崩溃。
- **审diff**：1210 行通读——边界红线实证守住：判据零发明全调钦定原件
  （depgate/dm_queue/dispatch_worker/dm_handoff/mirror_push/deploy_window_due
  /deploy_*），route 补拉与 mech_tick:102-141 逐字同源对拍实证；新事件词
  （席权/路由/收敛/升件路由/网关/心跳）相位投影 ELSE→NULL 不扰态，'拉起'
  有意进词；自写行 seat=gateway 自滤防回环；冻结断言复用
  escalation_unclaimed 量表同口径；部署幂等 sha:face 与 mech 同账共享；
  OnFailure 单元在件；meta 弃养死因查毕结论在案（非崩溃死=换代弃养——
  「交付未启用」同死法条款随案走，启用归本裁后部署径）。
- **机查**：本席独立复跑 scripts/check_all.sh mode=full 全链绿
  （不采信工人自述戳；rebase 后 HEAD=ac123c9e3 全绿→merge=626140363
  再跑全绿——两戳实证，非单戳移植）。
- **部署态注记**：merge commit 626140363 已上本地 master，推送滞留——
  绿戳闸当前被外因红 TestReportMacroEventsCount（ledger/report 周一边界，
  T-0484 挂起同锚，与本票零码面关系）所拒；check_all 并发同 SHA 互噬
  实证一发（并跑先毁旧戳后红=戳没——坑已升件）；红消后 check_all
  落戳即推，o-gateway.service 启用+实拍随推径后行。
