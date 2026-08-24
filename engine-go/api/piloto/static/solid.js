//#region ../node_modules/.pnpm/solid-js@1.9.14/node_modules/solid-js/dist/solid.js
var e = {
	context: void 0,
	registry: void 0,
	effects: void 0,
	done: !1,
	getContextId() {
		return t(this.context.count);
	},
	getNextContextId() {
		return t(this.context.count++);
	}
};
function t(t) {
	let n = String(t), r = n.length - 1;
	return e.context.id + (r ? String.fromCharCode(96 + r) : "") + n;
}
function n(t) {
	e.context = t;
}
function r() {
	return {
		...e.context,
		id: e.getNextContextId(),
		count: 0
	};
}
var i = (e, t) => e === t, a = Symbol("solid-proxy"), o = typeof Proxy == "function", s = { equals: i }, c = null, l = R, u = 1, d = 2, f = {
	owned: null,
	cleanups: null,
	context: null,
	owner: null
}, p = null, m = null, h = null, g = null, _ = null, v = null, y = null, b = 0;
function x(e, t) {
	let n = _, r = p, i = e.length === 0, a = t === void 0 ? r : t, o = i ? f : {
		owned: null,
		cleanups: null,
		context: a ? a.context : null,
		owner: a
	}, s = i ? e : () => e(() => T(() => H(o)));
	p = o, _ = null;
	try {
		return I(s, !0);
	} finally {
		_ = n, p = r;
	}
}
function S(e, t) {
	t = t ? Object.assign({}, s, t) : s;
	let n = {
		value: e,
		observers: null,
		observerSlots: null,
		comparator: t.equals || void 0
	};
	return [A.bind(n), (e) => (typeof e == "function" && (e = m && m.running && m.sources.has(n) ? e(n.tValue) : e(n.value)), j(n, e))];
}
function C(e, t, n) {
	let r = P(e, t, !1, u);
	h && m && m.running ? v.push(r) : M(r);
}
function w(e, t, n) {
	n = n ? Object.assign({}, s, n) : s;
	let r = P(e, t, !0, 0);
	return r.observers = null, r.observerSlots = null, r.comparator = n.equals || void 0, h && m && m.running ? (r.tState = u, v.push(r)) : M(r), A.bind(r);
}
function T(e) {
	if (!g && _ === null) return e();
	let t = _;
	_ = null;
	try {
		return g ? g.untrack(e) : e();
	} finally {
		_ = t;
	}
}
function E(e) {
	return p === null || (p.cleanups === null ? p.cleanups = [e] : p.cleanups.push(e)), e;
}
function D(e) {
	if (m && m.running) return e(), m.done;
	let t = _, n = p;
	return Promise.resolve().then(() => {
		_ = t, p = n;
		let r;
		return (h || k) && (r = m ||= {
			sources: /* @__PURE__ */ new Set(),
			effects: [],
			promises: /* @__PURE__ */ new Set(),
			disposed: /* @__PURE__ */ new Set(),
			queue: /* @__PURE__ */ new Set(),
			running: !0
		}, r.done ||= new Promise((e) => r.resolve = e), r.running = !0), I(e, !1), _ = p = null, r ? r.done : void 0;
	});
}
var [ee, O] = /* @__PURE__ */ S(!1), k;
function A() {
	let e = m && m.running;
	if (this.sources && (e ? this.tState : this.state)) if ((e ? this.tState : this.state) === u) M(this);
	else {
		let e = v;
		v = null, I(() => B(this), !1), v = e;
	}
	if (_) {
		let e = this.observers;
		if (!e || e[e.length - 1] !== _) {
			let t = e ? e.length : 0;
			_.sources ? (_.sources.push(this), _.sourceSlots.push(t)) : (_.sources = [this], _.sourceSlots = [t]), e ? (e.push(_), this.observerSlots.push(_.sources.length - 1)) : (this.observers = [_], this.observerSlots = [_.sources.length - 1]);
		}
	}
	return e && m.sources.has(this) ? this.tValue : this.value;
}
function j(e, t, n) {
	let r = m && m.running && m.sources.has(e) ? e.tValue : e.value;
	if (!e.comparator || !e.comparator(r, t)) {
		if (m) {
			let r = m.running;
			(r || !n && m.sources.has(e)) && (m.sources.add(e), e.tValue = t), r || (e.value = t);
		} else e.value = t;
		e.observers && e.observers.length && I(() => {
			for (let t = 0; t < e.observers.length; t += 1) {
				let n = e.observers[t], r = m && m.running;
				r && m.disposed.has(n) || ((r ? !n.tState : !n.state) && (n.pure ? v.push(n) : y.push(n), n.observers && V(n)), r ? n.tState = u : n.state = u);
			}
			if (v.length > 1e6) throw v = [], Error();
		}, !1);
	}
	return t;
}
function M(e) {
	if (!e.fn) return;
	H(e);
	let t = b;
	N(e, m && m.running && m.sources.has(e) ? e.tValue : e.value, t), m && !m.running && m.sources.has(e) && queueMicrotask(() => {
		I(() => {
			m && (m.running = !0), _ = p = e, N(e, e.tValue, t), _ = p = null;
		}, !1);
	});
}
function N(e, t, n) {
	let r, i = p, a = _;
	_ = p = e;
	try {
		r = e.fn(t);
	} catch (t) {
		return e.pure && (m && m.running ? (e.tState = u, e.tOwned && e.tOwned.forEach(H), e.tOwned = void 0) : (e.state = u, e.owned && e.owned.forEach(H), e.owned = null)), e.updatedAt = n + 1, K(t);
	} finally {
		_ = a, p = i;
	}
	(!e.updatedAt || e.updatedAt <= n) && (e.updatedAt != null && "observers" in e ? j(e, r, !0) : m && m.running && e.pure ? (m.sources.has(e) || (e.value = r), m.sources.add(e), e.tValue = r) : e.value = r, e.updatedAt = n);
}
function P(e, t, n, r = u, i) {
	let a = {
		fn: e,
		state: r,
		updatedAt: null,
		owned: null,
		sources: null,
		sourceSlots: null,
		cleanups: null,
		value: t,
		owner: p,
		context: p ? p.context : null,
		pure: n
	};
	if (m && m.running && (a.state = 0, a.tState = r), p === null || p !== f && (m && m.running && p.pure ? p.tOwned ? p.tOwned.push(a) : p.tOwned = [a] : p.owned ? p.owned.push(a) : p.owned = [a]), g && a.fn) {
		let e = a.fn, [t, n] = S(void 0, { equals: !1 }), r = g.factory(e, n);
		E(() => r.dispose());
		let i, o = () => D(n).then(() => {
			i &&= (i.dispose(), void 0);
		});
		a.fn = (n) => (t(), m && m.running ? (i ||= g.factory(e, o), i.track(n)) : r.track(n));
	}
	return a;
}
function F(e) {
	let t = m && m.running;
	if ((t ? e.tState : e.state) === 0) return;
	if ((t ? e.tState : e.state) === d) return B(e);
	if (e.suspense && T(e.suspense.inFallback)) return e.suspense.effects.push(e);
	let n = [e];
	for (; (e = e.owner) && (!e.updatedAt || e.updatedAt < b);) {
		if (t && m.disposed.has(e)) return;
		(t ? e.tState : e.state) && n.push(e);
	}
	for (let r = n.length - 1; r >= 0; r--) {
		if (e = n[r], t) {
			let t = e, i = n[r + 1];
			for (; (t = t.owner) && t !== i;) if (m.disposed.has(t)) return;
		}
		if ((t ? e.tState : e.state) === u) M(e);
		else if ((t ? e.tState : e.state) === d) {
			let t = v;
			v = null, I(() => B(e, n[0]), !1), v = t;
		}
	}
}
function I(e, t) {
	if (v) return e();
	let n = !1;
	t || (v = []), y ? n = !0 : y = [], b++;
	try {
		let t = e();
		return L(n), t;
	} catch (e) {
		n || (y = null), v = null, K(e);
	}
}
function L(e) {
	if (v &&= (h && m && m.running ? z(v) : R(v), null), e) return;
	let t;
	if (m) {
		if (!m.promises.size && !m.queue.size) {
			let e = m.sources, n = m.disposed;
			y.push.apply(y, m.effects), t = m.resolve;
			for (let e of y) "tState" in e && (e.state = e.tState), delete e.tState;
			m = null, I(() => {
				for (let e of n) H(e);
				for (let t of e) {
					if (t.value = t.tValue, t.owned) for (let e = 0, n = t.owned.length; e < n; e++) H(t.owned[e]);
					t.tOwned && (t.owned = t.tOwned), delete t.tValue, delete t.tOwned, t.tState = 0;
				}
				O(!1);
			}, !1);
		} else if (m.running) {
			m.running = !1, m.effects.push.apply(m.effects, y), y = null, O(!0);
			return;
		}
	}
	let n = y;
	y = null, n.length && I(() => l(n), !1), t && t();
}
function R(e) {
	for (let t = 0; t < e.length; t++) F(e[t]);
}
function z(e) {
	for (let t = 0; t < e.length; t++) {
		let n = e[t], r = m.queue;
		r.has(n) || (r.add(n), h(() => {
			r.delete(n), I(() => {
				m.running = !0, F(n);
			}, !1), m && (m.running = !1);
		}));
	}
}
function B(e, t) {
	let n = m && m.running;
	n ? e.tState = 0 : e.state = 0;
	for (let r = 0; r < e.sources.length; r += 1) {
		let i = e.sources[r];
		if (i.sources) {
			let e = n ? i.tState : i.state;
			e === u ? i !== t && (!i.updatedAt || i.updatedAt < b) && F(i) : e === d && B(i, t);
		}
	}
}
function V(e) {
	let t = m && m.running;
	for (let n = 0; n < e.observers.length; n += 1) {
		let r = e.observers[n];
		(t ? !r.tState : !r.state) && (t ? r.tState = d : r.state = d, r.pure ? v.push(r) : y.push(r), r.observers && V(r));
	}
}
function H(e) {
	let t;
	if (e.sources) for (; e.sources.length;) {
		let t = e.sources.pop(), n = e.sourceSlots.pop(), r = t.observers;
		if (r && r.length) {
			let e = r.pop(), i = t.observerSlots.pop();
			n < r.length && (e.sourceSlots[i] = n, r[n] = e, t.observerSlots[n] = i);
		}
	}
	if (e.tOwned) {
		for (t = e.tOwned.length - 1; t >= 0; t--) H(e.tOwned[t]);
		delete e.tOwned;
	}
	if (m && m.running && e.pure) U(e, !0);
	else if (e.owned) {
		for (t = e.owned.length - 1; t >= 0; t--) H(e.owned[t]);
		e.owned = null;
	}
	if (e.cleanups) {
		for (t = e.cleanups.length - 1; t >= 0; t--) e.cleanups[t]();
		e.cleanups = null;
	}
	m && m.running ? e.tState = 0 : e.state = 0;
}
function U(e, t) {
	if (t || (e.tState = 0, m.disposed.add(e)), e.owned) for (let t = 0; t < e.owned.length; t++) U(e.owned[t]);
}
function W(e) {
	return e instanceof Error ? e : Error(typeof e == "string" ? e : "Unknown error", { cause: e });
}
function G(e, t, n) {
	try {
		for (let n of t) n(e);
	} catch (e) {
		K(e, n && n.owner || null);
	}
}
function K(e, t = p) {
	let n = c && t && t.context && t.context[c], r = W(e);
	if (!n) throw r;
	y ? y.push({
		fn() {
			G(r, n, t);
		},
		state: u
	}) : G(r, n, t);
}
var q = !1;
function J(t, i) {
	if (q && e.context) {
		let a = e.context;
		n(r());
		let o = T(() => t(i || {}));
		return n(a), o;
	}
	return T(() => t(i || {}));
}
function Y() {
	return !0;
}
var X = {
	get(e, t, n) {
		return t === a ? n : e.get(t);
	},
	has(e, t) {
		return t === a ? !0 : e.has(t);
	},
	set: Y,
	deleteProperty: Y,
	getOwnPropertyDescriptor(e, t) {
		return {
			configurable: !0,
			enumerable: !0,
			get() {
				return e.get(t);
			},
			set: Y,
			deleteProperty: Y
		};
	},
	ownKeys(e) {
		return e.keys();
	}
};
function Z(e) {
	return (e = typeof e == "function" ? e() : e) ? e : {};
}
function Q() {
	for (let e = 0, t = this.length; e < t; ++e) {
		let t = this[e]();
		if (t !== void 0) return t;
	}
}
function $(...e) {
	let t = !1;
	for (let n = 0; n < e.length; n++) {
		let r = e[n];
		t ||= !!r && a in r, e[n] = typeof r == "function" ? (t = !0, w(r)) : r;
	}
	if (o && t) return new Proxy({
		get(t) {
			for (let n = e.length - 1; n >= 0; n--) {
				let r = Z(e[n])[t];
				if (r !== void 0) return r;
			}
		},
		has(t) {
			for (let n = e.length - 1; n >= 0; n--) if (t in Z(e[n])) return !0;
			return !1;
		},
		keys() {
			let t = [];
			for (let n = 0; n < e.length; n++) t.push(...Object.keys(Z(e[n])));
			return [...new Set(t)];
		}
	}, X);
	let n = {}, r = Object.create(null);
	for (let t = e.length - 1; t >= 0; t--) {
		let i = e[t];
		if (!i) continue;
		let a = Object.getOwnPropertyNames(i);
		for (let e = a.length - 1; e >= 0; e--) {
			let t = a[e];
			if (t === "__proto__" || t === "constructor") continue;
			let o = Object.getOwnPropertyDescriptor(i, t);
			if (!r[t]) r[t] = o.get ? {
				enumerable: !0,
				configurable: !0,
				get: Q.bind(n[t] = [o.get.bind(i)])
			} : o.value === void 0 ? void 0 : o;
			else {
				let e = n[t];
				e && (o.get ? e.push(o.get.bind(i)) : o.value !== void 0 && e.push(() => o.value));
			}
		}
	}
	let i = {}, s = Object.keys(r);
	for (let e = s.length - 1; e >= 0; e--) {
		let t = s[e], n = r[t];
		n && n.get ? Object.defineProperty(i, t, n) : i[t] = n ? n.value : void 0;
	}
	return i;
}
function te(e, ...t) {
	let n = t.length;
	if (o && a in e) {
		let r = n > 1 ? t.flat() : t[0], i = t.map((t) => new Proxy({
			get(n) {
				return t.includes(n) ? e[n] : void 0;
			},
			has(n) {
				return t.includes(n) && n in e;
			},
			keys() {
				return t.filter((t) => t in e);
			}
		}, X));
		return i.push(new Proxy({
			get(t) {
				return r.includes(t) ? void 0 : e[t];
			},
			has(t) {
				return r.includes(t) ? !1 : t in e;
			},
			keys() {
				return Object.keys(e).filter((e) => !r.includes(e));
			}
		}, X)), i;
	}
	let r = [];
	for (let e = 0; e <= n; e++) r[e] = {};
	for (let i of Object.getOwnPropertyNames(e)) {
		let a = n;
		for (let e = 0; e < t.length; e++) if (t[e].includes(i)) {
			a = e;
			break;
		}
		let o = Object.getOwnPropertyDescriptor(e, i);
		!o.get && !o.set && o.enumerable && o.writable && o.configurable ? r[a][i] = o.value : Object.defineProperty(r[a], i, o);
	}
	return r;
}
//#endregion
export { S as a, te as c, x as i, T as l, w as n, $ as o, C as r, e as s, J as t };
