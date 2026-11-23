// brokermap-domtest.mjs —— O-35250 步④兜底面 DOM 断言（范式同
// brokerboard-domtest.mjs：renderToStaticMarkup+esbuild 捆扎零新依赖；
// SymbolMappingStep 直挂+initial* 注面驱动态——异步 RPC 径不进静态渲）。
// 断言锚：①官名自由文本件（input+datalist——下拉册外手输同形闸
// [a-zA-Z0-9_-] 源码锚）；②候选空态=手输框+目录浏览钮+候选空文案；
// ③目录浏览=搜索框+原文符号清单渲出（BTCUSD#/OIL-JAN27 在列=机造清单
// 如实给席位挑）；④候选非空=radio 表+手输框并存（picked 单一挑定源）；
// ⑤提交走同一 RPC（源码 fs 断言——onSubmitMapping 单点调用 venueSymbol:
// picked，手输/下拉/浏览同径）。
// 运行：node scripts/brokermap-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, writeFileSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL, fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const comp = join(webRoot, "src", "react", "components", "brokermap.tsx").replaceAll("\\", "/");

const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h } from "react";
import { SymbolMappingStep } from "${comp}";
export { renderToStaticMarkup, h, SymbolMappingStep };
`;

const res = await build({
	stdin: { contents: entry, resolveDir: webRoot, loader: "tsx", sourcefile: "brokermap-domtest-entry.tsx" },
	bundle: true, write: false, format: "esm", platform: "node",
	jsx: "automatic", logLevel: "silent",
	banner: { js: `import { createRequire } from "node:module"; const require = createRequire(import.meta.url); globalThis.window ??= { location: { origin: "http://domtest" } };` },
});
const tmp = mkdtempSync(join(tmpdir(), "bmtdt-"));
const outFile = join(tmp, "entry.mjs");
writeFileSync(outFile, res.outputFiles[0].text);
const { renderToStaticMarkup, h, SymbolMappingStep } = await import(pathToFileURL(outFile).href);

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

const cand = (venueSymbol, score) => ({
	venueSymbol, base: "BTC", suffix: "usd", contractSize: 1, tickSz: 0.01,
	minSz: 0.01, lotStep: 0.01, digits: 2, tradeMode: 4, spread: 120, score,
});
const xmSymbols = ["BTCUSD#", "BTCUSD", "AUDUSD", "AUDUSD#", "GOLD#", "OIL-JAN27", "US500Cash#"];

// ①官名自由文本件（input+datalist——下拉册外手输同径）
const base = renderToStaticMarkup(h(SymbolMappingStep, { venue: "xm", login: "88001" }));
check("①官名输入框（placeholder 官名）", base.includes("官名（如 BTC-USDT-SWAP）"));
check("①datalist 建议件在", base.includes("<datalist"));
check("①旧 —选官名— 下拉退役（select 官名零渲染）", !base.includes("—选官名—"));

// ②候选空态（initialCanonical+空 rows——XM 死路形）
const empty = renderToStaticMarkup(h(SymbolMappingStep, {
	venue: "xm", login: "88001",
	initialCanonical: "BTC-USDT-SWAP", initialRows: [], initialScanned: 218,
	initialSymbols: xmSymbols, initialBrowse: false,
}));
check("②候选空文案（如实——非报错）", empty.includes("候选空") && empty.includes("218"));
check("②venue 手输框在（placeholder 手输）", empty.includes("venue 符号手输"));
check("②目录浏览钮在", empty.includes("目录浏览") || empty.includes("收目录"));
check("②硬验落行钮在", empty.includes("硬验并落行"));

// ③目录浏览（initialBrowse=true——原文清单渲出）
const browse = renderToStaticMarkup(h(SymbolMappingStep, {
	venue: "xm", login: "88001",
	initialCanonical: "BTC-USDT-SWAP", initialRows: [], initialScanned: 218,
	initialSymbols: xmSymbols, initialBrowse: true,
}));
check("③搜索框在（搜目录符）", browse.includes("搜目录符"));
check("③命中计数（/7 目录原文口径）", browse.includes("命中 7/7"));
check("③#尾符渲出（BTCUSD#——tradable 变体如实给席位挑）", browse.includes("BTCUSD#"));
check("③期货形渲出（OIL-JAN27——非#尾方言同列）", browse.includes("OIL-JAN27"));

// ④候选非空=radio 表+手输/浏览并存（picked 单一源——limit 截断死路同兜）
const filled = renderToStaticMarkup(h(SymbolMappingStep, {
	venue: "xm", login: "88001",
	initialCanonical: "BTC-USDT-SWAP", initialRows: [cand("BTCUSD#", 150)],
	initialScanned: 218, initialSymbols: xmSymbols,
}));
check("④候选表渲出（BTCUSD# 行+表头）", filled.includes("BTCUSD#") && filled.includes("候选方言名"));
check("④radio 挑件在", filled.includes('type="radio"'));
check("④手输框并存（候选非空亦兜——截断死路同防）", filled.includes("venue 符号手输"));

// ⑤提交走同一 RPC（源码 fs——onSubmitMapping 单点+venueSymbol:picked 同源）
const src = readFileSync(comp, "utf8");
check("⑤onSubmitMapping 单点调用", (src.match(/props\.onSubmitMapping\(/g) || []).length === 1);
check("⑤提交 venueSymbol=picked（手输/下拉/浏览同源）", src.includes("venueSymbol: picked"));
check("⑤canonGate 形闸源码锚", src.includes("[a-zA-Z0-9_-]"));
check("⑤目录浏览=symbols 原文渲（非候选行复用）", src.includes("browseHits") && src.includes("symbols.filter"));

console.log(`—— ${n - fail}/${n} PASS`);
process.exit(fail ? 1 : 0);
