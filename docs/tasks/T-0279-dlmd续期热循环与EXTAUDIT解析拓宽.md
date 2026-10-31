# T-0279 dlmd LEASE_RENEW 热循环根修 + EXTAUDIT 收口commit 解析拓宽

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-10-31（管理席令 `20261031-215533-MANAGE-3335`——门2-DM 落箱即点火） |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | **高**——journal 实测 8178 行/3min 持续空转烧 systemd 账；双仓票解析窄径=下次即复发 |
| 涉及面 | `/opt/meta/mbusd/cmd/dlmd/main.go`+`main_test.go`、`/opt/meta/mbus-dispatch.sh`（o 仓仅本票+回填，零码面） |
| 执行径 | 施工裁（meta 仓按现例直落 master——无分支流；o 仓票面随例） |

## 背景（根因已定位，勿重复诊断）

### 件① dlmd LEASE_RENEW 热循环

`mbusd/cmd/dlmd/main.go:380-389` lease 续期分支 CAS 只翻 `State=Armed`
**不滚 `FireAt`** → `armTimer(ob)` 拿已过的旧 fire_at 算 `secs<1` 钳 1s
（`:509-512`）→ 1s timer → fireq → `leaseRenew` 仍真 → 再续，永动。

产侧实证：`journalctl -u dlmd` `LEASE_RENEW id=lease:T-0276:1` 与
`TIMER_ARM … secs=1` 每秒成对，8178 行/3min（fire_at=21:47:28 仍 armed）；
VCHAINFIX 同型两条已由管理席 disarm 缓解。

对照组=hb 轴同型「延账重 arm」分支（`:1153-1162`）**有**滚
`o.FireAt = now+stale`——注释自称「同 leaseRenew 律」实为半残：lease 侧漏
写 FireAt。次生同病：两分支 `armTimer` 均传外层陈旧 `ob` 而非 CAS 返回值
——hb 侧一拍自纠（下拍 Load 到鲜账），属错顺手同改；lease 侧配合漏滚
FireAt 即成永动。

语义锚：lease=工件席日志进度轴 30min，`leaseRenew` 判据=
`last_activity+30min>now`（`:901-908`）——续期后 FireAt 应=
`last_activity+1800`（与判据同源，非 now+1800：la 陈旧窗内按实账滚）。

### 件② EXTAUDIT 收口commit 解析窄径

`mbus-dispatch.sh:436`（票面）与 `:447`（审计锚件比对）正则要求
`收口commit[：:]` 后紧邻裸 hex——双仓票 `o@fdea6669+meta@c1f5230` 式不可
析 → `failed:收口commit不可析`+report.failed（PIPEFIX 21:51 实证）。票面
已由管理席应急改裸 sha 重派（`run/extaudit-PIPEFIX.state`=
`dispatched:fdea6669` 在案），**正则面未修=下个双仓票复发**。

同文件 `door4_csha`(`:323`)/`route_done` gate(`:765`) 走「行内首个 hex」
宽径本可容——同字面两标准即漂移病灶（KB 教训：票面字段约定须与解析正则
同源核验）。

## 目标（钦定界）

### 件①

- `fireShadow` lease 续期 CAS 回调内（鲜账 `o` 上）：重读
  `o.Payload["last_activity"]`，`o.FireAt = la + 1800`；la 缺/≤0 兜底
  `time.Now().Unix()+1800`。
- `armTimer` 改传 **CAS 返回值**（`nob, err := d.store.CAS(...)`——禁传
  外层 `ob`，否则修完仍空转一拍）；`fireHb` 延账分支（`:1159`）同换。
- 测试：`main_test.go` 照 `TestFireHbRenewsFresh` 式（`d.timersOff =
  true`）加 lease 续期断言——armed+FireAt 已过+last_activity 鲜→
  `fireShadow` 后 state=armed 且 `FireAt` 前滚（>now+1700）；hb 延账断言
  补 FireAt 前滚核。
