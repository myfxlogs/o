# GW-001 内循环机械轨——o-mech 归一 + 事件接线 + 积压消解

> 独立基建票（GW 系，不占 T 编号、不进 dm_queue/ops_ticket_state 票队
> 判据——本件即管道本体，鸡生蛋故独立）。源头：T-0469 第三方外评改稿
> （git 历史在案）+ COS 评审 P1×2/P2 修补并入。原 T-0469 呈批件保留
> 为决策记录，施工以本票为准。
>
> **席位与例外授权（业主令 2026-11-08 钦定，留证防翻案）**：
> 施工=外派（禁无头席）；裁决=COS 交互席（业主直授，本件例外，
> 不引为先例）；merge+部署=COS 同席执行（两闸合一席=业主钦定例外，
> 证据全落 ops_event/票面）。
>
> **执行制**：业主逐段转发一行提示词 → 外派施工 → 收口报证据 →
> COS 独立复核（复跑断言非采信）→ 过则 merge+部署，败则退件。
> 完一项验一项，不批量化。

---

## 一、目标

全自动内循环：`案→草案→裁→票→已批→派单→拉起→收口→裁决→merge→
绿戳→push→部署→闭环回写案卷`，每跳①有事件唤下一席②下一席队列
谓词认这一跳③机械活不被判断会话的锁挡住。判断归席，机器只接线。

## 二、原则

- **单轨**：机械动作唯一执行者=o-mech；dm_tick 退化为纯判断会话壳；
- **不建长寿 bash 守护件**：事件由既有 eventbus/metabus + systemd
  `.path`/`ExecStopPost` 承载，不自建 LISTEN/NOTIFY、不自写 inotify；
- **相位单源**：票相位由生命周期词集推导，**锚在投影/VIEW 层**（见
  段 B-①修补），dm_queue.sh 只是消费方之一；
- 幂等键钉死（spawn=票单实例 / push=sha / deploy=sha:face / 升件=esc id）；
- 败即升件+面冻结+不自愈重试；判断永归席位。

## 三、诊断（实证引据——第三方外评已逐条复核为真）

收敛段 G1-G4：部署扳机挂 LLM 会话内（dm_tick L94-107 队空即退、
尾段无开火窗）；单例锁挡机械尾段（L60-89 拿不到锁 exit 0）；
eventbus 白名单只认 收口/移交/收口待审/升件/诊断草案
（cmd/eventbus/main.go `d` 源）；绿戳文件事件却挂 15min timer。
诊断前段 D1-D13：fast_close 单独不立案（diagnosis.go L124）、
裁定席 bug 案只落 ticket='' 升件、dm_queue review 谓词不认诊断草案、
归流卡 verdict_tick 会话闸、部署后无案卷回写。
积压段 B1-B5：ops_ticket_state=VIEW「DISTINCT ON 全事件 ts DESC」
即污染源本体；派单只在锁后且 rc75 break 全队；仓/部署单元
O_WORKER_PARALLEL_LIMIT 漂移；票翻取证面窄；水位律无机检
（37 活跃票中 17 票翻欠账+5 派单后被说明行踢出队列，18:00 实查）。

## 四、分段施工序（A→E，每段一行提示词一件收口）

### 段 A 收敛段——今日断链直接病灶，先行

- **A1 o-mech 机械轨**：新 `scripts/meta/mech_tick.sh` 原地承接
  dm_tick.sh 尾窗全部子件（depgate 补拉/dm_suspend/dm_handoff/
  ticketdraft/dm_autoapprove/dm_ledger/verdict_deliver，子件不改）；
  **自有 advisory 锁（新键≠42540354）**，拿不到=退；
  追加收敛窗（唯一新逻辑≈40 行）：
  `deploy_window_due.sh → state=应发 → 读 faces → image: 幂等查
  'deploy <sha> rc=0' 无→deploy_image.sh；oci: 'deploy-oci <sha>'
  无→deploy_oci.sh；rc≠0→ops_esc CTO DEPLOY-FIRE-FAIL:<sha12>:<face>
  +本面冻结24h去重键不重试`。判据全在 deploy_window_due；
  `deploy/o-mech.service`(oneshot)+`o-mech.timer`(5min 兜底)。
