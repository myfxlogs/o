-- 0067 md_macro_event 第八桥表 SG 镜像（T-0167/D-0032 §二裁：report 留 SG 本地读
-- 不加跨机池——全表投影 ~30 行 event_date PK，research 读 PG-md 正本本地）。
-- DDL 与 md 链 md_0020 正本归一化同形（check_migrations 桥表断言钉——回滚径
-- 「SG 镜像表即正表」前提）；种子行不落 fin 链——投影全表自带。
CREATE TABLE IF NOT EXISTS md_macro_event (
    event_date date PRIMARY KEY,
    kind       text NOT NULL CHECK (kind IN ('fomc','cpi','nfp')),
    title      text NOT NULL,
    source     text NOT NULL DEFAULT 'seed'
);
COMMENT ON TABLE md_macro_event IS '宏观高冲击日历事实域（md_ 前缀——种成事实非行情流）';

-- 保留账入册（T-0153 清道夫 fail-closed——新表不入账=报缺拒动；event_date PK
-- 全表 ~30 行=state 不删行）。
INSERT INTO fin_data_retention(table_name,class,note)
VALUES ('md_macro_event','state','状态表不删行——第八桥表镜像（report 本地读面）')
ON CONFLICT (table_name) DO NOTHING;
