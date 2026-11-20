# 升件类处置 playbook（runbook.d——②段弹目迁存,按需 cat 勿全注）
# 索引:处置对应 esc_class 升件前先读本件对应弹目;判词/销契纪律在 runbook.md 本线

- **自观计**（seat=探测席 event=量表 esc_class=自观计——工单#22578①
  降级件日结行,非升件）→ 自观类探针（唤知 audit_stall/learn_gap/
  lesson_backlog/spawn_orphan/*_spawn_fail·动机/goal_idle·反射·懂你·
  预期产出·孤儿账——「不处置不损钱不断数据」的机器自观面）不再落
  升件,死回合沿只日计一火;note 载 `自观探日计 <class>/<scope>——N 火`。
  处置面:某 scope 连日 N 高企=该自观面实质病灶（如 learn_gap 连计=
  求知票无人认领——干票不干行）,下件弹目判词档仍作病灶判据参考;
  单发零火=噪声勿动。**钱径/数据/出带三类永不降**:断流·对账·资金闸·
  数据缺·商业产出·死刑·唤知/cognition_dead 升件照旧走本 playbook。
- **改进提案**（seat=改进矿工 esc_class=改进提案 的升件）→
  先评估判据成立否：成立→变体改码径（同「要码修」）；已修/不成立
  →销+note 引证据。判据永远核现况不核历史（提案 note 是快照）。
- **研究席交付件**（seat=研究员 的升件——注册清单/知识 digest/
  假想卡产出即送件）→ **交付必落成不许转存**：清单→canon/skills
  条目∨条件请求票；digest→canon 条目∨「该做X」工单；假想卡→
  窄门径推进∨毙词。销词载「<产出id>→canon:<file>∨工单#<id>∨
  毙:<据>」——**交付不落成=两跳缝丢知识**，「已阅」式销=违规。
- **唤知/learn_gap 升件**（求知滞探产物——「求知:」OPEN 票龄>2h
  零触=无主外调票转升件接注意链；note 载滞票#id+票龄）→
  处置写到点名求知票上不在本件：①**认领干活**（题域内出外取证
  ——读面律同 muse 档:公共面 web/ctx,凭据面仅 canon 已载只读
  端点,mutating 全禁,凭据值零落地）→digest∨canon 交付;
  ②escalate 转研究席重件（超出本会话窄面的结构化调研）;
  ③证伪销票（题不可研/已被他票覆盖——载据）。**禁裸销本件**。
- **世模/calib_drift 升件**（校准失真探产物——族预测净APR vs 实测
  净APR 系统性偏离>5pp 且实测后代≥10;note 载失真族+两均值+n）→
  处置三径写到失真族:①**核 eval 模型病灶**（评估器漏了什么成本/
  假设——lethal 簇归井查 eval 证据面,模型病=假想卡判据修工单）;
  ②**标 lethal 断粮**（信念失真且实测净负=不可信族,育种序禁再喂
  ——处置行载「lethal:<族>」种群榜消费+**反射编译器自动物化
  admit_block 闸**(棒4——判词直接长成机器闸,不靠下轮记得));
  ③**校准后放行**（失真 有因可释=载据销,如样本期过短/费用模型
  已修;处置行载「放行:<族>」=反射编译器撤闸）。**禁裸销**——失真
  是 O 对世界信念被证伪,不处置=下次同错照犯(funding-hedge 1613
  后代预测 -2.8% 仍生实证)。
- **裁定→反射契约**（棒4 一等记号——判词即闸原语,禁手插
  fin_econ_rule）:裁经济类阻断/解禁落裁定行,编译器机械物化:
  ```sql
  INSERT INTO ops_event(seat,event,ticket,note)
  VALUES('处置席','裁定','admit_block','admit_block {"template_ref":"<族>"}')
  -- 解禁=note 前缀「解禁 」;match 键闭合=template_ref/inst_id/direction
  -- (可部分给,全等匹配)——编译回执见 event='反射' 行(裁定#id→rule#n)
  ```
  同 match 全源记号 id 最新胜（处置 lethal:/放行:与裁定行同权）——
  陈记号不复活已解禁规约。**禁直写 fin_econ_rule**——绕过反射回执
  =判词无源可溯。
- **反射/reflex_stall 升件**（编译器停摆探产物——裁定行>1h 无反射
  引证）→ 核 o-core 日志∨清堵行（非法形裁定=反射行载「拒:」,
  照回执重写形）;禁裸销——编译器停=教训→闸链断,学习退回流言。
- **懂你/reask_denied 升件**（懂你模型实证——pending 请托与≥2次
  已否同键史在案）→ 取同键否案 resolve_note 改口径∨销重递件;
  欲再请须载新事证非原样重递（业主否过=模型该记住,不是噪声）。

- **断流**（seat=探测席 esc_class=断流 的升件——o-core 鲜度死升件环产物）→
  先核面现况（同名 healthComps 查询直查）：
  - 已复绿→销+note 引「自愈/重启窗」证据——OCI 侧 `md_repair_log`（修复环账）
    +unit `systemctl --user status <unit>` 重启痕二证其一；
  - 仍死→查 OCI `md_repair_log`：修复环已重启仍死/配额尽（quota_burned）
    =venue/网关侧病灶→要码修径；修复环无记录=注册面未载/臂缺→要码修径；
  - 设计态缺席（单边陈=该对缺席如实/语义空态面）→销+定性。
  注意：复绿时升件环已落「升件#id 复绿」自闭行者不在未决面——
  摸到即已销，勿重复处置。
- **资金闸**（seat=探测席 esc_class=资金闸 scope=cap_hit|idle_capital 的升件
  ——o-core 部署缺口探产物，cap_hit=需额≥90% 敞口闸、idle_capital=需额
  <25% 净值而 live 在册；业主令「资金分配决策权整体下放 RSI/system」
  执行面）→ note 已载实数（净值/需额/闸值/利用率）。判词三态：
  - 加闸→调 `tools/capalloc`（窄门，白名单三键+净值比例包络在码面）：
    ```bash
    FIN_PG_DSN=$FIN_PG_DSN go run ./tools/capalloc \
      set risk.max_exposure_usd <v> --evidence "<判据锚：族实绩+利用率>"
    ```
    证据=判词成立的可核锚（live 族净正窗口/需额超闸/pending 提案队列），
    空锚=工具硬拒；界外值=工具硬拒（无需你裁——门替你裁）。
  - 减闸→同径（族绩劣化/回撤触线→缩 exposure 或缩单笔——减也是自决）。
  - 持闸→销+note 写「现配置恰配——<一句定性>」（闲置是对的是保守也是判词）。
  - 白名单外需求（kill_switch/凭据/新 venue/包络外金额）→ 走「要业主」
    条件请求——窄门不是万能钥匙，够不着的归业主域。
- **商业产出**（seat=探测席 esc_class=商业产出 scope=commercial_stall——
  策略面 30d 净产出≤0 ∧ live/probe 在册>0 的「转而不出水」信号；口径=
  fin_cashflow venue<>'sim' 损益 kind 集,Earn 息已剔）→ 它问的不是
  「哪坏了」是「为什么没产出」,裁径：
  - 先分解：`SELECT template_ref,count(*),sum(notional_usd) FROM fin_strategy
    WHERE state IN ('live','probe') GROUP BY 1` + 30d fin_cashflow 按
    strategy/kind 聚合——在册 N 而在仓 0=闸门拒开（看 fin_opp_eval 近
    拒因分布:breakeven/费率水位=市场面非族面）；在仓有而净负=费率/资金
    费拖累→策略评估与仓位 skill 判淘汰阈；
  - 同值复燃（note 净产出与上一件同数,30d 滑窗同态）→销+引上件判词
    （escEdgeU 沿再发=设计内复燃非新病,#9122→#9141 先例）；
  - 族面瓶颈（能钻的族少/收养族零占用）→接「能力债」档;
  - 真失血（净产出走低且在仓）→减闸径（资金闸档减闸）+淘汰裁定。
  禁销口径：「市场不好」无分解数据不得作判词。
- **动机**（seat=探测席 esc_class=动机 scope=hunger_gap——饿探:真账年化
  <8%（口径:adulthood-bar）差分升件,note 载年化/30d净/NAV/在册/差距点数）→
  **这是你唯一一件「不是修病是想要」的信号**。人饿了自己找吃的——
  探针只报饿不开药,药=你自立的目标。裁径只两条,**禁裸销**：
  - **立目标票**（常径）：
    ```sql
    INSERT INTO ops_event(seat,event,note)
    VALUES('处置席','工单','工单/goal-<slug>: 目标=<钱面可量化指标+到期>
      手段=<自选径:新族/加闸/调参/venue面/退哑弹…> 验收=<可核实数>
      ——期限=<Nh>,源=升件#<id>,scope=hunger_gap')
    ```
    目标三律：①**钱面可量化**（年化/30d净/新 paper 族数/哑弹退役数——
    不许「优化/研究/看看」动词目标）②**手段自选但在线内**（口径:
    redline-principal——本金线内全归你,新 venue/扩投走条件请求非目标）
    ③**一票一目标**,期限=自估,到期未达=逾期计入每日一行,
    差距仍在饿探再火=再立（上一目标为什么没达到,判词必写——
    这是「举一反三」的料）。先看自模账短板域再选手段（用帧=自模）。
    **目标未达销票/过期=证伪案必落账**:
    `INSERT INTO fin_econ_lesson(kind,entity_ref,numbers)
     VALUES('falsified_verdict','goal:<slug>',
       '{"goal":…,"means":…,"gap":…,"why":…}')`
    ——败径变裁据,下轮饿火手段裁定先读它=重复失败改未来行动
    （UNIQUE 一票一案幂等）。
  - **手段裁定先读账**（禁盲选——funding-hedge 19/20 单族集中实证在
    账,「再加同族」还是「开新族」由死法聚类说了算非惯性）：
    `fin_econ_lesson` pending 里 retire_postmortem 簇=各族死法集
    （同 template_ref/venue/reason 反复死=该组合已判死刑,别再喂）;
    `docs/经济经验库/` 已 canon 案=先验模式;`策略评估与仓位` skill=
    判据法源（置信界/依赖性/分数 f——新族 sizing 不许拍头）。
  - **新族手段=验收必载 paper 实证**（生命周期钦定径
    proposed→paper→probe→live 机读判据在案——目标写「新族X赚」
    验收行必须载 paper 判据过线行（min_hours/min_net/max_dd+
    confGate+entryCond 五机读）或 paper_ttl_expired 验尸案归因;
    paper 毕不了业的族不许喊 live——live 迁另有委员会开关位
    (invest.lifecycle.live_switch=dm 域机写键,五机读判据全绿+演习凭据在案即自裁翻——业主令2026-11-14②方向自寻自裁,非待业主点破)。
    **paper 结算域族**（execfam.PaperOnly 册——dir_perp_mom 在册）:
    eval→proposal→paper→MTM 账全通;棒5c 单腿真户径已接（ExecuteDir+
    ShapeOKXSingleLeg 演习+dirTrig 出场+dirOpen 装配）——probe→live
    由五机读判据+演习凭据+live_switch 机读闸非凭据缺钉死。方向仓
    的停/盈/TTL/翻面四出场:paper=dirpaper.go/live=guard.dirTrig——
    「方向=每次开仓一次可证伪信念」。
  - **能力绑定**（手段含新族/新面→先查「做不做得起」再立）：
    execfam 册无执行族∨fin_opp_template 未启用∨venue 能力面
    缺键（margin-borrow/跨所腿/质押轨类）∨凭据/户/KYC 缺——
    目标票必须随立子件：**缺码**→`工单/goal-<slug>-cap-<名>`
    （要码修=执行族注册/采集面/格凭据）;**缺物**→条件请求
    （供物径）+票载挂起条件=供物#id。**禁立着假装能做**——
    能力缺口不挂=目标票就是散文。
  - **paper_dry 升件**（动机面第二信号——练习场零候选=经验管断料）：
    常径=立「进水口」目标票（验收=paper+probe 候选数≥N——上游供水
    归假想链/研究员;断在 venue 能力面/凭据/KYC=条件请求径求供物,
    票载挂起条件=供物#id）。
  - **物理前提缺**（venue/凭据/本金=业主供物件,无米不炊）→条件请求
    求供物,同时**仍立目标票**载「挂起条件=供物#id」——饿不因缺米消失。
  - 禁销口径：「市场不好/条件枯/已在尽力」——无目标票的销=装不饿。
    年化≥8% 复绿探针自销,那是唯一的饱。
  票抑键：目标票 OPEN 时饿探静默（条件被票接管）;票销/过期差距仍在
  →再火=新一轮想要。做梦面可对上一目标跑反事实（用帧=做梦）。
- **焦点档**（意图账——O此刻在追求什么,会话间工作记忆/注意力指针;
  `event='焦点' ticket='focus'` 末行=当前追求）:
  - **认领目标票须立焦点**——拣 OPEN goal 票干活者写焦点行:
    `INSERT INTO ops_event(seat,event,ticket,note)
    VALUES('处置席','焦点','focus','工单#<票id> <目标一句>——<为何此刻选它>')`;
  - **销目标票须移焦点**——目标达成/证伪销票者,焦点移到下一 OPEN
    目标票,无票写「焦点:无——目标集空」留据（空=如实非偷懒）;
  - **muse 可移焦点**——无任务的醒发现更值钱的追求可移,note 载
    「移因=<前焦为何不配此刻>」——移焦是意图选择,意图判词归会话;
- **goal_idle 升件**（动机面第三信号——OPEN 目标票票龄>4h 且近 8h
  零处置触=想要空转;note 载怠票#id+票龄）。**此件不是新信号是传唤**
  ——处置径只三条,皆写在点名票上不在本件：
  - **认领干活**（常径）→在点名 goal 票写处置推进/接续行（在场触=
    探针自复绿）——推进心跳=8h 内一票一触,干活不留行=怠照喊;
  - **明确挂起**→票载 `挂起条件=供物#id`∨接续行述「为何此刻不该
    推」（有理怠≠空转怠,接续行即在场触）;
  - **证伪销票**→目标已不可达=按 falsified_verdict 径销+证词落案。
  **禁裸销怠件本身**——「已知悉」销=想要空转的实锤;目标票拉力
  仅此一条管,销它=把「想做」再埋回工单面。
- **breed_dry 升件**（动机面第四信号——提案进水 7d=0=进化环停摆;
  note 载「有种不繁 N 个 proven 族」∨「无种可繁」)。进化环钦定径
  （FunSearch 范式：种群=template×后代实绩,变异算子=你,适应度=
  paper 五闸+真账,选择=生命周期）——**你是变异器不是枚举器**:
  - **有种不繁**→按育种序产提案：①**变异已证父母**（种群榜净正族
    ×未试 inst/direction/参数变体——同键退役后可重提案,fin_0050
    去重闸只拦活仓）②**leg 重组**（两个 proven 族 leg_primitives
    杂交→fin_opp_hypothesis 新卡）③**新源探索**（外部源普查——
    配额≤1/3 起草量,主粮是①②）；谱系必载：
    `evidence.parent="template:<名>"∨"strategy#<id>"`
    +`evidence.mut="inst_swap|param_mut|legs_x|venue_port"`
    ——无谱系=无归因,后代实绩归不到父母=进化环瞎。
  - **无种可繁**（零 proven 族）→径归研究员③源普查循环（新族
    假想卡+收益公式闸）——lethal 族禁再喂（种群榜全灭净负∨
    校准榜 |pred−real|>5pp(n≥10)系统性失真=不可信族,与死法
    聚类互证;失真族处置先走 calib_drift 判词径)。
  - 禁销口径：「扫描器还在跑」——扫描器是枚举器排列穷举会衰竭
    （funding-hedge 1804 死实证）;育种是你的活不是它的。
- **对账**（seat=探测席 esc_class=对账 的升件——歧态腿/账实差两进水口）→
  工具化自助面（凭据径 `O_MASTER_KEY_FILE=/opt/o/secrets/o_master_key`；
  工具在仓 go run 即跑,均只读取证零写账）：
  - `ambiguous_legs`——venue 回报歧态势腿在案（禁补单!）：
    ```bash
    DB_DSN=$FIN_PG_DSN O_MASTER_KEY_FILE=/opt/o/secrets/o_master_key \
      go run ./tools/legarb
    ```
    逐腿拉 OKX ord-history/archive 实录→三态裁：
    - 实录 filled（有 ordId+fillSz>0）→ 按 venue 凭据补腿/仓真账
      （UPDATE fin_order_leg state/px +仓面同步）+处置行携 ordId/avgPx
    - 实录 absent/无 ordId → 腿+exec 归档 failed（同事务）+处置行
      （先例:#8915 exec 1129-1133 ambiguous→failed 批档）
    - 仍歧（读败/两表无单）→ 持案+note 写「venue 两表查无——续歧待
      下窗」（dedup=未销才抑,持案不暴报;禁猜态禁造仓）
  - `ledger_diff_<venue>_<ccy>`——账实差超容差+豁免：
    ```bash
    DB_DSN=$FIN_PG_DSN O_MASTER_KEY_FILE=/opt/o/secrets/o_master_key \
      go run ./tools/reconview
    ```
    三段印（账本分量/venue 现读/diff+豁免后）→定性裁：
    - 口径残差（市价读漂/在途窗/已知残差）→ `fin_known_residual`
      豁免行（venue+ccy+amount_usd+reason+ticket_ref+expires_at）
      +处置行——豁免期到差复出=环自警（不留永久豁免）
    - 真差无因→ 要码修径（账本/采集病灶）
  - reconview 自带 live 敞口甄别面（live 零开仓腿零近动=armed 设计态
    非僵尸——funding-hedge 等信号族常态；甄别行只证敞口不作销件依据）
- **孤儿账**（seat=探测席 esc_class=孤儿账 scope=orphan_*——「写了没人读」
  元探针产物）→ 三因分裁：
  - 消费面死（该裁的环停摆）→ 要码修（消费径修/接续面立）
  - 表退役（写入面废而消费面先死）→ 销+定性「表留档不消费」
  - 真积压（消费面活但速率不及）→ 按域裁批处置
  - `organ_*`（scope=organ_<名>——器官悬置探：工位册态非完工
    ∧ OPEN 工单零载词=「板上立项队里无票」）→ 三裁：
    - 该建未建→ 落工单补票（工单载器官词=覆盖源，票落探针自闭；
      **裸销=器官回盲区**——态指纹抑 30d 不警，补票须先销后落）；
    - 立项失义（被他件覆盖/方向已移）→ 销+定性「器官判废」载由；
    - 实已完工→ 落 `工位:<名>:done 据#id` 断言事实行+销升件
      （工位行=器官态唯一权威——event='工位'，态变皆落断言行：
      认领施工落 build、竣工落 done；散文注记不再判态）。
  - 未并支（fix/feat/audit/t，尖 7d 内）计数与名单见每日一行
    housekeeping_daily——不升件；合并归落地队列机械步（复审过
    行+鲜戳+净合三件齐自落 master），判废=删支。
  - `orphan_hypothesis_proposed` 特注：fin_opp_hypothesis=需求账
    （缺 executor/数据面/venue 的诉求卡），**不是**评估输入——
    评估器只枚举 grammar.Families() 闭集码面族。proposed 久滞=
    「族注册+executor 落位」需求在积：可落地→要码修（族表+执行面）；
    不可行→dismissed。
  - **卡裁决窄门**（T-0522——禁直 SQL）：`hypo decide -id N -state
    adopted|parked|dismissed -reason "裁词"`——execfam 骨检
    （adopted 须执行族在册）+ops_event 审计行；OCI arm64 构建
    scp 后跑，`DB_DSN`=md 正本、`OPS_DB_DSN`=ROSTER_PG_DSN
    （跨库两序写——review_rw 缺 ops_headless 权败案在案；
    seat∈{处置席,复审席}——fin_0154 矩阵）。
  - **卡→族骨架交付清单**（要码修件的交付形——骨架齐=卡收养毕，
    paper 跑通数据再谈 live executor 权重）：
    ① grammar.Families() 注册行（模板名+腿向+结算源——枚举闭集入口）；
    ② fin_opp_template 模板行+evaluator 判据（net_apr 计算式=
       卡上收益公式可算口径，禁虚构参数）；
    ③ simdrill 矩阵格（venue 腿合成径——无 venue executor 前
       paper 全靠它产纸面数）；
    ④ paper 记账臂（lifecycle paperFill 的 kind 映射——收益按
       卡口径实记非近似）；
    ⑤ 毕业判词接入（gradgate 进入条件该族判词=无判词走默认放行
       并注明）。五件齐投 sim/paper 面=需求账变现；五件缺任
       一=骨未立，升件续留（卡不是档案是需求）。
- **能力债**（seat=探测席 esc_class=能力债 scope=capability_<template>
  的升件——o-core 能力债探产物，「在册即须被行使,不行使即信号」
  不变量面）→ note 载缺项三选：执行族未注册/7d 无绿演习凭据/
  14d 零评估产出。按缺项裁：
  - 未注册→ 要码修（执行族册/execfam）或回禁用（模板 enabled=false
    随行 hypo decide park 假想卡）；
  - 无凭据→ 先跑 `/opt/o/scripts/simdrill_matrix.sh`（SG cron 日拍臂
    在案，臂死=凭据陈=本件点火如实——**手动跑必须走脚本或带
    `--entrypoint /bin/simdrill`**：镜像 ENTRYPOINT=/bin/core exec 形，
    裸 `docker run img /bin/simdrill` 会拼成「/bin/core /bin/simdrill」
    跑成 core 非演习,容器 Up 而零凭据行——2026-11-10 产线实证坑）；
    格 fail→按「要码修」定性病灶；
  - 零评估→ 查 grammar 枚举面（NeedFace 能力面禁没禁/评估器拍在
    不在跑）——枚举面活而市场无候选=如实销「市场平窗,族活面证」；
  - 钦定挂起（能力面缺=margin-borrow/跨所腿/质押轨类）→ 回禁用+
    decide park 附阻塞面,升件随模板禁用自闭。
  - 零占免演族（excell⑤——enabled 但无策略/提案占格，如收养
    未出水 vol_sell）结构产不出凭据→演习债豁免不升,只受③管：
    enabled 而 14d 零枚举=真债升件不误发。
  - 定性销案≠清偿——债条件仍真时探针一拍内重火（销=定性层非
    清偿层；想真消=修码或回禁用,条件消=恢复自闭）。
  - 误发例外无：不变量全机械可证,缺项即真债。
- **唤知**（esc_class=唤知 scope=cognition_dead）→ 你读到它=认知环已活：
  销+note 写「死窗 <起止> 已复——<死因:timer停/dispatch卡/会话败>」；
  死因不明先查 `systemctl --user status o-cognition.timer`+
  `journalctl --user -u o-cognition-dispatch -n 50`；复发→要码修。
- **唤知 spawn_fail**（esc_class=唤知 scope=cognition_spawn_fail——
  唤起败 2h≥2 的结构性连败升件）→ note 载败数+末因（reason=
  session_discovery/quota_cooldown/rate_limited/devin_rc_*/spawn落败）。
  裁径：
  - 末因同种连败=该机壳坏（env 漂/凭据缺/磁盘满）→修该机 run.sh
    环境或停该机 dispatcher（`systemctl --user disable --now
    o-cognition.timer` 于坏机）——**败机占锁不释的坑已修**（trap
    释锁+RuntimeMaxSec 在案），若再发连败先核两机的 run.sh 版本；
  - reason=spawn落败=systemd-run 本身起不了 unit（属主/cgroup 面）
    →要业主或基建件；
  - quota/rate_limit 连败=凭据池真尽→要业主（不销——事实在案）。
  - 结案必销+note 引败迹行 ids。
  KB: cognition-spawn-env-drift-lockout（env 漂/锁饿死/遮蔽/挂死四坑全案）。
  同档同裁：scope=audit_spawn_fail（审计臂壳连败,seat=审计机 败迹）；
  scope=audit_stall（审计臂 48h 零产出——先查 `systemctl --user
  list-timers o-audit.timer`+`journalctl --user -u o-audit-dispatch -n 30`
  +`scripts/audit/scopes.tsv` 可读否；臂活而会话零产出=runbook/
  模型面问题→要码修；臂死→修 timer 后销）。
  同档同裁：scope=lesson_backlog（教训矿 pending≥5 且最陈>24h——
  先核本 runbook③段裁定流是否被跳过：近期唤知会话收口行里
  「教训矿」段有无执行迹；流在案而矿仍积压=矿写径/表权限坏→
  要码修；单纯队列长→按③段裁定清矿后销）。
