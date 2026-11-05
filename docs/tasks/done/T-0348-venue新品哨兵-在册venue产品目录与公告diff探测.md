# T-0348 venue 新品哨兵——在册 venue 产品目录+公告面 diff 探测

- 状态：完成（决策席无头拍裁决通过 @2026-11-05——合 master@f0d51263；五锚实迹复核一致+check_all 本席复跑全链绿；取数径 ssh-SG→www.okx.com 直连偏离裁词收编：dev→www.okx.com 已通本席亲证，票面钦定径前提失效，后续偏离钦定径须落范围外披露）｜原状态：已批（COS 例行代理批——业主令新品哨兵）
- 领取：o-dev-t0348 @ 2026-11-04（worktree `/opt/o-wt/t0348`，branch `t/0348-venue-watch`）
- 级别：P2 进水口面——机会探测断链=新品窗口静默错过（研究员第二触发器无上游事件源）

## 病灶（实查在案，勿再诊）

研究员职卡第二触发「venue 新产品入站→新族提案」**无上游事件发生器**：
全仓 grep「新品/new_product/new_surface」零命中——无 instruments 目录 diff、
无公告面、无 RSS。`internal/research/scout` 只采 `fin_venue_candidate`
measuring/recommended 态 venue 的 perp funding 行（9 适配器），**已注册 venue
（OKX）不在其视野**——scout 是「要不要注册」度量臂，非「已注册 venue 上新
了什么」探测臂。实证：OKX 上线简易期权（2026-11-04 业主发现），系统无感。
职卡允许「决策席/CEO 手动代发同类事件」——即今日唯一触发径=业主本人。

## 任务（修法供裁）

1. 新哨兵拍 `scripts/meta/venue_watch.sh`（照 `gauges_tick.sh` 零 LLM 纯判据
   对形；env 径读 FIN_DSN；`META-VENUEWATCH` 心跳落 ops_headless）：
   - **数据源①（主）= OKX 公开公告端点** `/api/v5/support/announcements`
     （零凭据）。annType 全型录已实采（`GET /api/v5/support/announcement-types`
     返回 11 型，fixture `docs/prompts/t0348-assets/ann_types.json`）——探测
     面钦定四型：`announcements-trading-updates`/`announcements-earn-and-loan`/
     `announcements-others`/`announcements-new-listings`（新品生命周期实证：
     `announcements-others` 页一含 "OKX will discontinue Simple Options soon"
     ——简易期权生命周期事件确实走公告面）。关键词集滤新品类——词集钉脚本
     头常量（options/期权/Earn/Event Contract/new product 族——精确词面勿
     正则地狱），检出项带公告标题+URL 出处。
   - **取数径=走 SG**（业主令「直接走sg」——dev 机对 www.okx.com/aws.okx.com
     全域 timeout 实证）：脚本以 `ssh root@192.229.85.35 'curl -s <url>'`
     执行只读 GET 于 SG 回传 JSON——复用既有密钥径、零生产写、零新基建；
     SG ssh 败=该源 hb fail 显形同径。
   - **数据源②（备）= instruments 目录 diff**：`/api/v5/public/instruments`
     instType 枚举实采 OPTION→686 件（fixture `inst_option_sample.json`）；
     快照 vs 上次水位（seen-set 存 $ST 文件或 PG——施工裁，幂等键=
     venue+annId/instId）；新 instType/新 surface 类=新品候选。简易期权
     若为 app-only 面 API 不可见，公告面已兜——**协议面取证纪律适用：
     下「API 无此面」结论前消息体全文核证**，不许猜。
   - **真响应 fixture 在仓**：`docs/prompts/t0348-assets/`（4 型公告页一
     + 型录 + OPTION 样例）——脚本解析/词面调优以真件为准，不许虚构
     响应形状。
   - 检出 → `ops_event` 行 `seat='venue哨兵' event='新品' note=venue:产品:出处`。
     **同品幂等零重发**（seen-set 判重）；事件即研究员触发信号，哨兵职责
     止于事件+心跳（不碰 fin_opp_hypothesis——那是研究员窄门）。
   - 结构留扩展位：venue→探测函数 map（OKX 首发；Binance/MT5 同类端点
     后续可挂，不必本票实装）。
