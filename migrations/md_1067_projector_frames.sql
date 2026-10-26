-- 1067 projector 投影帧补全（T-0168/D-0030 §四——md 侧 wake 源）：0028 事件脊
-- 只盖 md_ticks/md_funding/fin_opp_eval，md_mark_index/md_signal/md_dcd_product
-- 无帧=projector 无法零轮询感知写拍；fin_proposal 正向 push 源（AFTER INSERT——
-- UPDATE 不发，回写不回环）。statement 级批合流同 0028 律（一拍一醒）。
-- 注意：fin_proposal SG 侧回写帧在 fin 链 fin_0070（同名异库，AFTER UPDATE OF state）。
CREATE OR REPLACE TRIGGER ev_markidx_trg AFTER INSERT ON md_mark_index
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_markidx');
CREATE OR REPLACE TRIGGER ev_signal_trg AFTER INSERT OR UPDATE ON md_signal
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_signal');
CREATE OR REPLACE TRIGGER ev_dcd_trg AFTER INSERT OR UPDATE ON md_dcd_product
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_dcd');
CREATE OR REPLACE TRIGGER ev_proposal_md_trg AFTER INSERT ON fin_proposal
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_proposal');
