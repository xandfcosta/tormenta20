import { n as e } from "./turn-juice.js";
//#region api/piloto/src/lib/token-move.ts
function t(e) {
	if (!e) return null;
	let t = /--col:\s*(-?\d+)/.exec(e), n = /--lin:\s*(-?\d+)/.exec(e);
	return !t || !n ? null : {
		col: Number(t[1]),
		lin: Number(n[1])
	};
}
function n(t, n, r, i) {
	if (!e(t)) return;
	let a = (n.col - r.col) * i, o = (n.lin - r.lin) * i;
	a === 0 && o === 0 || t.animate([{ transform: `translate(${a}px, ${o}px)` }, { transform: "translate(0, 0)" }], {
		duration: 200,
		easing: "cubic-bezier(0.22, 1, 0.36, 1)"
	});
}
//#endregion
//#region api/piloto/src/mesa.ts
var r = "(prefers-reduced-motion: reduce)";
function i() {
	let e = window.matchMedia(r);
	new MutationObserver((r) => {
		if (!e.matches) for (let e of r) {
			if (e.attributeName !== "style") continue;
			let r = e.target;
			if (!r.classList?.contains("tabuleiro-peca") || r.classList.contains("tabuleiro-peca-fantasma")) continue;
			let i = t(e.oldValue), a = t(r.getAttribute("style"));
			if (!i || !a) continue;
			let o = Number.parseFloat(getComputedStyle(r).getPropertyValue("--quadrado"));
			!Number.isFinite(o) || o <= 0 || n(r, i, a, o);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["style"]
	});
}
i();
//#endregion