- **A2 dm_tick 瘦身**：锁闸+队闸+devin+hb；尾段改一行
  `systemctl --user start --no-block o-mech.service`；**移除全部
  spawn/push/deploy 能力**（禁双轨）。
- **A3 戳位归一+推闸加固**（COS 评审 P1 修补，替代原稿 ②）：
  `check_all.sh` 戳改落 `git rev-parse --git-common-dir`
  （全 worktree 戳统一进 `/opt/o/.git/check-green-*`）；
  `mirror_push.sh` 候选追加闸 `git merge-base --is-ancestor "$sha"
  HEAD`——**只推已并入本地 master 的 sha**（防分支戳被自动推）；
  `deploy/o-green-stamp.path` `PathChangedGlob=/opt/o/.git/check-
  green-*`→`o-mirror-push.service`；`o-mirror-push.service` 加
  `ExecStartPost=systemctl --user start --no-block o-mech.service`；
  mirror timer 保留兜底。pre-push 验戳径不变（主 gitdir 即有戳）。
- **A4 事件白名单与路由**：eventbus `d` 源白名单 `+已批,派单,裁决,
  处置,部署,起草`；metabus wake 表加 `kind=mech→o-mech.service`
  （判断类仍 dm 帧双唤 CTO/CEO，机械类→mech）；
  **注意 eventbus=OCI 盒二进制，本段含 deploy_oci 发自证**。
- **A5 可观测**：`gauges.tsv` +`mech_hb`(MECH 心跳停鸣>15min 升件)、
  `loop_latency`(案 ts→部署行 ts 中位)；`handoff_pairs.tsv` +
  `诊断草案|裁决,驳回|7200`、`派单|拉起,领单|1800`、
  `拉起|收口|7200`（崩径僵尸票哨兵，COS 评审 P2 修补）。

**段 A 判据（COS 验收跑这六条）**：
① 人造绿戳→≤10s mirror_push 起→push→o-mech 起→应发→部署行，
全链无席位会话；② CTO 会话在飞时 kick→o-mech 照跑（锁分离实证）；
③ 同 sha 重触→部署行唯一（幂等）；④ deploy 面 rc≠0→升件+24h 零
重发；⑤ `stop o-mech.timer o-green-stamp.path`→旧 timer 仍转（降级）；
⑥ 分支戳（未 merge 的 worktree 戳）存在时 mirror_push **不推**它
（HEAD 包含闸实证——P1 修补必验）。

### 段 B 积压速效（可与 A 并行）

- **B1 相位投影层**：新 VIEW `ops_ticket_phase`（生命周期词集推导：
  正词 `起草→已批→派单→拉起/领单→收口/移交/收口待审→裁决→完成`，
  负词 `驳回/挂起/作废/异常`，说明类事件一律不进相位）——
  **不改 ops_ticket_state**（cockpit 等存量消费者语义不动，
  新相位面并行存在，COS 评审 P1 修补：锚 VIEW 层不锚消费方）；
  `dm_queue.sh` 改读新 VIEW，新增 `dispatched` 模式（派单未拉起）；
  dm_tick 补拉 SQL/dm_depgate 自带 SQL 删，统一调 dm_queue。
  **预期：5 张被踢票（0443/0444/0446/0449/0452）即回队**。
- **B2 漂移消**：`deploy/o-cto-seat.service` 补
  `Environment=O_WORKER_PARALLEL_LIMIT=2` 与部署单元对齐；
  并行硬顶 2 不动（业主令）。
- **B3 票翻取证扩两源**：dm_ledger A 段+(a)裁决通过∧无票支∧范围节
  无码件→裁决执行件翻完成；(b)票面「外审裁过归宗@<sha>」∧sha∈
  origin/master→外审 merge 证据。首拍清 17 张欠账的大部。

