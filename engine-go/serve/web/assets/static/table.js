//#region serve/web/assets/src/lib/float-card.ts
var e = "t20-float-card";
function t(e) {
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
function n(e, t, n) {
	return {
		x: Math.min(Math.max(0, e.x), Math.max(0, n.width - t.width)),
		y: Math.min(Math.max(0, e.y), Math.max(0, n.height - t.height))
	};
}
function r(e) {
	return getComputedStyle(e).position === "fixed";
}
function i(e, t) {
	let r = n(t, e.getBoundingClientRect(), {
		width: window.innerWidth,
		height: window.innerHeight
	});
	e.style.setProperty("--float-x", `${r.x}px`), e.style.setProperty("--float-y", `${r.y}px`);
}
function a(t) {
	return t.id === "table-attack" || !t.id ? e : `${e}:${t.id}`;
}
function o(e, n) {
	if (e.style.getPropertyValue("--float-x") || !r(e)) return;
	let o = t(n.getItem(a(e)));
	o && i(e, o);
}
function s(e = window.localStorage) {
	let n = null;
	document.addEventListener("pointerdown", (e) => {
		let t = e.target?.closest("[data-float-handle]"), i = t?.closest("[data-float]");
		if (!i || !r(i) || e.target.closest("button, a, input")) return;
		let a = i.getBoundingClientRect();
		n = {
			card: i,
			dx: e.clientX - a.left,
			dy: e.clientY - a.top,
			pointer: e.pointerId
		}, t.setPointerCapture(e.pointerId), i.dataset.dragging = "", e.preventDefault();
	}), document.addEventListener("pointermove", (e) => {
		!n || e.pointerId !== n.pointer || i(n.card, {
			x: e.clientX - n.dx,
			y: e.clientY - n.dy
		});
	});
	let s = (t) => {
		if (!n || t.pointerId !== n.pointer) return;
		let r = n.card.getBoundingClientRect();
		e.setItem(a(n.card), JSON.stringify({
			x: Math.round(r.left),
			y: Math.round(r.top)
		})), delete n.card.dataset.dragging, n = null;
	};
	document.addEventListener("pointerup", s), document.addEventListener("pointercancel", s);
	let c = () => {
		for (let t of document.querySelectorAll("[data-float]")) o(t, e);
	};
	new MutationObserver(c).observe(document.body, {
		subtree: !0,
		childList: !0
	}), window.addEventListener("resize", () => {
		for (let n of document.querySelectorAll("[data-float]")) {
			let o = t(e.getItem(a(n)));
			o && r(n) && i(n, o);
		}
	}), c();
}
//#endregion
//#region serve/web/assets/src/lib/turn-juice.ts
function c(e) {
	return !!e && typeof e.animate == "function";
}
function l(e, t) {
	if (!c(e)) return;
	let n = e.getBoundingClientRect();
	if (n.width === 0 || n.height === 0) return;
	let r = document.createElement("div");
	r.setAttribute("aria-hidden", "true"), r.setAttribute("data-vital-blink", t.curou ? "curou" : "feriu"), r.style.cssText = [
		"position:fixed",
		`top:${n.top}px`,
		`left:${n.left}px`,
		`width:${n.width}px`,
		`height:${n.height}px`,
		`border-radius:${getComputedStyle(e).borderRadius}`,
		"pointer-events:none",
		"z-index:40",
		`background:var(${t.curou ? "--hp-full" : "--hp-critical"})`
	].join(";"), document.body.appendChild(r), r.animate([{ opacity: .45 }, { opacity: 0 }], {
		duration: 380,
		easing: "ease-out"
	}).finished.then(() => r.remove()).catch(() => r.remove());
}
function u(e) {
	c(e) && e.animate([
		{
			transform: "scale(1)",
			boxShadow: "0 0 0 0 transparent"
		},
		{
			transform: "scale(1.015)",
			boxShadow: "0 0 14px 2px color-mix(in oklch, var(--ds-map-gold) 45%, transparent)",
			offset: .4
		},
		{
			transform: "scale(1)",
			boxShadow: "0 0 0 0 transparent"
		}
	], {
		duration: 250,
		easing: "ease-out"
	});
}
function d(e) {
	c(e) && e.animate([{
		opacity: 0,
		transform: "scale(0.92)"
	}, {
		opacity: 1,
		transform: "scale(1)"
	}], {
		duration: 150,
		easing: "ease-out"
	});
}
//#endregion
//#region serve/web/assets/src/lib/token-move.ts
function f(e) {
	if (!e) return null;
	let t = /--col:\s*(-?\d+)/.exec(e), n = /--lin:\s*(-?\d+)/.exec(e);
	return !t || !n ? null : {
		col: Number(t[1]),
		lin: Number(n[1])
	};
}
function p(e, t, n, r) {
	if (!c(e)) return;
	let i = (t.col - n.col) * r, a = (t.lin - n.lin) * r;
	i === 0 && a === 0 || e.animate([{ transform: `translate(${i}px, ${a}px)` }, { transform: "translate(0, 0)" }], {
		duration: 200,
		easing: "cubic-bezier(0.22, 1, 0.36, 1)"
	});
}
//#endregion
//#region serve/web/assets/src/table.ts
var m = "(prefers-reduced-motion: reduce)";
function h(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "style") continue;
			let t = e.target;
			if (!t.classList?.contains("board-token") || t.classList.contains("board-token-ghost")) continue;
			let n = f(e.oldValue), r = f(t.getAttribute("style"));
			if (!n || !r) continue;
			let i = Number.parseFloat(getComputedStyle(t).getPropertyValue("--quadrado"));
			!Number.isFinite(i) || i <= 0 || p(t, n, r, i);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["style"]
	});
}
function g(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-valuenow") continue;
			let t = e.target, n = Number(e.oldValue), r = Number(t.getAttribute("aria-valuenow"));
			if (!Number.isFinite(n) || !Number.isFinite(r) || n === r) continue;
			let i = t.closest("li, .ds-row");
			i && requestAnimationFrame(() => l(i, { curou: r > n }));
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["aria-valuenow"]
	});
}
function _(e) {
	let t = /^\s*(-?\d+)\s*\//.exec(e ?? "");
	return t ? Number(t[1]) : null;
}
function v(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			let t = e.target, n = t.parentElement;
			if (!n?.hasAttribute("data-vital")) continue;
			let r = _(e.oldValue), i = _(t.textContent);
			if (r === null || i === null || r === i) continue;
			let a = n.parentElement;
			a && requestAnimationFrame(() => l(a, { curou: i > r }));
		}
	}).observe(document.body, {
		subtree: !0,
		characterData: !0,
		characterDataOldValue: !0
	});
}
function y(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-current") continue;
			let t = e.target;
			t.getAttribute("aria-current") === "true" && u(t);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeFilter: ["aria-current"]
	});
}
function b(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) for (let t of e.addedNodes) {
			if (t.nodeType !== Node.ELEMENT_NODE) continue;
			let e = t, n = e.matches("[data-condicao]") ? [e] : [...e.querySelectorAll("[data-condicao]")];
			for (let e of n) d(e);
		}
	}).observe(document.body, {
		subtree: !0,
		childList: !0
	});
}
var x = window.matchMedia(m);
h(x), g(x), v(x), y(x), b(x), s();
//#endregion
