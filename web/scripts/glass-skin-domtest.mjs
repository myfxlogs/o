// glass-skin-domtest.mjs —— T-0411 液态玻璃皮肤机检三面：
// ①性能红线 CSS 纪律：backdrop-filter 只许出现在静态低频面白名单
//   （主面板/弹窗/搜索浮层/菜单/缩略条），高频刷新件（card-solid/
//   quote-tile/pipe-flow 嵌套卡/持仓行）零玻璃底；
// ②WCAG AA 数学实测（验收锚②）：玻璃底上钱数/告警文字对比度——令牌
//   从 00-base.css 实读（亮/暗双态），玻璃底=稿内透明度 45-60% 区间
//   合成色；正文 ≥4.5、大字（kpi-v 2em bold）≥3.0；
// ③交互③胶囊+视觉规格源锚（blur28/sat140、圆角 24、白光弧 8-12%、
//   拉长 1.13、弹性贝塞尔、减动效豁免）。
// 运行：node scripts/glass-skin-domtest.mjs（web/ 下）。
import { readFileSync, readdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const styleDir = join(webRoot, "src/react/styles");
let fail = 0, n = 0;
const check = (name, cond, extra = "") => {
	n++;
	console.log(`${cond ? "PASS" : "FAIL"} ${name}${cond || !extra ? "" : " —— " + extra}`);
	if (!cond) fail++;
};

// —— ①backdrop-filter 白名单扫描（CSS 全量规则级）——
const cssFiles = readdirSync(styleDir).filter((f) => f.endsWith(".css"));
const GLASS_ALLOWED = [
	/^\.card,\s*\.health-strip$/,            // 主面板玻璃卡
	/^\.modal\.glass-modal$/,                 // 弹窗玻璃面
	/^\.modal-overlay\.glass-overlay$/,       // 搜索浮层壳（整页模糊唯此一处）
	/^\.side$/,                               // 桌面菜单壳
	/^\.tabbar$/,                             // 移动菜单壳
	/^\.deck-strip$/,                         // 详情缩略条（静态低频）
];
const offenders = [];
const stripCssComments = (s) => s.replace(/\/\*[\s\S]*?\*\//g, "");
for (const f of cssFiles) {
	const css = stripCssComments(readFileSync(join(styleDir, f), "utf8"));
	// 规则级拆解（含 @media 内规则——剥 at-rule 头逐段扫；backdrop-filter:
	// none=显式撤玻璃不算玻璃底）
	for (const m of css.matchAll(/(^|})([^@{}]+)\{([^{}]*)\}/g)) {
		const sel = m[2].trim().replace(/\s+/g, " ");
		const body = m[3];
		if (!/backdrop-filter:(?!\s*none)/.test(body)) continue;
		const ok = GLASS_ALLOWED.some((re) => re.test(sel));
		if (!ok) offenders.push(`${f}: ${sel}`);
	}
}
check("①backdrop-filter 只在静态低频白名单面", offenders.length === 0, offenders.join(" | "));
const cardsCss = readFileSync(join(styleDir, "20-cards.css"), "utf8");
const solidRule = readFileSync(join(styleDir, "30-glass-skin.css"), "utf8")
	.match(/\.card-solid\s*\{[^}]+\}/s)?.[0] ?? "";
check("①高频刷新件 card-solid=实色描边零玻璃",
	/backdrop-filter:\s*none/.test(solidRule) && /var\(--surface\) 88%/.test(solidRule));
const pipeRule = stripCssComments(readFileSync(join(styleDir, "12-nox-shell.css"), "utf8"))
	.match(/\.pipe-flow \.pipe-card\s*\{[^}]+\}/s)?.[0] ?? "";
check("①嵌套模块不叠玻璃（pipe-card 层级律降实色）",
	pipeRule !== "" && !/backdrop-filter/.test(pipeRule) && /var\(--surface\) 88%/.test(pipeRule));

