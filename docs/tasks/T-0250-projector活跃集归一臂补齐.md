# T-0250 projector refreshSets 活跃集归一臂补齐——basis 现货腿+dcd uly（T-0216 根因②根修）

> 状态：**收口待审** | 领取：devin-t0250 @ 2026-10-30 | 优先级：**急诊**（钱径——fill 锚现役于
> dualwrite 旁写泵顶水，非正形不可长持；急诊抢占=管理令 MANAGE-82 钦定）
> 执行径：**无头** | 收口commit：3abe67647b7ca36377280876b05df02f83ddf272
> （双臂码面=510439e、KB 条目+索引=8dbb8d3、喂集断言桩=3abe676——撞车移交多 commit 合为交付，见结果回填偏离说明）
> 立项：决策席 @ 2026-10-30（管理令 20261030-012400-MANAGE-82）

## 面况

T-0216 上线排障实证根因两段：
- ① collector 旧件未订现货腿——已办（arm64 新件 17:12Z 重启，OCI md_ticks
  BTC-USDT 流实证）；
- ② **projector refreshSets 活跃集=裸 inst_id**（fin_position 未平∪
  fin_strategy 非retired，`internal/projector/projector.go:125`）——缺
  basis 现货腿归一+dcd uly 归一双臂，serve 谓词把 BTC-USDT 现货 tick
  滤在桥外；collector 侧 activeSet 第三/四 UNION 臂（T-0216 返修件，
  `internal/data/okxws/subscribe.go:78-83`）已有对应语义，projector
  侧漏配=T-0216 审计残差未捕段。

管理急件直办在役：`o-projector serve.conf -mode=dualwrite` 旁写泵通水
（SG BTC-USDT ticks 17:19Z 鲜、286990 首笔 funding fill
+0.006881@17:18:55Z 落地）——**dualwrite=旁写泵非正形，全量镜像写量
不可长持**，根修部署后须复原 `-mode=serve`。

## 钦定界

- `refreshSets` 活跃集补归一双臂，**语义对齐** subscribe.go activeSet
  第三/四 UNION 臂：
  - basis 现货腿：`regexp_replace(regexp_replace(inst_id,'-[0-9]{6}$',''),
    '-USD$','-USDT')`，限 `template_ref IN ('quarter_basis_long',
    'perp_quarter_spread')` 且 state 非 retired（口径含 paper——记账
    价源同 collector 臂）；
  - dcd uly 归一：`regexp_replace(uly,'-USD(G)?$','-USDT')`——**SG 侧
    表可用性先对账**：subscribe.go 臂 JOIN `md_dcd_product`（PG-md
    域表），SG fin 库若无此表/桥面对应物须如实改造取径（禁盲抄 SQL
    致 relation does not exist——同 T-0187 fin_position 坑型）。
- 桩测：refreshSets 断言归一臂入集（basis paper/live 策略→现货对入集；
  dcd 策略→uly 归一入集；retired 不入）；非 skip 实跑。
- 部署后配置复原衔接口径入结果段（serve.conf 切回 `-mode=serve`=
  部署面动作，施工交付=码面+桩测，复原由部署径执）。
- 链变即 `docs/数据流.md` 投影活跃集段同 commit 更新。

## 范围外

collector 侧（已修已重启）；dualwrite/复原配置实操（部署面）；basis
执行面其他件（T-0216 已审计过）。

## 验收锚

1. 双臂语义对齐实证（diff 对 subscribe.go:78-83 逐臂）+SG 表可用性
   对账注记；
2. 桩测：归一臂入集断言绿（非 skip）；
3. serve 模式生效后产验锚：SG `md_ticks` BTC-USDT 现货行续鲜+
   `fin_paper_fill` 续出（部署复原后由决策/管理验）；
4. 数据流.md 同 commit 更新；
5. 票面回填+KB（若产坑）。

## 结果回填（施工席·无头 devin-t0250 @2026-10-30）

### 实现摘要

refreshSets 活跃集两臂裸 inst_id → 四臂：新增 ③basis 现货腿归一臂
（inst_id 剥 `-[0-9]{6}$` 尾→剥 `-USD$`→`-USDT`，template_ref IN
('quarter_basis_long','perp_quarter_spread') 且 `state<>'retired'`——
票面钦定口径含 paper/probe）与 ④dcd uly 归一臂（fin_strategy JOIN
SG 镜像 md_dcd_product ON product_id=inst_id，uly 剥 `-USD(G)?$`
→`-USDT`，`state<>'retired' AND template_ref LIKE 'dcd%'`）；同件把
dcd 喂集 `='dcd_call'`→`LIKE 'dcd%'` 放宽使镜像覆盖面=JOIN 谓词域
自洽。serve 谓词下 basis/dcd 现货腿 tick 不再被滤桥外。

