# T-0446 探针升件自动起草：观测盲@scan:md_tick_scan→discovery

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:scan:md_tick_scan→discovery`，草案锚 fin_ticket_draft#5）

## 病灶（升件行证据引）

病灶签名：`观测盲:scan:md_tick_scan→discovery`（kind=探针类 `观测盲`，key=病灶键 `scan:md_tick_scan→discovery`）

末行升件原文：
> 依赖对失明 scan:md_tick_scan→discovery：产者 md_tick_scan 窗内行=0 而消费者 discovery 行=1——产者活消费者零视野依赖对失明（探针只报不修，修归常票/饥饿响应）

证据引（ops_event 行 id）：#3012

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='scan:md_tick_scan→discovery' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/research/（md_tick_scan 产者面——窗内行=0 而 discovery 消费者依赖失明根域）
   internal/portfolio/hunger/（dep_pair 册消费面）；
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

### 实现摘要

签名 `观测盲:scan:md_tick_scan→discovery` 的产者面=md_tick_scan 写入臂
`ScanPoller`（`internal/data/okxws/scan.go`，cmd/collector 进程内 60s
粗扫拍）。根修一行径：`Run` 拍败由「errc 上抛→cmd/collector
`log.Fatalf`→采集主进程全臂死」改为「让位记拍下拍续试」——对齐
sigdist/scout 姊妹臂拍败惯例。`ScanOnce` 本体零改动：族粒度
fail-closed、全族败 loud、空集拒收、upsert 最新赢全保留；持久真死
仍由 scanwm 族谱水位闸+depblind 探针升件收口，观测面不削弱、不造假
行。桩测 `TestScanRunBeatFailYields` 钉死病灶断言。

### 根因定位

- 钦定路径 `internal/research/` 全谱 grep 零 `md_tick_scan` 引用——
  产者面本体不在钦定目录，实际落点 `internal/data/okxws/scan.go`
  （偏离明示见下）。
- 升件当次（2026-11-06 18:31，ops_event #3012）病灶=部署错位窗：
  T-0401 projector twotier 桥未随部署上线，SG 侧 md_tick_scan 镜像
  空约 31h，探针按设计正确报出；后续部署自愈，生产实证同签名停报。
- 残留代码级根因（本票根修对象）：`ScanPoller.Run` 拍败径——任一拍
  败（全族 REST/目录瞬断、写败）即 errc 岗终，经 cmd/collector
  `log.Fatalf` 杀死整个采集进程，WS tick/sigdist/mt5 姊妹臂陪葬，
  md_tick_scan 产者停→同签名复发。全仓仅此臂拍败即进程死，属签名
  实体唯一残留代码病灶。

### 测试证据

- 红（修前）：`TestScanRunBeatFailYields`——前两拍全族瞬断后 Run
  errc 岗终退出，恢复拍永不到，`瞬断拍后未自愈落行` FAIL
  （5.51s，O_TEST_PG_DSN=localhost:5434 独立测试库）。
- 绿（修后）：同断言 PASS（3.72s）——拍败让位日志×2、第三拍
  `swap=1 spot=0 futures=0 upsert md_tick_scan` 自愈落行、errc 零发、
  Run 随 ctx 正常收摊。
- 回归：`go test ./internal/data/okxws/` 全包 ok（34.3s）；
  `go build ./...`+`go vet`（触改域）净；check_all 戳见收口行。

### 生产谱面零误报审计

- OCI `o-collecto.service` active running，粗扫拍日志每分钟
  `[okxscan] 粗扫拍 ... upsert md_tick_scan`；SG md_tick_scan
  窗内新鲜行 swap=528/spot=466/futures=6（2026-11-08 18:54 实测），
  depblind 同窗 prodN>0——签名不再报。
- 族粒度审计：单族目录/REST 败只让位该族（scanFam fail-closed 粒度
  到族），不拖死他族与主进程；空集仍拒收非真空；本修不放大任何单族
  缺席为全族失明，无误报面。
- 持久全族死径不变：拍败让位≠吞错——scanwm 水位闸按族升件、
  depblind 10min 零行窗照常报同签名（该签名本就是为此设计的探针）。

### 偏离说明（范围外发现——不修，呈决策席）

1. 钦定路径偏离：`internal/research/` 内无 md_tick_scan 产者/消费
   码（dep_pair 册消费面在 `internal/portfolio/hunger/` 已核——探针
   只报不修本票不动）；病灶本体落点 `internal/data/okxws/scan.go`，
   按「签名面病灶本体根修」施工并在此明示。
2. 面级设计观察（不修）：cmd/collector errc→`log.Fatalf` 的全臂死
   架构仍在——本票只收 ScanPoller 单臂让位；姊妹臂同型拍败上 errc
   脆弱面如需统一治理，另立单呈决策席。
3. 历史已知项（非本签名面）：MUU-USDT 无效订阅报错在案，与本票
   观测盲签名无因果，不动。

- 裁记：批径2件盖顶@490ef0522（T-0446+提案-KB向量检索——决策席 2026-11-09）
