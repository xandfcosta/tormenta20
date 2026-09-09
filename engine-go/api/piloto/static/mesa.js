import { i as e, n as t, r as n, t as r } from "./turn-juice.js";
//#region api/piloto/src/lib/token-move.ts
function i(e) {
	if (!e) return null;
	let t = /--col:\s*(-?\d+)/.exec(e), n = /--lin:\s*(-?\d+)/.exec(e);
	return !t || !n ? null : {
		col: Number(t[1]),
		lin: Number(n[1])
	};
}
function a(e, n, r, i) {
	if (!t(e)) return;
	let a = (n.col - r.col) * i, o = (n.lin - r.lin) * i;
	a === 0 && o === 0 || e.animate([{ transform: `translate(${a}px, ${o}px)` }, { transform: "translate(0, 0)" }], {
		duration: 200,
		easing: "cubic-bezier(0.22, 1, 0.36, 1)"
	});
}
//#endregion
//#region api/piloto/src/mesa.ts
var o = "(prefers-reduced-motion: reduce)";
function s(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "style") continue;
			let t = e.target;
			if (!t.classList?.contains("tabuleiro-peca") || t.classList.contains("tabuleiro-peca-fantasma")) continue;
			let n = i(e.oldValue), r = i(t.getAttribute("style"));
			if (!n || !r) continue;
			let o = Number.parseFloat(getComputedStyle(t).getPropertyValue("--quadrado"));
			!Number.isFinite(o) || o <= 0 || a(t, n, r, o);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["style"]
	});
}
function c(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-valuenow") continue;
			let t = e.target, n = Number(e.oldValue), i = Number(t.getAttribute("aria-valuenow"));
			if (!Number.isFinite(n) || !Number.isFinite(i) || n === i) continue;
			let a = t.closest("li");
			a && requestAnimationFrame(() => r(a, { curou: i > n }));
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["aria-valuenow"]
	});
}
function l(e) {
	let t = /^\s*(-?\d+)\s*\//.exec(e ?? "");
	return t ? Number(t[1]) : null;
}
function u(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			let t = e.target, n = t.parentElement;
			if (!n?.hasAttribute("data-vital")) continue;
			let i = l(e.oldValue), a = l(t.textContent);
			if (i === null || a === null || i === a) continue;
			let o = n.parentElement;
			o && requestAnimationFrame(() => r(o, { curou: a > i }));
		}
	}).observe(document.body, {
		subtree: !0,
		characterData: !0,
		characterDataOldValue: !0
	});
}
function d(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-current") continue;
			let t = e.target;
			t.getAttribute("aria-current") === "true" && n(t);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeFilter: ["aria-current"]
	});
}
function f(t) {
	new MutationObserver((n) => {
		if (!t.matches) for (let t of n) for (let n of t.addedNodes) {
			if (n.nodeType !== Node.ELEMENT_NODE) continue;
			let t = n, r = t.matches("[data-condicao]") ? [t] : [...t.querySelectorAll("[data-condicao]")];
			for (let t of r) e(t);
		}
	}).observe(document.body, {
		subtree: !0,
		childList: !0
	});
}
var p = window.matchMedia(o);
s(p), c(p), u(p), d(p), f(p);
//#endregion
