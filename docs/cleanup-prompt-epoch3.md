# 纪元3 全项目清洁施工提示词（外派径——业主令 2026-11-09）

你是施工席。本单是 O 仓（/opt/o）立项→纪元3 全量清洁——死件清理+经验库归纲。
读完 `/opt/o/AGENTS.md`、`/opt/o/docs/decisions/D-0045-RSI薄环钦定-工程径裁撤内联家电化.md`、
`/opt/o/scripts/cognition/runbook.md` 再动手。

## 背景三句

纪元3（D-0045）裁撤了票/相位/席位派发整套工程径官僚件——内联只剩
钱径+探测薄壳+唤知机+部署管道+PG。但仓里仍有纪元2 残留：引用死件的
文档段、退役脚本的部署件、席位卡 skills、经验库双层积压。

## 工作面

`git worktree add /opt/o-wt/cleanup -b t/cleanup origin/master`，
全程在 wt 施工，check_all 绿了再支推+部署树 merge（单写者律）。

## 任务一：死件清单先行（禁先删后列）

先产出 `/opt/o-wt/cleanup/docs/cleanup-inventory.md`——全仓死件
盘点表：`| 路径 | 死因 | 处置（删/归档/改写）| 引用面核验 |`。
死因判据（命中其一即候选，核验后定）：
- 被零文件引用（rg 全仓搜文件名/符号名零命中——自身引用不算）
- 语义只属纪元2（票机/相位机/席位派发/dispatch_worker/dm_tick/
  草案面/fin_ticket_seq）
- 已被替代件取代（写明替代者在哪）

已知死候选（核验后处置，勿照单全收——「零引用」必须实测）：
- `docs/工程经验库/待归纳.md`——业主令 10-25 已废待归纳层
  （「不待归纳——排队=下个 session 盲区」），文件本体仍存
- skills/ 席位卡七件（CEO席/COS/CTO席/施工席/分析席/外审席/裁定席）
  ——席位派关机已裁，但先核 runbook/AGENTS 有无活引用再裁；
  有残值内容归并进 runbook 或封存不删
- `docs/tasks/` 已清空（T-10000 纪元切换批清）——目录留否裁决
- deploy/ 逐件对账宿主 systemd 实况（`systemctl --user list-units`
  dev 宿主+`ssh ubuntu@149.118.58.98` OCI——单元不在=件死；
  o-cognition.timer/o-cognition-dispatch.service 现役勿动）
- tools/ 一次性探针（probe2/posprobe/fillsprobe/mt5probe 等）——
  取证完即弃件的裁；通用件（reconview/legarb/okxread 级）留
- scripts/ 逐件核引用面（check_all 引用链/systemd 单元/文档）
- docs/ 过期信息（collab.md 纪元2 段外的陈旧节/handoff/路线图/
  README 过期段）——改写优于删，删要有据
- 宿主侧：dev/OCI 的 disabled+dead 单元文件、env 死键、
  /opt/o-wt/* 陈树（`git worktree list` 对账——merge 毕的树收）

## 任务二：经验库归纲

- `条目/` 390 件按 risk/kind/status 分层核：kind=pitfall 主面；
  status=superseded 的归 `归档/`（目录不存则建，索引同步注归档位）
- `前身-nox/` 裁决：有引用价值留档并 README 注明归档态；
  无引用价值整体归 `归档/前身-nox/`
- `待归纳.md` 删除（死因在案）；存量条目内容逐条判：已 canon 化
  的删行，未 canon 的转正成条目或标弃
- `索引.md`/`检索物料.tsv`/`技能清单.md` 跑 `go run ./tools/kbgen gen`
  全量重建——生成面与条目录对拍零漂移
- README.md 更新为现行组织法（分层/归档规约/新条目流程——
  业主令 10-25「有坑即写条目+kbgen 重建」钦定径在文）

## 任务三：collab.md 真整理（非批注）

注意：master 上该件只有文首覆盖条无生死表（sigfix 支上 f01ba9945
有一版生死表批注，本单取代之——勿 merge 那支）。施工：
按覆盖条判词把死段正文实质移除或压入文末「纪元2 史证」附录，
活段（§6/§7/§10/§11/数据流正本及 §4/§9 存活部分）保留为正文。
目标是「全文皆可作现行法读」而非「全文需对照生死表读」。

## 红线（碰即停）

- 不动：`migrations/**`（链史 append-only）、`ops_event` 任何行、
  钱径码面（internal/trading、risk、guard、ledger 写径——
  除非确认死件且有引证）、`deploy/immune_paths.txt` 册内件、
  .git 历史
- 不删「可能被引用」的件——引用面核验必须实测贴命令+输出
- 删文件=git rm 留痕；改脚本=check_all 全绿才收口
- systemd 单元文件动前要核宿主实况（单元在跑=活件勿删）
- 完工验收锚：check_all mode=full 绿戳 + cleanup-inventory.md
  行数与 git rm 数对账 + 每项处置有死因引证

## 收口报告格式（发回审计）

```
1. inventory 行数 / 删件数 / 改件数 / 归档数
2. 每类处置的样本引证（命令+输出 3 条）
3. check_all 戳 sha
4. 拿不准搁置清单（若有——勿强裁）
```

## 自审记录（发件席填）

| 编号 | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 施工席可能凭「看起来旧」删活件 | 越权风险 | 钉死「零引用必须实测」+死因判据表+验收锚对账 |
| 2 | KB 归纲是判读面非机械面 | 质量风险 | 分两层——分层归位（机械）+拿不准进搁置清单（判读回审） |
| 3 | collab.md 有分支批注版冲突 | 合流风险 | 明文指定「真整理取代批注，勿 merge f01ba9945」 |
| 4 | 宿主单元实况本地不可全判 | 事实风险 | 钉 ssh 核实径+「单元在跑=活件」判词 |
| 5 | 提示词范围跨 docs/scripts/skills/deploy/宿主 | 蔓延风险 | 任务三件分界+红线列+搁置清单出口 |
