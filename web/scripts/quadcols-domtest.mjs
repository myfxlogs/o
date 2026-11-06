// quadcols-domtest.mjs —— T-0389 列联动解耦 DOM 几何桩测（范式同
// flowpulse-domtest.mjs：renderToStaticMarkup+esbuild 捆扎零新依赖+CSS
// 静态断言）。断言锚：
//   ①结构=独立列容器（每列一 cols-col 壳，列外无共享行轨）；
//   ②列内 gap=恒定值（CSS .cols-col gap 实测值驱动几何，无 space-between 拉距）；
//   ③展开不扰他列几何（右列瓦片加高→左列各卡 y 坐标零变）；
//   ④quad 桌面双列面（≥1100px grid-template-columns:1fr 1fr+align-items:start）；
//   ⑤窄屏单列回落（.cols 基面 flex-column）；
//   ⑥接线索引（System 页实走 Cols 面——防死件漂）。
// 运行：node scripts/quadcols-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL, fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const ui = join(webRoot, "src", "react", "components", "ui.tsx").replaceAll("\\", "/");

const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h, Fragment } from "react";
import { Cols } from "${ui}";
export { renderToStaticMarkup, h, Fragment, Cols };
`;

const res = await build({
	stdin: { contents: entry, resolveDir: webRoot, loader: "ts", sourcefile: "quadcols-domtest-entry.ts" },
	bundle: true, write: false, format: "esm", platform: "node",
	jsx: "automatic", logLevel: "silent",
	banner: { js: `import { createRequire } from "node:module"; const require = createRequire(import.meta.url);` },
});
const tmp = mkdtempSync(join(tmpdir(), "qcdt-"));
const outFile = join(tmp, "entry.mjs");
writeFileSync(outFile, res.outputFiles[0].text);
const { renderToStaticMarkup, h, Fragment, Cols } = await import(pathToFileURL(outFile).href);

// 瓦片桩：data-h=固有高（展开=右列首卡增高，模拟 exch-head 展开撑高本卡）。
const tile = (k, hh) => h("div", { key: k, className: "tile", "data-h": hh });
const page = (devH) =>
	renderToStaticMarkup(
		h(Cols, { className: "quad" }, [
			h(Fragment, { key: "l" }, tile("a", 120), tile("b", 90), tile("c", 60)),
			h(Fragment, { key: "r" }, tile("d", devH), tile("e", 50)),
		])
	);

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

// —— ①结构断言：每列一 cols-col 壳 ——
const html = page(80);
check("①容器类=cols quad", html.includes('class="cols quad"'));
check("①列壳数=2", (html.match(/class="cols-col"/g) || []).length === 2);
const colSegs = html.split('class="cols-col">').slice(1);
const colHs = colSegs.map((seg) => [...seg.matchAll(/data-h="(\d+)"/g)].map((m) => Number(m[1])));
check("①列内瓦片分组=[3,2]（左列生产/主机/管道，右列采集/席位）",
	JSON.stringify(colHs.map((c) => c.length)) === "[3,2]"
	&& JSON.stringify(colHs[0]) === "[120,90,60]");

// —— CSS 契约提取（20-cards.css 单源读）——
const css = readFileSync(join(webRoot, "src/react/styles/20-cards.css"), "utf8");
const rule = (sel) => css.match(new RegExp(sel.replace(/[.*]/g, "\\$&") + "\\s*\\{[^}]+\\}"))?.[0] ?? "";
const colsRule = rule(".cols");
const colRule = rule(".cols-col");
const quadMedia = css.match(/@media \(min-width: 1100px\)\s*\{\s*\.quad\s*\{[^}]+\}/s)?.[0] ?? "";
const gapM = colRule.match(/gap:\s*(\d+)px/);
check("②.cols-col=flex column+min-width:0（独立格式化上下文）",
	/display:\s*flex/.test(colRule) && /flex-direction:\s*column/.test(colRule) && /min-width:\s*0/.test(colRule));
check("②列内 gap=24px 恒值（无 space-between/around/evenly 拉距件）",
	gapM?.[1] === "24" && !/justify-content:\s*space-/.test(colRule) && !/justify-content/.test(colRule));
check("④quad ≥1100px 双列格 1fr 1fr+顶对齐",
	/grid-template-columns:\s*1fr\s+1fr/.test(quadMedia) && /align-items:\s*start/.test(quadMedia));
check("⑤.cols 基面=flex column 单列回落",
	/display:\s*flex/.test(colsRule) && /flex-direction:\s*column/.test(colsRule));

// —— ③几何桩：列容器独立语义下自顶向下堆叠，y_i=Σ(h_j+gap) ——
const gap = Number(gapM?.[1] ?? 0);
const layout = (markup) =>
	markup.split('class="cols-col">').slice(1).map((seg) => {
		let y = 0;
		return [...seg.matchAll(/data-h="(\d+)"/g)].map((m) => {
			const yy = y; y += Number(m[1]) + gap; return yy;
		});
	});
const yCold = layout(page(80));   // 收拢：右列首卡 80
const yOpen = layout(page(400));  // 展开：右列首卡 400（+320）
check("③展开→左列各卡 y 零变（他列几何不入本列）",
	JSON.stringify(yCold[0]) === JSON.stringify(yOpen[0]));
check("③展开→本列后续卡 y 增量=展开量（只增本卡高只推本列）",
	yOpen[1][1] - yCold[1][1] === 320);
check("②左列卡间间距恒=gap（y 差=h+24，无拉距）",
	yCold[0].every((y, i) => i === 0 || y - yCold[0][i - 1] === colHs[0][i - 1] + gap));

// —— ⑥接线索引：System 页实走 Cols 面 ——
const sys = readFileSync(join(webRoot, "src/react/pages/System.tsx"), "utf8");
check("⑥System 页接线 <Cols className=\"quad\">（非死件）",
	/<Cols className="quad">/.test(sys) && !/<div className="quad"/.test(sys));

console.log(`\n${n - fail}/${n} PASS${fail ? ` —— ${fail} FAIL` : ""}`);
process.exit(fail ? 1 : 0);
