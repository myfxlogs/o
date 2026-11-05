-- 0117 T-0377 xvenue mt5 腿 swap 持续记账——fin_paper_fill kind 枚举扩 'swap'
-- （0060 只放行 swap_snapshot=开户快照锁存；T-0377 断链补焊③新增持续摊提径：
-- md_signal mt5_swap 新行驱动按方向腿年化 Δh 摊提——与快照同源对拍锚②）。
ALTER TABLE fin_paper_fill DROP CONSTRAINT IF EXISTS fin_paper_fill_kind_check;
ALTER TABLE fin_paper_fill ADD CONSTRAINT fin_paper_fill_kind_check
  CHECK (kind IN ('open','funding','fee','close','swap_snapshot','swap'));
