//#region serve/web/assets/src/lib/float-card.ts
var e = "t20-float-card", t = "t20-float-fold", n = "t20-float-fold-once", r = 8, i = 12;
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
var c = /* @__PURE__ */ new Map();
function l(e, t) {
	e.style.setProperty("--float-x", `${t.x}px`), e.style.setProperty("--float-y", `${t.y}px`), e.id && c.set(e.id, t);
}
function u(e, t) {
	l(e, o(t, e.getBoundingClientRect(), {
		width: window.innerWidth,
		height: window.innerHeight
	}));
}
function d(e) {
	for (let t of document.querySelectorAll("[data-float]")) {
		h(t, e);
		let n = t.id ? c.get(t.id) : void 0;
		n && !t.style.getPropertyValue("--float-x") && l(t, n);
	}
}
function f(t) {
	return t.id === "table-attack" || !t.id ? e : `${e}:${t.id}`;
}
function p(e) {
	return `${e.hasAttribute("data-float-fleeting") ? n : t}:${e.id}`;
}
function m(e) {
	for (let t = e.length - 1; t >= 0; t--) {
		let r = e.key(t);
		if (!r?.startsWith(`${n}:`)) continue;
		let i = r.slice(20);
		document.querySelector(`#${CSS.escape(i)}[data-float]`) || e.removeItem(r);
	}
}
function h(e, t) {
	t.getItem(p(e)) && !e.hasAttribute("data-folded") && e.setAttribute("data-folded", ""), e.querySelector("[data-float-fold]")?.setAttribute("aria-expanded", e.hasAttribute("data-folded") ? "false" : "true");
}
function g() {
	let e = document.querySelector("[data-float-floor]");
	return e ? e.getBoundingClientRect().top : window.innerHeight;
}
function _(e) {
	m(e);
	let t = [];
	for (let n of document.querySelectorAll("[data-float]")) {
		if (h(n, e), !s(n)) continue;
		let r = a(e.getItem(f(n)));
		r ? u(n, r) : t.push(n);
	}
	let n = g() - i;
	for (let e of t.reverse()) {
		let t = e.getBoundingClientRect();
		n -= t.height, l(e, {
			x: Math.max(i, window.innerWidth - t.width - i),
			y: Math.max(i, n)
		}), n -= r;
	}
}
function v(e = window.localStorage) {
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
		!t || e.pointerId !== t.pointer || u(t.card, {
			x: e.clientX - t.dx,
			y: e.clientY - t.dy
		});
	});
	let n = (n) => {
		if (!t || n.pointerId !== t.pointer) return;
		let r = t.card.getBoundingClientRect();
		e.setItem(f(t.card), JSON.stringify({
			x: Math.round(r.left),
			y: Math.round(r.top)
		})), delete t.card.dataset.dragging, t = null;
	};
	document.addEventListener("pointerup", n), document.addEventListener("pointercancel", n), document.addEventListener("click", (t) => {
		let n = (t.target?.closest("[data-float-fold]"))?.closest("[data-float]");
		n && (n.toggleAttribute("data-folded") ? e.setItem(p(n), "1") : e.removeItem(p(n)), _(e));
	});
	let r = 0, i = () => {
		r ||= requestAnimationFrame(() => {
			r = 0, _(e);
		});
	}, a = new ResizeObserver(i), o = () => {
		d(e);
		for (let e of document.querySelectorAll("[data-float]")) a.observe(e);
		i();
	};
	new MutationObserver(o).observe(document.body, {
		subtree: !0,
		childList: !0
	}), window.addEventListener("resize", i), o();
}
//#endregion
//#region serve/web/assets/src/lib/turn-juice.ts
function y(e) {
	return !!e && typeof e.animate == "function";
}
function b(e, t) {
	if (!y(e)) return;
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
function x(e) {
	y(e) && e.animate([
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
function S(e) {
	y(e) && e.animate([{
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
function C(e) {
	if (!e) return null;
	let t = /--col:\s*(-?\d+)/.exec(e), n = /--lin:\s*(-?\d+)/.exec(e);
	return !t || !n ? null : {
		col: Number(t[1]),
		lin: Number(n[1])
	};
}
function w(e, t, n, r) {
	if (!y(e)) return;
	let i = (t.col - n.col) * r, a = (t.lin - n.lin) * r;
	i === 0 && a === 0 || e.animate([{ transform: `translate(${i}px, ${a}px)` }, { transform: "translate(0, 0)" }], {
		duration: 200,
		easing: "cubic-bezier(0.22, 1, 0.36, 1)"
	});
}
//#endregion
//#region serve/web/assets/src/table.ts
var T = "(prefers-reduced-motion: reduce)";
function E(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "style") continue;
			let t = e.target;
			if (!t.classList?.contains("board-token") || t.classList.contains("board-token-ghost")) continue;
			let n = C(e.oldValue), r = C(t.getAttribute("style"));
			if (!n || !r) continue;
			let i = Number.parseFloat(getComputedStyle(t).getPropertyValue("--quadrado"));
			!Number.isFinite(i) || i <= 0 || w(t, n, r, i);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["style"]
	});
}
function D(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-valuenow") continue;
			let t = e.target, n = Number(e.oldValue), r = Number(t.getAttribute("aria-valuenow"));
			if (!Number.isFinite(n) || !Number.isFinite(r) || n === r) continue;
			let i = t.closest("li, .ds-row");
			i && requestAnimationFrame(() => b(i, { curou: r > n }));
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeOldValue: !0,
		attributeFilter: ["aria-valuenow"]
	});
}
function O(e) {
	let t = /^\s*(-?\d+)\s*\//.exec(e ?? "");
	return t ? Number(t[1]) : null;
}
function k(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			let t = e.target, n = t.parentElement;
			if (!n?.hasAttribute("data-vital")) continue;
			let r = O(e.oldValue), i = O(t.textContent);
			if (r === null || i === null || r === i) continue;
			let a = n.parentElement;
			a && requestAnimationFrame(() => b(a, { curou: i > r }));
		}
	}).observe(document.body, {
		subtree: !0,
		characterData: !0,
		characterDataOldValue: !0
	});
}
function A(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) {
			if (e.attributeName !== "aria-current") continue;
			let t = e.target;
			t.getAttribute("aria-current") === "true" && x(t);
		}
	}).observe(document.body, {
		subtree: !0,
		attributes: !0,
		attributeFilter: ["aria-current"]
	});
}
function j(e) {
	new MutationObserver((t) => {
		if (!e.matches) for (let e of t) for (let t of e.addedNodes) {
			if (t.nodeType !== Node.ELEMENT_NODE) continue;
			let e = t, n = e.matches("[data-condicao]") ? [e] : [...e.querySelectorAll("[data-condicao]")];
			for (let e of n) S(e);
		}
	}).observe(document.body, {
		subtree: !0,
		childList: !0
	});
}
var M = window.matchMedia(T);
E(M), D(M), k(M), A(M), j(M), v();
//#endregion