- 部署：`cd /opt/meta/mbusd && go build -o ../bin/dlmd ./cmd/dlmd` +
  `systemctl --user restart dlmd`（指令特许——**唯 dlmd 可重启**）。

### 件②

抽共用提取函数（建议 `ticket_csha <file>`——单源除漂移）：

- 语义=文件内首个 `收口commit` 行：优先 `o@<hex>`（界锚
  `(^|[^A-Za-z0-9_-])o@` 防 `foo@hex` 子串误中）→ 落空取行内首个
  `[0-9a-f]{7,40}`（裸 sha/他 repo 前缀式按序首）。
- 四处同换：`:436`（extaudit_dispatch 票面 csha）、`:447`（审计锚件
  acsha）、`:323`（door4_csha）、`:765`（route_done gate）。禁各点自写
  正则再分叉。
- 生效径=mbus-dispatch 常驻进程下次重启（**本单禁重启**——指令边界；
  PIPEFIX 已 dispatched 无即时堵点，修=防下个双仓票复发）。

## 验收标准

- [ ] `cd /opt/meta/mbusd && go test ./...` 绿；新增 lease 续期测试断言
  FireAt 前滚过（旧码必红——对抗核）。
- [ ] `grep -n 'd\.armTimer(ob)'` 于 fireShadow lease 续期分支与 fireHb
  延账分支零命中（已改传 CAS 返回值）。
- [ ] ticket_csha 自证三式贴回填：
  `收口commit：o@fdea6669+meta@c1f5230`→`fdea6669`；裸
  `fdea6669`→`fdea6669`；`foo@aaa1111bb` 式不误作 o 侧。四处调用点
  grep 无残留旧正则。
- [ ] dlmd restart 后实证：`journalctl --user -u dlmd --since <重启点>`
  5min 窗 `LEASE_RENEW id=lease:T-0276:1` ≤数次（vs 修前 ~37行/秒），且
  `TIMER_ARM … secs=` 出现 ≈1800 级值非 1。
- [ ] mbus-dispatch.sh 改动 commit 落 meta 仓 master，票面注记「生效=下
  次重启」；**全程零重启 dlmd 以外服务**。
- [ ] 坑→`docs/工程经验库/条目/`（两处病灶合记或分记，施工裁）+
  `tools/kbgen gen` 重建索引。

## 范围外/禁线

- 在飞 PIPEFIX 审计与其票面**不碰**；T-0276 在飞生产部署**不碰**——其
  lease 账由修复后 dlmd 自理（last_activity 鲜=正常续期；陈旧=走
  fired/verdict 设计径），禁手工 disarm。
- 唯 `dlmd` 可重启；mbusd/mbus-dispatch/ghhookd 等零重启。
- leaseVerdict/hb 分诊/门4 五闸/伪票号解析/REWORK 等其他面不动。
- DM 箱 3290 冒烟探针已由管理席核结（唤径在飞让位正常，原件留箱待下唤
  消费），本单不含。

## 参考

- 指令原件：`/opt/meta/mail/DM/inbox/20261031-215533-MANAGE-3335.mb`
- 码位：`/opt/meta/mbusd/cmd/dlmd/main.go:378-389`（lease 续期）、
  `:901-908`（leaseRenew 判据）、`:505-524`（armTimer/secs 钳）、
  `:1153-1162`（hb 对照组）、`:993-1002`（lease 建账 FireAt=now+30m）；
  `/opt/meta/mbus-dispatch.sh:319-326,436,447,757-775`
- CAS 语义：`/opt/meta/mbusd/internal/obligation/ledger.go:167-196`
  （回调作用于鲜读副本并返回——续期字段必须在回调内写）
- KB：`docs/工程经验库/条目/meta-event-plane-external-blind-spots.md`
  （票面字段约定须与解析正则同源核验）
- 实证：journal 8178 行/21:55–21:58；`run/extaudit-PIPEFIX.state`=
  `dispatched:fdea6669`

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
