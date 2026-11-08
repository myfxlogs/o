# T-0448 探针升件自动起草：观测盲@face:service:仓巡

- 状态：待验
- 领取：devin-o-worker-T-0448 @ 2026-11-08
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:face:service:仓巡`，草案锚 fin_ticket_draft#7）

## 病灶（升件行证据引）

病灶签名：`观测盲:face:service:仓巡`（kind=探针类 `观测盲`，key=病灶键 `face:service:仓巡`）

末行升件原文：
> 面清单新面 face:service:仓巡：机器可列面 service 类新增 仓巡——上拍快照册外新面（「该建未建」类残余差集；探针只报不修，合法性人工过基线）

证据引（ops_event 行 id）：#3100

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='face:service:仓巡' ORDER BY id DESC LIMIT 20
```

## 疑似根域

评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/portfolio/hunger/facesnap.go+depblind*（面清单基线册——service:仓巡 新面合法性核+入册/拒册判）；
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

### 根因（生产取证 2026-11-08）

- 仓巡=guard Warden 在册员工（`fin_employee_registry` risk 域
  active，30s 心跳实证）——合法席位非幽灵；ops_event 写径=
  T-0419 裸袋回填（bag.go）+T-0423 守卫环自残探针
  （loop.go suspendOpen/loopEscalate）——两写径 2026-11-07 才
  首落行（裸袋 6 行+开仓挂起/closing 滞留升件），此前仓巡
  ops_event 零行史。
- 新面成因链：service 面=ops_event 近 7d `DISTINCT seat` 活跃
  投影（facesnap.go 设计口径「写者席即服务面投影」）——仓巡
  事件驱动间歇写（事故期每 2h、平静期零写），首写拍入集→上拍
  快照册外→「面清单新面」行+观测盲升件 #3100→T-0393 起草拍→本票。
- 病灶本体=合法间歇写席**无持久基线位**：depBlindWhitelist 有
  对级豁免册而面级无对应册——仓巡事故平息 >7d 出集→下次事故
  再写=再现新面→再升件再起草，无限循环（「该建未建」残余差集
  的册外盲区——面宇宙人工过基线后新增才值得看，间歇面需要
  登记位续判）。

### 实现摘要（枝 t/0448-face-svc-blind）

- `facesnap.go` 面清单人工基线册立册：`faceBaselineWhitelist`
  注册式——key='face:<kind>:<face>'（esc_scope 同键，'*' 尾
  前缀通配照 depBlindExempt 同式），value=理由键。册内面差集
  照落「面清单新面」行携「基线册=<理由键>」留账但**升件免**
  （drafter 只扫 event='升件'——断「窗摆→升件→起草→立票」
  循环）；拒册=不登记，册外新面照升件等人工判。
- 登记 `face:service:仓巡`（在册员工+T-0419/T-0423 写径合法+
  间歇写席窗摆不重复升）。
- `depenum.go` depConfigReaders 补「meta」读者册——T-0455
  fin_0143 种 `meta.autoapprove.registry` 键域未入册→
  `config:meta→∅` orphan 误报致 depblind 六桩红（master 既有
  破闸，本票 depblind* 票面面内同族解阻；读者=
  dm_autoapprove.sh 机批闸实证——非签名病灶本体）。

### 测试证据

- 先红桩（T-0388 律）：`TestProbeFaceDiffBaselineExempt` 修前
  红（册内面升件未免+新面行缺基线册理由键 n=0 实证）→修后绿
  ——册内 whsvc 新面行照落携理由键+升件零；册外 badsvc 对照
  新面行+升件齐=豁免不误伤。
- `go build`/`go vet`/`gofmt` 净；hunger 全包 `go test` 绿
  （depblind 六既有桩 meta 册补后复绿——修前 master 同跑同红
  实证非本 diff 引入）。
- check_all 分段：build/vet/import/gofmt/env-reads/rejects/
  config-write-domain/migrations/bridge-audit/ops-chain 全绿；
  **tolerate-report+seat-writes 两红=master 存量**（T-0445 同日
  同判——scripts/*.sh `|| true` 未册+决策席直 commit 越权档在案，
  与本 diff 零交集；tolerate-anchors 修复枝 t/tolerate-anchors@
  6e4e27c 已在案待 merge）；go test 定向谱（hunger+反向闭包+
  恒测册 17 包）手动同跑全绿（cmd/core/lifecycle/trading/
  guard/risk/ledger 全 ok）。

### 生产谱面零误报审计（范围④）

- 册面=单键精确匹配零通配——生产近 7d 面升件 12 键
  （dep_pair leg/subreq×4+service×5+table×3，兄弟票
  T-0449/50/53/54 在案）与册零交集，册外面照升件零误伤；
  service 册外四面（业主/配管/O-Dev交互席/--take）各有己票
  域不越权代判。
- 仓巡已入快照批 9/10（机制自愈在案）——册=窗摆期保险丝非
  现报抑制；面清单新面行照落留账（每 >7d 静默窗摆至多一行，
  drafter 只吃升件不误起草）。
- meta 册补预堵：fin_0143 未部署生产（app_config 无 meta.*
  实证）——部署后 `config:meta→∅` orphan 不炸票（读者实证
  在案）。

### 偏离/留案

- `depenum.go` meta 册补=票面面内 depblind* 同族解阻（master
  既有红不修 scoped 测试无法全绿）；scripts/ tolerate 破闸
  超票面面未碰（三禁守）。
- check_all 绿戳不可达=master 存量红（同 T-0445 判例）——
  本枝 HEAD commit 留本地共享 gitdir（决策席可直 merge），
  推送/发窗待存量红消化后补。
