//#region serve/web/assets/src/lib/copy-text.ts
async function e(e) {
	if (navigator.clipboard) try {
		return await navigator.clipboard.writeText(e), !0;
	} catch {}
	return t(e);
}
function t(e) {
	let t = document.activeElement, n = document.createElement("textarea");
	n.value = e, n.readOnly = !0, n.style.cssText = "position:fixed;top:0;left:0;opacity:0", (document.querySelector("dialog[open]") ?? document.body).append(n), n.select(), n.setSelectionRange(0, e.length);
	let r = !1;
	try {
		r = document.execCommand("copy");
	} catch {
		r = !1;
	}
	return n.remove(), t?.focus(), r;
}
//#endregion
//#region serve/web/assets/src/lib/fetch-busy.ts
var n = "button, a, input, select, textarea, [role=\"switch\"]";
function r(e) {
	let t = /* @__PURE__ */ new WeakMap();
	e.addEventListener("datastar-fetch", (e) => {
		let { type: r, el: i } = e.detail;
		if (!(i instanceof Element) || !i.matches(n)) return;
		let a = t.get(i) ?? 0;
		if (r === "started") {
			t.set(i, a + 1), i.setAttribute("aria-busy", "true");
			return;
		}
		if (r !== "finished") return;
		let o = Math.max(0, a - 1);
		t.set(i, o), o === 0 && i.removeAttribute("aria-busy");
	});
}
//#endregion
//#region serve/web/assets/src/lib/spatial-nav.ts
var i = 2, a = {
	left: !0,
	right: !0,
	up: !1,
	down: !1
};
function o(e) {
	return (e.left + e.right) / 2;
}
function s(e) {
	return (e.top + e.bottom) / 2;
}
function c(e, t) {
	return t === "grid" ? !0 : t === "row" ? a[e] : !a[e];
}
function l(e, t, n) {
	switch (n) {
		case "right": return o(t) > o(e) + .5;
		case "left": return o(t) < o(e) - .5;
		case "down": return s(t) > s(e) + .5;
		case "up": return s(t) < s(e) - .5;
	}
}
function u(e, t, n) {
	let r = Math.abs(o(t) - o(e)), c = Math.abs(s(t) - s(e));
	return (a[n] ? r : c) + i * (a[n] ? c : r);
}
function d(e, t, n, r) {
	if (!c(n, r)) return null;
	let i = e.find((e) => e.id === t);
	if (!i) return null;
	let a = null, o = Infinity;
	for (let r of e) {
		if (r.id === t || !l(i.rect, r.rect, n)) continue;
		let e = u(i.rect, r.rect, n);
		e < o && (o = e, a = r.id);
	}
	return a;
}
function f(e, t, n) {
	let r = null, i = Infinity;
	for (let a of e) {
		if (!l(t, a.rect, n)) continue;
		let e = u(t, a.rect, n);
		e < i && (i = e, r = a.id);
	}
	return r;
}
function ee(e, t) {
	let n = null, r = Infinity;
	for (let i of e) {
		let e = o(i.rect) - o(t), a = s(i.rect) - s(t), c = e * e + a * a;
		c < r && (r = c, n = i.id);
	}
	return n;
}
//#endregion
//#region serve/web/assets/src/lib/scene-nav.ts
var p = "(min-width: 1280px) and (pointer: fine)", m = "a[href], button:not([disabled]), [tabindex]:not([tabindex=\"-1\"]), [data-nav-item]";
function h(e) {
	let t = /* @__PURE__ */ new WeakMap(), n = (n) => g(n, e, t), r = typeof window.matchMedia == "function" ? window.matchMedia(p) : null, i = !1, a = () => {
		let t = (r?.matches ?? !1) && e.active?.() !== !1;
		t !== i && (i = t, t ? window.addEventListener("keydown", n, !0) : window.removeEventListener("keydown", n, !0));
	};
	return a(), r?.addEventListener("change", a), () => {
		r?.removeEventListener("change", a), i && window.removeEventListener("keydown", n, !0), i = !1;
	};
}
function g(e, t, n) {
	if (e.ctrlKey || e.metaKey || e.altKey || j()) return;
	let r = t.root();
	if (!r) return;
	let i = document.activeElement?.closest("[data-nav-region]") ?? null;
	if (t.delegated || i?.dataset.navMode === "delegated") {
		if (t.onKey?.(e) || A()) return;
		let n = M(e);
		if (!n) return;
		if (t.onCommand?.(n)) {
			N(e);
			return;
		}
		n.type === "back" && (N(e), t.onEscape());
		return;
	}
	if (A()) return;
	let a = M(e);
	a && te(e, a, t, i, r, n);
}
function te(e, t, n, r, i, a) {
	switch (t.type) {
		case "back":
			v(e, n, r, i, a);
			return;
		case "bumper":
			y(e, n, t.dir);
			return;
		case "edge":
			_(e, n, r, t.to, a);
			return;
		case "move":
			ne(e, n, r, t.dir, i, a);
			return;
		case "activate": return;
	}
}
function ne(e, t, n, r, i, a) {
	if (!n) {
		let n = document.activeElement;
		(!n || n === document.body) && b(e, t, i, a);
		return;
	}
	let o = T(n), s = document.activeElement, c = o.indexOf(s), l = c === -1 ? null : d(E(o), String(c), r, D(n));
	if (l !== null) {
		S(o[Number(l)], n, t, a, "hover"), N(e);
		return;
	}
	let u = re(n, r, O(s), i);
	u && (x(u, O(s), t, a, "select"), N(e));
}
function _(e, t, n, r, i) {
	if (!n) return;
	let a = T(n);
	a.length !== 0 && (S(r === "first" ? a[0] : a[a.length - 1], n, t, i, "hover"), N(e));
}
function v(e, t, n, r, i) {
	N(e);
	let a = r.querySelector("[data-nav-region=\"rail\"]");
	if (n && a && n !== a) {
		x(a, O(n), t, i, "back");
		return;
	}
	t.onEscape();
}
function y(e, t, n) {
	t.bumpers && (N(e), n === "next" ? t.bumpers.next() : t.bumpers.prev(), t.sfx("select"), requestAnimationFrame(() => {
		(t.root()?.querySelector("[data-nav-region=\"rail\"]"))?.querySelector(w)?.focus();
	}));
}
function b(e, t, n, r) {
	let i = n.querySelector("[data-nav-region=\"rail\"]") ?? n.querySelector("[data-nav-region]");
	i && (x(i, C, t, r, "select"), N(e));
}
function x(e, t, n, r, i) {
	let a = r.get(e);
	if (a && e.contains(a) && k(a)) {
		S(a, e, n, r, i);
		return;
	}
	let o = e.querySelector(w);
	if (o) {
		S(o, e, n, r, i);
		return;
	}
	let s = T(e);
	if (s.length === 0) return;
	let c = ee(E(s), t);
	c !== null && S(s[Number(c)], e, n, r, i);
}
function re(e, t, n, r) {
	let i = Array.from(r.querySelectorAll("[data-nav-region]")).filter((t) => t !== e && k(t)), a = e.getAttribute(`data-nav-edge-${t}`);
	if (a) return i.find((e) => e.dataset.navRegion === a) ?? null;
	let o = f(i.map((e, t) => ({
		id: String(t),
		rect: O(e)
	})), n, t);
	return o === null ? null : i[Number(o)];
}
function S(e, t, n, r, i) {
	e.focus(), r.set(t, e), n.sfx(i), e.scrollIntoView({
		block: "nearest",
		inline: "nearest"
	});
}
var C = {
	left: 0,
	top: 0,
	right: 0,
	bottom: 0
}, w = "[role=\"tab\"][data-selected], [role=\"tab\"][data-state=\"active\"]";
function T(e) {
	return Array.from(e.querySelectorAll(m)).filter((t) => t.closest("[data-nav-region]") === e && t.getAttribute("role") !== "tabpanel" && !t.hasAttribute("data-nav-skip") && k(t));
}
function E(e) {
	return e.map((e, t) => ({
		id: String(t),
		rect: O(e)
	}));
}
function D(e) {
	let t = e.dataset.navLayout;
	return t === "grid" || t === "row" ? t : "column";
}
function O(e) {
	let t = e.getBoundingClientRect();
	return {
		left: t.left,
		top: t.top,
		right: t.right,
		bottom: t.bottom
	};
}
function k(e) {
	let t = e.getBoundingClientRect();
	return t.width > 0 && t.height > 0 && e.checkVisibility();
}
function A() {
	let e = document.activeElement;
	return e instanceof HTMLElement ? e.tagName === "INPUT" || e.tagName === "TEXTAREA" || e.tagName === "SELECT" || e.isContentEditable : !1;
}
function j() {
	return !!document.querySelector([
		"[role=\"dialog\"][data-expanded]:not([data-nav-inline])",
		"[role=\"menu\"][data-expanded]",
		"[role=\"listbox\"][data-expanded]",
		"[popover]:popover-open",
		"dialog[open]"
	].join(", "));
}
function M(e) {
	switch (e.key) {
		case "ArrowUp": return {
			type: "move",
			dir: "up"
		};
		case "ArrowDown": return {
			type: "move",
			dir: "down"
		};
		case "ArrowLeft": return {
			type: "move",
			dir: "left"
		};
		case "ArrowRight": return {
			type: "move",
			dir: "right"
		};
		case "Enter": return { type: "activate" };
		case "Escape": return { type: "back" };
		case "PageUp": return {
			type: "bumper",
			dir: "prev"
		};
		case "PageDown": return {
			type: "bumper",
			dir: "next"
		};
		case "Home": return {
			type: "edge",
			to: "first"
		};
		case "End": return {
			type: "edge",
			to: "last"
		};
		default: return null;
	}
}
function N(e) {
	e.preventDefault(), e.stopPropagation();
}
//#endregion
//#region serve/web/assets/src/lib/audio-gate.ts
var P = ["pointerdown", "keydown"], F = class {
	ctx = null;
	constructor(e, t) {
		this.openContext = e;
		for (let e of P) t.addEventListener(e, () => this.arm(), { capture: !0 });
	}
	ready() {
		let e = this.ctx;
		return e && e.state === "running" ? e : null;
	}
	arm() {
		this.ctx ||= this.openContext(), this.ctx?.state === "suspended" && this.ctx.resume();
	}
}, I = {
	hover: {
		shape: "sweep",
		type: "triangle",
		from: 1300,
		to: 1100,
		dur: .05,
		gain: .03
	},
	select: {
		shape: "sweep",
		type: "sine",
		from: 520,
		to: 780,
		dur: .12,
		gain: .08
	},
	transition: {
		shape: "sweep",
		type: "sine",
		from: 620,
		to: 200,
		dur: .28,
		gain: .06
	},
	open: {
		shape: "sweep",
		type: "sine",
		from: 320,
		to: 760,
		dur: .22,
		gain: .07
	},
	back: {
		shape: "sweep",
		type: "triangle",
		from: 660,
		to: 240,
		dur: .18,
		gain: .06
	},
	turn: {
		shape: "bell",
		partials: [880, 1320],
		dur: .9,
		gain: .1
	}
};
function L(e, t, n) {
	if (n <= 0) return;
	let r = I[t];
	if (r.shape === "bell") {
		z(e, r, n);
		return;
	}
	R(e, r, n);
}
function R(e, { type: t, from: n, to: r, dur: i, gain: a }, o) {
	let s = e.currentTime, c = e.createOscillator();
	c.type = t, c.frequency.setValueAtTime(n, s), c.frequency.exponentialRampToValueAtTime(r, s + i), c.connect(B(e, a * o, s, .012, i)).connect(e.destination), c.start(s), c.stop(s + i + .02);
}
function z(e, { partials: t, dur: n, gain: r }, i) {
	let a = e.currentTime;
	t.forEach((t, o) => {
		let s = r * i * (o === 0 ? 1 : .45), c = n * (o === 0 ? 1 : .55), l = e.createOscillator();
		l.type = "sine", l.frequency.setValueAtTime(t, a), l.connect(B(e, s, a, .005, c)).connect(e.destination), l.start(a), l.stop(a + c + .02);
	});
}
function B(e, t, n, r, i) {
	let a = e.createGain();
	return a.gain.setValueAtTime(1e-4, n), a.gain.exponentialRampToValueAtTime(Math.max(t, 2e-4), n + r), a.gain.exponentialRampToValueAtTime(1e-4, n + i), a;
}
//#endregion
//#region serve/web/assets/src/lib/sfx-player.ts
var V = class {
	constructor(e = H()) {
		this.gate = e;
	}
	play(e, t) {
		try {
			let n = this.gate.ready();
			n && L(n, e, t);
		} catch {}
	}
};
function H() {
	return new F(U, window);
}
function U() {
	let e = window.AudioContext ?? window.webkitAudioContext;
	return e ? new e() : null;
}
function W() {
	return new V();
}
//#endregion
//#region serve/web/assets/src/lib/ui-store.ts
var G = "t20-ui";
function K(e, t = globalThis.localStorage) {
	t?.setItem(G, JSON.stringify({ state: e }));
}
var q = 100;
function J(e) {
	if (e) try {
		return JSON.parse(e).state;
	} catch {
		return;
	}
}
function Y(e) {
	return J(e)?.sfx === !0;
}
function X(e) {
	return ie(J(e)?.volume);
}
function ie(e) {
	return typeof e != "number" || Number.isNaN(e) ? q : Math.min(100, Math.max(0, Math.round(e)));
}
//#endregion
//#region serve/web/assets/src/scene.ts
function ae() {
	let e = globalThis.localStorage?.getItem("t20-ui") ?? null;
	return {
		som: Y(e),
		volume: X(e)
	};
}
var Z = ae(), oe = W();
function Q(e) {
	Z.som && (e === "hover" && window.matchMedia?.("(pointer: coarse)").matches || oe.play(e, Z.volume / 100));
}
function se(e) {
	if (e.hasAttribute("data-sheet")) return;
	let t = document.querySelector(`[popovertarget="${e.id}"]`);
	if (!t) return;
	let n = t.getBoundingClientRect(), r = n.top - 16, i = window.innerHeight - n.bottom - 16, a = r > i;
	e.style.top = a ? "auto" : `${Math.round(n.bottom + 8)}px`, e.style.bottom = a ? `${Math.round(window.innerHeight - n.top + 8)}px` : "auto", e.style.maxHeight = `${Math.max(96, Math.round(a ? r : i))}px`;
	let o = e.offsetWidth || 224;
	e.style.left = `${Math.round(Math.max(8, Math.min(n.left, window.innerWidth - o - 8)))}px`;
}
window.cena = {
	som() {
		return Z.som = !Z.som, K({
			sfx: Z.som,
			volume: Z.volume
		}), Z.som && Q("select"), Z.som;
	},
	volume(e) {
		return Z.volume = Math.min(100, Math.max(0, Math.round(e))), K({
			sfx: Z.som,
			volume: Z.volume
		}), Z.volume;
	},
	telaCheia() {
		return document.fullscreenElement ? (document.exitFullscreen(), !1) : (document.documentElement.requestFullscreen?.(), !0);
	},
	temTelaCheia() {
		return typeof document.documentElement.requestFullscreen == "function";
	},
	preferencias() {
		return { ...Z };
	},
	cue: Q,
	copyText: e
}, r(document), window.addEventListener("popstate", (e) => {
	e.state?.softNav && window.location.reload();
});
var $ = () => document.querySelector("[data-slot=\"scene-shell\"]");
h({
	root: $,
	onEscape: () => {
		let e = $()?.dataset.voltar;
		e && (window.location.href = e);
	},
	sfx: Q
}), document.addEventListener("pointerover", (e) => {
	e.target?.closest?.("[data-cue-hover]") && Q("hover");
}, !0), document.addEventListener("click", (e) => {
	e.target?.closest?.("[data-cue-select]") && Q("select");
});
for (let e of ["beforetoggle", "toggle"]) document.addEventListener(e, (e) => {
	let t = e.target;
	t?.matches?.("[popover]") && e.newState === "open" && se(t);
}, !0);
//#endregion
