// quoteboard-domtest.mjs —— T-0382 报价卡 DOM 断言（范式同
// flowpulse-domtest.mjs：renderToStaticMarkup+esbuild 捆扎，零新依赖）
// + T-0391 自选报价条断言（NOX QuoteStrip/WatchlistModal 语义移植）
// + T-0442 行情卡三件断言（①删除软删折叠/②双阈同条分组/③venue chip 源选）。
// 断言锚：①last/bid/ask 渲染+鲜度标（服务端派生龄直译人话）；②人话
// 面（涨到/跌破/盯梢中/已提醒——裸态键 active/fired 禁入可见面）；③
// 无行情行=「暂无行情」显缺不假零；④预设表单 symbol=可过滤输入
// （input+datalist——业主令「缺少搜索功能」）+上限/下限双阈输入（T-0442
// 件②）+删除钮（件①——fired 史行无删）；⑤自选瓦片：成员价渲+零成员缺态
// +成员零行情显缺；⑥浮层：搜索「trx」子串过滤候选（大小写不敏）/已选剔出
// 候选/↑↓✕件/保存全量替换文案/venue chip 亮暗+全部源（件③）；⑦提醒表单
// 候选=watchlist 成员优先（成员空退回全候选）；⑧T-0406：自选瓦片去 last
// 只显 bid/ask+bid绿ask红（u-good/u-bad 两处）；浮层作卡外兄弟渲
// （backdrop-filter 包含块阱修复——保存键不被平铺卡盖）+modal 90vh 封顶列
// 内滚；UtcClock 共享件 UTC 时标+待开仓卡接线断言；⑨T-0412 未订阅候选
// 「晋升中」标；⑩T-0442：删除钮只在 active 行/已停用折叠区恢复钮/双阈
// rowSpan 分组一卡/双空禁建源闸/瓦片源约束标+@来源/浮层 chip 集候选源文本。
// 运行：node scripts/quoteboard-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL, fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const comp = join(webRoot, "src", "react", "components", "QuoteBoard.tsx").replaceAll("\\", "/");
const wcomp = join(webRoot, "src", "react", "components", "QuoteWatch.tsx").replaceAll("\\", "/");
const uicomp = join(webRoot, "src", "react", "components", "ui.tsx").replaceAll("\\", "/");

