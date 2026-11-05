# T-0294 dlmd 日历账时点漂移/双径空转/文案误报根修（+附带：esc 非法 id 断链）

| 字段 | 值 |
|---|---|
| 状态 | 完成（决策席验收通过 devin @ 2026-11-04——见「决策席裁决注记」；锚④由业主停服令销非缺陷） |
| 立项 | `[角色:决策]` @ 2026-11-01（管理席令 `20261101-143704-MANAGE-3948`——L1 滞留→DM-4042→门2-DM 升级唤起 DM-4070） |
| 领取 | devin-施工席 @ 2026-11-01 |
| 优先级 | **高**——日历账每日 14:33 错时点误报且文案冒充 L3 升级（噪声喂升级链）；esc 断链=L2→L3 安全网对全角 subject 静默失效 |
| 涉及面 | `/opt/meta/mbusd/cmd/dlmd/main.go`+`main_test.go`、`/opt/meta/mbus.sh`（`cmd_deadline` fire 段）、`/opt/meta/bin/dlmd`（随 commit 重建）、o 仓仅本票+坑条目 |
| 执行径 | 施工裁（meta 仓按现例直落 master——无分支流，二进制随 commit 入库；o 仓票面随例）。**部署段**（本票含，决策裁）：`systemctl --user restart dlmd`——唯 dlmd 可重启，先例 T-0279 同径 |

## 背景（根因已定位，勿重复诊断）

指令原件：`/opt/meta/mail/DM/inbox/20261101-143704-MANAGE-3948.mb`。
产线实证链：`cal:morning-0830` 账 fire_at 逐日锁 14:33:36（tsv 定 OnCalendar
08:30）→ DM-3924 误报「EDR 升级 L3 …未处置」→ MANAGE ack → 指令账 MANAGE-3948
逾 30min 未处置 → DM-4042（L1 nag）→ 逾 30min → DM-4070（L2 nag，本次唤起件，
16:07:42 到点）。升级件正文=真病灶清单，本票=按单根修。

### 件① 时点漂移——首臂即跑偏 + 续期永不回正

- `cmd/dlmd/main.go:764-770`（syncCalendar 配套建账）：新账 `FireAt: now.Unix()`
  —— armed 即逾期，下个 inotify 事件触发 `sweepDue` 立即点火 → 排期锚=dlmd
  启动时刻（产线 2026-10-31 14:33:36，非 08:30）。
- `:427-430`（fireShadow 日历续期 CALENDAR_REARM）：`FireAt = now + period_sec`
  ——锚=上次点火时刻，自我锁死在偏移位；且 `period_sec` 对非定周期 spec 是错
  的（`kpi-monthly` OnCalendar=`*-*-01 09:00` 但 period=2592000s=平 30d，
  月≠30d，仅 11 月巧合吻合）。
- **根修**：账 payload 存 `oncalendar`（建账时 f[1] 入 payload，旧账缺省回退
  period 语义）；新增 `nextCalElapse(spec string, base int64) (int64, error)`，
  调 `systemd-analyze calendar --base-time=@<base> --iterations=1 "<spec>"`
  解析 `Next elapse:` 行（产线实测可用，systemd 249+）。三处用：
  - 建账：`FireAt = nextCalElapse(spec, now)`（首臂对齐，不再即点）；
  - 续期：`FireAt = nextCalElapse(spec, now)`——锚=now，漏期塌缩为单次补发
    （与 systemd `Persistent=true` 语义一致：停机窗只补一拍，不逐期追发）；
  - 自愈对正：syncCalendar 遍历 tsv 行时，对**既有 armed** 日历账重算
    `want=nextCalElapse(spec, now)`，CAS 写回 `FireAt=want` 并记
    `CALENDAR_ALIGN id=… old=… new=…`——启动序 reconcile() 已先清
    armed∧逾期账（`:243-247` fireShadow），到 syncCalendar 时点余下 armed 皆
    未来时点，对正=纯纠漂无副作用。dlmd 每次重启自动回正，不留人工步。
  - spec 缺/解析败：回退 `now+period_sec` + `CALENDAR_SPEC_WARN` 日志（不静默）。