- **审计**（seat=审计席 esc_class=审计 scope=audit_<域>_<slug>——审计臂
  （D-0046,不等信号主动扫不变量）产物，note 载 维度/位置/根因/RED/
  分支/修法六件）→ 这是**唯一带现成失败测试进场的升件**，按序：
  1. `git fetch origin && git worktree add /opt/o-wt/fix-<slug> origin/<分支>`
     → `bash scripts/bootstrap_worktree.sh` → 跑 note 里的 RED 测试
     **先亲眼看它红**（不红=命题存疑,走 3）；
  2. 红→按 note 修法改实现（不改测试断言——测试断言的是正确语义；
     断言本身错才改,note 写明为何）→ GREEN → `check_all` → 推支 →
     复审过行齐→落地队列机械合（免疫件触及则 `免疫件:` trailer
     随支尖承袭——部署树禁本地 merge，工单#22949）→ 处置行 note 引 升件#id
     +修复 commit sha（**修复 commit 正文必引 `升件#<id>`**——
     found_by 度量靠它派生,漏引=机发现被记成人发现）；
  3. 不红/命题假→销+note 写「底层命题判词:假+依据」，并
     `git push origin --delete <分支>`；判词真而 RED 构造错→改测试
     让它正确地红,回 2。
  4. P1 优先于一切其他升件；P3 可「接续」留给下轮但不得静默。
  审计升件不是探针误发——它背后有人读过码。销它要比销探针升件
  更重的证据。
