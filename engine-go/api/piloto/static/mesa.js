import { n as e, r as t, t as n } from "./turn-juice.js";
//#region api/piloto/src/lib/token-move.ts
function r(e) {
	if (!e) return null;
	let t = /--col:\s*(-?\d+)/.exec(e), n = /--lin:\s*(-?\d+)/.exec(e);
	return !t || !n ? null : {
		col: Number(t[1]),
		lin: Number(n[1])
	};
}
function i(t, n, r, i) {
	if (!e(t)) return;
	let a = (n.col - r.col) * i, o = (n.lin - r.lin) * i;
	a === 0 && o === 0 || t.animate([{ transform: `translate(${a}px, ${o}px)` }, { transform: "translate(0, 0)" }], {
		duration: 200,
		easing: "cubic-bezier(0.22, 1, 0.36, 1)"
	});
}
//#endregion
//#region api/piloto/src/mesa.ts
var a = "(prefers-reduced-motion: reduce)";
function o(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "style") continue;
			let t = e.target;
			if (!t.classList?.contains("tabuleiro-peca") || t.classList.contains("tabuleiro-peca-fantasma")) continue;
			let n = r(e.oldValue), a = r(t.getAttribute("style"));
			if (!n || !a) continue;
			let o = Number.parseFloat(getComputedStyle(t).getPropertyValue("--quadrado"));
			!Number.isFinite(o) || o <= 0 || i(t, n, a, o);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["style"]
	});
}
function s(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-valuenow") continue;
			let t = e.target, r = Number(e.oldValue), i = Number(t.getAttribute("aria-valuenow"));
			if (!Number.isFinite(r) || !Number.isFinite(i) || r === i) continue;
			let a = t.closest("li");
			a && requestAnimationFrame(() => n(a, { curou: i > r }));
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["aria-valuenow"]
	});
}
function c(e) {
	new MutationObserver((n) => {
		if (!e.matches) for (let e of n) {
			if (e.attributeName !== "aria-current") continue;
			let n = e.target;
			n.getAttribute("aria-current") === "true" && t(n);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeFilter: ["aria-current"]
	});
}
var l = window.matchMedia(a);
o(l), s(l), c(l);
//#endregion