**段 B 判据**：① `dm_queue dispatched` 列出人造「已批→派单→处置」
票；② cockpit 读旧 VIEW 行数不变（零误伤）；③ 两型欠账票各一
翻完成入 done/；④ 全谱零新增 FAIL/SKIP。

### 段 C 批派段（依赖 A 的 o-mech）

- **C1 ExecStopPost 链**：`dispatch_worker.sh` systemd-run 加
  `--property=ExecStopPost=systemctl --user start --no-block
  o-mech.service`；补拉循环 rc75 改「本票候水续试下一票」不 break。
- **C2 机批册按族入册**：`meta.autoapprove.registry` +`rootcause:
  <family>`（族谱=diagnosis.go diagFamilies）+`postmortem:*`；
  **负目录表落码**（非散文）：internal/exec、ledger 写径、venue
  下单面=永不入册（机检断言）；起草滞留阈 24h→2h。
- **C3 水位机检**：ticketdraft 物料化前查 已批未领+候选≥5→草案
  留 pending 不占票号；同签名族 24h 合并一票；已批 7 天未动→回
  候选+升件；派单序改 (优先级,已批ts)，族机械映射
  `postmortem:*/rootcause:* > 写败:*/账链断:* > 观测盲:*`，CTO 可覆写。

**段 C 判据**：两张零交集已批票，首张退场→≤10s 次张拉起；
rc75 不阻断次票；负目录签名人造草案→机批断言拒；WIP 闸到 5 停出票。

### 段 D 诊断前段（依赖 A；打通「案→票」）

- **D1** `diagnosisWorthy`：`fast_close` 附复发条件入闸（同
  strategy_ref∨inst_id 7d≥N 次∨settled<0，N=热启键
  `guard.postmortem.fast_close_repeat_n`）——单次秒平不烧诊断额度。
- **D2** 裁定席 SKILL §平仓监察案 bug 分支：升件行→**INSERT
  fin_ticket_draft**（signature=`postmortem:<anomaly>`），升件行
  保留携 `draft#<id>`。
- **D3 dm_queue review 扩诊断支**：`fin_econ_lesson
  kind='rootcause_diag' AND status='drafted'`→输出伪键
  `DIAG-<lesson_id>`（COS 评审 P2 修补：不坍缩到 T-0431，逐案一键，
  幂等天然）；会话壳按伪键开班，裁词对案不对票。
- **D4 verdict_reflow.sh**：归流执行剥离确定性脚本（扫未归流
  裁决/驳回诊断草案行→翻案/INSERT draft/三轮升件，幂等键
  `ruling#<rid>`），挂 o-mech 尾窗；裁定席 SKILL 归流段退役。

**段 D 判据**：人造 `rootcause_diag drafted`→CTO 会话开→裁决行→
verdict_reflow 产 draft→ticketdraft 出票；两并发草案各得独立伪键。

### 段 E 闭环与吞吐（依赖 A、D）

- **E1 案卷回写**：部署成功后 old..new 票号集查 draft→lesson 谱系，
  `verdict_ref||=' deployed:<sha12>'`+ops_event 归流行；复发=新案+
  diag_attempts 累计。
- **E2 验收零复跑**：施工收口前 rebase origin/master 盖戳→merge
  `--ff-only`→戳复用不复跑；非 ff 才复跑。落两职卡。
  **前置=A3 已生效**（戳在主 gitdir 可验，否则本项不成立）。

**段 E 判据**：ff merge 后 push 零复跑过闸（A3 闸下戳可见）；
案卷行含 deployed: 后缀；非 ff 路径仍走复跑不降级。

### 段 F 漂移对账（业主令 2026-11-08 增段——GitOps 式期望态/实态
### 对账；依赖 A 的 o-mech 尾窗）

> 病灶实证：unit 文件 6/20 漂、二进制无 sha 查询面、迁移链 145 vs
> 树 147 靠撞发现——drift 五类各散各域无一本账。

- **F1 期望态声明**：`deploy/manifest.tsv`——一行一类：unit 名→repo
  件路径 / binary 名→cmd 构建路径 / 迁移域→migrate_table 期望版 /
  timer→应 enabled 态。repo 件=唯一真源，live 件=被对账面。
