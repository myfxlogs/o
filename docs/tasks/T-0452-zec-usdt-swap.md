# T-0452 探针升件自动起草：观测盲@ZEC-USDT-SWAP

- 状态：已批（DM 裁 2026-11-08——范围节钦定回填，签名面病灶成立候选过基线）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:ZEC-USDT-SWAP`，草案锚 fin_ticket_draft#18）

## 病灶（升件行证据引）

病灶签名：`观测盲:ZEC-USDT-SWAP`（kind=探针类 `观测盲`，key=病灶键 `ZEC-USDT-SWAP`）

末行升件原文：
> 腿行情覆盖失明 ZEC-USDT-SWAP：live/paper 策略腿所需行情键近 10min md_ticks 零新鲜行——归一链订阅面/镜像谓词/写入闸断点待查（探针只报不修）

证据引（ops_event 行 id）：#3451

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='ZEC-USDT-SWAP' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/data/okxws/（activeSet×md_ticks 归一链——ZEC-USDT-SWAP 腿键覆盖缺）
   internal/portfolio/hunger/（腿覆盖探针域）；
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
  1. **根因定位（生产取证 2026-11-09 四径亲证）**：升件 #3451=
     2026-11-07 16:24:13.016 探针拍落账——`fin_strategy`
     `ZEC-USDT-SWAP` funding-hedge 新实例 created_at=16:24:12.08，
     腿键入所需集与判盲间隔 **0.9s**，与派生现货腿 `ZEC-USDT`
     （#3449）同拍齐报。根因=**T-0451 同根**
     `probe-new-key-startup-race`：入所需集→订阅→采集→桥投影→
     消费面落行全链延迟分钟级，首见键即时判盲=结构性假阳。
     **病灶本体已修**：`coverNewGraceMin=15min` 首见宽限判据逐键
     一体无键形分支——`-SWAP` inst 本键与派生现货腿同效；
     T-0451 merge 已部署生产 @`668704438`（2026-11-08 17:24
     deploy-oci rc=0，git 祖先核证含 0ca7a2989）。
  2. **归一链健康实证（范围①钦定面零断点）**：md_ticks 亲读——
     `ZEC-USDT-SWAP` venue=okx 活跃期鲜行持续（末鲜 11-08 18:01，
     策略退役退订后断供符合预期）；`ZEC-USDT` 末鲜 19:23；
     md_funding/md_mark_index 双键全窗鲜（universe 级订阅面
     funding-rate/mark-price 独立佐证 venue 侧在挂）；OKX
     /public/instruments 亲核双键 state=live 在册。
  3. **本票增量=签名面钉死桩**（范围③）：T-0451 宽限桩只钉
     派生现货腿盲形，`-USDT-SWAP` 本键盲形无断言——补
     `TestProbeLegCoverageSwapLegNewGrace` 复刻生产签名
     （ZEC-USDT-SWAP live 策略双键齐入所需集零 tick：宽限内
     零报＋宽限过双键照报＋esc_scope 钉死 -SWAP 本键面）。
     码面零改——宽限判据本体 T-0451 已落，本桩防签名形状回归。
- 测试证据：
  - 先红桩（T-0388 律）：修前径=e89465b35（T-0451 impl 父
    commit）scratch wt 同桩（coverNewGraceMin→字面量 20min
    等价语义）→ **FAIL** `coverage_test.go:226: 新挂腿宽限内
    误报: 覆盖失明=2 升件=2`——生产 #3449/#3451 双报复刻；
    修后径=本枝同断言 **PASS**（`go test
    ./internal/portfolio/hunger -run TestProbeLegCoverage` 全 8
    桩绿；全包 `ok 45.5s`）。
  - 生产谱面零误报审计（范围④）：修复部署后（11-08 17:24 起）
    `覆盖失明`/`观测盲:T-0392` 升件**零新增**（升件全谱仅存量
    #3449/#3451/#3497/#3007 四件历史行）；ActiveSetUnion 亲跑
    对拍——ZEC 双键已随全数策略 retired 出集（9 例全 retired，
    末例 entered 11-08 19:23:58 后退），探针静默=键出集非失明；
    宇宙外键（dcd 期权 inst 形/MUU-USDT 幽灵）经 universe 闸
    正确滤除零误报——谱面干净。
- 机检：gofmt✓ vet✓ build✓（hunger 包）｜go test
  `./internal/portfolio/hunger` 全包 ok（45.5s）｜check_all
  **mode=scoped 全链绿**（变更面=coverage_test.go+票面，
  scoped 谱 19 包 ok 零败；戳对口收口 commit）。
- 自检 A–I：A 复用既有桩件/helper 零新造✓；B 直击签名病灶本体
  （-SWAP 本键形状宽限断言缺口）✓；C 零死码/TODO✓；D 红桩修前
  红修后绿双点在案✓；E 红线逐条：只碰本票签名面（hunger 测试
  件）/未部署未并/探针阈签名面类册零动/钱径凭据零碰✓；F 票面
  回填同步✓；G 三锚逐条上对账✓；H 生产亲证四径：ops_event
  #3451 行 psql 亲读/fin_strategy 9 例 retired 亲读/md_ticks
  +md_funding+md_mark_index 三表亲读/OKX instruments API 亲核
  ——非听闻✓；I 非展示值面 N/A✓。
- 涉及面与范围外确认：实际触碰=
  `internal/portfolio/hunger/coverage_test.go`（+1 钉死桩
  58 行）+本票面。范围①钦定 internal/data/okxws 面**零触**——
  归一链订阅/镜像/写入闸三段对拍零断点（T-0451 同证+本票
  md_ticks 复证），病灶不在 okxws 本体。范围外发现（只记不修，
  归决策席裁）：T-0393 起草拍按 esc_scope 签名逐键产票——同根
  病灶跨腿键（ZEC-USDT+ZEC-USDT-SWAP→T-0451/T-0452；MUU 同型
  →T-0445）会产姊妹票各烧一施工位；draft 层根因去重/同根归并
  可为后续提案面（本票不动）。
- 偏离说明：无——范围②「病灶根因定位与根修」落定=根因已修
  （T-0451 同根部署在案），本票交付=签名面钉死断言+生产复证，
  未超单新增码面改动。

分支: t/0452-zec-usdt-swap（impl+回填本地，check_all scoped 绿戳对口）
移交: 决策席（收口移交归票面钦定）
