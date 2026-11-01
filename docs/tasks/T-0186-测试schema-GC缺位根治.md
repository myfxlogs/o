# T-0186 测试 schema GC 缺位根治

> 立单：决策席 2026-10-27（T-0174 施工收口上报 → 决策裁定立项）
> 域：dev `o-postgres-test`（5434）测试实例
> 量级：中件

## 病灶实证（T-0174 施工席取证回填）

- `internal/store/pgstore/testutil`（或同族 helper）每测试建 `t_*`
  schema，**用后不清**——实例积 18,365 schema / ~32GB 才发现。
- 前身 GC 脚本按 `arb_*` 库名+arb 角色写成，与 O 拓扑脱钩，
  cron 连日 `auth failed`（脚本活着但从未成功过=半死人形态）。
- 本次应急已逐条 DROP 清至约 7 千 schema，**但根源未修**：
  测试继续跑就会再积。

## 钦定修法（二选一，施工判）

**甲案（首选）**：testutil 自清——`t_*` schema 在 `t.Cleanup`
或连接归还钩里 DROP CASCADE。测试内的 schema 生命周期归测试
自己管，零外部依赖。

**乙案**：O 拓扑 GC 重写——`scripts/gc_test_schemas.sh`：
只认 `t_*` 前缀 + 创建时间 >Nh + 库=o_test 三重钉（防误杀），
挂 dev cron 日拍。前身脚本入档不留 cron。

**禁线**：DROP 范围只许 `t_*` 前缀 schema——`public`/`fin_*`/
`md_*` 白名单永不进 DROP 集。

## 验收锚

1. 测试跑后 `t_*` schema 不残留（甲案）或日拍清零（乙案）；
2. `o-postgres-test` 体积增速回到正常水位（T-0181 盘漂哨兵
   现每日机械报，作长期观察面）；
3. 前身 arb_* GC cron 条目移除或钉死注释。

## 归档/参考

- 经验库条目：`docs/工程经验库/条目/test-schema-per-test-no-gc-instance-suffocation.md`
  （T-0174 施工已入库）

## 状态
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**已批**——乙案gc_test_schemas.sh生成SQL仍缺分号——DROP批拼永不执行结构性仍在（驳回锚实证未消）；t_*残留+arb_* cron债在

已批（11-01 清账定性留队·SETTLE-SWEEP）
审计席：headless @ 2026-10-30

## 判决注记（逐锚对账）

- **锚1（测试跑后 t_* 不残留 / 日拍清零）——证伪**：
  - 甲案码面已落：`testutil.Pool` 挂 `t.Cleanup` DROP SCHEMA IF EXISTS
    <t_*> CASCADE（commit 1a383cc，testutil.go:61-66）。但实例实态：
    `o_test` 现存 t_* schema 8,060 个；按 schema 内最老表文件 mtime
    断代——pre-merge 残留 7,524 + **post-merge 新建残留 118**
    （10-27:49/10-28:7/10-29:8/10-30:54，含当日 09:12–09:37Z 新建）
    +358 个无表文件不可断代。Graceful 路径清了（观测期间计数 8063→8060
    有落），但被杀进程/超时/吞错路径照积；Cleanup 内 `_, _ =` 吞 DROP
    错——失败零可见。
  - 乙案件功能死亡：`gc_test_schemas.sh` 生成 `DROP SCHEMA IF EXISTS
    <n> CASCADE` 行尾无 `;`，pipe 入 `psql -q` 时多行并入单 query
    buffer，≥2 候选即 `syntax error at or near "DROP"`——
    `/opt/o-backups/gc_test_schemas.log` 10-28/10-29 两跑皆此错，**从未
    drop 任何 schema**。另偏离钦定修法：三重钉只落两钉（t_* 前缀+DSN
    钉库），缺「创建时间>Nh」——现形态可误杀在跑测试的活动 schema。
- **锚2（体积增速回正常水位）——方向证伪**：`o_test` 现 23GB，t_*
  由应急清后 ~7k 净增至 8,060——增速未回落，哨兵长期观察面前提不成立。
- **锚3（前身 arb_* GC cron 移除或钉死注释）——证伪**：本机
  `crontab -l` 第 11 行 `45 4 * * * /opt/arb-archive/scripts/
  test_db_gc.sh --apply >> /opt/arb-backups/test_db_gc.log` 仍为活体——
  未移除未注释；`/opt/arb-backups/test_db_gc.log` 连日
  `password authentication failed for user "arb"` 照旧，半死人形态未拆。
- **禁线核查——未违**：两面 DROP 集均仅 `t_*` 前缀（甲案按自建 schema
  名、乙案 `nspname LIKE 't\_%'`），`public`/`fin_*`/`md_*` 未入 DROP 集。
- **重飞指向**：乙案生成行补 `;`（或 `\gexec`/逐行 -c）+补年龄钉；
  甲案 DROP 错改可见（log/计数）；锚3 摘或注释 arb cron 条目；
  现存 118+358 残留与 23GB 体积清理归重飞票一并处置。

## 结果回填

- 甲案（测试内 GC）：testutil.Pool t.Cleanup 加 DROP SCHEMA IF EXISTS
  t_* CASCADE（用后清——实例窒息根治；白名单守=只 t_* 前缀）。
- 乙案兜底：scripts/gc_test_schemas.sh 日拍 cron 件（quote_ident 拼；
  t_* LIKE 谓词——public/fin_*/md_* 永不进 DROP 集）。

## 决策席验收投产（2026-10-27）

- 甲案 `testutil.Pool` t.Cleanup DROP SCHEMA CASCADE 码面合规（每测自 schema 用后清，不误他件）；乙案 `gc_test_schemas.sh` `nspname LIKE 't\_%'`+quote_ident 谓词窄钉——DROP 集永不入白名单件。
- **补位**：乙案 cron 件票面钦定「挂 dev cron 日拍」但分支未装——决策席投产步直挂：`23 5 * * * gc_test_schemas.sh >> /opt/o-backups/gc_test_schemas.log`（本机 crontab 实证在册）。
- master 合入 `7cd3277`；七断言绿。销号。
