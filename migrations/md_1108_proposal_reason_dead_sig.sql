-- md_1108 fin_proposal 退役死因词表扩 'dead_signature'（工单O-38942——
-- fin_0251 对偶：md 副本 fin_proposal 镜像表同词表，projector
-- uProposalReverse 写 retire_reason='dead_signature' 不撞 CHECK；
-- 两链 ALTER 归一化全等同 fin_0240/md_1104 判例）。

ALTER TABLE fin_proposal DROP CONSTRAINT IF EXISTS fin_proposal_reason_vocab;
ALTER TABLE fin_proposal ADD CONSTRAINT fin_proposal_reason_vocab
	CHECK (retire_reason IS NULL OR retire_reason IN (
		'eval_rejected','stale_expired','armed_ttl_expired','falsified_回炉',
		'superseded_armed','venue_untradeable','manual','other','legacy_unrecorded',
		'dead_signature')) NOT VALID;
ALTER TABLE fin_proposal VALIDATE CONSTRAINT fin_proposal_reason_vocab;