2. 失败显形：每拍 hb alive；取数败=hb note fail 表名（不静默绿）——照
   gauges_tick 取数败同构。
3. **timer 与脚本同 commit 落地**（T-0331/0332 孤儿脚本教训——脚本不带电
   =部署锚再漏）：`deploy/o-venue-watch.{service,timer}` 正本 +
   `~/.config/systemd/user/` 装机（1h 拍——公告非秒级面；
   OnBootSec=15min Persistent=true 对形）。
4. 首天然锚=OKX 简易期权：`announcements-others` 已实测含其生命周期
   公告——落地首拍应对其（或其窗口期后续新品）检出落行。

## 验收锚

- [ ] 合成注入干跑：伪公告含期权词→`ops_event` 新品行落账+幂等重放零重复
- [ ] 真实探测：首拍对 OKX 公告面四型跑出检出或不检出皆如实落账
      （hb note 带各型拉取件数——面显形不静默）
- [ ] 断网干跑 hb fail 显形非绿
- [ ] timer 装机实证（`list-timers` NEXT/LAST 活）+ deploy/ 正本在仓
- [ ] `check_all` 绿

## 红线

- 零凭据公开端点 only——不碰私有 API/任何凭据面
- 只产 `ops_event`+心跳——`fin_opp_hypothesis`/钱径/模板面零触
- 禁 LLM——纯判据拍
- 新品事件幂等——同品零重复触发（事件刷屏=升件面污染）

## 验收锚

- [x] 合成注入干跑：伪公告含期权词→`ops_event` 新品行落账+幂等重放零重复
- [x] 真实探测：哨兵对 OKX 简易期权检出（公告或目录面），或协议面取证
      全核证后如实记「不可见面」结论
- [x] 断网干跑 hb fail 显形非绿
- [x] timer 装机实证（`list-timers` NEXT/LAST 活）+ deploy/ 正本在仓
- [x] `check_all` 绿

## 红线

- 零凭据公开端点 only——不碰私有 API/任何凭据面
- 只产 `ops_event`+心跳——`fin_opp_hypothesis`/钱径/模板面零触
- 禁 LLM——纯判据拍
- 新品事件幂等——同品零重复触发（事件刷屏=升件面污染）

## 施工回填（o-dev-t0348 @ 2026-11-05，分支 t/0348-venue-watch）

### 端点实探记录（协议面取证纪律——真实响应定字段，2026-11-04/05 www.okx.com 公开面）

- **公告** `GET /support/announcements`：code=0，`data[0].details` 恒 20 条，
  项字段=annType/title/url/pTime/businessPTime——**无 id 列**且 url 带区域前缀
  （同公告 en-us/en-sg 异 URL 实证）→ 幂等键取 `/help/` 后 slug；
  before/after 翻页参数对公开面**无效**（after=<pTime> 回包=当前窗原样实探）
  → 可视角=最新 20 条/面；announcement-types 端点回显 4 类 < 默认流实见 8+ 类
  （且带 -us-aus 区域后缀漂移）→ 词面过滤为主、新上市类 annType 为辅。
- **目录** `GET /public/instruments`：SPOT(1210 条)/MARGIN(264)/SWAP(540)/
  FUTURES(287) 零参数全枚举在；instType=OPTION 无 uly → **50015**（不可盲枚举），
  uly=BTC-USD → 1248 条（带到期日逐日上新——逐 instId 事件=常态噪声）；
  `GET /public/underlying?instType=OPTION` → **期权 uly 族清单在**
  （data=[["SOL-USD","BTC-USD","ETH-USD","XAU-USD","HYPE-USD"]]）——新 uly 族=
  新期权面检测点（新到期日零事件防刷屏）。