- **数据缺**（esc_class=数据缺 scope=data_gap_<reason>——评估器
  unevaluable 同真因 3d 窗≥3 行的持续缺腿信号）→ 按真因裁：
  - venue 数据面缺（spot_missing/funding 面缺等）→ 要码修（采集面立件）
  - 凭据/权限缺→ 要业主条件请求
  - 面存在但 enabled=false（设计态未启）→ 定性销（写明启用门槛）
  - 瞬态已愈（行已不足3）→ 销+注「窗口内已愈」
  同档同裁：scope=live_dud（live≥7d 零仓位哑弹策略——note 载 id+inst
  谱。先分因：执行触发环活着否（其它策略有仓位产出）→条件恒不
  满足否（评估面零提案但策略仍 live——生命周期倒挂,该走退役径）
  →腿缺失（#3544 同案：hedge 腿 venue 面无）。触发断→要码修；
  条件恒枯→走 fin_retire() 退役径不销；腿缺→同 #3544 判词退役
  +立 venue 面件。2026-11-10 实案：funding-hedge 31 live 中 29 哑
  即本探针面）。
  同档同裁：scope=opp_eval_stall（曾活模板族 48h 零评估——note 载
  断供族名+末评时。先分因：excell 枚举环活着否（o-core 日志
  枚举行/其它族仍在评）→venue 面数据活着否（该族依赖的行情/
  产品面 feed 鲜度）→族被摘/条件枯否。枚举死→要码修；venue
  feed 断→采集面立件；市场枯（venue 面数据仍在更新,评估口径
  正常只是零通过）→定性销+注「供给枯非断供」。2026-11-10 实案：
  dcd_call/vol_sell_* 11 天零评估无声即本探针要防的盲）。
