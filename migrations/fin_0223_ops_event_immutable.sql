-- fin_0223: ops_event 不可变机械闸（首性审计 2026-11-21 整改件）。
-- 病灶:append-only 铁律在 grant 层是约定——o 表主权含
-- UPDATE/DELETE/TRUNCATE,任意持 DSN 会话物理可改,检出只靠
-- ops_chain_verify() 周期验(可溯非预防)。本件把「事实只插不改」
-- 升库层前置闸:UPDATE/DELETE/TRUNCATE 一律 RAISE——对全角色含
-- 表属主生效,与 sem_gate 同型(SQL 触发器非应用层自觉)。
--
-- 旁路一:测试实例(o_test_s<N> 库名)放行——27 个篡改现场测试零改动;
-- current_database() 会话内不可伪造,且测试库内的 ops_event 与产线
-- 真相无关,放行无泄面。
-- 旁路二:合法订正/回填=迁移 tx 内显式 DISABLE TRIGGER:
--   ALTER TABLE ops_event DISABLE TRIGGER ops_event_immutable_trg;
--   ... UPDATE/DELETE ...
--   ALTER TABLE ops_event ENABLE TRIGGER ops_event_immutable_trg;
-- (DISABLE 须属主权——随手 UPDATE 绝径,蓄意破闸=DDL 留痕+哈希链
--  周期验检出,与同族闸的威胁模型同构)。
CREATE OR REPLACE FUNCTION ops_event_immutable() RETURNS trigger AS $$
BEGIN
    IF current_database() LIKE 'o\_test\_%' THEN
        IF TG_OP = 'TRUNCATE' THEN RETURN NULL; END IF;
        RETURN COALESCE(NEW, OLD);
    END IF;
    RAISE EXCEPTION 'ops_event append-only: % 禁——事实只插不改(订正径=迁移 tx 内 DISABLE TRIGGER 明示)', TG_OP;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ops_event_immutable_trg ON ops_event;
CREATE TRIGGER ops_event_immutable_trg
    BEFORE UPDATE OR DELETE ON ops_event
    FOR EACH ROW EXECUTE FUNCTION ops_event_immutable();

DROP TRIGGER IF EXISTS ops_event_immutable_trunc_trg ON ops_event;
CREATE TRIGGER ops_event_immutable_trunc_trg
    BEFORE TRUNCATE ON ops_event
    FOR EACH STATEMENT EXECUTE FUNCTION ops_event_immutable();
