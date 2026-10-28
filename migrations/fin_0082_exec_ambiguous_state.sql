-- fin_0082 fin_execution.state CHECK 集扩 'ambiguous'（T-0208 三巡：
-- 歧态势写径既有即有——CHECK 集从来不含=静默败病灶根修）。
ALTER TABLE fin_execution DROP CONSTRAINT fin_execution_state_check;
ALTER TABLE fin_execution ADD CONSTRAINT fin_execution_state_check
  CHECK (state = ANY (ARRAY['open','legged','rolled_back','closed','failed','rollback_failed','ambiguous']));
