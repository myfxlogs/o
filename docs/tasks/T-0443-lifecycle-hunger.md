# T-0443 探针升件自动起草：写败@LIFECYCLE-HUNGER

- 状态：已批（DM 裁 2026-11-08——范围节钦定回填，签名面病灶成立候选过基线）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `写败:LIFECYCLE-HUNGER`，草案锚 fin_ticket_draft#1）

## 病灶（升件行证据引）

病灶签名：`写败:LIFECYCLE-HUNGER`（kind=探针类 `写败`，key=病灶键 `LIFECYCLE-HUNGER`）

末行升件原文：
> hungertick depblind_probe 败: 腿对键集读败: 宇宙读败: okx.symbols=* 宇宙扩件探针未挂——fail-closed 不虚构标的集

证据引（ops_event 行 id）：#2630,#2631

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='写败' AND esc_scope='LIFECYCLE-HUNGER' ORDER BY id DESC LIMIT 20
```

## 疑似根域

lifecycle 饥饿拍写径（hungerEscalate 自报面）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/portfolio/hunger/（depblind_probe 宇宙扩件探针挂接面——okx.symbols=* 扩件缺挂 fail-closed 根域）
   tools/hungertick/（hunger 拍写径面）；
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
  1. **根因定位（生产取证+码面对拍）**：签名 `写败:LIFECYCLE-HUNGER`
     唯一写径=`internal/portfolio/lifecycle/hunger.go` `hungerEscalate`
     （ops_event esc_class/esc_scope 落行）——tools/hungertick 拍壳
     只 stderr 不升件，不可能产本签名。生产实证 #2630/#2631
     （2026-11-06 09:08Z 同拍两行）：o-core 内 hungerTick 的
     legcover_probe+depblind_probe 双双 fail-closed「okx.symbols=*
     宇宙扩件探针未挂」——当时部署二进制早于 T-0416 挂线
     （9c69dffb5 落 master 17:54Z，部署归窗后）——o-core 侧根因
     T-0416 已根修并部署复电（今日 12:48Z 依赖对清单 census
     pairs=107 在流、leg:* 对枚举实证；升件行 48h+ 零复鸣——
     哨兵 #3613 在案）。
  2. **残余病灶（本票范围内根修）**：`tools/hungertick` 拍壳
     `hunger.New(pool,cfg)` 永不挂 `AttachUniverse`—— Sensor
     `expandUniverse` 恒 nil；且探针面漂移：hungerTick 同槽序
     （opschain→ProbeFamilies→**ProbeLegCoverage→ProbeDeps**→
     ProbeFeedStale→DraftTickets→Detect→Respond）中拍壳缺
     legcover/depblind 两宇宙读探针（签名证据 #2630/#2631 正是
     这两个探针）——手动拍壳跑到宇宙读探针=同签名病灶必复现，
     不跑=静默覆盖缺口（「该挂未挂」同型）。
  3. **根修**（tools/hungertick，票面钦定域）：抽 `newSensor`
     装配件——`okxws.UniverseCatalog` 单源供给件（与 cmd/core
     `uni.Symbols`/cmd/collector 同件同读，TTL 常驻缓存；探针
     宽谱判据不滤 venue 黑名单=T-0416 语义）+`AttachUniverse`
     挂线；`okxexec.NewClient(Credentials{},false).PublicGet`
     公开面零凭据（collector 同例）。hungerTick 同位补
     `ProbeLegCoverage`→`ProbeDeps` 两拍（ProbeFamilies 后、
     ProbeFeedStale 前），败情 stderr 让位不阻 Detect 主拍同
     现例。internal/portfolio/hunger 探针挂接面本体（AttachUniverse
     缝+subUniverse fail-closed 语义）T-0416 已正确，零改动。
- 测试证据：
  - 先红桩（T-0388 律）：`tools/hungertick/main_test.go`
    `TestShellUniverseAttach`——okx.symbols=["*"] 桩景下
    ①红点：裸 `hunger.New` 未挂扩件→ProbeDeps/ProbeLegCoverage
    双双 fail-closed「宇宙扩件探针未挂」（病灶存在断言=生产
    #2630/#2631 同签名根域复刻，修前修后同绿——fail-closed
    哲学钉死不动）；②绿点：`newSensor` 同装配挂线（桩
    ScanGet 三族目录件）→双探针复电零未挂错。
  - 修前红实证：main.go 还原 HEAD 版跑测=`build failed:
    undefined: newSensor`（挂线面缺席=病灶本体——callsite
    接线件修复类红为编译级，如实记）；修后 `go test
    ./tools/hungertick/` PASS（4.67s）。
  - 邻域谱：`go build ./tools/... ./internal/portfolio/...`+
    `go vet`+`gofmt` 零报；`go test ./internal/portfolio/hunger/
    ./internal/portfolio/lifecycle/` 全绿（488s/434s——本票
    零 internal 改动基线回归证）。
- 验收锚对账：
  - ①桩测断言全绿：PASS——双点证据上节（红=病灶复刻+编译级
    修前红；绿=挂线复电）。
  - ②探针同窗不再报同签名病灶：PASS——生产侧 T-0416 部署后
    零复鸣（#2630/#2631 后 ops_event 同签名升件零新增，48h+
    哨兵 #3613 在案）；本票修拍壳侧残余挂接面防手动径复发。
  - ③check_all 绿戳对口：见收口行（scoped 谱——变更面=tools/
    .go 两件+票面）。
- 生产谱面零误报审计（范围④）：Sensor 实例全谱对拍——
  `hunger.New` 调用点仅 lifecycle/run.go（Runner.AttachUniverse
  ←cmd/core `uni.Symbols` 已挂，T-0416）+tools/hungertick
  （本票新挂）——无第三未挂面；`okx.symbols=["*"]` 现行值下
  未挂径已绝。目录读败径=fail-closed 让位探针侧自决（stderr/
  升件语义不变非误报）。
- 偏离/留案（范围外发现不自修——呈决策席议）：
  - 拍壳探针面仍缺 `DraftTickets`（hungerTick 同槽第六件）与
    `MineLessons`（独立 12h 矿工拍）镜像——签名病灶域外漂移，
    是否补齐拍壳全镜像归决策席裁（另立票议）；
  - internal/portfolio/hunger 本票零改动——探针挂接面
    （AttachUniverse 缝/subUniverse fail-closed/depLegKeys∩
    legRequiredKeys 近重复枚举）T-0416 后语义正确，归一重构
    非本票签名病灶；