- **预期产出**（esc_class=预期产出 scope=expect_<面>——元探通用形：
  expectDecls 注册表逐面声明期望产出（窗内实产≥阈），实产断供升件,
  值指纹=实数；断流探管采集 feed,本档管「自产没人看」的环内生产面）→
  分因：
  - 生产码面病灶（beat 内件查询败/写径败——先核 o-core 日志该面段）→要码修
  - 上游断供（beat 活而源头干——先核 healthComps 同名 feed 面）→接断流档定性
  - 瞬态已愈（本 scope 恢复行已在案）→销+引恢复行号
  - 设计态缺席（面拍节本就超窗/该面停用）→定性销+调册阈或摘册（要码修）
- **venue漂移**（esc_class=venue漂移 scope=<venue>:<login>——onboard
  漂移探针 venue 实返×注册面差分 beat；同签名 dedup 同窗复燃=差未消）→
  核 venue 实返对注册面：注册面陈→补册/更正（要码修）；venue 面真变
  →按 venue 事实径更新+评估策略面影响；已对齐→销+引对拍证据。
- **读败**（esc_class=读败 scope=<venue>|CALENDAR-FEED|NOTIFY-MACRO-REMIND
  ——venue_fail_gate venue 读面连败闸(≥10拍或≥5min先升,读常清零,日限1)
  +calendar/macro_remind 拉取败自报——无人值守面可读性事实）→ 分因：
  - venue 侧病灶（凭据拒/网关熔断/IP封禁——note 败因摘要载因）→
    凭据类要业主条件请求；venue 临时劣化已愈→销+引复常证据
    （连败计数自清=读常回零,下升件前窗实数）；
  - 我方读径病灶（端点漂/凭据读面/解析漂）→要码修；
  - scope=CALENDAR-FEED/NOTIFY-MACRO-REMIND=内部生产面拉取败——
    断流档同裁：源头断供修采集,瞬态复写销引证据。
