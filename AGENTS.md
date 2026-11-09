# O — 纪元3家电律

套利基金系统（前身 NOX）。**纪元3（2026-11-09，D-0045）**：内联家电化——
本仓=钱径家电+探测+免疫闸+事实账；工程认知全在盒外会话。
纪元2官僚层（票/相位/席位/草案/物料化/metabus）已裁撤，
归档锚=`arch/pre-appliance-2026-11-09` tag（旧树一 checkout 可回）。

## 形态（D-0045 钦定五环）

```
感知: 探测薄壳——探针写 ops_event 信号行，只述事实
认知: 外席会话——被唤进场「扫信号→处置→落账→收口→死」
行动: 修码/修数/部署——同会话内完成
验证: check_all→绿戳→merge→pre-push 免疫链（钱径全谱不议价）
记录: ops_event append-only + KB条目 + D档
```

常驻盒内件（全部）：o-core 钱径 / 采集数据面 / PG / 探测薄壳 /
部署管道（check_all·green-stamp·mirror_push·tunnel，被动件有 merge 才醒）。

## 进仓协议

1. 读本文件 + `docs/decisions/D-0045-RSI薄环钦定-工程径裁撤内联家电化.md`
2. `git pull`
3. 扫信号面：`SELECT id,ts,seat,event,ticket,note FROM ops_event
   WHERE event='升件' AND ts>now()-interval '72 hours' ORDER BY id DESC`
   + `fin_eng_lesson` pending（教训矿候选）——未决信号即工作入口
4. 开工：`git worktree add /opt/o-wt/<名>` → `bash scripts/bootstrap_worktree.sh`

## 角色（纪元3收敛）

| 面 | 谁 | 干 | 禁 |
|---|---|---|---|
| 外席 | 业主授权会话（devin/外审/施工同格） | 信号处置/修码/部署/记账 | 钱径直改须过闸；红线三件归业主 |
| 探测 | 盒内薄壳 | 写信号行 | 不分类不起草不落件 |
| 业主 | 人 | 出资/供凭据/红线/方向锚 | ——（不插手运行） |

- 旧角色词表（管理/决策/施工/COS 席位拍）随纪元2退役；
  commit 标头沿用 `[域:摘要]` 形——域=钱径/探测/部署/簿记/法典 等自由词，
  修复类码面件仍须 `坑处置:` 显式行（D-585 闸在位）。

## 仓库地图（纪元3）

| 位置 | 内容 |
|---|---|
| `docs/总纲.md` | 方向锚——业主钦定层（禁擅改） |
| `docs/collab.md` | 协作纪律现行法（决策档/自检/工作面/会话尾/部署SOP）+纪元2史证附录（勿作流程依据） |
| `docs/工程纪律.md` / `架构约束.md` / `数据流.md` | 工程法典/结构律/链路正本（沿用） |
| `docs/decisions/` | 决策档案（D-0045 起新章） |
| `docs/工程经验库/` `经济经验库/` | 坑/案 canon——有坑即写条目+kbgen 重建 |
| `skills/` | 能力账 canon（在用中证错必修） |
| `internal/portfolio/hunger` | 感知+钱维护自治环（探针/Detect/Respond/教训矿工） |
| `internal/portfolio/lifecycle` | 策略生命周期（开仓→paper→live→退役——钱径） |
| `cmd/{core,collector,projector,research,sweeper}` | 部署件 |
| `scripts/` | 部署管道+免疫闸（无 meta/ 席位件） |

## 铁律（纪元3不变的）

- **钱径全谱闸**：动 `internal/{risk,execution,ledger,trading}` / cmd/core
  =check_all 全谱+戳+merge，无快道。
- **ops_event append-only**：事实只插不改；相位/状态靠事件推导非 UPDATE。
- **部署树戒律**：/opt/o 只 pull/merge/deploy，码面改动一律 worktree。
- **红线三件**：资金扩投/新平台/席位立换=业主件，外席禁裁。
- **私有记忆禁单存**：坑/口径/判据当下落共享层（KB/D档/ops_event），
  会话蒸发=对项目不存在。
