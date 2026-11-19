-- 1098 perp-trend-daily 模板回 park（工单#20664 取舍(a)）：族 enabled 但
-- 文法无 eval 分支→能力债探针恒复火；假想卡#172286 已 parked——模板行
-- 回 candidate=禁用非退役（eval 臂另棒落地可再 enabled 复役，语义与卡
-- parked 同格）。探针读面 enabled AND state='enabled'（meta_esc.go:122）
-- ——本 UPDATE 落即探针灭源，票载「禁用随票自闭信号」。
-- 行依赖 UPDATE：行在案（md_1096 登记），无行=空集幂等不炸（fin_0207
-- 生造词坑鉴——本件词面皆 CHECK 闭集内）。

UPDATE fin_opp_template SET enabled=false, state='candidate'
  WHERE template='perp-trend-daily';