- **F2 只读探针**：`scripts/meta/drift_probe.sh`——实态采集零写径：
  `systemctl --user cat` 哈希 vs repo 件哈希；`go version -m` 取
  二进制 vcs.sha vs 构建期 sha（无嵌入=drift 类「sha 不可考」）；
  `schema_migrations` max vs 登记表 max；unit/timer active+enabled
  实态 vs 期望。drift 行→`gauges.tsv`+ops_event 观测行（说明类词
  形——不进票相位，段 B1 先例）；挂 o-mech 尾窗每拍跑。
- **F3 自收敛分级**：安全类（unit 文件按 repo sync、二进制按已并入
  HEAD 的绿戳 sha 重建）可机修——执行+留账+回探转绿才算闭环；
  非安全类（迁移欠账、master↔origin 落差、sha 不可考、钱径域任何
  文件）**只升件不碰**——持续漂移 ≥2 拍→升件行（接段 D 案→票管）。
  机修须双拍确认 drift 仍在才动手（探针误报不得触发写动作）。

**段 F 判据**：①人造 unit 漂移（改 live 件一字）→探针下拍报出含
该单元 drift 行；②人造迁移欠账→报；③机修类收敛留执行账+回探
转绿；④钱径域人造漂移→只有升件行零写动作实证。

## 五、判据总则

- 每段桩测新件随码落（mech_tick_test/dm_queue_phase_test/
  dm_ledger_proof_test 等，O_SHIFTEND_WINDOW_SH 同式喂三态壳）；
- 全谱跑零新增 FAIL/SKIP，报绿必附 RUN/FAIL/SKIP 数；
- 时延阈按跳写：事件→唤醒≤10s，部署行落账不设时延阈（部署本身
  分钟级，判「发生」不判「快」）。

## 六、禁线

- 禁新建长寿 bash 守护件/自建事件机制；禁双执行者（o-mech 上线后
  dm_tick 不留 spawn/push/deploy）；禁自立判断语义；
- 禁动并行硬顶 2；禁碰钱径；禁用无头席施工/裁决本件；
- 禁自愈重试/禁冻结期强发/禁无戳部署/禁推未绿 sha/禁推未并 HEAD 的 sha；
- 回退=git revert 对应段 commit（①搬尾段后无「stop 即回现行制」，
  票面不承诺零迁移回退）。

## 七、裁决记录

- [x] COS 段 A 验收：通过（桩测全绿+判据六条实证；merge eb4c3e6，单元
  装起，HEAD 包含闸现役）
- [x] COS 段 B 验收：通过（桩测 38/38+ledger/mech/dispatch 四套全绿；
  merge d6b7b90，部署期逮 roster_rw 漏授已补——半成品窗实证）
- [x] COS 段 C 验收：通过（五套桩测全绿+负目录落码实证；merge af9b4e5，
  merge 侧迁号 fin_0148→fin_0149 避让 t0485）
- [ ] COS 段 D 验收：＿＿
- [ ] COS 段 E 验收：＿＿
- [ ] COS 段 F 验收：＿＿

## 八、段C 施工结果（回填 2026-11-08 施工席——外派施工，待 COS 独立复审）

分支 `gw/001-mech-segC`（基于 master@f7fd69559；串行三 commit：C1/C2/C3；
未 merge 未部署——merge+部署归 COS 同席，迁移 fin_0148 随发窗生效）。

**C1 退场唤轨**：dispatch_worker.sh systemd-run 加
`--property=ExecStopPost=-/usr/bin/systemctl --user start --no-block o-mech.service`
（字面实证 systemd 255 整串拆 argv——o-c1probe 探针 `systemctl show -p
ExecStopPost` argv[] 五参齐+ignore_errors=yes 在案）。**半成品窗评估**：
属性只罩改码后新拉起单元；改码瞬间在飞 worker（实证 T-0488/T-0489）退场
无此钩——由 o-mech.timer 5min 兜底拍吸收，候队留 dispatched 账不留断链
（最多迟一拍不丢派；窗随在飞单元自然退场闭合，无需人工迁移）。补拉循环
rc75 break→continue（本票候水续试下一票；dm_depgate 即派循环 break 语义
不动——票面钦定「补拉循环」为限，depgate 派单行先落+补拉窗承接）。

