# T-0498 deploy窗挡件谱系根修——非T前缀票unknown→永双挡img+oci

- 状态：已批（业主令 2026-11-09「立根修票」）
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
