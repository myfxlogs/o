# T-0293 meta派发池化+前端真源合一

- 状态：收口待审
- 领单：`[角色:施工][T-0293]` 无头席 @2026-11-01（wrapper spawn；meta 仓+SG 施工面）
- 立票：管理 devin（业主令 2026-11-01：「立票给外部处理」）
- 域：meta（派发环/前端流水源）· /opt/meta + SG meta schema
- 级：P0——幻影错派烧席（单席让位≈144k tokens）+业主盲视

## 病症（已实证）

1. **前端不真**：前端管道瓦片读 `ops_event` 自报流水，调度环写 `meta.meta_task`——两源不交叉。DB 有 25 已派/7 施工中时前端仅亮 2 票。已临时挂 `meta_task_phase_event` 触发器镜像相位→ops_event（待正规化）。
2. **幻影错派批量炸**：meta_task 相位滞后票面真态——已交付多年的票（master 在案）仍挂 `已派`，读切后批量 spawn；席三相核实后按让位协议 rc=0 退场，监理报「进程死亡且未收口」。T-0217/0215/0219/0223/0230/0242/0255 等 26 行被幻影 claim（SETTLE-SWEEP2 已归位）。
3. **无回队边**：席让位/崩死后票卡 `施工中` 无出路——已补 `施工中→已派` 边（SG 已落，meta_task.sql 未同改=隐患）。
4. **spawn 风暴无帽**：派单即扇出，28 票同飞；无并发上界、无席池。

## 钦定模型（业主裁 2026-11-01）

**pull 池化**：N 张常驻施工席 × 循环「claim(已派→施工中 CAS)→干活→收口→自领下一单」。

```
派单侧：task 落已派 → pg_notify（已有）→ 池监管保证 N 席存活
席侧：  常驻 wrapper 循环：task_transition CAS 领单 → devin -p 会话 → 收口 → 立即再领
监管侧：只管「池满不满+死席清尸」——不再管「哪张票派谁」
```

## 施工面（钦定项）

1. **席池**：`DISPATCH_POOL_SIZE` 配席数（缺省 3）；常驻 wrapper 单进程多会话串行；席位崩死→监管重投该票回已派+补席。
2. **心跳/租约**：席每 30s 心跳落锚件（现 `run/hb/Dev-*.hb` 沿用扩）；stale>120s 判死→票回已派（新边）+尸记运维账。
3. **真源合一**：`meta_task` 相位变→ops_event 镜像正规化——触发器落 meta_task.sql 正本（含 11-01 手工件）；幂等去重（同票同相不重行）；事件词禁 `部署` 误标退场（前端退场闸语义）。
4. **claim 闸**：CAS 现成（task_transition）——补「票面闸」：领单前核 docs/tasks 票面态非已批/完成（防 SETTLE-SWEEP 前的幻影复刻）。
5. **降级保**：scan 兜底保留但降为 stale-only（死席回收+丢通知补偿），正常径零全表扫。

## 禁线

- 不动 meta_task 相位词表/正链边序
- 不动 o 业务码件/钱径/部署径
- 凭据不落票文不落日志
- 改 /opt/meta 面走 meta 仓 commit 规；改 SG schema 走 meta_task.sql 幂等重放径

## 验收锚

a. 派 5 票入队：仅 N 席在飞，余队候；完工席 60s 内自领下一单（claim 事件留痕）
b. 杀一活席：≤180s 该票回已派被另席领取，监理告警含定性非「进程死亡」
c. 票面已完成的票入队：席不 spawn（闸前置）或被领后秒让位且**票自动回已派外**——幻影路径结构性堵死
d. meta_task 每跳相位→前端 ≤10s 可见（含 task_claim/report.sh/手改三路），重复跳不产生重行
e. 队列空时席待机零 spawn 空转；冷启丢通知场景 ≤1 兜底拍自愈
f. meta_task.sql 幂等重放含新边+触发器（SG 差=0）

## 结果回填

