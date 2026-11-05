-- 1089 T-0356：根因实证为 okx/dcd 目录面 disabled，grammar vol-sell 两族
-- NeedFace=okx/dcd 因而静默不枚举。只由含 T-0356 担保域闸的新 research
-- binary 在 boot migration 重启时执行，恢复目录读面；SG volsell.enabled=false
-- 独立挡住提案，故启用目录不形成旧码裸奔窗。
DO $$
DECLARE n integer;
BEGIN
  UPDATE fin_capability_surface SET enabled=true
    WHERE venue='okx' AND surface='dcd' AND rw_class='read';
  GET DIAGNOSTICS n = ROW_COUNT;
  IF n <> 1 THEN
    RAISE EXCEPTION 'T-0356 expects exactly one okx/dcd read capability row; got %', n;
  END IF;
END $$;