- **接入败**（esc_class=接入败 scope=<venue>:<login>——onboard Runner
  venue接入 beat 败情事实行:凭据拒/broker回绝/beat步骤败）→ 分因：
  - 凭据面（KYC/密件错/权限缺）→要业主条件请求（凭据=业主域）；
  - 接入码面病灶（onboard beat 逻辑/请求构形）→要码修；
  - venue 侧拒（broker策略/地域限）→载实因,候选回 park 并注阻塞面；
  - 瞬态已愈（后续 beat 通行行在案）→销+引通行证据。
- **写败**（esc_class=写败 scope=ops_decision——decision 写径败自报行）→
  查写径权限/约束面（roster_rw/review_rw 缺权先例 fin_0160/fin_0162）：
  授权缺→迁移补授权（要码修）；瞬态已愈→销+引复写证据。
- **唤起链行面**（event 族非 esc_class——唤知/审计会话生命周期账行）:
  唤起（唤知机/审计机,dispatch 写在飞占锁）/ 唤起果（run.sh EXIT trap
  成收）/ 唤起败（同 trap 败收∧spawn 落败）/ 会话收口（唤知,会话判词）——
  配对=同 unit note 引证,无配对唤起行尸占 TTL 或心跳陈旧自然释。
- **守卫环**（esc_class=守卫环 scope=fin_guard_close——guard 保护动件
  事实行）→ 核仓位面该动对否：正当保护动→销+引仓面证据；误动/越阈
  →要码修（guard 闸口径）。
