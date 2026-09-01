//#region api/piloto/src/lib/spatial-nav.ts
var e = 2, t = {
	left: !0,
	right: !0,
	up: !1,
	down: !1
};
function n(e) {
	return (e.left + e.right) / 2;
}
function r(e) {
	return (e.top + e.bottom) / 2;
}
function i(e, n) {
	return n === "grid" ? !0 : n === "row" ? t[e] : !t[e];
}
function a(e, t, i) {
	switch (i) {
		case "right": return n(t) > n(e) + .5;
		case "left": return n(t) < n(e) - .5;
		case "down": return r(t) > r(e) + .5;
		case "up": return r(t) < r(e) - .5;
	}
}
function o(i, a, o) {
	let s = Math.abs(n(a) - n(i)), c = Math.abs(r(a) - r(i));
	return (t[o] ? s : c) + e * (t[o] ? c : s);
}
function s(e, t, n, r) {
	if (!i(n, r)) return null;
	let s = e.find((e) => e.id === t);
	if (!s) return null;
	let c = null, l = Infinity;
	for (let r of e) {
		if (r.id === t || !a(s.rect, r.rect, n)) continue;
		let e = o(s.rect, r.rect, n);
		e < l && (l = e, c = r.id);
	}
	return c;
}
function c(e, t, n) {
	let r = null, i = Infinity;
	for (let s of e) {
		if (!a(t, s.rect, n)) continue;
		let e = o(t, s.rect, n);
		e < i && (i = e, r = s.id);
	}
	return r;
}
function l(e, t) {
	let i = null, a = Infinity;
	for (let o of e) {
		let e = n(o.rect) - n(t), s = r(o.rect) - r(t), c = e * e + s * s;
		c < a && (a = c, i = o.id);
	}
	return i;
}
//#endregion
//#region api/piloto/src/lib/scene-nav.ts
var u = "(min-width: 1280px) and (pointer: fine)", d = "a[href], button:not([disabled]), [tabindex]:not([tabindex=\"-1\"]), [data-nav-item]";
function f(e) {
	let t = /* @__PURE__ */ new WeakMap(), n = (n) => p(n, e, t), r = typeof window.matchMedia == "function" ? window.matchMedia(u) : null, i = !1, a = () => {
		let t = (r?.matches ?? !1) && e.active?.() !== !1;
		t !== i && (i = t, t ? window.addEventListener("keydown", n, !0) : window.removeEventListener("keydown", n, !0));
	};
	return a(), r?.addEventListener("change", a), () => {
		r?.removeEventListener("change", a), i && window.removeEventListener("keydown", n, !0), i = !1;
	};
}
function p(e, t, n) {
	if (e.ctrlKey || e.metaKey || e.altKey || k()) return;
	let r = t.root();
	if (!r) return;
	let i = document.activeElement?.closest("[data-nav-region]") ?? null;
	if (t.delegated || i?.dataset.navMode === "delegated") {
		if (t.onKey?.(e) || O()) return;
		let n = A(e);
		if (!n) return;
		if (t.onCommand?.(n)) {
			j(e);
			return;
		}
		n.type === "back" && (j(e), t.onEscape());
		return;
	}
	if (O()) return;
	let a = A(e);
	a && m(e, a, t, i, r, n);
}
function m(e, t, n, r, i, a) {
	switch (t.type) {
		case "back":
			_(e, n, r, i, a);
			return;
		case "bumper":
			v(e, n, t.dir);
			return;
		case "edge":
			g(e, n, r, t.to, a);
			return;
		case "move":
			h(e, n, r, t.dir, i, a);
			return;
		case "activate": return;
	}
}
function h(e, t, n, r, i, a) {
	if (!n) {
		let n = document.activeElement;
		(!n || n === document.body) && ee(e, t, i, a);
		return;
	}
	let o = C(n), c = document.activeElement, l = o.indexOf(c), u = l === -1 ? null : s(w(o), String(l), r, T(n));
	if (u !== null) {
		b(o[Number(u)], n, t, a, "hover"), j(e);
		return;
	}
	let d = te(n, r, E(c), i);
	d && (y(d, E(c), t, a, "select"), j(e));
}
function g(e, t, n, r, i) {
	if (!n) return;
	let a = C(n);
	a.length !== 0 && (b(r === "first" ? a[0] : a[a.length - 1], n, t, i, "hover"), j(e));
}
function _(e, t, n, r, i) {
	j(e);
	let a = r.querySelector("[data-nav-region=\"rail\"]");
	if (n && a && n !== a) {
		y(a, E(n), t, i, "back");
		return;
	}
	t.onEscape();
}
function v(e, t, n) {
	t.bumpers && (j(e), n === "next" ? t.bumpers.next() : t.bumpers.prev(), t.sfx("select"), requestAnimationFrame(() => {
		(t.root()?.querySelector("[data-nav-region=\"rail\"]"))?.querySelector(S)?.focus();
	}));
}
function ee(e, t, n, r) {
	let i = n.querySelector("[data-nav-region=\"rail\"]") ?? n.querySelector("[data-nav-region]");
	i && (y(i, x, t, r, "select"), j(e));
}
function y(e, t, n, r, i) {
	let a = r.get(e);
	if (a && e.contains(a) && D(a)) {
		b(a, e, n, r, i);
		return;
	}
	let o = e.querySelector(S);
	if (o) {
		b(o, e, n, r, i);
		return;
	}
	let s = C(e);
	if (s.length === 0) return;
	let c = l(w(s), t);
	c !== null && b(s[Number(c)], e, n, r, i);
}
function te(e, t, n, r) {
	let i = Array.from(r.querySelectorAll("[data-nav-region]")).filter((t) => t !== e && D(t)), a = e.getAttribute(`data-nav-edge-${t}`);
	if (a) return i.find((e) => e.dataset.navRegion === a) ?? null;
	let o = c(i.map((e, t) => ({
		id: String(t),
		rect: E(e)
	})), n, t);
	return o === null ? null : i[Number(o)];
}
function b(e, t, n, r, i) {
	e.focus(), r.set(t, e), n.sfx(i), e.scrollIntoView({
		block: "nearest",
		inline: "nearest"
	});
}
var x = {
	left: 0,
	top: 0,
	right: 0,
	bottom: 0
}, S = "[role=\"tab\"][data-selected], [role=\"tab\"][data-state=\"active\"]";
function C(e) {
	return Array.from(e.querySelectorAll(d)).filter((t) => t.closest("[data-nav-region]") === e && t.getAttribute("role") !== "tabpanel" && !t.hasAttribute("data-nav-skip") && D(t));
}
function w(e) {
	return e.map((e, t) => ({
		id: String(t),
		rect: E(e)
	}));
}
function T(e) {
	let t = e.dataset.navLayout;
	return t === "grid" || t === "row" ? t : "column";
}
function E(e) {
	let t = e.getBoundingClientRect();
	return {
		left: t.left,
		top: t.top,
		right: t.right,
		bottom: t.bottom
	};
}
function D(e) {
	let t = e.getBoundingClientRect();
	return t.width > 0 && t.height > 0;
}
function O() {
	let e = document.activeElement;
	return e instanceof HTMLElement ? e.tagName === "INPUT" || e.tagName === "TEXTAREA" || e.tagName === "SELECT" || e.isContentEditable : !1;
}
function k() {
	return !!document.querySelector([
		"[role=\"dialog\"][data-expanded]:not([data-nav-inline])",
		"[role=\"menu\"][data-expanded]",
		"[role=\"listbox\"][data-expanded]",
		"[popover]:popover-open",
		"dialog[open]"
	].join(", "));
}
function A(e) {
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
function j(e) {
	e.preventDefault(), e.stopPropagation();
}
//#endregion
//#region api/piloto/src/lib/audio-gate.ts
var M = ["pointerdown", "keydown"], N = class {
	ctx = null;
	constructor(e, t) {
		this.openContext = e;
		for (let e of M) t.addEventListener(e, () => this.arm(), { capture: !0 });
	}
	ready() {
		let e = this.ctx;
		return e && e.state === "running" ? e : null;
	}
	arm() {
		this.ctx ||= this.openContext(), this.ctx?.state === "suspended" && this.ctx.resume();
	}
}, P = {
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
function F(e, t, n) {
	if (n <= 0) return;
	let r = P[t];
	if (r.shape === "bell") {
		L(e, r, n);
		return;
	}
	I(e, r, n);
}
function I(e, { type: t, from: n, to: r, dur: i, gain: a }, o) {
	let s = e.currentTime, c = e.createOscillator();
	c.type = t, c.frequency.setValueAtTime(n, s), c.frequency.exponentialRampToValueAtTime(r, s + i), c.connect(R(e, a * o, s, .012, i)).connect(e.destination), c.start(s), c.stop(s + i + .02);
}
function L(e, { partials: t, dur: n, gain: r }, i) {
	let a = e.currentTime;
	t.forEach((t, o) => {
		let s = r * i * (o === 0 ? 1 : .45), c = n * (o === 0 ? 1 : .55), l = e.createOscillator();
		l.type = "sine", l.frequency.setValueAtTime(t, a), l.connect(R(e, s, a, .005, c)).connect(e.destination), l.start(a), l.stop(a + c + .02);
	});
}
function R(e, t, n, r, i) {
	let a = e.createGain();
	return a.gain.setValueAtTime(1e-4, n), a.gain.exponentialRampToValueAtTime(Math.max(t, 2e-4), n + r), a.gain.exponentialRampToValueAtTime(1e-4, n + i), a;
}
//#endregion
//#region api/piloto/src/lib/sfx-player.ts
var z = class {
	constructor(e = B()) {
		this.gate = e;
	}
	play(e, t) {
		try {
			let n = this.gate.ready();
			n && F(n, e, t);
		} catch {}
	}
};
function B() {
	return new N(V, window);
}
function V() {
	let e = window.AudioContext ?? window.webkitAudioContext;
	return e ? new e() : null;
}
function H() {
	return new z();
}
//#endregion
//#region api/piloto/src/lib/ui-store.ts
var U = "t20-ui";
function W(e, t = globalThis.localStorage) {
	t?.setItem(U, JSON.stringify({ state: e }));
}
var G = 100;
function K(e) {
	if (e) try {
		return JSON.parse(e).state;
	} catch {
		return;
	}
}
function q(e) {
	return K(e)?.sfx === !0;
}
function J(e) {
	return Y(K(e)?.volume);
}
function Y(e) {
	return typeof e != "number" || Number.isNaN(e) ? G : Math.min(100, Math.max(0, Math.round(e)));
}
//#endregion
//#region api/piloto/src/cena.ts
function X() {
	let e = globalThis.localStorage?.getItem("t20-ui") ?? null;
	return {
		som: q(e),
		volume: J(e)
	};
}
var Z = X(), ne = H();
function Q(e) {
	Z.som && (e === "hover" && window.matchMedia?.("(pointer: coarse)").matches || ne.play(e, Z.volume / 100));
}
function re(e) {
	let t = document.querySelector(`[popovertarget="${e.id}"]`);
	if (!t) return;
	let n = t.getBoundingClientRect(), r = n.top > window.innerHeight / 2;
	e.style.top = r ? "auto" : `${Math.round(n.bottom + 8)}px`, e.style.bottom = r ? `${Math.round(window.innerHeight - n.top + 8)}px` : "auto";
	let i = e.offsetWidth || 224;
	e.style.left = `${Math.round(Math.max(8, Math.min(n.left, window.innerWidth - i - 8)))}px`;
}
window.cena = {
	som() {
		return Z.som = !Z.som, W({
			sfx: Z.som,
			volume: Z.volume
		}), Z.som && Q("select"), Z.som;
	},
	volume(e) {
		return Z.volume = Math.min(100, Math.max(0, Math.round(e))), W({
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
	cue: Q
};
var $ = () => document.querySelector("[data-slot=\"scene-shell\"]");
f({
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
}), document.addEventListener("beforetoggle", (e) => {
	let t = e.target;
	t?.matches?.("[popover]") && e.newState === "open" && re(t);
}, !0);
//#endregion
