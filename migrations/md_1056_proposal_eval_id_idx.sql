-- 0056 fin_proposal(eval_id) 索引入册（T-0127——T-0126 迁移事故处置时手建
-- 索引生产在、迁移序列无：新环境部署不自带重演嵌套循环风险面；
-- CONCURRENTLY 迁移器不可用退普通 CREATE——181k 行秒级）。
CREATE INDEX IF NOT EXISTS fin_proposal_eval_id ON fin_proposal(eval_id);
