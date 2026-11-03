[角色:施工] T-0312 返修①——fin_0100 GRANT 补件+notify 空批次收窄

## 任务单
/opt/o/docs/tasks/T-0312-三服务器终态-OCI事件总线与无头决策席.md
（一轮裁决节）。原分支 t/0312-regime-push 续提，不切新支。

## 驳回件

**R1（硬伤）fin_0100 漏 GRANT**——SG 实盘已实锤：`projector_rw` 对现存
全部九桥镜像表持 SELECT/INSERT/UPDATE/DELETE 四权（information_schema
table_privileges 实查）；新 md_regime_fact/md_regime_memo 零授权=
部署后投影每拍 permission denied 静默败。fin_0098/0099 修过的同款坑。

修=migrations/fin_0100_regime_mirror.sql 末尾补条件 GRANT 块，照
fin_0098 先例（角色存在才授——测试库无角色零失败）：

    DO $$
    BEGIN
      IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'projector_rw') THEN
        GRANT SELECT,INSERT,UPDATE,DELETE ON md_regime_fact TO projector_rw;
        GRANT SELECT,INSERT,UPDATE,DELETE ON md_regime_memo TO projector_rw;
      END IF;
    END $$;

md_regime_push 为 SG fin 域表、写者=o-core 本体池——**不在授列**。

**R2（小疵）pg_notify 空批次也唤**——internal/projector/regime.go：
`SELECT pg_notify('ev_regime','')` 当前每 wake 必发（两表零新行也唤
推送面空跑一次 Scan）。收窄=两表 batch 合计非空才发，一行标志即可。

## 验收
- go build ./... + check_migrations + scout/notify/projector 包 test 全过
- 部署拍后实证件（决策席跑）：OCI projector 投影两表无 permission denied、
  SG 镜像落行、O 群实收、重启不重推——三重硬迹照旧票面锚

## 自审记录
| 编号 | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | R1 修法给全文 SQL 未只说原则 | 去歧义 | SQL 全文钉死免二次偏差 |
| 2 | md_regime_push 是否同授易误判 | 范围边界 | 明示不在授列（写者=o-core 池非 projector_rw） |
| 3 | 序列 USAGE 是否需授 | 排疑 | 显式 id 插入不走 bigserial 默认——不授，注释已明 |
| 4 | 部署实迹由谁跑 | 职责界 | 明示决策席部署拍后自验——施工只交码 |
| 5 | 原分支续提防散件 | 流程 | 钉原分支不切新支 |
