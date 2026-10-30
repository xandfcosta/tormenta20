//#region serve/web/assets/src/lib/fetch-busy.ts
var e = "button, a, input, select, textarea, [role=\"switch\"]";
function t(t) {
	let n = /* @__PURE__ */ new WeakMap();
	t.addEventListener("datastar-fetch", (t) => {
		let { type: r, el: i } = t.detail;
		if (!(i instanceof Element) || !i.matches(e)) return;
		let a = n.get(i) ?? 0;
		if (r === "started") {
			n.set(i, a + 1), i.setAttribute("aria-busy", "true");
			return;
		}
		if (r !== "finished") return;
		let o = Math.max(0, a - 1);
		n.set(i, o), o === 0 && i.removeAttribute("aria-busy");
	});
}
//#endregion
//#region serve/web/assets/src/lib/spatial-nav.ts
var n = 2, r = {
	left: !0,
	right: !0,
	up: !1,
	down: !1
};
function i(e) {
	return (e.left + e.right) / 2;
}
function a(e) {
	return (e.top + e.bottom) / 2;
}
function o(e, t) {
	return t === "grid" ? !0 : t === "row" ? r[e] : !r[e];
}
function s(e, t, n) {
	switch (n) {
		case "right": return i(t) > i(e) + .5;
		case "left": return i(t) < i(e) - .5;
		case "down": return a(t) > a(e) + .5;
		case "up": return a(t) < a(e) - .5;
	}
}
function c(e, t, o) {
	let s = Math.abs(i(t) - i(e)), c = Math.abs(a(t) - a(e));
	return (r[o] ? s : c) + n * (r[o] ? c : s);
}
function l(e, t, n, r) {
	if (!o(n, r)) return null;
	let i = e.find((e) => e.id === t);
	if (!i) return null;
	let a = null, l = Infinity;
	for (let r of e) {
		if (r.id === t || !s(i.rect, r.rect, n)) continue;
		let e = c(i.rect, r.rect, n);
		e < l && (l = e, a = r.id);
	}
	return a;
}
function u(e, t, n) {
	let r = null, i = Infinity;
	for (let a of e) {
		if (!s(t, a.rect, n)) continue;
		let e = c(t, a.rect, n);
		e < i && (i = e, r = a.id);
	}
	return r;
}
function d(e, t) {
	let n = null, r = Infinity;
	for (let o of e) {
		let e = i(o.rect) - i(t), s = a(o.rect) - a(t), c = e * e + s * s;
		c < r && (r = c, n = o.id);
	}
	return n;
}
//#endregion
//#region serve/web/assets/src/lib/scene-nav.ts
var f = "(min-width: 1280px) and (pointer: fine)", p = "a[href], button:not([disabled]), [tabindex]:not([tabindex=\"-1\"]), [data-nav-item]";
function m(e) {
	let t = /* @__PURE__ */ new WeakMap(), n = (n) => h(n, e, t), r = typeof window.matchMedia == "function" ? window.matchMedia(f) : null, i = !1, a = () => {
		let t = (r?.matches ?? !1) && e.active?.() !== !1;
		t !== i && (i = t, t ? window.addEventListener("keydown", n, !0) : window.removeEventListener("keydown", n, !0));
	};
	return a(), r?.addEventListener("change", a), () => {
		r?.removeEventListener("change", a), i && window.removeEventListener("keydown", n, !0), i = !1;
	};
}
function h(e, t, n) {
	if (e.ctrlKey || e.metaKey || e.altKey || A()) return;
	let r = t.root();
	if (!r) return;
	let i = document.activeElement?.closest("[data-nav-region]") ?? null;
	if (t.delegated || i?.dataset.navMode === "delegated") {
		if (t.onKey?.(e) || k()) return;
		let n = j(e);
		if (!n) return;
		if (t.onCommand?.(n)) {
			M(e);
			return;
		}
		n.type === "back" && (M(e), t.onEscape());
		return;
	}
	if (k()) return;
	let a = j(e);
	a && g(e, a, t, i, r, n);
}
function g(e, t, n, r, i, a) {
	switch (t.type) {
		case "back":
			ee(e, n, r, i, a);
			return;
		case "bumper":
			te(e, n, t.dir);
			return;
		case "edge":
			v(e, n, r, t.to, a);
			return;
		case "move":
			_(e, n, r, t.dir, i, a);
			return;
		case "activate": return;
	}
}
function _(e, t, n, r, i, a) {
	if (!n) {
		let n = document.activeElement;
		(!n || n === document.body) && y(e, t, i, a);
		return;
	}
	let o = T(n), s = document.activeElement, c = o.indexOf(s), u = c === -1 ? null : l(E(o), String(c), r, ne(n));
	if (u !== null) {
		S(o[Number(u)], n, t, a, "hover"), M(e);
		return;
	}
	let d = x(n, r, D(s), i);
	d && (b(d, D(s), t, a, "select"), M(e));
}
function v(e, t, n, r, i) {
	if (!n) return;
	let a = T(n);
	a.length !== 0 && (S(r === "first" ? a[0] : a[a.length - 1], n, t, i, "hover"), M(e));
}
function ee(e, t, n, r, i) {
	M(e);
	let a = r.querySelector("[data-nav-region=\"rail\"]");
	if (n && a && n !== a) {
		b(a, D(n), t, i, "back");
		return;
	}
	t.onEscape();
}
function te(e, t, n) {
	t.bumpers && (M(e), n === "next" ? t.bumpers.next() : t.bumpers.prev(), t.sfx("select"), requestAnimationFrame(() => {
		(t.root()?.querySelector("[data-nav-region=\"rail\"]"))?.querySelector(w)?.focus();
	}));
}
function y(e, t, n, r) {
	let i = n.querySelector("[data-nav-region=\"rail\"]") ?? n.querySelector("[data-nav-region]");
	i && (b(i, C, t, r, "select"), M(e));
}
function b(e, t, n, r, i) {
	let a = r.get(e);
	if (a && e.contains(a) && O(a)) {
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
	let c = d(E(s), t);
	c !== null && S(s[Number(c)], e, n, r, i);
}
function x(e, t, n, r) {
	let i = Array.from(r.querySelectorAll("[data-nav-region]")).filter((t) => t !== e && O(t)), a = e.getAttribute(`data-nav-edge-${t}`);
	if (a) return i.find((e) => e.dataset.navRegion === a) ?? null;
	let o = u(i.map((e, t) => ({
		id: String(t),
		rect: D(e)
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
	return Array.from(e.querySelectorAll(p)).filter((t) => t.closest("[data-nav-region]") === e && t.getAttribute("role") !== "tabpanel" && !t.hasAttribute("data-nav-skip") && O(t));
}
function E(e) {
	return e.map((e, t) => ({
		id: String(t),
		rect: D(e)
	}));
}
function ne(e) {
	let t = e.dataset.navLayout;
	return t === "grid" || t === "row" ? t : "column";
}
function D(e) {
	let t = e.getBoundingClientRect();
	return {
		left: t.left,
		top: t.top,
		right: t.right,
		bottom: t.bottom
	};
}
function O(e) {
	let t = e.getBoundingClientRect();
	return t.width > 0 && t.height > 0;
}
function k() {
	let e = document.activeElement;
	return e instanceof HTMLElement ? e.tagName === "INPUT" || e.tagName === "TEXTAREA" || e.tagName === "SELECT" || e.isContentEditable : !1;
}
function A() {
	return !!document.querySelector([
		"[role=\"dialog\"][data-expanded]:not([data-nav-inline])",
		"[role=\"menu\"][data-expanded]",
		"[role=\"listbox\"][data-expanded]",
		"[popover]:popover-open",
		"dialog[open]"
	].join(", "));
}
function j(e) {
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
function M(e) {
	e.preventDefault(), e.stopPropagation();
}
//#endregion
//#region serve/web/assets/src/lib/audio-gate.ts
var N = ["pointerdown", "keydown"], P = class {
	ctx = null;
	constructor(e, t) {
		this.openContext = e;
		for (let e of N) t.addEventListener(e, () => this.arm(), { capture: !0 });
	}
	ready() {
		let e = this.ctx;
		return e && e.state === "running" ? e : null;
	}
	arm() {
		this.ctx ||= this.openContext(), this.ctx?.state === "suspended" && this.ctx.resume();
	}
}, F = {
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
function I(e, t, n) {
	if (n <= 0) return;
	let r = F[t];
	if (r.shape === "bell") {
		R(e, r, n);
		return;
	}
	L(e, r, n);
}
function L(e, { type: t, from: n, to: r, dur: i, gain: a }, o) {
	let s = e.currentTime, c = e.createOscillator();
	c.type = t, c.frequency.setValueAtTime(n, s), c.frequency.exponentialRampToValueAtTime(r, s + i), c.connect(z(e, a * o, s, .012, i)).connect(e.destination), c.start(s), c.stop(s + i + .02);
}
function R(e, { partials: t, dur: n, gain: r }, i) {
	let a = e.currentTime;
	t.forEach((t, o) => {
		let s = r * i * (o === 0 ? 1 : .45), c = n * (o === 0 ? 1 : .55), l = e.createOscillator();
		l.type = "sine", l.frequency.setValueAtTime(t, a), l.connect(z(e, s, a, .005, c)).connect(e.destination), l.start(a), l.stop(a + c + .02);
	});
}
function z(e, t, n, r, i) {
	let a = e.createGain();
	return a.gain.setValueAtTime(1e-4, n), a.gain.exponentialRampToValueAtTime(Math.max(t, 2e-4), n + r), a.gain.exponentialRampToValueAtTime(1e-4, n + i), a;
}
//#endregion
//#region serve/web/assets/src/lib/sfx-player.ts
var B = class {
	constructor(e = V()) {
		this.gate = e;
	}
	play(e, t) {
		try {
			let n = this.gate.ready();
			n && I(n, e, t);
		} catch {}
	}
};
function V() {
	return new P(H, window);
}
function H() {
	let e = window.AudioContext ?? window.webkitAudioContext;
	return e ? new e() : null;
}
function U() {
	return new B();
}
//#endregion
//#region serve/web/assets/src/lib/ui-store.ts
var W = "t20-ui";
function G(e, t = globalThis.localStorage) {
	t?.setItem(W, JSON.stringify({ state: e }));
}
var K = 100;
function q(e) {
	if (e) try {
		return JSON.parse(e).state;
	} catch {
		return;
	}
}
function J(e) {
	return q(e)?.sfx === !0;
}
function Y(e) {
	return X(q(e)?.volume);
}
function X(e) {
	return typeof e != "number" || Number.isNaN(e) ? K : Math.min(100, Math.max(0, Math.round(e)));
}
//#endregion
//#region serve/web/assets/src/scene.ts
function re() {
	let e = globalThis.localStorage?.getItem("t20-ui") ?? null;
	return {
		som: J(e),
		volume: Y(e)
	};
}
var Z = re(), ie = U();
function Q(e) {
	Z.som && (e === "hover" && window.matchMedia?.("(pointer: coarse)").matches || ie.play(e, Z.volume / 100));
}
function ae(e) {
	let t = document.querySelector(`[popovertarget="${e.id}"]`);
	if (!t) return;
	let n = t.getBoundingClientRect(), r = n.top - 16, i = window.innerHeight - n.bottom - 16, a = r > i;
	e.style.top = a ? "auto" : `${Math.round(n.bottom + 8)}px`, e.style.bottom = a ? `${Math.round(window.innerHeight - n.top + 8)}px` : "auto", e.style.maxHeight = `${Math.max(96, Math.round(a ? r : i))}px`;
	let o = e.offsetWidth || 224;
	e.style.left = `${Math.round(Math.max(8, Math.min(n.left, window.innerWidth - o - 8)))}px`;
}
window.cena = {
	som() {
		return Z.som = !Z.som, G({
			sfx: Z.som,
			volume: Z.volume
		}), Z.som && Q("select"), Z.som;
	},
	volume(e) {
		return Z.volume = Math.min(100, Math.max(0, Math.round(e))), G({
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
}, t(document), window.addEventListener("popstate", (e) => {
	e.state?.softNav && window.location.reload();
});
var $ = () => document.querySelector("[data-slot=\"scene-shell\"]");
m({
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
	t?.matches?.("[popover]") && e.newState === "open" && ae(t);
}, !0);
//#endregion
