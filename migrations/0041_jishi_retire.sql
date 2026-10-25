-- 0041 集使销编（T-0094：「集使」=internal/facts 批拉采集调度岗，T-0041 上岗；
-- T-0048 周期扫描环全拆事件化整包删除 internal/facts——心跳写径（Register+
-- BeatLoop）随包亡，fin_employee_registry 行残留成幽灵红灯 last_beat 死寂
-- 18h+）。编外在册→status='retired' 销编（行留台账可审；组织图
-- status='active' 过滤自然下卡，不留永久红灯）。幂等可重放。
UPDATE fin_employee_registry SET status='retired' WHERE name='集使';