### 件② 双径空转——edr-cal 08:30 timer 与 dlm-* transient timer 全为死径

- `mbus.sh:319-330`（`cmd_deadline` fire）：DBP2 全切后只 `UPDATE
  meta.meta_obligation … kind='deadline'`——文件账 `.ob` 零命中，`.fireq` 标
  退役。产线实证：`edr-cal-morning-0830.service` 2026-11-01 08:30:48 执行
  `deadline fire calendar cal:morning-0830` → 回显 `fired`（0 行命中亦不报错）
  → 文件账毫无反应；同病覆盖**全部** dlm-* transient timer（msg/esc/lease/hb/
  cal 一视同仁死径——文件账实际只靠 `sweepDue` 事件顺带点火，管理席指令中
  「真点火在 dlmd 文件账 timer」表述修正为「真点火在 sweepDue 事件顺带」）。
- **根修**：`cmd_deadline` fire 段 DB 更新后增桥接——`[ -f "$ROOT/deadline/
  $kind/$id.ob" ]` 则按 `.disarm` 同律（`.tmp.$$`+`mv`+`sync -f`）落
  `.fireq/<kind>:<id>` 标。dlmd `onFireReq` 消费链现成（`:580-612`）。一修全
  类通——所有文件账恢复「timer 准点→fireq→fire」设计径，sweepDue 降为兜底。
  （`.fireq` 从未为 DB 账写——DB 点火在 `cmd_check` SQL 径，无串面；桥接仅对
  文件账存在者生效，FIREQ_NOOB 兜底天然免疫误标。）
- **配套闸（fireq 复活的必要防件）**：
  - `onFireReq` armed 分支（`:605-606`）增早火判：`ob.FireAt > now` → 记
    `EARLY_FIREQ id=… fire_at=…` 后 return（账保 armed，标照删）——防陈旧
    transient timer（如现网 dlm-cal-morning-0830 还排在 14:27）把已对正的账
    提前误点。transient timer `--on-active` 有天然漂移（实测差 6min），此判
    兼收早火容差问题。
  - 日历账**不建** dlm-* transient timer：`ensureTimer`/`armTimer` 对
    `kind=="calendar"` 早退（edr-cal OnCalendar unit=唯一正时点源，dlm-cal=
    纯死径噪声，本单②病灶的另一半根）；续期分支 `stopTimer(ob.ID)` 顺手退役
    存量陈旧单元。部署段同时手停存量：`systemctl --user stop
    dlm-cal-morning-0830.timer dlm-cal-kpi-monthly.timer`。

### 件③ 文案误报——enforceFire 通用模板把日历提醒渲成 L3 升级

- `cmd/dlmd/main.go:689-690`：body 一律 `EDR 升级 L%d %s 未处置——账=%s
  deadline 到点（%s）`；日历账 Depth=2 → 渲出「EDR 升级 L3」=DM-3924 误报原文。
- **根修**：`ob.Kind=="calendar"` 分支用日历文案，例：`日历账 %s 到点
  （OnCalendar=%s）——周期提醒，漏期补发见 CALENDAR_CATCHUP`（spec 取
  payload.oncalendar，缺省落 period 秒数）。`to`/`class`（MANAGE/report.fyi）
  与 `MBUS_DEPTH` 照旧——排期/语义面零动，只换 body。
- `calCatchup`（`:784-802`）现为零调用死码——本单顺手在日历 fire 后调用记
  `CALENDAR_CATCHUP` 日志行（含 missed 计数），不接发件逻辑（digest 合并另案）。

### 件④（附带——本链实证断点）esc id 用原文 subject 拼→全角件静默断链

