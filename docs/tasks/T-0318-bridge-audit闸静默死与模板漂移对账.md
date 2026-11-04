# T-0318 bridge-audit 闸静默死修+模板漂移对账——master 存量红追修件

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` devin @ 2026-11-04（T-0316 裁决门禁对账实证） |
| 领取 | （空=未领） |
| 优先级 | 高——check_all bridge-audit 段 master 恒红=全谱闸阻一切后续票 |
| 涉及面 | `scripts/check_bridge_registry.sh`、`deploy/o-research.env.example`（+可能 deploy 其余模板键）；**实机 env 面只读对账不直改**（漂移去向裁后另定） |

## 背景（裁决实证，勿再诊）

T-0316 裁决亲跑 bridge-audit：①授权列 ②存在性列绿后脚本静默 exit 1
零 FAIL 行——③模板列 `av=$(env_field)`/`tv=$(tpl_field)` 于
`set -euo pipefail` 下，grep 无匹配 rc=1→pipefail→set -e 直接杀死，
设计意匠 `[ -z "$av" ] && red` 永不到达。master HEAD 同诊复现。

绕过静默死逐对拍实查三真况：

1. `/home/ubuntu/o-collecto.env` **SG_PG_DSN 键整缺**（对拍①杀死点）；
2. `o-research/env ROSTER_PG_DSN` 实=`roster_rw` vs 模板
   `o-research.env.example` 声=`research_rw`；
3. `o-research/env FIN_PG_DSN` 实=`review_rw` vs 模板声=`research_rw`。

漂移去向待裁：模板=图（生产用户名细分为 roster_rw/review_rw 大概率
是图旧实新——以 PG 角色实存+代码引用面定改哪侧）。

## 目标

bridge-audit ③模板列 fail-closed 语义保、静默死灭：读败/键缺/漂移
全部走 `red` 显报路径不再吞行；模板×实机 DSN 对拍归一（改图或改实，
裁后执行）；check_all master 转绿。

## 验收标准

- [ ] 模板列六对拍全走显报径——人为断一键实测 `FAIL[bridge-audit]` 行
  出现非静默死（贴实证）。
- [ ] 漂移三行裁定归一（改 .example 或改实机 env——裁词落回填）后
  ③模板列+④水印列实跑绿（列 OK 行贴出）。
- [ ] `scripts/check_all.sh` master 全谱绿盖戳（bridge-audit 段含）。

## 范围外

- 不动 PG 角色授权本身（roster_rw/review_rw 存废归 DBA 面另票）。
- 不改桥域表集/授权矩阵语义（①②列零动）。

## 追加件：worktree 坟场清除（业主令 2026-11-04 随棒）

`git worktree list` 实测 **376 条滞留**——meta 时代各 harness 攒的坟场。
分两段：

- **A 段·零险**：`git worktree prune`——目录已不在的条目直接摘
  （git 自证安全）。
- **B 段·判死**：目录仍在的逐条判——脏树（`git status --porcelain`
  非空）一律跳过保留；净树且分支已合/已删/mtime>30d 者摘；
  近 30d 有动者保留并列表报出（活人票不碰）。O 域在飞支
  （t/0316 链、T-0317 返修支）绝对豁免。

### 验收

- [ ] `git worktree list` 降至在飞集（剩件逐条列名+保留由）。
- [ ] A 段 prune 数、B 段摘数、跳数三数回填票面。
- [ ] 零误摘——被摘件必须「净树+分支死/老」双证在案。