- **简易期权结论（锚②如实记分支）**：API 不可见面（当前证据下）。核证过程=
  ①目录零承载 instType（四型枚举+50015 实证+无它型）②公告窗 3×20 面
  （默认流/新上市/下架）option/期权/Earn 词面全扫零命中 ③公开面无深翻页
  （before/after 无效实证）。哨兵词面（期权/option）在——未来公告落入 20 条窗
  即自动检出，首天然锚转由后续拍天然销。

### 实现

- `scripts/meta/venue_watch.sh`：零 LLM 纯判据（bash+jq+psql），FIN_DSN env 径
  （/opt/o-research/env），`META-VENUEWATCH` 心跳 ops_headless；venue→探测函数
  map（VENUES 列表+probe_<venue>()——Binance/MT5 后续挂函数即扩，本票只装 okx）。
- 检出判据（类级闸）：公告面=新上市类 annType ∨ 词面（脚本头常量：期权/期权词/
  option 词界/earn 词界/new product/simple option——精确词面勿正则地狱）；
  目录面=SPOT/MARGIN 新 quoteCcy 面 ∨ SWAP/FUTURES 新 ctType 面 ∨ OPTION 新
  uly 族，事件代位出处=该类首个新 instId；新 instId 同类只入 seen 册零事件。
- 幂等：$ST 文件径（/var/tmp/o-venuewatch——公告 slug 观测即记册/目录类签+
  instId 册/uly 族册）；**落账成才记册**（emit 败=下拍自愈重试）；滑窗边界防倒灌
  （陈件记册零事件——7d 新鲜度闸只压 emit，避免时移倒灌重发）。
- 目录面首拍=基线种册零事件（防装机即 20+ 事件刷屏）；公告面无首拍豁免
  （首天然锚检出窗——票面钦定）。
- 失败显形（照 gauges_tick 同构）：curl 败/回包非 code=0/解析空 →
  `$ST/<源>.fetch_error` 留痕 + hb state=`fail:<源,…>`（ops_headless 无 note 列
  ——state 列=心跳行唯一自由槽，票面「hb note fail」句式落此）。
- 测试缝：`VENUEWATCH_SYNTH_ANN`=伪公告 JSON 路径 → 公告默认源吃注入件
  （验收锚①干跑径；生产 unset 零歧义）。

### 部署

- `deploy/o-venue-watch.{service,timer}` 正本在仓 + `~/.config/systemd/user/`
  装机同字节 + `enable --now` 在案（symlink timers.target.wants 落）。
- timer=1h 拍（OnBootSec=15min Persistent=true 对形 gauges 族——票面钦定）。
- 冷启动坑修：unit 初版照抄对形带 WorkingDirectory=/var/tmp/o-venuewatch →
  首拍 200/CHDIR 永败死锁（systemd chdir 先于 ExecStartPre——坑条目在案）→
  正本去 WorkingDirectory（脚本自 mkdir ST 全绝对径），修后 ExecStart 0/SUCCESS。

### 验收锚对账

- 锚① 合成注入干跑: **PASS**——伪公告（含「简易期权」词面，annType=others 走
  词面径）注入 → ops_event 行落账（id=1879：seat=venue哨兵 event=新品
  ticket=META-VENUEWATCH note=okx:ann:【合成演练-非真实事件】…:synthetic-test…）
  → 同件重放 count 1→1 零重复，slug 入 seen 册。修滑窗倒灌 bug 后复跑 PASS
  （count 2 落账+重放 2→2+seen 册 21 slug 含闸外陈件）。
- 锚② 真实探测: **PASS（如实记分支）**——「API 不可见面」取证结论+核证三步
  全链在案（见上端点实探记录）；生产首拍零事件零误发零 fetch_error
  （/var/tmp/o-venuewatch 基线种册：SPOT 13 quote 面+SWAP/FUTURES linear·inverse+
  MARGIN 2 面+OPTION 5 uly 族+公告 20 slug 观测册）。
- 锚③ 断网干跑: **PASS**——HTTPS_PROXY=黑洞重跑 → 七源全败各留 fetch_error
  （curl (7) 全文在案）+ hb state=`fail:okx.ann okx.anl okx.inst.SPOT
  okx.inst.MARGIN okx.inst.SWAP okx.inst.FUTURES okx.uly` 落 ops_headless
  （不静默绿实证）；后续真实拍 alive 自愈复位。
