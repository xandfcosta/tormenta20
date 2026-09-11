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
function a(e, t, r, i) {
	if (!n(e)) return;
	let a = (t.col - r.col) * i, o = (t.lin - r.lin) * i;
	a === 0 && o === 0 || e.animate([{ transform: `translate(${a}px, ${o}px)` }, { transform: "translate(0, 0)" }], {
		duration: 200,
		easing: "cubic-bezier(0.22, 1, 0.36, 1)"
	});
}
//#endregion
//#region api/piloto/src/table.ts
var o = "(prefers-reduced-motion: reduce)";
function s(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "style") continue;
			let t = e.target;
			if (!t.classList?.contains("board-token") || t.classList.contains("board-token-ghost")) continue;
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
	new MutationObserver((n) => {
		if (!e.matches) for (let e of n) {
			if (e.attributeName !== "aria-valuenow") continue;
			let n = e.target, r = Number(e.oldValue), i = Number(n.getAttribute("aria-valuenow"));
			if (!Number.isFinite(r) || !Number.isFinite(i) || r === i) continue;
			let a = n.closest("li");
			a && requestAnimationFrame(() => t(a, { curou: i > r }));
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
	new MutationObserver((n) => {
		if (!e.matches) for (let e of n) {
			let n = e.target, r = n.parentElement;
			if (!r?.hasAttribute("data-vital")) continue;
			let i = l(e.oldValue), a = l(n.textContent);
			if (i === null || a === null || i === a) continue;
			let o = r.parentElement;
			o && requestAnimationFrame(() => t(o, { curou: a > i }));
		}
	}).observe(document.body, {
		subtree: !0,
		characterData: !0,
		characterDataOldValue: !0
	});
}
function d(t) {
	new MutationObserver((n) => {
		if (!t.matches) for (let t of n) {
			if (t.attributeName !== "aria-current") continue;
			let n = t.target;
			n.getAttribute("aria-current") === "true" && e(n);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeFilter: ["aria-current"]
	});
}
function f(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) for (let t of e.addedNodes) {
			if (t.nodeType !== Node.ELEMENT_NODE) continue;
			let e = t, n = e.matches("[data-condicao]") ? [e] : [...e.querySelectorAll("[data-condicao]")];
			for (let e of n) r(e);
		}
	}).observe(document.body, {
		subtree: !0,
		childList: !0
	});
}
var p = window.matchMedia(o);
s(p), c(p), u(p), d(p), f(p);
//#endregion
