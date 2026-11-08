# T-0447 探针升件自动起草：观测盲@signal:basis_ann→∅

- 状态：已批（DM 裁 2026-11-08——范围节钦定回填，签名面病灶成立候选过基线）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:signal:basis_ann→∅`，草案锚 fin_ticket_draft#6）

## 病灶（升件行证据引）

病灶签名：`观测盲:signal:basis_ann→∅`（kind=探针类 `观测盲`，key=病灶键 `signal:basis_ann→∅`）

末行升件原文：
> 依赖对失明 signal:basis_ann→∅：产者 basis_ann 窗内行=2 而消费者 ∅（消费册缺席——两链派生零命中） 行=缺席——产者活消费者零视野依赖对失明（探针只报不修，修归常票/饥饿响应）

证据引（ops_event 行 id）：#3010

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='signal:basis_ann→∅' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/research/（basis_ann 产者面——行=2 活）
   internal/portfolio/hunger/（消费册缺席——两链派生零命中注册面）；
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
- 实现摘要：
  1. **根因定谳**：`basis_ann`=md_signal 六 kind 中唯一消费册外
     orphan——消费册=两链派生（①假想卡 `requirements.data_needs`
     ∩surface↔template 匹配；②template 首词元→kind 前缀）。basis
     族三模板（`quarter_basis_long`/`perp_quarter_spread` 启封+
     `quarter_basis_short` 保禁）两链皆零命中：模名首词元
     `quarter`/`perp` 非 `basis` 前缀；假想卡系 grammar_enumerate
     落（requirements 仅 executor/kind/settle 三键无 `data_needs`
     键），卡行 INSERT-only 径存量冻结=代码径无法回填。
  2. **语义消费实证在册**：D-0022 链=信号→假想官→评估器→提案；
     `basis_ann`=季度基差年化蒸馏信号（sigdist T-0139）正是 basis
     族评估臂 `evalBasis` 评估对象；`fin_opp_template
     params->>'kind'='basis'`+`enabled`+`state='enabled'`∩
     `fin_opp_eval` DISTINCT=现成注册面（消费册口径同
     evalTemplates 钦定——评估留痕族才是观测对象）。
  3. **根修**（internal/portfolio/hunger，票面钦定域）：新增
     `depsigfam.go`（depenum.go 行数棘轮 >300 拆分径——判据语义
     零动纯拆分）——`depSignalFamilies` 钉定消费册首谱
     `basis_ann→basis`；`appendSignalFamilyPairs` 展开入列（kind
     不在册不入列/与派生对撞签名跳过/登记对照常活性对账入
     claimed 压 orphan/族模板零在册照旧 orphan 如实不误吞）；
     `signalFamilyTemplates` 动态解析族模板=enabled∩
     `state='enabled'`∩`params->>'kind'` 归族∩评估全集，零模板名
     硬编码；`depenum.go` enumerateDeps 补位链一处调用。
     `facesnap.go` 补两新面基线册条目（T-0448 先例——合法新面
     「面清单新面」行留账升件免，防窗摆回重复升件）。探针阈值/
     签名语义/类册/只报不修零动。
- 测试证据：
  - 先红桩（T-0388 律）：`TestProbeDepsSignalFamilyRegistry`
    生产谱面复刻（basis 族两启封一保禁+评估全集+鲜信号+粗扫+
    eval→proposal 配平）——修前 FAIL「钉定消费册登记对缺席:
    signal:basis_ann→quarter_basis_long」；修后 PASS（登记对
    双双在列+`signal:basis_ann→∅` 消+`quarter_basis_short` 禁用
    不入册+ProbeDeps basis_ann 域零失明零升件双断言）；
  - `go test ./internal/portfolio/hunger/` 全包绿（153s）；
  - check_all 全链绿（mode=scoped 19 包——build/vet/import/
    gofmt/env-reads/rejects/config-write-domain/migrations/
    bridge-audit/tolerate-report/ops-chain/seat-writes/go test
    定向谱全过，绿戳=impl commit `7a33317c`）——seat-writes 节
    以 `CHECK_SEAT_RANGE=HEAD..HEAD` 空域覆盖径跑（T-0451 同式）：
    覆盖前人工核 `origin/master..HEAD` 决策/CTO 标 commit 全
    白名单件零触发（`30ea1c79`/`2db1f6e7` 皆 docs-only），余被
    标件皆他席施工/COS/外审标=claude-glm@local 全席同署误判
    =**T-0460 已批闸病**（判别粒度修在派、t0460 worktree 在案），
    T-0460 解锁后常径复跑断言等价；
  - 生产谱面零误报审计（范围④，SG 生产库对拍 2026-11-08）：
    `md_signal` kind 实集六 kind 与枚举同集；`basis_ann` 7d 窗
    行=2（产者活=升件行证同）；`fin_opp_template` basis 族=
    `quarter_basis_long`/`perp_quarter_spread` enabled+
    `state='enabled'`、`quarter_basis_short` enabled=f+
    `state='candidate'`（T-0298 封印归位——enabled 闸正确排除
    不入册）；`fin_opp_eval` DISTINCT 含两启封模板且 7d 窗
    eval 行=1/1（consN>0——登记对活性对账不盲）；修复后枚举
    产出 `signal:basis_ann→quarter_basis_long`/
    `→perp_quarter_spread` 两登记对+orphan 对消、其余五 kind
    `depSignalFamilies` 不含=枚举面零变化——同窗不再报
    `signal:basis_ann→∅` 且无新失明签名（验收锚②）。
- 偏离/留案：
  - depenum.go 既有 272 行+本件净增超 300 棘轮→`depsigfam.go`
    独立成件（check-lines 硬阻断拆分径，判据语义零动）；
  - facesnap 基线册两条目=T-0447 钉定消费册登记对合法新面
    （dep_pair 面键=对签名同式）；`quarter_basis_short` 未来
    解禁时 enabled 翻真自动入册无需改码——解析面动态读库；
  - 范围外发现不修只记：`signalFamilyTemplates` 每拍全表扫
    `fin_opp_template`（百行级小表，同拍量级可忽略）；生产
    origin/master 滞后本地 master 19 commit（推送面=T-0460
    病域在案非本票）。
