# 施工返修②提示词：T-0314——fin_0103 补权链+skills/索引.md 收

**角色**：`[角色:施工]`　**票**：T-0314 债清　**分支**：`t/0314-econ-lesson-loop` 续提
**债务源**：裁定席首拍验收时手工补的权（经验库条目
`grant-not-in-migration-silent-dead` 在案）——手工权=解锁术非正本，
须回 canonical 链；skills 索引生成件卡 D-651 部署树进不了 master。

## ①`migrations/fin_0103_roster_grants.sql`

生产已手工执行的权写回链（条件 DO 块照 fin_0100 同款形——role 存在判）：

```sql
-- 0103 T-0314 债清：fin_econ_lesson/app_config 消费角色补权回链
-- （roster_rw=dev 裁定席隧道凭据——裁定三步写面钦定窄白名单：
--   lesson SELECT+UPDATE 裁行；app_config UPDATE 机械回改点 gate.* 键。
--   手工补权 2026-11-03 已生效，本件回 canonical 链防全新部署哑死）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT SELECT,UPDATE ON fin_econ_lesson TO roster_rw';
        EXECUTE 'GRANT UPDATE ON app_config TO roster_rw';
    END IF;
END $$;
```

- migrate_table.go `{103, "fin_0103_roster_grants.sql"}` 登记；
- domain.tsv 无新增表——本件纯 GRANT。

## ②`skills/索引.md` 收编

master 工作树现留 `kbgen skills` 生成的新索引（含裁定席条目——
D-651 部署树拒收故滞留）。本分支里 `go run ./tools/kbgen skills`
重跑后 `git add skills/索引.md` 一并提交——worktree 径可收。

## 验收

- `check_migrations` 七断言过（fin_0103 链内唯一/序号实存）；
- fresh 库 migrate 后 `roster_rw` 权面=生产现态（SELECT/UPDATE lesson+
  UPDATE app_config）；
- skills/索引.md 含裁定席行入库。

## 停线

- 纯 GRANT 件——禁改表结构/插数据；
- 幂等——DO 块 role 存在判+重跑零错。