// —— 视觉规格参数锚（v2 稿钉值）——
const cardGlass = cardsCss.match(/\.card,\s*\.health-strip\s*\{[^}]+\}/s)?.[0] ?? "";
check("②玻璃参数 blur(28px) saturate(140%)", /blur\(28px\) saturate\(140%\)/.test(cardGlass));
check("②玻璃圆角 24px+半透明白描边", /border-radius:\s*24px/.test(cardsCss)
	&& /color-mix\(in srgb, #ffffff 30%, transparent\)/.test(cardsCss));
check("②背景白光弧 8-12% 两道", (cardsCss.match(/#ffffff (8|10)%, transparent/g) || []).length >= 2);
const shellCss = readFileSync(join(styleDir, "10-shell.css"), "utf8");
check("③菜单胶囊：拉长 1.13（10-15% 区间）+弹性贝塞尔+高光描边",
	/scaleY\(1\.13\)/.test(shellCss) && /scaleX\(1\.13\)/.test(shellCss)
		&& /cubic-bezier\(0\.22, 1, 0\.36, 1\)/.test(shellCss)
		&& /nav-capsule/.test(shellCss));
check("③胶囊描边 1px 半透高光", /nav-capsule[\s\S]{0,400}1px solid color-mix\(in srgb, #ffffff 38%/.test(shellCss));
const shellSrc = readFileSync(join(webRoot, "src/react/Shell.tsx"), "utf8");
check("③胶囊=实测锚位滑动件（NavCapsule 接线双 nav）",
	/NavCapsule/.test(shellSrc) && (shellSrc.match(/<NavCapsule/g) || []).length === 2);
check("③减动效豁免（胶囊/deck/缩略条）",
	/prefers-reduced-motion/.test(shellCss) && /prefers-reduced-motion/.test(readFileSync(join(styleDir, "30-glass-skin.css"), "utf8")));
const stripRule = readFileSync(join(styleDir, "30-glass-skin.css"), "utf8").match(/\.deck-strip\s*\{[^}]+\}/s)?.[0] ?? "";
check("②缩略条溢出横向滚动（上限 8 枚稿钉）", /overflow-x:\s*auto/.test(stripRule));

// —— ②WCAG AA 数学实测（锚②：钱数/告警文字玻璃底可读）——
// 亮度（WCAG 2.x 相对亮度）+对比度。
const lum = (r, g, b) => {
	const f = (c) => {
		c /= 255;
		return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
	};
	return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
};
const ratio = (a, b) => {
	const [l1, l2] = [lum(...a), lum(...b)].sort((x, y) => y - x);
	return (l1 + 0.05) / (l2 + 0.05);
};
const hex2rgb = (h) => {
	const s = h.replace("#", "");
	return [0, 2, 4].map((i) => parseInt(s.slice(i, i + 2), 16));
};
const mix = (a, b, wa) => a.map((v, i) => Math.round(v * wa + b[i] * (1 - wa)));
// 令牌实读（00-base.css 亮/暗双态）
const base = readFileSync(join(styleDir, "00-base.css"), "utf8");
const rootBlock = base.match(/:root\s*\{([^}]+)\}/s)?.[1] ?? "";
const darkBlock = base.match(/:root\[data-theme="dark"\]\s*\{([^}]+)\}/s)?.[1] ?? "";
const tok = (block, name) => {
	const m = block.match(new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})`));
	if (m) return hex2rgb(m[1]);
	const mm = block.match(new RegExp(`--${name}:\\s*rgba?\\(([^)]+)\\)`));
	if (mm) return mm[1].split(",").slice(0, 3).map((v) => parseInt(v.trim(), 10));
	return null;
};
const themes = {
	亮: { page: tok(rootBlock, "page"), surface: tok(rootBlock, "surface"), ink: tok(rootBlock, "ink"),
		goodInk: tok(rootBlock, "good-ink"),
		delta: tok(rootBlock, "delta-down"), warn: tok(rootBlock, "warn"), critical: tok(rootBlock, "critical") },
	// 状态色固定不随主题（00-base 头注）——warn/critical 暗态同 root 块
	暗: { page: tok(darkBlock, "page"), surface: tok(darkBlock, "surface"), ink: tok(darkBlock, "ink"),
		goodInk: tok(darkBlock, "good-ink"),
		delta: tok(darkBlock, "delta-down"), warn: tok(rootBlock, "warn"), critical: tok(rootBlock, "critical") },
};
for (const [tn, t] of Object.entries(themes)) {
	if (!t.page || !t.surface || !t.ink) { check(`②${tn} 态令牌实读`, false, "00-base 令牌解析败"); continue; }
	// 玻璃底合成（稿内透明度区间：surface 46% 主玻璃面；58% KPI 面）
	const glass46 = mix(t.surface, t.page, 0.46);
	const glass58 = mix(t.surface, t.page, 0.58);
	const solid88 = mix(t.surface, t.page, 0.88);
	// 告警病因注折算（30-glass-skin：warn 55% × ink 45%——亮态；暗态原 warn）
	const compNote = tn === "亮" ? mix(t.warn, t.ink, 0.55) : t.warn;
	const pairs = [
		["钱数主字 ink@玻璃", t.ink, glass46, 4.5],
		["钱数正 u-good→good-ink@玻璃", t.goodInk, glass58, 3.0], // kpi-v 2em=大字 3.0
		["钱数负 u-bad→delta-down@玻璃", t.delta, glass58, 3.0],
		["告警文字→delta-down@玻璃", t.delta, glass46, 4.5],
		["病因注折算@玻璃", compNote, glass46, 4.5],
		["实色卡正文 ink@card-solid", t.ink, solid88, 4.5],
	];
	for (const [name, fg, bg, min] of pairs) {
		const r = ratio(fg, bg);
		check(`②WCAG ${tn}·${name} = ${r.toFixed(2)}:1 ≥ ${min}`, r >= min);
	}
}

console.log(`\n${n - fail}/${n} PASS`);
process.exit(fail ? 1 : 0);