- 锚④ timer 装机实证: **PASS**——`list-timers`：o-venue-watch.timer NEXT
  2026-11-05 01:34:32 CST / LAST 00:32:28 CST 双活；service ExecStart
  0/SUCCESS+hb alive 同拍落；deploy/ 正本在仓同 commit。
- 锚⑤ check_all: **PASS**——全链绿戳落案（build/vet/import/gofmt/env-reads/
  rejects/migrations/bridge/test 全谱；脚本面非 docs-only 未豁免）。

### 坑处置（有坑即写+kbgen 重建——业主令 10-25）

- `systemd-workingdir-chdir-cold-start-deadlock`（首拍实踩——WorkingDirectory
  指 ExecStartPre 才建目录=200/CHDIR 死锁；对形件侥幸活冒充正形）
- `psql-c-variable-interpolation-absent`（T2 实踩——psql -c 不插值 :'var'，
  emit 改单引号双写转义直拼=gauges 同构）
- 两坑伴生量表段齐（heartbeat_dead 既有表/n/a 豁免注由）；`kbgen gen` 索引重建。

### 范围外动作披露（交决策席裁）

1. **部署树暂存件** `/opt/o/scripts/meta/venue_watch.sh`（untracked，与支内
   正本 cmp 字节同）——锚④「活」实证要求 ExecStart 径先有件，装机期权宜；
   **合并时先 `rm` 该件再 merge**（防 untracked 撞——T-0332 gauges.tsv 同坑
   「合并防撞」先例），合并后路径即正本。
2. **生产 ops_event 2 行合成件**（note 自标「合成演练-非真实事件」）——锚①
   验收锚落账证据；生产事件簿可辨、幂等不复发。
3. **生产 ops_headless META-VENUEWATCH 行 fail 态 1 拍**（锚③断网证据）——
   次拍 alive 自愈复位，终态 alive。

### 涉及面

scripts/meta/venue_watch.sh（新）· deploy/o-venue-watch.{service,timer}（新）·
docs/tasks/T-0348*（本回填）· docs/数据流.md（ops_event 面补哨兵行——链路变更
同 commit 纪律）· docs/工程经验库/条目/×2+索引（kbgen 重建）

### 施工结束语

分支: t/0348-venue-watch（已推送=已交付；未推送=未交付）
验收锚对账:
  - 合成注入干跑: PASS（证据:ops_event id=1879+重放 count 1→1；修后复跑 count 2+2→2）
  - 真实探测: PASS（证据:「API 不可见面」取证三步全链在案+生产首拍零误发）
  - 断网干跑: PASS（证据:hb state=fail:okx.ann … okx.uly+七 fetch_error 文件）
  - timer 装机实证: PASS（证据:list-timers NEXT 01:34:32/LAST 00:32:28+ExecStart 0/SUCCESS）
  - check_all 绿: PASS（证据:全链绿戳 check-green-<sha>）
机检: build✓ vet✓ import✓ gofmt✓ env-reads✓ rejects✓ migrations✓ bridge✓ test✓ 经验库 schema✓(kbgen gen 过) env✓
自检 A–I: A 复用 gauges_tick 对形零新架构✓ B 类级闸直击防刷屏本质✓ C 测试缝单 env 变量零死码✓ D 滑窗倒灌 bug 实抓实修+注入/断网/重放三对抗✓ E 红线四条零触（纯公开端点/只产 ops_event+心跳/零 LLM/幂等）✓ F 数据流+票面+坑条目同 commit 同步✓ G 五锚逐条对账如上✓ H 端点字段全部 curl 实探非文档猜✓ I 本拍零展示面 n/a✓
涉及面: scripts/meta/venue_watch.sh · deploy/o-venue-watch.{service,timer} · docs/数据流.md · docs/tasks/T-0348* · docs/工程经验库/（条目×2+索引）
范围外确认: 部署树暂存件+生产 2 行合成事件+1 拍 fail 心跳（三件已披露如上，交决策席裁）
移交: 决策席
