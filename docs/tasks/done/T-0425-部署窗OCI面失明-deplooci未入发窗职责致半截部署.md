# T-0425 部署窗 OCI 面失明——deploy_oci 未入发窗职责致半截部署

- 状态：完成（ops_event 完成行 2026-11-09 在案——机械轨票翻）
  没有发 OCI」）
- 领取：t/0425-deploywin-oci @ /opt/o-wt/t0425
- 优先级：P1——部署机制失明（实证半截部署在产）
- 收口移交归：决策席
- 关联：T-0418（deploy_oci %h 修毕径通）、T-0416（票面留
  「SG镜像+collecto换件」双锚被机制漏掉）

## 背景（生产实证 2026-11-07）

部署窗 02:57 发 SG `o-core:06ab66a4`——但 OCI 侧二进制
（collector/projector/research/sweeper，host systemd 件）
**没换**：projector 老二进制不含 `tick_scan` 投影（strings
0 命中）→SG md_tick_scan 恒 0→选符域塌回订阅集→
「搜不到未订阅符+报价不实时」用户面实证。

根因：`deploy_window_due.sh`+决策席发窗动作只钦定
`deploy_image.sh`——**OCI 二进制部署（deploy_oci.sh）
不在发窗职责面**，两半部署制机制只盖了一半。票面生产锚
写「SG镜像+deploy_oci collecto换件」也救不了——机制不读
票面锚。

## 钦定语义

```
批面二分:
  镜像面(现状): diff碰cmd/internal/web/proto/migrations/
    Dockerfile*→deploy_image.sh
  OCI二进制面(补): diff碰 OCI二进制构建源
    (cmd/collector|research|eventbus|sweeper、
     internal/data|projector|research|mt5|venue——
     即deploy_oci.sh构建谱系)→同窗发deploy_oci.sh
  两集求并:任一面非空=批非空;两面各发各的;
    面不可判=保守按双发
挡件谓词:  不变(在飞码件+待裁码件)
半截升件:  镜像面发了OCI面没发(或反之)→
         必落ops_event「半截部署」升件——
         禁静默半部署
```

## 范围

1. `deploy_window_due.sh`：批面判定扩二分——镜像面集
   +OCI 面集各自机读（路径清单钉壳头同式）；
2. 决策席 SKILL 发窗动作节改：应发→各非空面各发
   （deploy_image.sh∥deploy_oci.sh），半截必升件；
3. deploy_oci.sh 自检补：构建源 diff 空时可跳（幂等
   快退零烧）；
4. docs/数据流.md 或部署档补「两半部署制」段——
   机制与票面锚从此同源。

## 验收

- 红桩：diff 只碰 OCI 构建源→窗判应发+deploy_oci 径
  走通；
- 红桩：diff 只碰镜像面→OCI 面跳过不误发；
- 半截闸桩：一面发一面缺→升件行实落；
- 生产实证：本次半截补发后 SG md_tick_scan 起灌；
- 绿戳对口+全链。

## 施工回填（2026-11-07）

- `deploy/deploy-faces.tsv`（新，路径分类正本——钦定「机制与票面
  锚从此同源」单本）：class 词表 nondeploy/image_only/<arm>；
  arm 谱系=go list -deps 五件实测（collecto/projector/research/
  eventbus/sweeper）；未列名=面不可判→保守按双发。
- `scripts/meta/deploy_window_due.sh`：批面判定扩二分——镜像面
  现状不动，OCI 面=镜像面∧¬image_only（tsv 否列，件缺=全面双发）；
  OCI 部署基线三源 env→盒上三臂 $ep.sha 最老件→ops_event
  deploy-oci 行，mb 基线兜未并支部署件，三源皆败=面不可判按非空
  保守计；批=两面并集（commits/票号/首件龄同锚）；次行 faces=
  机读面集；半截检出=两基线可序时 lag-range 碰陈面（含谱系岔疑升）
  →ops_esc「半截部署」前缀升件（24h 去重）。
- `scripts/deploy_oci.sh`：①b 谱系自检——臂部署 SHA（盒面
  $ep.sha 远端自取）merge-base..HEAD diff 全件证外（nondeploy/
  image_only/他臂行）=跳臂，不可判面/.sha 缺/diff 败=保守照换，
  O_DEPLOY_FORCE=1 压过；完工回执 deploy-oci 前缀 ops_event 部署行
  +feishu 同 deploy_image 径（SWAP_HAPPENED 闸防快退误落簿）。
- `skills/决策席/SKILL.md` 部署窗节：应发→读 faces= 各面各发
  （image→deploy_image.sh oci→deploy_oci.sh），半截必升件。
- `docs/collab.md` §11 两半部署制段、`docs/工程纪律.md` 部署窗
  触发律条扩径、`docs/数据流.md` ops_event 面 deploy-oci 行。
- 桩测两新件全绿：`deploy_window_faces_test` ⑤面（红桩①OCI 单面
  应发+半截升件实落/红桩②镜像单面不误发/双面批/批空/基线不可判
  保守计）、`deploy_oci_skip_test` ④面（全件证外幂等快退/臂级
  粒度/不可判面全换/.sha 缺照换）；`deploy_oci_execstart_test`
  回归绿。
- 生产实证：真盒跑窗壳判 `应发 faces=oci`+「半截部署」升件径走通
  （img=4fce5538 已新、oci=50800409 臂陈于 T-0420 合入码——机制
  正检出在产 OCI 欠账）；md_tick_scan 起灌锚成——PG-md 正本
  02:28:27 lag 18s 每 60s 一拍、SG 镜像 02:27:28 随行推进
  （投影含 tick_scan 换件生效实证）。

## 备注

- eventbus/sweeper 钦定谱系列 OCI 面但现无 deploy_oci 臂位——
  窗面发窗臂级全跳（其部署另径归另票，tsv 注在案）。
- 双基线谱系岔开期每 24h 至多一升（前缀去重）——下次 deploy_oci
  自 master 换件后谱系复序自消。