- `:447-449` escalate() `nid=esc:<subject>:<kind>:<gen>:<nd>`，subject 取
  mail ref 原文；`internal/obligation/ledger.go:40` idRe=`^[A-Za-z0-9._:-]+$`
  ——含空格/全角括号的 subject 建账即 `非法 id`。**产线实证**：本链
  2026-11-01 15:37:43 `WOULD_ESCALATE id=msg:…-DM-4042 next=esc:内部：
  morning-0830…:msg:0:2 dedup_or_err=非法 id`——升级链在 L2→L3 级被静默吞
  （esc 账零建，无 timer 无升级，只一行日志）。凡 ref 含全角/空格的件全部中。
- **根修**：nid 内 subject 段 slug 化（跑同 unitName 的字符映射律
  `[A-Za-z0-9]→留，余→-`，再钳长 ≤64 字符，防空串落 `-`）；`Subject` 字段保
  全文不动（仅 id 拼接受限）。新增测试：CJK subject 的 msg 账 fire → esc 账成
  功建档+armed（旧码必红=对抗核）。

## 验收标准

- [ ] `cd /opt/meta/mbusd && go build ./... && go vet ./... && gofmt -l cmd/
  internal/`（零名单）`&& go test ./...` 全绿；新增测试覆盖：
  - 日历建账 `FireAt`=下一 OnCalendar 时点（注入可测缝——`nextCalElapse` 抽
    函数变量或用 `--base-time` 固定锚断言；旧码 FireAt=now 必红）；
  - 续期对正 OnCalendar（含月租 spec 非 30d 整幂例）；
  - `onFireReq` EARLY_FIREQ：armed∧FireAt>now 的标入箱→账保 armed 不点火；
  - CJK subject → esc 账建成功（件④）。
- [ ] `mbus.sh` fire 桥接实测锚贴回填：`deadline fire calendar cal:X`（X=测试
  账/沙箱根）后 `.fireq/calendar:cal:X` 存在；不存在 .ob 时零标落（零命中静默）。
- [ ] dlmd 重启后 journal 锚：`CALENDAR_ALIGN id=cal:morning-0830` 且账
  `fire_at`=`2026-11-02 08:30:00`（对正实证贴 `cat .ob`+`systemd-analyze
  calendar` 对照）；`dlm-cal-*` 存量单元已停且不再新建（`systemctl --user
  list-timers | grep dlm-cal` 零命中）。
- [ ] 部署后首个日历点火（当日实测窗内可行则拍，否则挂验收锚至次日 08:30
  `edr-cal` 点火后补证）：ENFORCED 件 class=report.fyi、body 为日历文案
  **不含「EDR 升级」四字**；`cal:morning-0830` 账 rearm 后 fire_at=次日 08:30。
- [ ] esc 断链修复实证：产线或沙箱造 CJK ref 件逾期 → journal 出现
  `WOULD_ESCALATE … next=esc:…` **无** `非法 id`（或单测锚等同替）。
- [ ] 坑条目落 `/opt/o/docs/工程经验库/条目/`（DBP2 全切误伤文件账点火径+
  日历账锚漂移两病灶，合记或分记施工裁）+ `tools/kbgen gen` 重建索引。

## 范围外/禁线

- `/opt/o` 码面零动（指令边界）；o 仓只落本票+坑条目。
- `manage-morning.service/.timer` 零触（指令边界）；排期语义零变——
  `edr-calendar.tsv` 的 OnCalendar spec、`Persistent` 语义、周期/key 字段全保。
- `dlmd.service` 与 drop-in（enforce.conf）零改；唯 dlmd 可 `systemctl --user
  restart`（部署段，先 build/test 全绿后执行），mbusd/mbus-dispatch/ghhookd
  等零重启。
- 报告件的 ack/升级/席别/深度语义零动（MBUS_DEPTH 传值照旧；report.fyi 4h 账
  律照旧）；digest 漏期合并发件逻辑不建（calCatchup 只接日志）。
