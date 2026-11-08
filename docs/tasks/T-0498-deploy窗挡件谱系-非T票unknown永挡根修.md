# T-0498 deploy窗挡件谱系根修——非T前缀票unknown→永双挡img+oci

- 状态：收口待审（施工收口——回决策席常径裁）
- 面：工程基建（机械轨部署闸）；急迫度=中（当前靠24h兜底节拍放行，非死锁但批积窗压长）

## 病灶（2026-11-09 实发在案）

`scripts/meta/deploy_window_due.sh` 判据②挡件谱系：
`ticket_block_faces` 对 pending/review 相位的票调 `ticket_refs`（支式
`t/NNNN*`）+`/opt/o-wt/tNNNN` 工位两面找代码面；两皆空=`unknown`→
`mark_blocked` 保守双挡 img+oci。

**非 T 前缀票结构性中招**：跟踪票/总票（GW-001）、伪键票
（DEPLOYNF/DOOR4/GATEFIX/PIPEFIX/BASESWP/DBP3a 族）无 T-NNNN
编号 → `num=${1#T-}` 剥离失效 → 支式 `t/GW-001*` 永不存在 →
`saw=0` → `unknown` → **凡挂 pending/review 相位即永双挡**。

实证：GW-001 六段全 merge 后，各段收口移交行把它反复顶回 review
相位 → deploy_window state=有挡 rc=11 持续，批 9 件龄 3.9h 堆积，
全靠 ≥24h 兜底节拍逃生门放行（DEPLOY-WINDOW-OVERRUN 升件随发）。

## 范围

1. `ticket_block_faces` 判据补「无代码面正证」分支：支/树皆无时，
   查票文件涉及面——票面涉及路径全非部署面（deploy-faces.tsv 否列）
   → `none` 不挡；无法判明才保留 `unknown` 双挡现行保守语义。
   （「支树皆不可定→双挡」的保守只对「可能有码而看不到」成立——
   票面自证零部署面的票不在此列。）
2. 非 T 前缀票的 `num` 剥离面：支式/工位推导对非 T-NNNN 票应
   显式不可定而非误拼 `t/GW-001*`（字面错拼面虽小，语义要钉死）。

## 判据

- 人造 pending 票：票面仅 docs/** 涉及+无支无树 → 不挡窗；
- 人造 pending 票：票面无涉及面可判+无支无树 → 仍双挡（保守不松）；
- GW-001 形态（跟踪票 review 态+仅 docs 涉及）→ 不挡窗；
- 既有 T-NNNN 票支/树分类行为零回改；
- 全谱零新增 FAIL/SKIP。

## 备注

- 连案：`deploy_window_due` 是部署闸——改动面属闸件域，判据②
  保守性红线=宁可误挡不可误放，修的是「误挡有正证可消」的子集，
  非放松判据本身。
- 立票依据：2026-11-09 批 9 件积压 3.9h 实证 + GW-001 永挡鬼影
  取证（ticket_refs 支式源码+相位投影复核）。

## 结果回填

**实现摘要**（`scripts/meta/deploy_window_due.sh` 单件+桩测扩，零新进程）：

1. `ticket_block_faces` 支/树推导收窄至 T-NNNN 规整形（≥4 位数字+可选
   字母后缀，同 check_ticket_numbers 字母后缀形）——非 T 前缀票
   （GW-001/DEPLOYNF/DOOR4/GATEFIX/PIPEFIX/BASESWP/DBP3a 族）与异形
   id 支式/工位推导**显式不可定**，不再误拼 `t/<id>*`/`/opt/o-wt/t<id>`；
2. 新增 `ticket_doc`/`ticket_doc_faces` 票面自证判据：支/树皆无
   （saw=0）时查票文件涉及路径——票件定位 `docs/tasks[ /done]`→
   `docs/infra/` 三域序首配（infra 域=GW-001/提案族跟踪票正本位；
   glob 元字符票=不可定防过配）；路径 token 抽取先归一仓根引用
   （`/opt/o[/wt/<树>]/`→`o/`，剥 `o/`、`./` 前缀），slash-token
   首段顶层目录闸（cmd/deploy/docs/internal/migrations/proto/scripts/
   skills/tools/web/.github/.devin）+根级部署件名单补抽
   （Dockerfile*/docker-compose*/.dockerignore/go.mod/go.sum/buf*.ya?ml）；
   路径集全非部署面=none 不挡、涉部署面=diff_block_faces 同谱按标挡
   （img/both 与支 diff 同判）、票件缺席/零 token=unknown 保守双挡
   现行语义不退化；