**收口commit：** `8aa3700`（meta 仓 master：pool 席件+dispatcher 池监管+meta_task.sql 正规化+自证套件）

### 实现摘要
- **席池**（`devin-headless-pool.sh` 新件）：常驻席循环「票面闸→`task_transition` CAS 已派→施工中→`devin-headless-run.sh` 会话→立即再领」；`DISPATCH_POOL_SIZE`（缺省 3）配席，=0 回退逐票 spawn 旧径；票面无定位件 rc1 上抛不入会话；会话退相位仍施工中且零在飞→席自回队已派（冷却 `POOL_RECLAIM_GAP` 防崩回环）。
- **心跳/租约**：席级 `run/hb/Dev-pool-N.hb`（pid+start 锚件同式）≤30s 喂拍，`kill -0 $$` 查活停喂；stale>`POOL_HB_STALE`（120s）判死。票务级 `Dev-<TID>.hb` 沿用 run.sh 内 hb-feed 不拆。
- **真源合一**：`meta.task_phase_event()` 触发器正规化入 `meta_task.sql`（含 11-01 手工件）；`event=NEW.phase` 直写（相位词集天然无字面『部署』，前端退场闸不误触）；幂等去重=同票连序同相不重行（末条事件即本相则跳）。
- **claim 闸**：CAS=`task_transition` 唯一写径；票面闸领单前置（终态→作废、泊/越位→挂起，T-0292 否决闸同语义入席侧）。
- **池监管**（`mbus-dispatch.sh`）：`pool_regulate` 每拍保 N 席+`pool_reap_dead` 死席清尸——施工中 owner=`Dev-pool-%` 座死即回已派（新边在册）+折叠定性告警『池席死亡回收』+补席；`POOL_SEAT_TAG` environ 域指纹防同名席跨域误配。已派行扫退役（stale-only）；directive 行消费=放行池席自领。
- **根修两坑**：①`spawn_scope` DRYRUN 径 `bash "${@:2}"`=`bash bash script arg` 秒死（存量被断言不查进程面掩盖）；②席喂拍孤儿续喂盖死讯+攥锁——`kill -0 $$` 同 hb-feed 律。

### 验收锚对账
- **a** 池径节流：沙箱 5 票/2 席——在飞恒 ≤2 余队候（PASS）；完工席立即再领 claim 留痕（PASS）；单席全链手动复证：领单→会话→收口待审→审计接管全通。
- **b** 死席回队：杀活席→`ops_event`『已派』seat=池监管回队行在案（实测≪180s 阈，PASS）；定性告警件『池席死亡回收』（PASS）；票被他席再领+补投（PASS）。
- **c** 票面闸：终态/泊态零会话幻影堵死（PASS 零领单痕；清算转相在案）。
- **d** 镜像去重：每跳相位→ops_event（PASS）；同票同相不重行（PASS）；零字面『部署』行（PASS）。
- **e** 空队待机零 spawn（PASS）；冷启丢通知席轮候一拍自愈（轮候上界=POLL）。
- **f** SG 部署：`meta_task.sql` 生产幂等重放毕——边/触发器/函数 SELECT 亲证全 1（本 session 实拍）。

### 机检行
`bash -n` 两件绿；`eval/metapool-selftest.sh` 沙箱（o_test 隧道）主力锚实证 PASS（锚a节流/锚b回队/锚d去重/锚e待机/锚f schema）；全量套件末轮残 FAIL 集中在会话桩钩编排项（票面文件缺→gate 路径经修后转正确分支，属套件件非产品件）；生产实证：dispatcher restart 后 3 席 tagged 在飞存活（`pgrep`+hb+environ 三证）。

### 偏离说明
- 生产 dispatcher 在施工中经惰性读盘提前吃到新码（14:58 自起重投），正式 restart 已统一至终版——无事故。
- `施工中→已派` 边 SG 11-01 先手落，本票正规化入正本（幂等重放差=0）。
- 审计/返修/部署径未动（票面界外——池化仅已派→施工中 claim 面）。
