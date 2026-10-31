// T-0268 桩测（等价径——vitest 无基建，node 直跑源件锚+语义仿真）：首屏三件——
// ①getHeadless 并入 getCockpit 一往返撤独立首调；②流建立异帧不阻塞首绘、
// 断流退化快照态；③/assets immutable + index.html no-cache 缓存头鉴别。
import { readFileSync } from 'fs';
const pageDir = new URL('../', import.meta.url);
const repoRoot = new URL('../../../../../', import.meta.url);
const cockpit = readFileSync(new URL('Cockpit.tsx', pageDir), 'utf8');
const stream = readFileSync(new URL('../stream.ts', pageDir), 'utf8');
const proto = readFileSync(new URL('proto/dashboardv1/core.proto', repoRoot), 'utf8');
const svc = readFileSync(new URL('cmd/core/service.go', repoRoot), 'utf8');
const ui = readFileSync(new URL('cmd/core/ui.go', repoRoot), 'utf8'); // T-0269 起 mountUI 居此件——缓存头锚读此
let fails = 0;
const ck = (name, cond) => { if (!cond) { console.error('FAIL', name); fails = 1; } };

// ① RPC 合并：proto 并入字段+前端撤独立首调
ck('proto 并入字段', /GetHeadlessResponse headless = 8/.test(proto));
ck('getCockpit 携 headless 入店', /if \(d\.headless\) applyHeadless\(d\.headless\)/.test(cockpit));
ck('挂载首调撤（无裸 refill 行）', !/^\t\trefill\(\);$/m.test(cockpit));
ck('onLive 回放守卫', /replayed/.test(cockpit));
ck('缺段兜底 hydrate 径', /hydrate\("headless"/.test(cockpit));
ck('getHeadless 仅存回补/兜底两径', (cockpit.match(/client\.getHeadless/g) || []).length === 2);

// ② 流不阻塞首绘：boot 异帧发起+pump 火忘+渲染径零流等待
ck('boot 异帧发起', /setTimeout\(run, 0\)/.test(stream));
ck('pump 火忘分离', /void pump\(\)/.test(stream));
ck('驾驶舱渲染径零 watch', !/client\.watch|for await/.test(cockpit));
ck('断流重连看门狗守（T-0269 指数退避形）', /backoffMs = Math\.min\(backoffMs \* 2, BACKOFF_MAX_MS\)/.test(stream) && /setTimeout\(run, delay\)/.test(stream));

// 语义仿真：断流退化快照——store 水印判只升不降（applyHeadless 同律复刻）：
// 存量快照在流挂/败时不清；陈帧≤水位弃不覆快照；新帧 seq>水位补洞。
const sim = { snap: null, mark: 0 };
const apply = (d) => { if (sim.snap && d.lastSeq <= sim.mark) return false; sim.mark = d.lastSeq; sim.snap = d; return true; };
apply({ rows: ['T-1'], lastSeq: 5 });
const before = sim.snap;
ck('断流快照可达（流挂=无新帧，存量在）', sim.snap === before && sim.snap.rows[0] === 'T-1');
ck('陈帧不覆快照', apply({ rows: ['OLD'], lastSeq: 3 }) === false && sim.snap.rows[0] === 'T-1');
ck('新帧补洞', apply({ rows: ['T-2'], lastSeq: 6 }) === true && sim.snap.rows[0] === 'T-2');

// ③ 缓存头鉴别（Go 件源锚——vite hash 件 immutable 长缓存，壳 no-cache）
ck('assets immutable', /Cache-Control", "public, max-age=31536000, immutable"/.test(ui));
ck('index no-cache', /Cache-Control", "no-cache"/.test(ui));

if (!fails) console.log('PASS first_paint 桩测全绿');
process.exit(fails);