const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h } from "react";
import { QuoteBoardView } from "${comp}";
import { WatchEditView } from "${wcomp}";
import { UtcClock } from "${uicomp}";
export { renderToStaticMarkup, h, QuoteBoardView, WatchEditView, UtcClock };
`;

const res = await build({
	stdin: { contents: entry, resolveDir: webRoot, loader: "ts", sourcefile: "quoteboard-domtest-entry.ts" },
	bundle: true, write: false, format: "esm", platform: "node",
	jsx: "automatic", logLevel: "silent",
	banner: { js: `import { createRequire } from "node:module"; const require = createRequire(import.meta.url); globalThis.window ??= { location: { origin: "http://domtest" } };` },
});
const tmp = mkdtempSync(join(tmpdir(), "qbdt-"));
const outFile = join(tmp, "entry.mjs");
writeFileSync(outFile, res.outputFiles[0].text);
const { renderToStaticMarkup, h, QuoteBoardView, WatchEditView, UtcClock } = await import(pathToFileURL(outFile).href);

const nowS = Math.floor(Date.now() / 1000);
const row = (id, symbol, dir, target, state, { last, bid, ask, ageS, firedAtS, note = "" } = {}) => ({
	id: BigInt(id), symbol, direction: dir, target, state, note,
	createdBy: "桩", createdAt: { seconds: BigInt(nowS - 600) },
	firedAt: firedAtS ? { seconds: BigInt(nowS - firedAtS) } : undefined,
	hasQuote: last !== undefined,
	last: last ?? 0, bid: bid ?? 0, ask: ask ?? 0,
	quoteTs: last !== undefined ? { seconds: BigInt(nowS - (ageS ?? 0)) } : undefined,
	quoteAgeS: BigInt(ageS ?? 0),
});
const wrow = (symbol, q, { venues = [], quoteVenue = "", pending = false } = {}) => ({
	symbol,
	venues, quoteVenue, pending, // T-0442 件③源约束/来源实证/源感知晋升标
	hasQuote: q !== undefined,
	last: q?.last ?? 0, bid: q?.bid ?? 0, ask: q?.ask ?? 0,
	quoteTs: q ? { seconds: BigInt(nowS - (q.ageS ?? 0)) } : undefined,
	quoteAgeS: BigInt(q?.ageS ?? 0),
});
const render = (rows, symbols, extra = {}) =>
	renderToStaticMarkup(h(QuoteBoardView, {
		rows, symbols, watchRows: [], watchCands: symbols,
		onCreate: () => {}, onToggle: () => {}, onSaveWatch: async () => {},
		...extra,
	}));

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

const rows = [
	row(1, "BTC-USDT-SWAP", ">=", 60000, "active", { last: 61000.5, bid: 60999, ask: 61001, ageS: 4 }),
	row(2, "TRX-USDT-SWAP", "<=", 0.2, "fired", { last: 0.28, bid: 0.279, ask: 0.281, ageS: 90, firedAtS: 300, note: "空单观察" }),
	row(3, "ETH-USDT-SWAP", ">=", 4000, "disabled", {}),
	row(4, "XRP-USDT-SWAP", ">=", 1, "active", {}), // 零行情 active 行——显缺/晋升标锚
];
const allSyms = ["BTC-USDT-SWAP", "ETH-USDT-SWAP", "TRX-USDT-SWAP"];
const html = render(rows, allSyms);

// ①报价渲+鲜度标
check("①last/bid/ask 渲出", html.includes("61,000.5") && html.includes("60,999") && html.includes("61,001"));
check("①鲜度<10s=刚刚", html.includes("刚刚"));
check("①鲜度90s=分前档", html.includes("1 分前"));
// ②人话面
check("②方向人话", html.includes("涨到 ≥") && html.includes("跌破 ≤"));
check("②状态人话", html.includes("盯梢中") && html.includes("已提醒") && html.includes("已停用"));
check("②裸态键禁入可见面", !html.includes(">active<") && !html.includes(">fired<") && !html.includes(">disabled<"));
check("②备注入悬停", html.includes("空单观察"));
// ③无行情行显缺不假零
check("③无行情显缺", html.includes("暂无行情") && !html.includes(">0<"));
// ④表单：symbol=可过滤输入（input+datalist——非裸 select）+T-0442 件②双阈输入
check("④symbol=可过滤输入 datalist", html.includes('name="symbol"') && html.includes('list="alert-symbols"')
	&& html.includes("<datalist") && !html.includes('<select name="symbol"'));
check("④零成员时候选=全在采", (html.match(/<datalist[\s\S]*?<\/datalist>/)?.[0].match(/<option/g) || []).length === 3);
check("④双阈输入 上限/下限", html.includes('name="upper"') && html.includes('name="lower"')
	&& !html.includes('name="direction"') && !html.includes('name="target"'));
// ⑩T-0442 件①：删除钮（active 行）——fired 史行无删（禁删禁改审计面）
check("⑩删除钮在 active 行", html.includes("删除"));
const firedTr = html.split("<tbody>")[1]?.split("</tbody>")[0]
	.split("<tr>").find((seg) => seg.includes("已提醒")) ?? "";
check("⑩fired 行无删钮", firedTr !== "" && !firedTr.includes(">删除<"));
check("⑩已停用折叠区+恢复钮", html.includes("已停用 1 条（软删留痕——点开可恢复）")
	&& html.includes("<details") && html.includes("恢复"));

// ⑤自选瓦片面（成员价渲/零行情显缺/非成员不建瓦片——零成员态上行已断缺态文案）
check("⑤零成员缺态文案", html.includes("未设自选品种") && html.includes("自选✎"));
const watchRows = [
	wrow("TRX-USDT-SWAP", { last: 0.1505, bid: 0.15, ask: 0.151, ageS: 3 }),
	wrow("BTC-USDT-SWAP", undefined), // 成员零行情——显缺不假零
];
const whtml = render(rows, allSyms, { watchRows });
// T-0406④：瓦片只显 bid/ask（业主令去 last——「那个价格不知道取自哪」），
// 鲜度行保留；⑧ bid=绿 u-good/ask=红 u-bad（CEX 定调）。
check("⑤瓦片无 last 仅 bid/ask", !whtml.includes("0.1505") && !whtml.includes('title="最新价"')
	&& whtml.includes('class="num u-good" title="买一">0.15') && whtml.includes('class="num u-bad" title="卖一">0.151'));
check("⑤瓦片鲜度行保留", whtml.includes("刚刚"));
check("⑧提醒表买一绿卖一红", /<td[^>]*class="u-good"[^>]*>60,999/.test(whtml)
	&& /<td[^>]*class="u-bad"[^>]*>61,001/.test(whtml));
check("⑤成员零行情显缺", whtml.includes('class="quote-tile"') && whtml.includes("暂无行情"));
// 非成员不建瓦片：瓦片区只含两成员符号各一次（表体行另含 ETH——计数 quote-tile 数=成员数）
check("⑤非成员不建瓦片", (whtml.match(/quote-tile/g) || []).length === 2);
// ⑦提醒表单候选=watchlist 成员优先（成员在→候选=成员集，非全在采）
const dl = whtml.match(/<datalist[\s\S]*?<\/datalist>/)?.[0] ?? "";
check("⑦表单候选=成员集", (dl.match(/<option/g) || []).length === 2 && dl.includes("TRX-USDT-SWAP") && !dl.includes("ETH-USDT-SWAP"));

// ⑥浮层面（WatchEditView 受控渲——query='trx' 子串过滤大小写不敏）
const ehtml = renderToStaticMarkup(h(WatchEditView, {
	items: [{ symbol: "BTC-USDT-SWAP", venues: [] }], candidates: allSyms, query: "trx",
	onQuery: () => {}, onAdd: () => {}, onMove: () => {}, onRemove: () => {},
	onSave: () => {}, onCancel: () => {},
}));
check("⑥搜索 trx→候选含 TRX", ehtml.includes("TRX-USDT-SWAP"));
check("⑥搜索 trx→BTC 已选剔候选+ETH 不匹配不入", !ehtml.includes("ETH-USDT-SWAP"));
check("⑥已选行+排序删件", ehtml.includes("已选（1）") && ehtml.includes("↑") && ehtml.includes("↓") && ehtml.includes("✕"));
check("⑥保存=全量替换文案", ehtml.includes("保存（全量替换生效）"));
const ehtml2 = renderToStaticMarkup(h(WatchEditView, {
	items: [], candidates: allSyms, query: "",
	onSave: () => {}, onCancel: () => {},
}));
check("⑥零已选缺态+全候选", ehtml2.includes("未选品种") && ehtml2.includes("ETH-USDT-SWAP"));

// ⑩T-0442 件③浮层 chip 集（Nox 形制引进——srcSets 派生、chip 亮=含暗=不含、
// 空集=全部源、候选行携源集文本预览）：
const chtml = renderToStaticMarkup(h(WatchEditView, {
	items: [
		{ symbol: "TRX-USDT-SWAP", venues: ["okx"] }, // 约束单源——mt5 暗 chip
		{ symbol: "BTC-USDT-SWAP", venues: [] }, // 空集=全部源
	],
	candidates: ["SOL-USDT"],
	srcSets: {
		"TRX-USDT-SWAP": ["mt5", "okx"],
		"BTC-USDT-SWAP": ["mt5", "okx"],
		"SOL-USDT": ["okx"],
	},
	query: "", onSave: () => {}, onCancel: () => {}, onToggleVenue: () => {},
}));
check("⑩chip 集渲+亮暗态", (chtml.match(/class="chip chip-active"/g) || []).length === 1
	&& (chtml.match(/class="chip"/g) || []).length === 3);
check("⑩空集=全部源标+约束源摘要", chtml.includes("全部源") && chtml.includes("okx"));
check("⑩候选行携源集文本", chtml.includes("源 okx"));
check("⑩零源集缺态", renderToStaticMarkup(h(WatchEditView, {
	items: [{ symbol: "NEW-USDT", venues: [] }], candidates: [], srcSets: {}, query: "",
	onSave: () => {}, onCancel: () => {},
})).includes("源集待实采"));

// ⑧T-0406①：浮层作卡外兄弟节点渲——.card backdrop-filter 成 fixed 包含块+
// 叠层阱病灶修复（overlay 出 </section> 后才出现=不被卡界截尾/平铺卡盖）。
const ohtml = render(rows, allSyms, { watchRows, editOpen: true });
check("⑧浮层出卡渲(overlay 在 card 外)", ohtml.includes("modal-overlay")
	&& ohtml.indexOf("modal-overlay") > ohtml.indexOf("</section>"));
check("⑧保存键在浮层可达面", ohtml.includes("保存（全量替换生效）") && ohtml.includes("modal-actions"));
// 弹窗任意列高下保存键可达：CSS 钉 modal-overlay fixed 置顶 + modal 90vh
// 封顶+列表区 min-height:0 可缩（列内滚）——静态断言同 flowpulse 范式。
const btnCss = readFileSync(join(webRoot, "src/react/styles/06-buttons.css"), "utf8");
const overlayRule = btnCss.match(/\.modal-overlay\s*\{[^}]+\}/s)?.[0] ?? "";
const modalRule = btnCss.match(/^\.modal\s*\{[^}]+\}/ms)?.[0] ?? "";
check("⑧overlay fixed 视口置顶", /position:\s*fixed/.test(overlayRule)
	&& /inset:\s*0/.test(overlayRule) && /z-index:\s*2\d\d/.test(overlayRule));
check("⑧modal 90vh 封顶+列内滚可缩", /max-height:\s*90vh/.test(modalRule)
	&& /min-height:\s*0/.test(btnCss));

// ⑨T-0412 未订阅候选「晋升中」标（pendingSyms→行/瓦片/表单/浮层四面——
// 用户该知道此符报价要稍后接通，禁伪装有行情）。
const phtml = render(rows, allSyms, {
	watchRows: [wrow("DOGE-USDT-SWAP", undefined)],
	pendingSyms: ["XRP-USDT-SWAP", "DOGE-USDT-SWAP"],
});
check("⑨提醒行未订阅=晋升中显", phtml.includes("晋升中——订阅已提名，报价接通后可评估"));
check("⑨瓦片未订阅=晋升中显", phtml.includes("晋升中…") && !phtml.includes(">0<"));
check("⑨表单候选 label 携晋升标", phtml.includes("晋升中（选建即提名订阅）"));
const pehtml = renderToStaticMarkup(h(WatchEditView, {
	items: [], candidates: allSyms, pending: new Set(["ETH-USDT-SWAP"]), query: "",
	onSave: () => {}, onCancel: () => {},
}));
check("⑨浮层候选·晋升中标（仅未订阅员）", pehtml.includes("ETH-USDT-SWAP") && pehtml.includes("·晋升中")
	&& !/BTC-USDT-SWAP[^<]*·晋升中/.test(pehtml));

// ⑩T-0442 件②一品种双阈同条：同品种两条件 rowSpan 合并一卡（品种 cell=1、
// 条件行=2）——条目减半=分组呈现；异品种各占卡。
const dualRows = [
	row(11, "BTC-USDT-SWAP", ">=", 70000, "active", { last: 61000.5, bid: 60999, ask: 61001, ageS: 4 }),
	row(12, "BTC-USDT-SWAP", "<=", 60000, "active", { last: 61000.5, bid: 60999, ask: 61001, ageS: 4 }),
	row(13, "TRX-USDT-SWAP", ">=", 0.2, "active", { last: 0.28, bid: 0.279, ask: 0.281, ageS: 90 }),
];
const dhtml = render(dualRows, allSyms);
check("⑩双阈 rowSpan=2 合并一卡", dhtml.includes('rowSpan="2"'));
check("⑩双阈条件两行齐渲", dhtml.includes("涨到 ≥ 70,000") && dhtml.includes("跌破 ≤ 60,000"));
check("⑩双阈品种 cell 惟一（条目减半）", (dhtml.match(/<td[^>]*>BTC-USDT-SWAP<\/td>/g) || []).length === 1);
check("⑩异品种各占卡", (dhtml.match(/<td[^>]*>TRX-USDT-SWAP<\/td>/g) || []).length === 1);
// 件②双空禁建前端闸=submit 内守卫（运行时逻辑——源面静态断言在案）
const qbSrc = readFileSync(comp, "utf8");
check("⑩双空禁建守卫在案", qbSrc.includes("双空禁建"));

// ⑩T-0442 件③瓦片源标：约束源+报价来源 @venue（选源后只显所选源价实证面）
const vhtml = render(rows, allSyms, {
	watchRows: [wrow("TRX-USDT-SWAP", { last: 0.1505, bid: 0.15, ask: 0.151, ageS: 3 },
		{ venues: ["okx"], quoteVenue: "okx" })],
});
check("⑩瓦片源约束标+@来源", vhtml.includes('title="源约束——只取所选源报价">okx')
	&& vhtml.includes('title="报价来源 venue（空约束=最新跨源）">@okx'));
// 源感知晋升标：pending=true 且零行情→晋升中（服务端源感知派生直译）
const vphtml = render(rows, allSyms, {
	watchRows: [wrow("SOL-USDT", undefined, { venues: ["okx"], pending: true })],
});
check("⑩源感知 pending=晋升中", vphtml.includes("晋升中…"));

// ⑧T-0406②：UtcClock 共享件渲 UTC 时标+Cockpit 待开仓卡接线在案。
const ck = renderToStaticMarkup(h(UtcClock));
check("⑧UtcClock 渲 HH:MM:SS UTC", /\d{2}:\d{2}:\d{2} UTC/.test(ck));
const cockpitSrc = readFileSync(join(webRoot, "src/react/pages/Cockpit.tsx"), "utf8");
check("⑧待开仓卡右上挂 UtcClock", /待开仓"[\s\S]{0,400}<UtcClock/.test(cockpitSrc));

console.log(`\n${n - fail}/${n} PASS`);
process.exit(fail ? 1 : 0);
