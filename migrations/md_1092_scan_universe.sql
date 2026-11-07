-- 1092 T-0427 粗扫扩全宇宙——md_tick_scan 携族标 inst_type（票面钦定形：
-- 显式 > inst_id 键内隐式——族判禁散落成 inst_id 正则复制件，单位错配
-- 同族病预防）。取值=OKX instType 小写归一：swap/spot/futures
-- （OPTION/MARGIN 票面不收——无消费面）。
-- 存量行回填：粗扫层历史行清一色 SWAP 族（md_1091 起单族扫），
-- NOT NULL DEFAULT 'swap' 直填即回填，零 UPDATE。
-- fin_0133 镜像同形（八桥 DDL 同形断言⑥）。

ALTER TABLE md_tick_scan ADD COLUMN IF NOT EXISTS inst_type text NOT NULL DEFAULT 'swap';
COMMENT ON COLUMN md_tick_scan.inst_type IS
  '族标（T-0427 显式>键内隐式——OKX instType 小写：swap/spot/futures）';
