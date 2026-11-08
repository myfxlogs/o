# T-0495 探针升件自动起草：观测盲@signal:basis_ann→∅

- 状态：候选（T-0393 起草拍自动起草——非已批，COS/DM 过目才进派发队）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:signal:basis_ann→∅`，草案锚 fin_ticket_draft#38）

## 病灶（升件行证据引）

病灶签名：`观测盲:signal:basis_ann→∅`（kind=探针类 `观测盲`，key=病灶键 `signal:basis_ann→∅`）

末行升件原文：
> 依赖对失明 signal:basis_ann→∅：产者 basis_ann 窗内行=2 而消费者 ∅（消费册缺席——两链派生零命中） 行=缺席——产者活消费者零视野依赖对失明（探针只报不修，修归常票/饥饿响应）

证据引（ops_event 行 id）：#3500

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='signal:basis_ann→∅' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（起草骨架——DM 裁时可改判，起草非诊断）

1. 机批类册钦定文件径：internal/research/sigdist internal/portfolio/hunger——签名 观测盲:signal:basis_ann→∅（T-0455 册内候选，DM 裁时可改判）；
2. 病灶根因定位与根修（签名面病灶本体——不超签名外衍）；
3. 桩测钉死断言（病灶存在断言修前红+同断言修后绿——T-0388 修复类先红桩律）；
4. 生产谱面零误报审计。

## 验收锚（骨架——DM 裁时补足）

① 桩测断言全绿（红/绿双点证据落结果回填）；
② 探针同窗不再报同签名病灶（修复后探针行随停）；
③ check_all 绿戳对口；并入发窗批。

## 红线（继承模板——不改不删）

- 只修本票签名病灶——不超单施工、不碰范围外文件；
- 不部署、不并 master——收口回决策席常径裁；
- 探针只报不修——本票修病灶本体，禁顺手改探针阈/签名面/类册；
- 钱径/凭据域/真单径语义禁碰。

## 结果回填

- 状态：待验
- 施工回填（2026-11-09 无头施工席 @ wt /opt/o-wt/t0495 枝 t/0495-basis-ann-consumers）：
  - **定性=陈旧证据重复起草件——签名病灶本体已由 T-0447 根修在案并生产
    实证，代码面零改动**（diff 仅票面回填）：
    1. 时间线实证：升件 #3500 落账 2026-11-07 22:24Z，早于 T-0447 施工
       commit `7a33317c`（2026-11-08 03:55Z）与其 merge/部署窗——系修复
       部署前 24h 去重窗届满的合法再报（dedup 语义内），非修复失效复鸣；
       同签名前案 #3010（11-06 18:31Z）即 T-0447 立案证据引。
    2. 病灶本体现状：`internal/portfolio/hunger/depsigfam.go`
       `depSignalFamilies` 钉定消费册在 master——kind=basis_ann→族
       kind=basis 解析面=fin_opp_template enabled∩state='enabled'∩
       params->>'kind' ∩ fin_opp_eval DISTINCT，登记对
       `signal:basis_ann→quarter_basis_long`/`→perp_quarter_spread`
       入列压 orphan，`signal:basis_ann→∅` 枚举面结构性不再产出。
    3. 生产谱面实证（SG o-postgres 2026-11-08 ~21:0xZ 对拍）：
       - 依赖对清单 census #4906（o-core 重启后首拍 19:34:58Z）sigs
         含两登记对、signal 域 `→∅` orphan=0（pairs=106）；
       - `md_signal` basis_ann 7d 行=2（产者活=升件行证同）；
       - `fin_opp_template` basis 族=quarter_basis_long/
         perp_quarter_spread enabled+state='enabled'、
         quarter_basis_short enabled=f+state='candidate'（保禁正确排除）；
       - `fin_opp_eval` 镜像 DISTINCT=12 模板含 basis 族；两启封模板
         7d 各 1 行 proposed verdict、ts=11-08 19:23 新鲜（登记对
         consN>0——活性对账不盲）；
       - 失明/升件面：`signal:basis_ann` 域历史仅 #3009/#3499 两行皆
         修复前落账；部署后同签名零复鸣——枚举面不再产该对=
         probeDep 无对象可报（验收锚②结构性满足，dedup 24h 窗虽未
         全程走完亦不构成可报路径）。
    4. 桩测复跑：`TestProbeDepsSignalFamilyRegistry` PASS 2.71s
       （本枝=master 含 T-0447 修复面；修前红证据=T-0447 回填在案
       「钉定消费册登记对缺席」——T-0388 先红桩律双点齐）。
  - 偏离/留案（范围外不修只记）：
    - depenum `evalTemplates`/`depCounts` consN/facesnap DISTINCT 三面
      仍读 SG `fin_opp_eval` 镜像（T-0485 回填已记档「已知残面——归
      CTO 裁量另票」）：镜像谓词=proposal 引用行→全拒族对 depblind
      欠报向，且镜像行 ts 陈>7d 时登记对 consN=0 可致
      `signal:basis_ann→<template>` 异签名误报——本票签名（→∅
      orphan）不在其射程（fin_opp_eval 镜像行无删除径、模板全集不失），
      不超单施工；
    - 起草侧观察：T-0393 起草拍对「已修未部署窗内落账」升件不去重——
      同签名 T-0447/T-0495 双票并存；判例闸/起草 dedup 域归另票裁量
      （T-0499 ticket-key-hygiene 相邻域在案）；
    - 生产 origin/master 滞后本地 master（推送面=T-0460 病域在案非本票，
      同 T-0447 留案）。