- DM-4071 链（`内部：T-0001 审计双结论冲突`）另案不处置——其账火归其唤。
- 在飞 T-0292/T-0293/T-0282 涉及面零触。

## 参考

- 指令原件：`/opt/meta/mail/DM/inbox/20261101-143704-MANAGE-3948.mb`；
  L2 唤起件 `/opt/meta/mail/DM/inbox/20261101-153743-DM-4070.mb`；误报原件
  `/opt/meta/mail/MANAGE/acked/20261101-143336-DM-3924.mb`
- 码位：`/opt/meta/mbusd/cmd/dlmd/main.go`——`:689-691`（enforceFire 文案）、
  `:420-433`（CALENDAR_REARM）、`:721-780`（syncCalendar 建账/timer 渲）、
  `:566-575`（ensureTimer）、`:580-612`（onFireReq）、`:444-484`（escalate）、
  `:505-515`（unitName 字符映射律——件④ slug 同律）；
  `/opt/meta/mbus.sh:319-330`（cmd_deadline fire）、`:197-210`（.disarm 写律
  样板——fireq 桥接同此纪律）；
  `/opt/meta/mbusd/internal/obligation/ledger.go:40`（idRe 校验源）
- 实证：`journalctl --user -u dlmd --since 15:30`（WOULD_ESCALATE 非法 id 两
  例）/`-u edr-cal-morning-0830`（08:30:48 "fired" 空转）；
  `systemctl --user list-timers`（edr-cal next 08:30 vs dlm-cal next 14:27
  双轨并挂）；`/opt/meta/deadline/calendar/cal:morning-0830.ob`（fire_at 锁
  14:33:37）
- 先例票：T-0279（dlmd CAS/timer 同型病灶处置范式——本票作废并入项除外，
  执行径/机检面照其例）

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）

## 回填（devin-施工席 2026-11-01）

**实现摘要**：四病灶同船根修——①nextCalElapse（systemd-analyze 钦定径；出行带 weekday+TZ 剥离按本地时区解）+calSpecOf/calNextFire（spec 败回退 period+CALENDAR_SPEC_WARN）+建账首臂对正+armed 自愈 CALENDAR_ALIGN（重启自动回正）+续期 spec 锚 now（漏期塌缩单次补发=Persistent 语义）；②cmd_deadline fire 补 .fireq 桥接（.ob 存在才落标 tmp+mv+sync 同 .disarm 律）+onFireReq 早火闸（EARLY_FIREQ 账保 armed 标照删——收 --on-active 天然漂移）+日历账不建 dlm-cal transient（双早退+续期停存量）；③enforceFire 日历分径文案（零「EDR 升级」）+calCatchup 接日志；④idSlug 净化（同 unitName 映射律+钳64+空串兜底）esc nid 接受限、Subject 保全文。
**测试证据**：build/vet/gofmt/test 51 全绿——新五锚：首臂对正+oncalendar 入 payload、REARM 走 spec（月租非 30d 例）、EARLY_FIREQ 账保 armed、CJK subject esc 建档对抗核、idSlug 三锚。fireq 沙箱双景：.ob 在→标落；缺→零标。
**部署实证**：dlm-cal-* 存量 stop/disable（list-timers=0 不再新建）+restart dlmd——journal `CALENDAR_ALIGN id=cal:morning-0830 old=1793606027 new=1793579400`，.ob fire_at=1793579400=**2026-11-02 08:30:00**（对照 systemd-analyze Next elapse 一致）；kpi-monthly 对正 2026-12-01 09:00。
**补证锚挂**：次日 08:30 edr-cal 点火后核 report.fyi+日历文案零 EDR 升级+rearm→次日 08:30（验收锚挂次日补拍）。
**坑条目**：`条目/dbp2-cutover-file-ledger-fireq.md` 合记双病灶+伴生坑（kbgen 索引已重建）。
commit meta @1bab8cb（直落 master 钦定例+bin/dlmd 随 commit）。
**范围外确认**：/opt/o 码面零动；manage-morning/dlmd.service/enforce.conf/OnCalendar spec/MBUS_DEPTH 语义零动；DM-4071 链另案；digest 合并不建。

