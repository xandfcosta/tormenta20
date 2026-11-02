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
function a(n, a) {
	if (n.style.getPropertyValue("--float-x") || !r(n)) return;
	let o = t(a.getItem(e));
	o && i(n, o);
}
function o(n = window.localStorage) {
	let o = null;
	document.addEventListener("pointerdown", (e) => {
		let t = e.target?.closest("[data-float-handle]"), n = t?.closest("[data-float]");
		if (!n || !r(n) || e.target.closest("button, a, input")) return;
		let i = n.getBoundingClientRect();
		o = {
			card: n,
			dx: e.clientX - i.left,
			dy: e.clientY - i.top,
			pointer: e.pointerId
		}, t.setPointerCapture(e.pointerId), n.dataset.dragging = "", e.preventDefault();
	}), document.addEventListener("pointermove", (e) => {
		!o || e.pointerId !== o.pointer || i(o.card, {
			x: e.clientX - o.dx,
			y: e.clientY - o.dy
		});
	});
	let s = (t) => {
		if (!o || t.pointerId !== o.pointer) return;
		let r = o.card.getBoundingClientRect();
		n.setItem(e, JSON.stringify({
			x: Math.round(r.left),
			y: Math.round(r.top)
		})), delete o.card.dataset.dragging, o = null;
	};
	document.addEventListener("pointerup", s), document.addEventListener("pointercancel", s);
	let c = () => {
		for (let e of document.querySelectorAll("[data-float]")) a(e, n);
	};
	new MutationObserver(c).observe(document.body, {
		subtree: !0,
		childList: !0
	}), window.addEventListener("resize", () => {
		for (let a of document.querySelectorAll("[data-float]")) {
			let o = t(n.getItem(e));
			o && r(a) && i(a, o);
		}
	}), c();
}
//#endregion
//#region serve/web/assets/src/lib/turn-juice.ts
function s(e) {
	return !!e && typeof e.animate == "function";
}
function c(e, t) {
	if (!s(e)) return;
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
function l(e) {
	s(e) && e.animate([
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
function u(e) {
	s(e) && e.animate([{
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
function d(e) {
	if (!e) return null;
	let t = /--col:\s*(-?\d+)/.exec(e), n = /--lin:\s*(-?\d+)/.exec(e);
	return !t || !n ? null : {
		col: Number(t[1]),
		lin: Number(n[1])
	};
}
function f(e, t, n, r) {
	if (!s(e)) return;
	let i = (t.col - n.col) * r, a = (t.lin - n.lin) * r;
	i === 0 && a === 0 || e.animate([{ transform: `translate(${i}px, ${a}px)` }, { transform: "translate(0, 0)" }], {
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
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-valuenow") continue;
			let t = e.target, n = Number(e.oldValue), r = Number(t.getAttribute("aria-valuenow"));
			if (!Number.isFinite(n) || !Number.isFinite(r) || n === r) continue;
			let i = t.closest("li, .ds-row");
			i && requestAnimationFrame(() => c(i, { curou: r > n }));
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
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			let t = e.target, n = t.parentElement;
			if (!n?.hasAttribute("data-vital")) continue;
			let r = g(e.oldValue), i = g(t.textContent);
			if (r === null || i === null || r === i) continue;
			let a = n.parentElement;
			a && requestAnimationFrame(() => c(a, { curou: i > r }));
		}
	}).observe(document.body, {
		subtree: !0,
		characterData: !0,
		characterDataOldValue: !0
	});
}
function v(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-current") continue;
			let t = e.target;
			t.getAttribute("aria-current") === "true" && l(t);
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
			for (let e of n) u(e);
		}
	}).observe(document.body, {
		subtree: !0,
		childList: !0
	});
}
var b = window.matchMedia(p);
m(b), h(b), _(b), v(b), y(b), o();
//#endregion
