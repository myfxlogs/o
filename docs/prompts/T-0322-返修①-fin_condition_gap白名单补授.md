# T-0322 返修①——o_fin_ro 白名单补授 `fin_condition_gap`（升件补裁件，非驳回）

> 事由：T-0322 施工升件行1564——钦定白名单不含 `fin_condition_gap`（COS
> 收件箱源②条件请求件承载表，D-0014/T-0177/fin_0072），停线禁自扩名单
> 在案。**补裁（决策席无头拍 devin @ 2026-11-04）**：采纳打包件
> `docs/cos/packs/2026-11-04-o_fin_ro-condition-gap-grant.md` 建议A——
> 白名单补授。技术域件非红线三件：SELECT-only、非凭据面、正是秘书席
> 应读之面，与 o_fin_ro 设计同源。

## 工作树/分支

`git worktree add /opt/o-wt/t0322r -b t/0322-condgap-grant master`——
原树 `/opt/o-wt/t0322` 票毕可收。

## 须落四件（勿扩）

1. `migrations/fin_0108_o_fin_ro_condition_gap.sql`——一行授：
   `GRANT SELECT ON fin_condition_gap TO o_fin_ro;`
   （注释引 T-0322 升件行1564+打包件+本补裁；fin_condition_gap 生产
   真存——fin_0072 钦定建表，勿用条件授权块）。
2. `internal/store/pgstore/migrate_table.go`——`{108,
   "fin_0108_o_fin_ro_condition_gap.sql"}` 登记。
3. `docs/数据流.md` 凭据面 o_fin_ro 行——白名单串补 `fin_condition_gap`
   （同 commit 硬律）。
4. `skills/COS/SKILL.md` 源②注——删「授权落地前缺席=如实记读面缺」
   改「已授」（迁移落地后实话）。

## 验收锚（亲跑实录贴回填段，禁口报）

- [ ] dev 库 `o-postgres@127.0.0.1:5433`：`SET ROLE o_fin_ro;
  SELECT count(*) FROM fin_condition_gap;` 通（迁移后）。
- [ ] 回归钉：`SET ROLE o_fin_ro; SELECT 1 FROM app_secrets LIMIT 1;`
  仍拒；`INSERT INTO ops_event` 仍拒——补授不伤黑名单/写禁。
- [ ] `check_all` 全谱绿戳随收口。

## 停线

- 他表一概不授（本件只补 `fin_condition_gap` 一行）；
- o_fin_ro 写权仍零、凭据域仍结构性不授——越此=停。
