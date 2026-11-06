// moduledeck-domtest.mjs —— T-0411 模块 deck DOM 断言（范式同
// quoteboard-domtest.mjs：renderToStaticMarkup+esbuild 捆扎，零新依赖）。
// 断言锚：①静止形态=等大格网格全模块在格（首屏密度锚——可见数据件数
// 不减：8 模块 8 格全渲）；②展开态=deck 宿主+底部玻璃缩略条（缩略图=
// 模块名+状态点，不渲实时数；上限 8 枚）+网格让位（is-dimmed 格位保挂）
// +关闭件；③分阈/拖拽/吸附/持久化源锚（6px 分阈、200ms ease 让位、
// -3° 硬阴影、localStorage 全排列验败弃用）。
// 运行：node scripts/moduledeck-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL, fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const deckComp = join(webRoot, "src", "react", "components", "moduledeck.tsx").replaceAll("\\", "/");
const uiComp = join(webRoot, "src", "react", "components", "ui.tsx").replaceAll("\\", "/");

const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h } from "react";
import { ModuleDeck } from "${deckComp}";
import { Card, StateDot } from "${uiComp}";
export { renderToStaticMarkup, h, ModuleDeck, Card, StateDot };
`;

const res = await build({
	stdin: { contents: entry, resolveDir: webRoot, loader: "ts", sourcefile: "moduledeck-domtest-entry.ts" },
	bundle: true, write: false, format: "esm", platform: "node",
	jsx: "automatic", logLevel: "silent",
	banner: { js: `import { createRequire } from "node:module"; const require = createRequire(import.meta.url); globalThis.window ??= { location: { origin: "http://domtest" }, matchMedia: () => ({ matches: false }) };` },
});
const tmp = mkdtempSync(join(tmpdir(), "mddt-"));
const outFile = join(tmp, "entry.mjs");
writeFileSync(outFile, res.outputFiles[0].text);
const { renderToStaticMarkup, h, ModuleDeck, Card, StateDot } = await import(pathToFileURL(outFile).href);

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

// 8 模块桩（座舱同构：4 KPI+管道+行情+待开仓+平台明细——缩略条上限 8 正好）
const mod = (id, title, dotCls) => ({
	id, title,
	el: h(Card, { title },
		h("div", { className: "card-h" }, h("h2", null, title), h(StateDot, { state: 0, breathe: true })),
		h("p", { className: "kpi-v" }, "42"),
	),
});
const modules = [
	mod("safe", "① 钱安全吗"), mod("mtd", "② 赚了多少（MTD）"),
	mod("d30", "② 近 30 天"), mod("alive", "① 存活"),
	mod("pipeline", "③ 工作流管道"), mod("quotes", "行情·价格提醒"),
	mod("armed", "④ 待开仓"), mod("money", "①② 钱面平台明细"),
];
const render = (mods, extra = {}) =>
	renderToStaticMarkup(h(ModuleDeck, { storageKey: "o_cockpit_modules", modules: mods, ...extra }));

// ①静止形态：8 格 8 卡全渲（密度锚——可见数据件数不减）
const html = render(modules);
check("①等大格网格+8 格全渲", (html.match(/module-cell/g) || []).length === 8
	&& html.includes("module-grid") && !html.includes("is-dimmed"));
check("①模块全在格（data-module-id 锚）", ["safe", "mtd", "d30", "alive", "pipeline", "quotes", "armed", "money"]
	.every((id) => html.includes(`data-module-id="${id}"`)));
check("①卡面原样入格", (html.match(/class="card/g) || []).length >= 8);

// ②展开态：宿主+缩略条+让位+关闭件；缩略图=名+状态点不渲实时数
const ehtml = render(modules, { initialExpanded: "pipeline" });
check("②deck 宿主+详情落点", ehtml.includes("deck-host") && ehtml.includes("deck-detail"));
check("②网格让位（is-dimmed 格位保挂）", ehtml.includes("is-dimmed")
	&& (ehtml.match(/module-cell/g) || []).length === 8);
check("②缩略条=玻璃 toolbar", ehtml.includes("deck-strip") && ehtml.includes('aria-label="模块缩略条"'));
check("②缩略图 8 枚=名+状态点", (ehtml.match(/deck-thumb/g) || []).length >= 8
	&& ehtml.includes("deck-thumb-name") && /deck-thumb[\s\S]*?class="dot s\d/.test(ehtml));
check("②缩略图不渲实时数（kpi-v 数不入条）",
	!/(deck-strip[\s\S]*)42/.test(ehtml));
check("②当前项高亮+切换件", /deck-thumb on/.test(ehtml));
check("②关闭件在案（✕+收起）", ehtml.includes("deck-close-x") && ehtml.includes("收起"));

// ③源锚：分阈/动效/吸附/持久化（稿内钉值随码走；T-0411 棘轮拆分——
// 拖拽控制面在 usedeckdrag.ts，deck 壳+FLIP 回放在 moduledeck.tsx）
const src = readFileSync(join(webRoot, "src/react/components/moduledeck.tsx"), "utf8");
const dragSrc = readFileSync(join(webRoot, "src/react/components/usedeckdrag.ts"), "utf8");
check("③点按/拖拽分阈=6px", /DRAG_THRESHOLD_PX = 6/.test(dragSrc));
check("③让位 200ms ease", /YIELD_MS = 200/.test(dragSrc) && /transform \$\{YIELD_MS\}ms ease/.test(src));
check("③FLIP 原位放大+沿原路收回（双向 first/last 逆向回放）",
	/pendingFirst/.test(src) && /closing/.test(src) && /transformOrigin/.test(src));
check("③序持久化=localStorage 全排列验败弃用",
	/localStorage\.setItem/.test(src) && /new Set\(kept\)\.size !== ids\.length/.test(src));
const css = readFileSync(join(webRoot, "src/react/styles/30-glass-skin.css"), "utf8");
check("③拖起旋转 -3°+硬阴影 14px（CSS 钉）",
	/rotate:\s*-3deg/.test(css) && /0 14px 28px/.test(css));
check("③触屏纵向滚让位（touch-action: pan-y）", /touch-action:\s*pan-y/.test(css));
check("③状态点=卡头点 DOM 镜像（MutationObserver 只读零判据）",
	/MutationObserver/.test(src) && /\.card-h \.dot/.test(src));

console.log(`\n${n - fail}/${n} PASS`);
process.exit(fail ? 1 : 0);
