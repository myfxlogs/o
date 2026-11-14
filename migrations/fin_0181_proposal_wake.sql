-- fin_0181_proposal_wake.sql —— 提案落账自唤缝补（棒5 观察实拍案——
-- dir_perp_mom 提案 425711/425712 与 dir eval 同 projector 事务 08:07:54
-- 落账,scan 竞在 commit 前跑完;其后 76 条 eval 皆同值 UPDATE 触发器
-- 抑制=零唤醒,提案饿死 fin_proposed ~12min 直至人工 pg_notify 即开两户。
-- 缝:提案消费全押 eval 实质变唤醒,写提案的人自己不唤——eval 静默期
-- 提案可饿死到下一批(dirmom 拍距 240min)。
-- 修:fin_proposed 落账即唤消费环(谁生产谁唤);复用 ev_eval 通道=
-- 消费侧零改。UPDATE 不唤(提案只 INSERT 进 fin_proposed;armed/retired
-- 迁移不属新生产)。

CREATE OR REPLACE FUNCTION ev_notify_proposal() RETURNS trigger AS $$
BEGIN
  IF NEW.state='fin_proposed' THEN
    PERFORM pg_notify('ev_eval','');
  END IF;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ev_proposal_trg ON fin_proposal;
CREATE TRIGGER ev_proposal_trg AFTER INSERT ON fin_proposal
  FOR EACH ROW EXECUTE FUNCTION ev_notify_proposal();
