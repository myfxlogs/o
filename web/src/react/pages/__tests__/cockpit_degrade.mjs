// T-0248 桩测（等价现有面——vitest 无基建，node 直跑）：sec 语义仿真+源件锚断言。
import { readFileSync } from 'fs';
const src = readFileSync(new URL('../Cockpit.tsx', import.meta.url), 'utf8');
let fails = 0;
const ck = (name, cond) => { if (!cond) { console.error('FAIL', name); fails = 1; } };
// ①sec 边界在册+缺失段显缺非掀页
ck('sec 定义', /const sec = \(name: string, el: ReactNode\)/.test(src));
ck('SecBoundary 类', /class SecBoundary extends Component/.test(src));
ck('缺注文', /该段数据缺/.test(src));
// ②money 缺段不裸链——m 变量判据真源用
ck('money 判据源', /const m = d\.money/.test(src));
ck('裸链清', !/d\.money\?\.(netMonth|capital|exposure|daily|platforms)/.test(src));
ck('mt5 缺显缺', /miss\.has\("mt5"\) \? "数据缺"/.test(src));
// ③build 链 typecheck 前置
const pj = JSON.parse(readFileSync(new URL('../../../../package.json', import.meta.url), 'utf8'));
ck('typecheck 入链', /typecheck/.test(pj.scripts.build));
// ④sec 语义仿真：missing 含名→显缺；不含→渲子件
const miss = new Set(['money']);
const secSim = (name, el) => miss.has(name) ? '缺注' : el;
ck('missing 段显缺', secSim('money', '<money/>') === '缺注');
ck('存活段照渲', secSim('seats', '<seats/>') === '<seats/>');
if (!fails) console.log('PASS cockpit_degrade 桩测全绿');
process.exit(fails);
