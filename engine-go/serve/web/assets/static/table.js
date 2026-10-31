import { i as e, n as t, r as n, t as r } from "./turn-juice.js";
//#region serve/web/assets/src/lib/float-card.ts
var i = "t20-float-card";
function a(e) {
	if (!e) return null;
	try {
		let t = JSON.parse(e);
		return typeof t.x != "number" || typeof t.y != "number" || !Number.isFinite(t.x) || !Number.isFinite(t.y) ? null : {
			x: t.x,
			y: t.y
		};
	} catch {
		return null;
	}
}
function o(e, t, n) {
	return {
		x: Math.min(Math.max(0, e.x), Math.max(0, n.width - t.width)),
		y: Math.min(Math.max(0, e.y), Math.max(0, n.height - t.height))
	};
}
function s(e) {
	return getComputedStyle(e).position === "fixed";
}
function c(e, t) {
	let n = o(t, e.getBoundingClientRect(), {
		width: window.innerWidth,
		height: window.innerHeight
	});
	e.style.setProperty("--float-x", `${n.x}px`), e.style.setProperty("--float-y", `${n.y}px`);
}
function l(e, t) {
	if (e.style.getPropertyValue("--float-x") || !s(e)) return;
	let n = a(t.getItem(i));
	n && c(e, n);
}
function u(e = window.localStorage) {
	let t = null;
	document.addEventListener("pointerdown", (e) => {
		let n = e.target?.closest("[data-float-handle]"), r = n?.closest("[data-float]");
		if (!r || !s(r) || e.target.closest("button, a, input")) return;
		let i = r.getBoundingClientRect();
		t = {
			card: r,
			dx: e.clientX - i.left,
			dy: e.clientY - i.top,
			pointer: e.pointerId
		}, n.setPointerCapture(e.pointerId), r.dataset.dragging = "", e.preventDefault();
	}), document.addEventListener("pointermove", (e) => {
		!t || e.pointerId !== t.pointer || c(t.card, {
			x: e.clientX - t.dx,
			y: e.clientY - t.dy
		});
	});
	let n = (n) => {
		if (!t || n.pointerId !== t.pointer) return;
		let r = t.card.getBoundingClientRect();
		e.setItem(i, JSON.stringify({
			x: Math.round(r.left),
			y: Math.round(r.top)
		})), delete t.card.dataset.dragging, t = null;
	};
	document.addEventListener("pointerup", n), document.addEventListener("pointercancel", n);
	let r = () => {
		for (let t of document.querySelectorAll("[data-float]")) l(t, e);
	};
	new MutationObserver(r).observe(document.body, {
		subtree: !0,
		childList: !0
	}), window.addEventListener("resize", () => {
		for (let t of document.querySelectorAll("[data-float]")) {
			let n = a(e.getItem(i));
			n && s(t) && c(t, n);
		}
	}), r();
}
//#endregion
//#region serve/web/assets/src/lib/token-move.ts
function d(e) {
	if (!e) return null;
	let t = /--col:\s*(-?\d+)/.exec(e), n = /--lin:\s*(-?\d+)/.exec(e);
	return !t || !n ? null : {
		col: Number(t[1]),
		lin: Number(n[1])
	};
}
function f(e, t, r, i) {
	if (!n(e)) return;
	let a = (t.col - r.col) * i, o = (t.lin - r.lin) * i;
	a === 0 && o === 0 || e.animate([{ transform: `translate(${a}px, ${o}px)` }, { transform: "translate(0, 0)" }], {
		duration: 200,
		easing: "cubic-bezier(0.22, 1, 0.36, 1)"
	});
}
//#endregion
//#region serve/web/assets/src/table.ts
var p = "(prefers-reduced-motion: reduce)";
function m(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "style") continue;
			let t = e.target;
			if (!t.classList?.contains("board-token") || t.classList.contains("board-token-ghost")) continue;
			let n = d(e.oldValue), r = d(t.getAttribute("style"));
			if (!n || !r) continue;
			let i = Number.parseFloat(getComputedStyle(t).getPropertyValue("--quadrado"));
			!Number.isFinite(i) || i <= 0 || f(t, n, r, i);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["style"]
	});
}
function h(e) {
	new MutationObserver((n) => {
		if (!e.matches) for (let e of n) {
			if (e.attributeName !== "aria-valuenow") continue;
			let n = e.target, r = Number(e.oldValue), i = Number(n.getAttribute("aria-valuenow"));
			if (!Number.isFinite(r) || !Number.isFinite(i) || r === i) continue;
			let a = n.closest("li, .ds-row");
			a && requestAnimationFrame(() => t(a, { curou: i > r }));
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["aria-valuenow"]
	});
}
function g(e) {
	let t = /^\s*(-?\d+)\s*\//.exec(e ?? "");
	return t ? Number(t[1]) : null;
}
function _(e) {
	new MutationObserver((n) => {
		if (!e.matches) for (let e of n) {
			let n = e.target, r = n.parentElement;
			if (!r?.hasAttribute("data-vital")) continue;
			let i = g(e.oldValue), a = g(n.textContent);
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
function v(t) {
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
function y(e) {
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
var b = window.matchMedia(p);
m(b), h(b), _(b), v(b), y(b), u();
//#endregion