### 验收锚对账

1. **双臂语义对齐实证 PASS**（diff 对 subscribe.go:78-83 逐臂）：
   - 臂③逐字对 :78-80——`regexp_replace(d.uly,'-USD(G)?$','-USDT')`+
     `JOIN md_dcd_product ON product_id=inst_id`+`state<>'retired'`+
     `template_ref LIKE 'dcd%'` 四件全同语义。
   - 臂④逐字对 :81-83——双剥归一链同式、模板集同；state 取票面
     「非 retired」非 collector 逐字 `IN ('paper','live','degraded')`
     ——超集（probe 记账同需腿价/proposed），与 projector 臂②同域。
   - **SG 表可用性对账注记**：md_dcd_product SG fin 库**实存**
     （fin_0049_md_state_tables.sql 建、与 PG-md md_1049 同形——
     check_migrations 断言⑥八桥 DDL 同形机检钉）；uly/product_id
     列齐（fin_0049:50-51）。该表=本臂 uDcd upsert 写之镜像件
     （写权唯一 projector），覆盖面=r.dcd 喂集——故同件放宽 dcd%
     使覆盖面=JOIN 谓词域（原 `='dcd_call'` 口径下 dcd_put 策略
     产品永不镜像=JOIN 臂静默漏——同型坑换姿势重演，桩测
     DCD-LTC-2 live 行即此场景断言位）。真冷启动首拍 uly 臂滞后
     一拍（Reconcile 序 refreshSets→dcd 镜像→次刷收敛，
     ev_active_set/10min 活性对账——非死锁：dcd 镜像只赖
     fin_strategy 不赖 active 集）。
2. **桩测 PASS**：`TestActiveSetNormalizedLegs` 非 skip 实跑绿——
   basis paper/live→现货对入集（BTC-USDT/ETH-USDT），retired 不入
   （SOL-USDT/SOL-USD-270626），非钦定模板不归一（XRP-USDT 不入、
   XRP-USD-270626 裸键入），dcd_call/dcd_put→uly 归一入集
   （AVAX-USDT/LTC-USDT，含 -USDG 方言形），retired dcd 不误取
   （DOT-USDT——镜像残行豁免）；dcd 喂集域断言同桩（dcdSet 含
   DCD-AVAX-1/DCD-LTC-2、排 DCD-DOT-3——放宽断言位实证）；端到端
   BTC-USDT tick 落 SG 镜像=1、SOL-USDT=0。
3. **serve 产验锚（部署径待验）**：serve.conf 切回 `-mode=serve`=
   部署面动作（票面钦定：施工交付=码面+桩测，复原由部署径执）——
   复原后 SG md_ticks BTC-USDT 现货行续鲜+fin_paper_fill 续出，
   由决策/管理验。
4. **数据流.md 同 commit 更新 PASS**：serve 谓词段四臂注记
   （510439e 同件）。
5. **回填+KB PASS**：本段；工程经验库条目
   `active-set-dual-owner-arm-drift` 同 commit 入库+`kbgen gen`
   索引机生重建（7a44546）——双端同语义谓词集单端补臂另端
   静默漏+跨域 JOIN 臂移植覆盖面自洽两点固化。

### 机检行

check_all 全链绿戳盖 HEAD：build✓ vet✓ import 方向✓ gofmt✓ env✓
rejects✓ migrations 七断言✓ bridge-audit 四列✓ go test 全谱✓
（internal/projector 域含新桩实跑 PASS）；check-lines
projector.go=283 行<300✓。

### 偏离说明

- **撞车移交残件**：Dev 席交互施工与本无头 spawn 撞单（relay
  实录 MANAGE-75 双发案——对方 01:46 自报「撞车退场，补丁移交」），
  其在飞补丁（双臂 SQL+桩测+数据流.md）落 `510439e` 移交本席；
  本席复核语义无误后续作：dcd 喂集 `dcd%` 放宽+KB 条目+本回填。
  `510439e` 消息头误书「领单」实为码面移交件；KB commit 曾被
  对方 amend 改题（7a44546→8dbb8d3，内容同一——历史不改写，
  收口commit 字段以现链 8dbb8d3 为准）。
- **dcd 喂集 `='dcd_call'`→`LIKE 'dcd%'` 放宽**：票面钦定界字面无
  此项，为 JOIN 臂 SG 侧「镜像覆盖面=谓词域」自洽所必需；dcd_put
  现无 paper 径（paperTemplates 白名单外）故零实效——防未来启用
  踩同型静默滤（KB 条目复用面②）。**自定待裁**。
- **basis 臂 state=`非 retired`**：取票面字面口径非 collector 逐字
  三态集——超集含 probe/proposed（probe 态记账同需腿价），与
  projector 臂②同域。