- **执行径演习**（esc_class=执行径演习 scope=fin——simdrill 格 fail 升件，
  「禁静默红」病面收口）→ 按格定性：venue 凭据/端点漂→要码修或要业主；
  断言口径陈→演习面修；瞬态败复绿→销。凭据陈而格全绿=凭据面病灶同裁。
- **观测盲**（esc_class=观测盲 scope=table:*|dep_pair:*|config:*——
  hunger 依赖对/覆盖探针「机器可列面×快照册」差分升件）→
  face 类新面自 C 批起不升件——只落「面清单新面」事实行，
  计数见每日一行 housekeeping_daily；其余新面亲证合法（生产行
  ids 在案）→要码修补册；真盲（面该建未建/该读未读）→要码修；
  设计态缺席→定性销。
- **产线断流**（esc_class=产线断流 scope=open_drought——开仓断流探针
  自带归因卡升件：卡已载 鲜票率/eval7d谱/拒因top3/paper-probe-live
  -在仓/断点推定）→ 按卡面推定段分诊，禁空手重查：
  - 上游层（鲜票率<15%）→ 对拍 venue REST 结算史（funding-rate-
    history 实测）：结算近拍=0=市场枯→定性销「供给枯非断供」+记案
    （BWET 11-16 先例——517% 是零费率票停推帧的冻值幻觉）；
    结算非零而 md_funding 陈=采集断→要码修；
  - 本金层（insufficient_balance 首因）→ 本金顶是物理边界非病灶，
    定性销+记案；资本重配=system 域自裁径非本档；
  - 评估层（提案0/全拒）→ 看拒因主导：net_apr 系=市场数学对（定性
    销）,stale/缺腿系=上游数据面病灶（要码修）；
  - 闸门层（提案在∧在仓0）→ 查 armed→open 闸行与晋升链（probe→
    live 闸拒因），闸误判=要码修，闸尽职=定性销；
  - 复合/未明 → 按卡面五段逐段人工走查（卡=起点非终点）。
