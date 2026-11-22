-- fin_0232: md_klines/md_funding roster_rw SELECT 授权补齐（工单O-32603）。
-- 体制读数器官（regimedigest.sh 细胞+唤知会话 regime_prompt.md）输入面钦定
--   =md_funding 近窗+md_klines 主品种+ops_event 市场面事件；认知宿主机
--   （models/ant）唯一 PG 径=FIN_PG_DSN(roster_rw@SG)——md_klines bridge-
--   reverse 正本本在 SG 面但 roster_rw 零授予（has_table_privilege 实证
--   permission denied），md_funding 同。授权前读数件/对拍件全哑。
-- 单件双令：只授 SELECT 两表本体给 roster_rw（读径消费面），不动其他
--   role/其他表；pg_tables/pg_roles 存在性闸=镜像面/异域部署缺席不炸。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw')
       AND EXISTS (SELECT FROM pg_tables WHERE tablename='md_klines') THEN
        EXECUTE 'GRANT SELECT ON md_klines TO roster_rw';
    END IF;
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw')
       AND EXISTS (SELECT FROM pg_tables WHERE tablename='md_funding') THEN
        EXECUTE 'GRANT SELECT ON md_funding TO roster_rw';
    END IF;
END $$;
