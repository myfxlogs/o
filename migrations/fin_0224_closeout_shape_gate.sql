-- fin_0224_closeout_shape_gate.sql —— 施工收口形闸（业主令2026-11-21:
-- 派单/收口三模板钦定「[施工收口:O-<票号>] @ sha=<40位全sha>」定稿形——
-- 既往漂例:收口报回不带sha/带短码/走记事事件,落地闸无从机读消费）。
-- 两段机械:
--   ①矩阵白名单:event='施工收口'×(施工/施工席) 入册——新相位词
--     不入册=闸外直通,入册后非册seat写该event被拒;
--   ②函数反向断言:note含「[施工收口:」者,event必须='施工收口'
--     且note匹配定稿形(长码40位sha),否则RAISE——错event/错形/
--     短码物理拒,塞记事绕行径断。
-- 与fin_0154矩阵闸同函数体末段前置——矩阵内事件语义不变。

CREATE OR REPLACE FUNCTION ops_event_sem_gate() RETURNS trigger AS $$
BEGIN
	-- fin_0224 收口形闸(先于矩阵判——note载施工收口签者双断)
	IF NEW.note LIKE '[施工收口:%' THEN
		IF NEW.event <> '施工收口'
		   OR NEW.note !~ '^\[施工收口:O-[0-9]+\] @ sha=[0-9a-f]{40}' THEN
			RAISE EXCEPTION 'ops_event_sem_gate: 施工收口错形——定稿形「[施工收口:O-<票号>] @ sha=<40位全sha>」(fin_0224;event=% note=%)',
				NEW.event, left(NEW.note, 80);
		END IF;
	END IF;
	IF EXISTS (SELECT 1 FROM ops_event_sem_matrix m WHERE m.event = NEW.event) THEN
		IF NOT EXISTS (
			SELECT 1 FROM ops_event_sem_matrix m
			WHERE m.event = NEW.event AND m.seat = NEW.seat
			  AND (m.note_like = '' OR NEW.note LIKE m.note_like)
		) THEN
			RAISE EXCEPTION 'ops_event_sem_gate: 拒写非法 (seat,event)——seat=% event=% ticket=%（合法性矩阵 fin_0154；相位驱动词×seat 白名单落库执行，说明类/矩阵外事件不受闸）',
				NEW.seat, NEW.event, NEW.ticket;
		END IF;
	END IF;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

INSERT INTO ops_event_sem_matrix(event,seat,note_like,basis) VALUES
('施工收口','施工','','fin_0224 施工收口相位入册——定稿形由函数形闸钉(note_like空=形归函数管)'),
('施工收口','施工席','','fin_0224 同上——机/席两写面在案(27499/27500实证)')
ON CONFLICT (event,seat) DO NOTHING;