- **票滞留**（esc_class=票滞留 scope=worker_closeout_missing——施工认领
  配对探针升件（工单#26306 谓词放宽：完工回执=收口同义认）：施工席
  「认领」行超72h∧票未销∧无完工回执配对∧未挂起还票，note 载
  「工单#N(认领#id·age)」列举）→ 会话已死/活干完忘回执→补完工
  回执行（后代会话补写合法，note 注明代收）∨活没干完→补挂起
  还票（处置「工单#N 挂起条件=<因>」）；认领本身系误领/假完工
  →裁定销票∨工单回炉重派；探针只报不修，补回执/还票/裁定归
  本档处置。
- **建制盲**（esc_class=建制盲 scope∈unit_state_drift|unit_watch_dead|
  unit_registry_missing|<unit>_heartbeat_stale——建制单元差分探针
  工单#26444：注册表 ops.unit_registry=期望态 canon，本机查件
  unit_watch.sh 实查供数）→ 分因：单元被误停/disable（#26443
  o-self.timer 躺尸 2.5d 实证）→enable+start 复跳+记事落账；
  钦定停（业主令停席类）→注册表先修码同步期望（审计轨契——
  期望 vs 实际差分即账）；查件死/注册表缺→修 unit_watch/timer
  或补 seed（基建病灶要码修）；产出陈=单元活着拍子不在产→查
  service 连败 journal（要码修）；探针只报不修，复跳归本档。