**C2 按族入册+负目录**：fin_0148 registry 增键 `rootcause:unit_mismatch`
（diagFamilies 全仓唯一族）+`postmortem:*`，值=经验库条目录（诊断/复盘类
安全面）；族↔册对账桩（diagnosis_test）——新族漏补册=测试红。负目录落码
`scripts/meta/autoapprove_negative_catalog.tsv`（internal/exec 字面+
internal/trading 全谱+internal/venue+internal/ledger 全包 fail-closed），
dm_autoapprove 开拍全册扫描，值径命中=册坏级零批 rc1 升件（判据③机检
断言）。滞留阈 86400→7200。已批 7 天未动→回候选+升件（批件保鲜律）。

**C3 水位机检+派单序**：dm_queue.sh 加 `candidate` 面（view+legacy 双径，
同刻并列反词同族）；tools/ticketdraft 物料化前查 approved+candidate≥5
（O_TICKETDRAFT_WIP_MAX）→留 pending 不占号+「水位闸:」升件（拥塞/
TICKETDRAFT-WIP，24h 去重；计数败同律 fail-closed）；同签名族 24h 并票
（status='merged'+ticket 载吸收票号+「并票」说明类行，批内内存镜防同拍
双物料化）；派单序单源 `scripts/meta/dispatch_order.sh`（(优先级rank,
已批ts,票号)——P0/P1/P3 显式覆写，P2=族映射档；postmortem:*/rootcause:*=
0>写败:*/账链断:*=1>观测盲:*=2>其余=3；已批ts 查败 fail-open），mech_tick
补拉窗+dm_depgate 即派面双消费。dm_queue dispatched 语义零回改（只增
candidate 面）。

**桩测账**：mech_tick_test ①–⑩（⑨rc75 续试/⑩派单序过链）、
worker_runtime_test（ExecStopPost 属性断言）、dm_autoapprove_test 六场景
（负目录命中零批/表缺同律/2h 阈边界/7天回候选+升件一次/扫描败滑行）、
dispatch_order_test 七场景、dm_queue_waipai_test 43 断言（candidate 双径+
同刻并列）、dm_depgate_test（+派单序场景；**随手修**：fbfc39e22 模换
swe-2-flash-high 未同切本测试三处字面 grep——在用中证错必修，master 上
本测试先于本票已红）、ledger 包族↔册对账桩、ticketdraft go test（闸纯
函数+wipCount 壳测+并票/闸 SQL 面）。

**判据四条预证**（COS 复跑为准）：①首张退场→次张拉起：ExecStopPost 链
机检在案（worker_runtime 断言+o-c1probe 实证），时延链=worker 退场→
ExecStopPost 即起 o-mech→补拉窗秒级续派（live 复跑归 COS）；②rc75 不阻
断次票：mech_tick_test ⑨；③负目录签名人造草案→机批断言拒：
dm_autoapprove_test ②（册值径 internal/trading/okxexec→零批 rc1 升件）；
④WIP 闸到 5 停出票：ticketdraft TestWipBlocked（5=true 边界）+TestWipCount
（双面计数单源）。

**偏离说明**：①段C2「已批 7 天回候选」落在 dm_autoapprove.sh（批生命周期
管家件）非 ticketdraft——回候选是批面动作，起草物料化件不辖批后生命周期；
②registry 新族键值=经验库条目录（诊断类安全面）——若段D 修法票沿用
rootcause:* 签名，其钦定径将由 DM 裁时改判（模板行语义保留），负目录断言
兜底防钱径误映射；③dm_depgate 即派面同受派单序（「派单序」钦定覆盖两派
径，仅补拉面排序=死信面）。