3. 支/树分类路径零回改——saw=1 时票面判据不介入（桩⑮钉死：docs-only
   票面+t/0907 internal 支→支 diff 判 both 双挡）。

**测试证据**（`deploy_window_faces_test.sh` ①-⑮ 全绿）：

- **先红桩双点**：同断言对旧版 deploy_window_due.sh（master 版）跑=
  ⑪红 `❌ ⑪rc=11≠0 … 待裁:T-0904(不可判→双挡) 待裁:GW-900(不可判→双挡)`
  ——docs-only 票被 unknown 永双挡=病灶复现；本枝同断言=⑪-⑮ 全绿 PASS；
- ⑪ pending docs-only 票（T-0904 tasks 件+GW-900 非T infra 件）→none
  不挡，双面全发 faces=image,oci；
- ⑫ 票面不可判（T-0905 无票件+T-0906 票件零路径 token）→unknown 双挡
  exit11（保守不松）；
- ⑬ 非T 票涉 internal/（GW-901）→both 双面挡 exit11（非T 不再永
  unknown，票面自证生效）；
- ⑭ 票面涉 web/（T-0908）→img 单面挡，oci 净面照发 faces=oci；
- ⑮ 支分类优先（T-0907 docs-only 票面+t/0907 internal 支）→支 diff
  判 both 双挡（零回改锚）。

**判据对账**：

- 人造 pending 票 docs/** 涉及+无支无树→不挡窗：⑪ PASS；
- 人造 pending 票无涉及面可判+无支无树→仍双挡：⑫ PASS；
- GW-001 形态（跟踪票 review 态+仅 docs 涉及）→不挡窗：⑪ GW-900
  同构（非T 前缀+infra 域票件+docs-only 涉及）PASS；
- 既有 T-NNNN 票支/树分类行为零回改：①-⑩ 原断言全绿+⑮ 支优先锚 PASS；
- 全谱零新增 FAIL/SKIP：check_all 绿戳对口（戳 sha 见收口行）。

**偏离说明**：

- ①「票面涉及面」按**全文件路径 token 抽取**而非仅解析涉及面字段——
  票面格式异构（`| 涉及面 |` 表行/`### 涉及面` 节/`- 面：` 行并存无统一
  钦定解析面），全文件抽取方向保守（散文提及部署路径也挡=误挡向安全
  侧），「无法判明→unknown」兜底不变；
- ② 票件检索域含 `docs/infra/`——docs/tasks[ /done] 惯例之外的必要
  补域（GW-001/提案族跟踪票正本位，票面病灶实证对象）；
- ③ 实证对照（真仓复跑）：GW-001→both（票面散见 internal/cmd 部署
  路径=自证非零部署面按标挡——「误挡有正证可消」子集外，保守语义
  保留，24h 兜底节拍仍其逃生门）；DEPLOYNF/DBP3a 等无票件伪键→
  unknown 双挡保留；T-0498 自身票面→none；
- ④ 范围外留案（呈裁量不代修）：`docs/tasks/` 之外票件（如
  docs/infra 外散件）与 `prop/`、`提案-*` 含斜杠/前缀 id 的 glob
  解析盲区——现行落 unknown 保守双挡，是否扩检索域归决策裁。
