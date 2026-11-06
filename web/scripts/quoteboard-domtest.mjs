// quoteboard-domtest.mjs —— T-0382 报价卡 DOM 断言（范式同
// flowpulse-domtest.mjs：renderToStaticMarkup+esbuild 捆扎，零新依赖）。
// 断言锚：①last/bid/ask 渲染+鲜度标（服务端派生龄直译人话）；②人话
// 面（涨到/跌破/盯梢中/已提醒——裸态键 active/fired 禁入可见面）；③
// 无行情行=「暂无行情」显缺不假零；④预设表单（品种下拉+方向+目标价）
// +停用/恢复翻位钮。
// 运行：node scripts/quoteboard-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL, fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const comp = join(webRoot, "src", "react", "components", "QuoteBoard.tsx").replaceAll("\\", "/");

const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h } from "react";
import { QuoteBoardView } from "${comp}";
export { renderToStaticMarkup, h, QuoteBoardView };
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
const { renderToStaticMarkup, h, QuoteBoardView } = await import(pathToFileURL(outFile).href);

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
const render = (rows, symbols, extra = {}) =>
	renderToStaticMarkup(h(QuoteBoardView, { rows, symbols, onCreate: () => {}, onToggle: () => {}, ...extra }));

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

const rows = [
	row(1, "BTC-USDT-SWAP", ">=", 60000, "active", { last: 61000.5, bid: 60999, ask: 61001, ageS: 4 }),
	row(2, "TRX-USDT-SWAP", "<=", 0.2, "fired", { last: 0.28, bid: 0.279, ask: 0.281, ageS: 90, firedAtS: 300, note: "空单观察" }),
	row(3, "ETH-USDT-SWAP", ">=", 4000, "disabled", {}),
];
const html = render(rows, ["BTC-USDT-SWAP", "ETH-USDT-SWAP", "TRX-USDT-SWAP"]);

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
// ④表单+翻位钮
check("④品种下拉在采集", (html.match(/<option/g) || []).length >= 5);
check("④方向双选+目标价输入", html.includes('name="direction"') && html.includes('name="target"'));
check("④停用钮在 active 行", html.includes("停用"));
check("④恢复钮在 disabled 行", html.includes("恢复"));

console.log(`\n${n - fail}/${n} PASS`);
process.exit(fail ? 1 : 0);
