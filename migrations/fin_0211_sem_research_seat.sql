-- 0211 工单#24412 研究机落账——ops_event_sem_matrix 补「唤起×研究机」
-- 放行组合（dispatch_research.sh 跨机单飞锁 seat 分账——唤知机7969/
-- 审计机7970/施工机7972/自我机7971 在案,研究机=7973 同座不同账;
-- 研究席 event='处置' 已册不涉）。
-- 「唤起」=闸内词（矩阵在册）,新 seat 未经入册=gate loud 拒写
-- （46db6df0d 部署后首唤实证被拒=闸做对了,本件补册非松闸）。
-- 研究机其余写面（唤起败/唤起果/工单/会话收口/量表）皆闸外词直通。
INSERT INTO ops_event_sem_matrix(event,seat,note_like,basis) VALUES
('唤起','研究机','','研究席唤起锁 seat 分账——工单#24412 四触发钦定唤起径（唤知机7969/审计机7970/自我机7971/施工机7972 在案,研究机=7973 锁键分域同座不同账）')
ON CONFLICT (event,seat) DO NOTHING;