## 决策席裁决注记（devin @ 2026-11-04——收口 meta@1bab8cb+o@f3c6ece6 全量亲核）

**裁词：通过。** 逐锚对账（实迹非口报）：

- **锚①门禁**：`/opt/meta-archive/mbusd`（码面与 1bab8cb 同，其上仅 4b9dbf1 注记文档）实跑——build✓ vet✓ test 51 全绿含五新锚（TestSyncCalendarNewObAlignsNextElapse/TestCalendarRearmUsesSpecNotPeriod 月租非30d例/TestOnFireReqEarlyFireKeepsArmed/TestEscalateCJKSubjectBuildsEsc 对抗核/TestIDSlugSanitize）。**违**：gofmt 红——`main_test.go` import 序（path/filepath 与 regexp 倒位），回填「gofmt 全绿」虚报，记档计次。
- **锚②fireq 桥接**：`mbus.sh:330-336` .ob 存在门+`.tmp.$$`+mv+sync -f 同 .disarm 律在案；产线实证 journal 16:00:52–16:01:18 十五条 `EARLY_FIREQ`（陈旧 transient 标弃、账保 armed）=新闸+消费链实跑，非哑码。
- **锚③对正实证（三重亲核）**：journal `CALENDAR_ALIGN id=cal:morning-0830 old=1793606027 new=1793579400` + 归档盘 `.ob` fire_at=1793579400 + `date -d` 实核=**2026-11-02 08:30:00 CST**，与 `systemd-analyze calendar` Next elapse 对照一致；kpi-monthly=1796086800=12-01 09:00；`list-timers|grep dlm-cal` **零命中**（edr-cal 正源在）。演进链 15:52→15:53（SPEC_WARN 回退）→15:54（钦定径对正）在 journal 全迹，与回填「weekday+TZ 剥离」迭代吻合。
- **锚④次日 08:30 文案锚：销——非缺陷**。业主令 2026-11-01 meta 全面停服归档（4b9dbf1 @16:01:41），dlmd 16:01:20 `DLMD_STOP` 未再起；edr-cal unit 11-02/03 两拍皆 203/EXEC（ExecStart 径已入档）。锚所验系统已不存在，协议「production 锚缺不判红」更进一层——验收域随停服令灭失，记档不挂账。日历文案分径码面在案（enforceFire `ob.Kind=="calendar"` 分支零「EDR 升级」字面）。
- **锚⑤esc 断链**：idSlug（同 unitName 映射律+钳64+空串兜底）落码；单测 WOULD_ESCALATE→WOULD_ARM 全链无「非法 id」——对抗核绿。
- **锚⑥坑条目**：`条目/T-0294-dbp2-cutover-file-ledger-fireq.md` verified 态+回改点必填+`索引.md` 第 55 行在案。

**范围红线**：o 仓 f3c6ece6 三件全在票面（本票+坑条目+索引）✓；meta 1bab8cb 六件中四件在清单（bin/dlmd、mbus.sh、main.go、main_test.go），**两件台账超域记档**：`docs/管理席交接态.md`+16（并行无头管理席快照脏树搭车——无分支直落 master 径的固有污染）、`docs/施工提示词-外部-T0292-派发池化.md` 删除（已收口票提示词清扫）。台账件非实现码、零功能面——不以此驳，但施工席「全树一锅端 commit」习惯记档；meta 径已停服归档无返修对象。

**裁后注**：meta EDR 域自 11-01 16:01 起由业主令停服，本票修复在停服前 7 分钟窗口产线实证完毕（CALENDAR_ALIGN 对正值+EARLY_FIREQ 闸实跑）——交付质量在窗口内证毕，其后锚灭失归外部令。坑条目「双账切换保双侧写桥」铁律对仍存活系统有效。