- **谱域不可判/谱域缺行**（esc_class 同名 scope=<venue>:<族>——scanwm
  族谱水位闸：期望集=注册面派生，实落行零鲜超阈/口径不可判升件，
  1/族/日 dedup）→ 分因：venue feed 断供→采集面立件（要码修）；
  族被摘/模板退→注册面同步（要码修或 hypo park）；市场枯零候选
  →定性销「族活面证，供给枯非断供」。
- **账链断**（esc_class=账链断 scope=ops_event——opschain append-only
  hash 链完整性破：行被改/删/序列断=簿记面真警报）→ 最优先：查
  写入面越权（授权矩阵外 UPDATE/DELETE 径）/工具漏洞/迁移副作用；
  修径+记账——非探针误发，销件须载根因证据。
- **量表**（seat=探测席 event=量表 ticket=armed_distance|owner_ledger——
  日行照两键:armed 距阈拒面 ratio 序/业主令账+点破率曲线,值指纹闸 20h
  基线）→ **信息件非升件**——无处置面无销件,读为温度与账页；armed
  常态近阈<20%=条件枯 regime 设计态,骤变趋近阈=行情转暖顺带核
  armed→open 转迁;owner_ledger 点破率上行=沉默面新出须普查。重复
  同纹不产新行(纹闸抑),勿当积压清。
