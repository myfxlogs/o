-- 0188 md_klines 补授 collector_rw 写径——工单#11481 挂起条件载「RW 双拒
-- 待补授随 fin_0171 径」之一:采集腿最小权限角色对日K表零授权(写径
-- 现走属主 o 直通,4480 行实证数据不断流——债=最小权限卫生非断流)。
-- 授权形照 fin_0071 先例(角色存在才授,幂等可重放)。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'collector_rw') THEN
    GRANT SELECT, INSERT, UPDATE ON md_klines TO collector_rw;
  END IF;
END $$;
