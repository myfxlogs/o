-- 0137（合流顺延——原0133撞T-0422已占） T-0427 粗扫扩全宇宙 fin 链镜像侧（md_1092 同形——桥表投影终表；
-- 语句文本照 md_1092 逐字）。md_tick_scan 携族标 inst_type：SG 镜像同列
-- 归位，选符域（candSets/symDomain）按 inst_id 读不加族谓词——零改动
-- 自动见新族符（票面钦定）。

ALTER TABLE md_tick_scan ADD COLUMN IF NOT EXISTS inst_type text NOT NULL DEFAULT 'swap';
COMMENT ON COLUMN md_tick_scan.inst_type IS
  '族标（T-0427 显式>键内隐式——OKX instType 小写：swap/spot/futures）';
