-- 0118 T-0379 fin_venue_candidate 状态机补尾——registered 后 probed/mapped/paper
-- 三态（broker 接入标准径：凭据→探测→映射→paper 全链）。桥表 DDL 同形断言⑥：
-- md_1090 正本链同语句逐字。
ALTER TABLE fin_venue_candidate DROP CONSTRAINT IF EXISTS fin_venue_candidate_state_check;
ALTER TABLE fin_venue_candidate ADD CONSTRAINT fin_venue_candidate_state_check
    CHECK (state IN ('candidate','measuring','recommended','registered','rejected',
                     'probed','mapped','paper'));
