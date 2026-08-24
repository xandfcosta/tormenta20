import { a as e, c as t, i as n, l as r, n as i, o as a, r as o, s, t as c } from "./solid.js";
//#region ../node_modules/.pnpm/solid-js@1.9.14/node_modules/solid-js/web/dist/web.js
var l = /* @__PURE__ */ new Set([
	"className",
	"value",
	"readOnly",
	"noValidate",
	"formNoValidate",
	"isMap",
	"noModule",
	"playsInline",
	"adAuctionHeaders",
	"allowFullscreen",
	"browsingTopics",
	"defaultChecked",
	"defaultMuted",
	"defaultSelected",
	"disablePictureInPicture",
	"disableRemotePlayback",
	"preservesPitch",
	"shadowRootClonable",
	"shadowRootCustomElementRegistry",
	"shadowRootDelegatesFocus",
	"shadowRootSerializable",
	"sharedStorageWritable",
	.../* @__PURE__ */ "allowfullscreen.async.alpha.autofocus.autoplay.checked.controls.default.disabled.formnovalidate.hidden.indeterminate.inert.ismap.loop.multiple.muted.nomodule.novalidate.open.playsinline.readonly.required.reversed.seamless.selected.adauctionheaders.browsingtopics.credentialless.defaultchecked.defaultmuted.defaultselected.defer.disablepictureinpicture.disableremoteplayback.preservespitch.shadowrootclonable.shadowrootcustomelementregistry.shadowrootdelegatesfocus.shadowrootserializable.sharedstoragewritable".split(".")
]), u = /* @__PURE__ */ new Set([
	"innerHTML",
	"textContent",
	"innerText",
	"children"
]), d = /* @__PURE__ */ Object.assign(Object.create(null), {
	className: "class",
	htmlFor: "for"
}), f = /* @__PURE__ */ Object.assign(Object.create(null), {
	class: "className",
	novalidate: {
		$: "noValidate",
		FORM: 1
	},
	formnovalidate: {
		$: "formNoValidate",
		BUTTON: 1,
		INPUT: 1
	},
	ismap: {
		$: "isMap",
		IMG: 1
	},
	nomodule: {
		$: "noModule",
		SCRIPT: 1
	},
	playsinline: {
		$: "playsInline",
		VIDEO: 1
	},
	readonly: {
		$: "readOnly",
		INPUT: 1,
		TEXTAREA: 1
	},
	adauctionheaders: {
		$: "adAuctionHeaders",
		IFRAME: 1
	},
	allowfullscreen: {
		$: "allowFullscreen",
		IFRAME: 1
	},
	browsingtopics: {
		$: "browsingTopics",
		IMG: 1
	},
	defaultchecked: {
		$: "defaultChecked",
		INPUT: 1
	},
	defaultmuted: {
		$: "defaultMuted",
		AUDIO: 1,
		VIDEO: 1
	},
	defaultselected: {
		$: "defaultSelected",
		OPTION: 1
	},
	disablepictureinpicture: {
		$: "disablePictureInPicture",
		VIDEO: 1
	},
	disableremoteplayback: {
		$: "disableRemotePlayback",
		AUDIO: 1,
		VIDEO: 1
	},
	preservespitch: {
		$: "preservesPitch",
		AUDIO: 1,
		VIDEO: 1
	},
	shadowrootclonable: {
		$: "shadowRootClonable",
		TEMPLATE: 1
	},
	shadowrootdelegatesfocus: {
		$: "shadowRootDelegatesFocus",
		TEMPLATE: 1
	},
	shadowrootserializable: {
		$: "shadowRootSerializable",
		TEMPLATE: 1
	},
	sharedstoragewritable: {
		$: "sharedStorageWritable",
		IFRAME: 1,
		IMG: 1
	}
});
function ee(e, t) {
	let n = f[e];
	return typeof n == "object" ? n[t] ? n.$ : void 0 : n;
}
var p = /* @__PURE__ */ new Set([
	"beforeinput",
	"click",
	"dblclick",
	"contextmenu",
	"focusin",
	"focusout",
	"input",
	"keydown",
	"keyup",
	"mousedown",
	"mousemove",
	"mouseout",
	"mouseover",
	"mouseup",
	"pointerdown",
	"pointermove",
	"pointerout",
	"pointerover",
	"pointerup",
	"touchend",
	"touchmove",
	"touchstart"
]), m = /* @__PURE__ */ new Set(/* @__PURE__ */ "altGlyph.altGlyphDef.altGlyphItem.animate.animateColor.animateMotion.animateTransform.circle.clipPath.color-profile.cursor.defs.desc.ellipse.feBlend.feColorMatrix.feComponentTransfer.feComposite.feConvolveMatrix.feDiffuseLighting.feDisplacementMap.feDistantLight.feDropShadow.feFlood.feFuncA.feFuncB.feFuncG.feFuncR.feGaussianBlur.feImage.feMerge.feMergeNode.feMorphology.feOffset.fePointLight.feSpecularLighting.feSpotLight.feTile.feTurbulence.filter.font.font-face.font-face-format.font-face-name.font-face-src.font-face-uri.foreignObject.g.glyph.glyphRef.hkern.image.line.linearGradient.marker.mask.metadata.missing-glyph.mpath.path.pattern.polygon.polyline.radialGradient.rect.set.stop.svg.switch.symbol.text.textPath.tref.tspan.use.view.vkern".split(".")), h = {
	xlink: "http://www.w3.org/1999/xlink",
	xml: "http://www.w3.org/XML/1998/namespace"
};
function g(e, t, n) {
	let r = n.length, i = t.length, a = r, o = 0, s = 0, c = t[i - 1].nextSibling, l = null;
	for (; o < i || s < a;) {
		if (t[o] === n[s]) {
			o++, s++;
			continue;
		}
		for (; t[i - 1] === n[a - 1];) i--, a--;
		if (i === o) {
			let t = a < r ? s ? n[s - 1].nextSibling : n[a - s] : c;
			for (; s < a;) e.insertBefore(n[s++], t);
		} else if (a === s) for (; o < i;) (!l || !l.has(t[o])) && t[o].remove(), o++;
		else if (t[o] === n[a - 1] && n[s] === t[i - 1]) {
			let r = t[--i].nextSibling;
			e.insertBefore(n[s++], t[o++].nextSibling), e.insertBefore(n[--a], r), t[i] = n[a];
		} else {
			if (!l) {
				l = /* @__PURE__ */ new Map();
				let e = s;
				for (; e < a;) l.set(n[e], e++);
			}
			let r = l.get(t[o]);
			if (r != null) if (s < r && r < a) {
				let c = o, u = 1, d;
				for (; ++c < i && c < a && !((d = l.get(t[c])) == null || d !== r + u);) u++;
				if (u > r - s) {
					let i = t[o];
					for (; s < r;) e.insertBefore(n[s++], i);
				} else e.replaceChild(n[s++], t[o++]);
			} else o++;
			else t[o++].remove();
		}
	}
}
var _ = "_$DX_DELEGATE";
function v(e, t, n, i) {
	let a, o = () => {
		let t = i ? document.createElementNS("http://www.w3.org/1998/Math/MathML", "template") : document.createElement("template");
		return t.innerHTML = e, n ? t.content.firstChild.firstChild : i ? t.firstChild : t.content.firstChild;
	}, s = t ? () => r(() => document.importNode(a ||= o(), !0)) : () => (a ||= o()).cloneNode(!0);
	return s.cloneNode = s, s;
}
function te(e, t = window.document) {
	let n = t[_] || (t[_] = /* @__PURE__ */ new Set());
	for (let r = 0, i = e.length; r < i; r++) {
		let i = e[r];
		n.has(i) || (n.add(i), t.addEventListener(i, le));
	}
}
function y(e, t, n) {
	D(e) || (n == null ? e.removeAttribute(t) : e.setAttribute(t, n));
}
function ne(e, t, n, r) {
	D(e) || (r == null ? e.removeAttributeNS(t, n) : e.setAttributeNS(t, n, r));
}
function re(e, t, n) {
	D(e) || (n ? e.setAttribute(t, "") : e.removeAttribute(t));
}
function b(e, t) {
	D(e) || (t == null ? e.removeAttribute("class") : e.className = t);
}
function x(e, t, n, r) {
	if (r) Array.isArray(n) ? (e[`$$${t}`] = n[0], e[`$$${t}Data`] = n[1]) : e[`$$${t}`] = n;
	else if (Array.isArray(n)) {
		let r = n[0];
		e.addEventListener(t, n[0] = (t) => r.call(e, n[1], t));
	} else e.addEventListener(t, n, typeof n != "function" && n);
}
function ie(e, t, n = {}) {
	let r = Object.keys(t || {}), i = Object.keys(n), a, o;
	for (a = 0, o = i.length; a < o; a++) {
		let r = i[a];
		!r || r === "undefined" || t[r] || (O(e, r, !1), delete n[r]);
	}
	for (a = 0, o = r.length; a < o; a++) {
		let i = r[a], o = !!t[i];
		!i || i === "undefined" || n[i] === o || !o || (O(e, i, !0), n[i] = o);
	}
	return n;
}
function ae(e, t, n) {
	if (!t) return n ? y(e, "style") : t;
	let r = e.style;
	if (typeof t == "string") return r.cssText = t;
	typeof n == "string" && (r.cssText = n = void 0), n ||= {}, t ||= {};
	let i, a;
	for (a in n) t[a] ?? r.removeProperty(a), delete n[a];
	for (a in t) i = t[a], i !== n[a] && (r.setProperty(a, i), n[a] = i);
	return n;
}
function S(e, t, n) {
	n == null ? e.style.removeProperty(t) : e.style.setProperty(t, n);
}
function C(e, t = {}, n, r) {
	let i = {};
	return r || o(() => i.children = k(e, t.children, i.children)), o(() => typeof t.ref == "function" && oe(t.ref, e)), o(() => T(e, t, n, !0, i, !0)), i;
}
function oe(e, t, n) {
	return r(() => e(t, n));
}
function w(e, t, n, r) {
	if (n !== void 0 && !r && (r = []), typeof t != "function") return k(e, t, r, n);
	o((r) => k(e, t(), r, n), r);
}
function T(e, t, n, r, i = {}, a = !1) {
	t ||= {};
	for (let r in i) if (!(r in t)) {
		if (r === "children") continue;
		i[r] = ce(e, r, null, i[r], n, a, t);
	}
	for (let o in t) {
		if (o === "children") {
			r || k(e, t.children);
			continue;
		}
		let s = t[o];
		i[o] = ce(e, o, s, i[o], n, a, t);
	}
}
function E(e) {
	let t, n;
	return !D() || !(t = s.registry.get(n = N())) ? e() : (s.completed && s.completed.add(t), s.registry.delete(n), t);
}
function D(e) {
	return !!s.context && !s.done && (!e || e.isConnected);
}
function se(e) {
	return e.toLowerCase().replace(/-([a-z])/g, (e, t) => t.toUpperCase());
}
function O(e, t, n) {
	let r = t.trim().split(/\s+/);
	for (let t = 0, i = r.length; t < i; t++) e.classList.toggle(r[t], n);
}
function ce(e, t, n, r, i, a, o) {
	let s, c, f, m, g;
	if (t === "style") return ae(e, n, r);
	if (t === "classList") return ie(e, n, r);
	if (n === r) return r;
	if (t === "ref") a || n(e);
	else if (t.slice(0, 3) === "on:") {
		let i = t.slice(3);
		r && e.removeEventListener(i, r, typeof r != "function" && r), n && e.addEventListener(i, n, typeof n != "function" && n);
	} else if (t.slice(0, 10) === "oncapture:") {
		let i = t.slice(10);
		r && e.removeEventListener(i, r, !0), n && e.addEventListener(i, n, !0);
	} else if (t.slice(0, 2) === "on") {
		let i = t.slice(2).toLowerCase(), a = p.has(i);
		if (!a && r) {
			let t = Array.isArray(r) ? r[0] : r;
			e.removeEventListener(i, t);
		}
		(a || n) && (x(e, i, n, a), a && te([i]));
	} else if (t.slice(0, 5) === "attr:") y(e, t.slice(5), n);
	else if (t.slice(0, 5) === "bool:") re(e, t.slice(5), n);
	else if ((g = t.slice(0, 5) === "prop:") || (f = u.has(t)) || !i && ((m = ee(t, e.tagName)) || (c = l.has(t))) || (s = e.nodeName.includes("-") || "is" in o)) {
		if (g) t = t.slice(5), c = !0;
		else if (D(e)) return n;
		t === "class" || t === "className" ? b(e, n) : s && !c && !f ? e[se(t)] = n : e[m || t] = n;
	} else {
		let r = i && t.indexOf(":") > -1 && h[t.split(":")[0]];
		r ? ne(e, r, t, n) : y(e, d[t] || t, n);
	}
	return n;
}
function le(e) {
	if (s.registry && s.events && s.events.find(([t, n]) => n === e)) return;
	let t = e.target, n = `$$${e.type}`, r = e.target, i = e.currentTarget, a = (t) => Object.defineProperty(e, "target", {
		configurable: !0,
		value: t
	}), o = () => {
		let r = t[n];
		if (r && !t.disabled) {
			let i = t[`${n}Data`];
			if (i === void 0 ? r.call(t, e) : r.call(t, i, e), e.cancelBubble) return;
		}
		return t.host && typeof t.host != "string" && !t.host._$host && t.contains(e.target) && a(t.host), !0;
	}, c = () => {
		for (; o() && (t = t._$host || t.parentNode || t.host););
	};
	if (Object.defineProperty(e, "currentTarget", {
		configurable: !0,
		get() {
			return t || document;
		}
	}), s.registry && !s.done && (s.done = _$HY.done = !0), e.composedPath) {
		let n = e.composedPath();
		a(n[0]);
		for (let e = 0; e < n.length - 2 && (t = n[e], o()); e++) {
			if (t._$host) {
				t = t._$host, c();
				break;
			}
			if (t.parentNode === i) break;
		}
	} else c();
	a(r);
}
function k(e, t, n, r, i) {
	let a = D(e);
	if (a) {
		!n && (n = [...e.childNodes]);
		let t = [];
		for (let e = 0; e < n.length; e++) {
			let r = n[e];
			r.nodeType === 8 && r.data.slice(0, 2) === "!$" ? r.remove() : t.push(r);
		}
		n = t;
	}
	for (; typeof n == "function";) n = n();
	if (t === n) return n;
	let s = typeof t, c = r !== void 0;
	if (e = c && n[0] && n[0].parentNode || e, s === "string" || s === "number") {
		if (a || s === "number" && (t = t.toString(), t === n)) return n;
		if (c) {
			let i = n[0];
			i && i.nodeType === 3 ? i.data !== t && (i.data = t) : i = document.createTextNode(t), n = M(e, n, r, i);
		} else n = n !== "" && typeof n == "string" ? e.firstChild.data = t : e.textContent = t;
	} else if (t == null || s === "boolean") {
		if (a) return n;
		n = M(e, n, r);
	} else if (s === "function") return o(() => {
		let i = t();
		for (; typeof i == "function";) i = i();
		n = k(e, i, n, r);
	}), () => n;
	else if (Array.isArray(t)) {
		let s = [], l = n && Array.isArray(n);
		if (A(s, t, n, i)) return o(() => n = k(e, s, n, r, !0)), () => n;
		if (a) {
			if (!s.length) return n;
			if (r === void 0) return n = [...e.childNodes];
			let t = s[0];
			if (t.parentNode !== e) return n;
			let i = [t];
			for (; (t = t.nextSibling) !== r;) i.push(t);
			return n = i;
		}
		if (s.length === 0) {
			if (n = M(e, n, r), c) return n;
		} else l ? n.length === 0 ? j(e, s, r) : g(e, n, s) : (n && M(e), j(e, s));
		n = s;
	} else if (t.nodeType) {
		if (a && t.parentNode) return n = c ? [t] : t;
		if (Array.isArray(n)) {
			if (c) return n = M(e, n, r, t);
			M(e, n, null, t);
		} else n == null || n === "" || !e.firstChild ? e.appendChild(t) : e.replaceChild(t, e.firstChild);
		n = t;
	}
	return n;
}
function A(e, t, n, r) {
	let i = !1;
	for (let a = 0, o = t.length; a < o; a++) {
		let o = t[a], s = n && n[e.length], c;
		if (!(o == null || o === !0 || o === !1)) if ((c = typeof o) == "object" && o.nodeType) e.push(o);
		else if (Array.isArray(o)) i = A(e, o, s) || i;
		else if (c === "function") if (r) {
			for (; typeof o == "function";) o = o();
			i = A(e, Array.isArray(o) ? o : [o], Array.isArray(s) ? s : [s]) || i;
		} else e.push(o), i = !0;
		else {
			let t = String(o);
			s && s.nodeType === 3 && s.data === t ? e.push(s) : e.push(document.createTextNode(t));
		}
	}
	return i;
}
function j(e, t, n = null) {
	for (let r = 0, i = t.length; r < i; r++) e.insertBefore(t[r], n);
}
function M(e, t, n, r) {
	if (n === void 0) return e.textContent = "";
	let i = r || document.createTextNode("");
	if (t.length) {
		let r = !1;
		for (let a = t.length - 1; a >= 0; a--) {
			let o = t[a];
			if (i !== o) {
				let t = o.parentNode === e;
				!r && !a ? t ? e.replaceChild(i, o) : e.insertBefore(i, n) : t && o.remove();
			} else r = !0;
		}
	} else e.insertBefore(i, n);
	return [i];
}
function N() {
	return s.getNextContextId();
}
var ue = "http://www.w3.org/2000/svg";
function P(e, t = !1, n = void 0) {
	return t ? document.createElementNS(ue, e) : document.createElement(e, { is: n });
}
function de(e, t) {
	let n = i(e);
	return i(() => {
		let e = n();
		switch (typeof e) {
			case "function": return r(() => e(t));
			case "string":
				let n = m.has(e), i = s.context ? E() : P(e, n, r(() => t.is));
				return C(i, t, n), i;
		}
	});
}
function F(e) {
	let [, n] = t(e, ["component"]);
	return de(() => e.component, n);
}
//#endregion
//#region ../node_modules/.pnpm/component-register@0.8.8/node_modules/component-register/dist/component-register.js
function I(e) {
	return Object.keys(e).reduce((t, n) => {
		let r = e[n];
		return t[n] = Object.assign({}, r), ge(r.value) && !_e(r.value) && !Array.isArray(r.value) && (t[n].value = Object.assign({}, r.value)), Array.isArray(r.value) && (t[n].value = r.value.slice(0)), t;
	}, {});
}
function L(e) {
	return e ? Object.keys(e).reduce((t, n) => {
		let r = e[n];
		return t[n] = ge(r) && "value" in r ? r : { value: r }, t[n].attribute || (t[n].attribute = he(n)), t[n].parse = "parse" in t[n] ? t[n].parse : typeof t[n].value != "string", t;
	}, {}) : {};
}
function R(e) {
	return Object.keys(e).reduce((t, n) => (t[n] = e[n].value, t), {});
}
function fe(e, t) {
	let n = I(t);
	return Object.keys(t).forEach((t) => {
		let r = n[t], i = e.getAttribute(r.attribute), a = e[t];
		i != null && (r.value = r.parse ? pe(i) : i), a != null && (r.value = Array.isArray(a) ? a.slice(0) : a), r.reflect && me(e, r.attribute, r.value, !!r.parse), Object.defineProperty(e, t, {
			get() {
				return r.value;
			},
			set(e) {
				let n = r.value;
				r.value = e, r.reflect && me(this, r.attribute, r.value, !!r.parse);
				for (let r = 0, i = this.__propertyChangedCallbacks.length; r < i; r++) this.__propertyChangedCallbacks[r](t, e, n);
			},
			enumerable: !0,
			configurable: !0
		});
	}), n;
}
function pe(e) {
	if (e) try {
		return JSON.parse(e);
	} catch {
		return e;
	}
}
function me(e, t, n, r) {
	if (n == null || n === !1) return e.removeAttribute(t);
	let i = r ? JSON.stringify(n) : n;
	e.__updating[t] = !0, i === "true" && (i = ""), e.setAttribute(t, i), Promise.resolve().then(() => delete e.__updating[t]);
}
function he(e) {
	return e.replace(/\.?([A-Z]+)/g, (e, t) => "-" + t.toLowerCase()).replace("_", "-").replace(/^-/, "");
}
function ge(e) {
	return e != null && (typeof e == "object" || typeof e == "function");
}
function _e(e) {
	return Object.prototype.toString.call(e) === "[object Function]";
}
function ve(e) {
	return typeof e == "function" && e.toString().indexOf("class") === 0;
}
var z;
function B() {
	Object.defineProperty(z, "renderRoot", { value: z });
}
function ye(e, t) {
	let n = Object.keys(t);
	return class extends e {
		static get observedAttributes() {
			return n.map((e) => t[e].attribute);
		}
		constructor() {
			super(), this.__initialized = !1, this.__released = !1, this.__releaseCallbacks = [], this.__propertyChangedCallbacks = [], this.__updating = {}, this.props = {};
			for (let e of n) this[e] = void 0;
		}
		connectedCallback() {
			if (this.__initialized) return;
			this.__releaseCallbacks = [], this.__propertyChangedCallbacks = [], this.__updating = {}, this.props = fe(this, t);
			let e = R(this.props), n = this.Component, r = z;
			try {
				z = this, this.__initialized = !0, ve(n) ? new n(e, { element: this }) : n(e, { element: this });
			} finally {
				z = r;
			}
		}
		async disconnectedCallback() {
			if (await Promise.resolve(), this.isConnected) return;
			this.__propertyChangedCallbacks.length = 0;
			let e = null;
			for (; e = this.__releaseCallbacks.pop();) e(this);
			delete this.__initialized, this.__released = !0;
		}
		attributeChangedCallback(e, n, r) {
			if (this.__initialized && !this.__updating[e] && (e = this.lookupProp(e), e in t)) {
				if (r == null && !this[e]) return;
				this[e] = t[e].parse ? pe(r) : r;
			}
		}
		lookupProp(e) {
			if (t) return n.find((n) => e === n || e === t[n].attribute);
		}
		get renderRoot() {
			return this.shadowRoot || this.attachShadow({ mode: "open" });
		}
		addReleaseCallback(e) {
			this.__releaseCallbacks.push(e);
		}
		addPropertyChangedCallback(e) {
			this.__propertyChangedCallbacks.push(e);
		}
	};
}
function be(e, t = {}, n = {}) {
	let { BaseElement: r = HTMLElement, extension: i, customElements: a = window.customElements } = n;
	return (n) => {
		if (!e) throw Error("tag is required to register a Component");
		let o = a.get(e);
		return o ? (o.prototype.Component = n, o) : (o = ye(r, L(t)), o.prototype.Component = n, o.prototype.registeredTag = e, a.define(e, o, i), o);
	};
}
//#endregion
//#region ../node_modules/.pnpm/solid-element@1.9.2_solid-js@1.9.14/node_modules/solid-element/dist/index.js
function xe(t) {
	let n = Object.keys(t), r = {};
	for (let i = 0; i < n.length; i++) {
		let [a, o] = e(t[n[i]]);
		Object.defineProperty(r, n[i], {
			get: a,
			set(e) {
				o(() => e);
			}
		});
	}
	return r;
}
function Se(e) {
	if (e.assignedSlot && e.assignedSlot._$owner) return e.assignedSlot._$owner;
	let t = e.parentNode;
	for (; t;) {
		if (t._$owner) return t._$owner;
		if (t.assignedSlot && t.assignedSlot._$owner) return t.assignedSlot._$owner;
		t = t.parentNode;
	}
	return e._$owner;
}
function Ce(e) {
	return (t, r) => {
		let { element: i } = r;
		return n((n) => {
			let a = xe(t);
			i.addPropertyChangedCallback((e, t) => a[e] = t), i.addReleaseCallback(() => {
				i.renderRoot.textContent = "", n();
			});
			let o = e(a, r);
			return w(i.renderRoot, o);
		}, Se(i));
	};
}
function V(e, t, n) {
	return arguments.length === 2 && (n = t, t = {}), be(e, t)(Ce(n));
}
//#endregion
//#region ../node_modules/.pnpm/clsx@2.1.1/node_modules/clsx/dist/clsx.mjs
function we(e) {
	var t, n, r = "";
	if (typeof e == "string" || typeof e == "number") r += e;
	else if (typeof e == "object") if (Array.isArray(e)) {
		var i = e.length;
		for (t = 0; t < i; t++) e[t] && (n = we(e[t])) && (r && (r += " "), r += n);
	} else for (n in e) e[n] && (r && (r += " "), r += n);
	return r;
}
function Te() {
	for (var e, t, n = 0, r = "", i = arguments.length; n < i; n++) (e = arguments[n]) && (t = we(e)) && (r && (r += " "), r += t);
	return r;
}
//#endregion
//#region ../node_modules/.pnpm/class-variance-authority@0.7.1/node_modules/class-variance-authority/dist/index.mjs
var Ee = (e) => typeof e == "boolean" ? `${e}` : e === 0 ? "0" : e, De = Te, Oe = (e, t) => (n) => {
	if (t?.variants == null) return De(e, n?.class, n?.className);
	let { variants: r, defaultVariants: i } = t, a = Object.keys(r).map((e) => {
		let t = n?.[e], a = i?.[e];
		if (t === null) return null;
		let o = Ee(t) || Ee(a);
		return r[e][o];
	}), o = n && Object.entries(n).reduce((e, t) => {
		let [n, r] = t;
		return r === void 0 || (e[n] = r), e;
	}, {});
	return De(e, a, t?.compoundVariants?.reduce((e, t) => {
		let { class: n, className: r, ...a } = t;
		return Object.entries(a).every((e) => {
			let [t, n] = e;
			return Array.isArray(n) ? n.includes({
				...i,
				...o
			}[t]) : {
				...i,
				...o
			}[t] === n;
		}) ? [
			...e,
			n,
			r
		] : e;
	}, []), n?.class, n?.className);
}, ke = (e, t) => {
	let n = Array(e.length + t.length);
	for (let t = 0; t < e.length; t++) n[t] = e[t];
	for (let r = 0; r < t.length; r++) n[e.length + r] = t[r];
	return n;
}, Ae = (e, t) => ({
	classGroupId: e,
	validator: t
}), je = (e = /* @__PURE__ */ new Map(), t = null, n) => ({
	nextPart: e,
	validators: t,
	classGroupId: n
}), Me = "-", Ne = [], Pe = "arbitrary..", Fe = (e) => {
	let t = Re(e), { conflictingClassGroups: n, conflictingClassGroupModifiers: r } = e;
	return {
		getClassGroupId: (e) => {
			if (e.startsWith("[") && e.endsWith("]")) return Le(e);
			let n = e.split(Me);
			return Ie(n, +(n[0] === "" && n.length > 1), t);
		},
		getConflictingClassGroupIds: (e, t) => {
			if (t) {
				let t = r[e], i = n[e];
				return t ? i ? ke(i, t) : t : i || Ne;
			}
			return n[e] || Ne;
		}
	};
}, Ie = (e, t, n) => {
	if (e.length - t === 0) return n.classGroupId;
	let r = e[t], i = n.nextPart.get(r);
	if (i) {
		let n = Ie(e, t + 1, i);
		if (n) return n;
	}
	let a = n.validators;
	if (a === null) return;
	let o = t === 0 ? e.join(Me) : e.slice(t).join(Me), s = a.length;
	for (let e = 0; e < s; e++) {
		let t = a[e];
		if (t.validator(o)) return t.classGroupId;
	}
}, Le = (e) => e.slice(1, -1).indexOf(":") === -1 ? void 0 : (() => {
	let t = e.slice(1, -1), n = t.indexOf(":"), r = t.slice(0, n);
	return r ? Pe + r : void 0;
})(), Re = (e) => {
	let { theme: t, classGroups: n } = e;
	return ze(n, t);
}, ze = (e, t) => {
	let n = je();
	for (let r in e) {
		let i = e[r];
		Be(i, n, r, t);
	}
	return n;
}, Be = (e, t, n, r) => {
	let i = e.length;
	for (let a = 0; a < i; a++) {
		let i = e[a];
		Ve(i, t, n, r);
	}
}, Ve = (e, t, n, r) => {
	if (typeof e == "string") {
		He(e, t, n);
		return;
	}
	if (typeof e == "function") {
		Ue(e, t, n, r);
		return;
	}
	We(e, t, n, r);
}, He = (e, t, n) => {
	let r = e === "" ? t : Ge(t, e);
	r.classGroupId = n;
}, Ue = (e, t, n, r) => {
	if (Ke(e)) {
		Be(e(r), t, n, r);
		return;
	}
	t.validators === null && (t.validators = []), t.validators.push(Ae(n, e));
}, We = (e, t, n, r) => {
	let i = Object.entries(e), a = i.length;
	for (let e = 0; e < a; e++) {
		let [a, o] = i[e];
		Be(o, Ge(t, a), n, r);
	}
}, Ge = (e, t) => {
	let n = e, r = t.split(Me), i = r.length;
	for (let e = 0; e < i; e++) {
		let t = r[e], i = n.nextPart.get(t);
		i || (i = je(), n.nextPart.set(t, i)), n = i;
	}
	return n;
}, Ke = (e) => "isThemeGetter" in e && e.isThemeGetter === !0, qe = (e) => {
	if (e < 1) return {
		get: () => void 0,
		set: () => {}
	};
	let t = 0, n = Object.create(null), r = Object.create(null), i = (i, a) => {
		n[i] = a, t++, t > e && (t = 0, r = n, n = Object.create(null));
	};
	return {
		get(e) {
			let t = n[e];
			if (t !== void 0) return t;
			if ((t = r[e]) !== void 0) return i(e, t), t;
		},
		set(e, t) {
			e in n ? n[e] = t : i(e, t);
		}
	};
}, Je = "!", Ye = ":", Xe = [], Ze = (e, t, n, r, i) => ({
	modifiers: e,
	hasImportantModifier: t,
	baseClassName: n,
	maybePostfixModifierPosition: r,
	isExternal: i
}), Qe = (e) => {
	let { prefix: t, experimentalParseClassName: n } = e, r = (e) => {
		let t = [], n = 0, r = 0, i = 0, a, o = e.length;
		for (let s = 0; s < o; s++) {
			let o = e[s];
			if (n === 0 && r === 0) {
				if (o === Ye) {
					t.push(e.slice(i, s)), i = s + 1;
					continue;
				}
				if (o === "/") {
					a = s;
					continue;
				}
			}
			o === "[" ? n++ : o === "]" ? n-- : o === "(" ? r++ : o === ")" && r--;
		}
		let s = t.length === 0 ? e : e.slice(i), c = s, l = !1;
		s.endsWith(Je) ? (c = s.slice(0, -1), l = !0) : s.startsWith(Je) && (c = s.slice(1), l = !0);
		let u = a && a > i ? a - i : void 0;
		return Ze(t, l, c, u);
	};
	if (t) {
		let e = t + Ye, n = r;
		r = (t) => t.startsWith(e) ? n(t.slice(e.length)) : Ze(Xe, !1, t, void 0, !0);
	}
	if (n) {
		let e = r;
		r = (t) => n({
			className: t,
			parseClassName: e
		});
	}
	return r;
}, $e = (e) => {
	let t = /* @__PURE__ */ new Map();
	return e.orderSensitiveModifiers.forEach((e, n) => {
		t.set(e, 1e6 + n);
	}), (e) => {
		let n = [], r = [];
		for (let i = 0; i < e.length; i++) {
			let a = e[i], o = a[0] === "[", s = t.has(a);
			o || s ? (r.length > 0 && (r.sort(), n.push(...r), r = []), n.push(a)) : r.push(a);
		}
		return r.length > 0 && (r.sort(), n.push(...r)), n;
	};
}, et = (e) => ({
	cache: qe(e.cacheSize),
	parseClassName: Qe(e),
	sortModifiers: $e(e),
	...Fe(e)
}), tt = /\s+/, nt = (e, t) => {
	let { parseClassName: n, getClassGroupId: r, getConflictingClassGroupIds: i, sortModifiers: a } = t, o = [], s = e.trim().split(tt), c = "";
	for (let e = s.length - 1; e >= 0; --e) {
		let t = s[e], { isExternal: l, modifiers: u, hasImportantModifier: d, baseClassName: f, maybePostfixModifierPosition: ee } = n(t);
		if (l) {
			c = t + (c.length > 0 ? " " + c : c);
			continue;
		}
		let p = !!ee, m = r(p ? f.substring(0, ee) : f);
		if (!m) {
			if (!p) {
				c = t + (c.length > 0 ? " " + c : c);
				continue;
			}
			if (m = r(f), !m) {
				c = t + (c.length > 0 ? " " + c : c);
				continue;
			}
			p = !1;
		}
		let h = u.length === 0 ? "" : u.length === 1 ? u[0] : a(u).join(":"), g = d ? h + Je : h, _ = g + m;
		if (o.indexOf(_) > -1) continue;
		o.push(_);
		let v = i(m, p);
		for (let e = 0; e < v.length; ++e) {
			let t = v[e];
			o.push(g + t);
		}
		c = t + (c.length > 0 ? " " + c : c);
	}
	return c;
}, rt = (...e) => {
	let t = 0, n, r, i = "";
	for (; t < e.length;) (n = e[t++]) && (r = it(n)) && (i && (i += " "), i += r);
	return i;
}, it = (e) => {
	if (typeof e == "string") return e;
	let t, n = "";
	for (let r = 0; r < e.length; r++) e[r] && (t = it(e[r])) && (n && (n += " "), n += t);
	return n;
}, at = (e, ...t) => {
	let n, r, i, a, o = (o) => (n = et(t.reduce((e, t) => t(e), e())), r = n.cache.get, i = n.cache.set, a = s, s(o)), s = (e) => {
		let t = r(e);
		if (t) return t;
		let a = nt(e, n);
		return i(e, a), a;
	};
	return a = o, (...e) => a(rt(...e));
}, ot = [], H = (e) => {
	let t = (t) => t[e] || ot;
	return t.isThemeGetter = !0, t;
}, st = /^\[(?:(\w[\w-]*):)?(.+)\]$/i, ct = /^\((?:(\w[\w-]*):)?(.+)\)$/i, lt = /^\d+(?:\.\d+)?\/\d+(?:\.\d+)?$/, ut = /^(\d+(\.\d+)?)?(xs|sm|md|lg|xl)$/, dt = /\d+(%|px|r?em|[sdl]?v([hwib]|min|max)|pt|pc|in|cm|mm|cap|ch|ex|r?lh|cq(w|h|i|b|min|max))|\b(calc|min|max|clamp)\(.+\)|^0$/, ft = /^(rgba?|hsla?|hwb|(ok)?(lab|lch)|color-mix)\(.+\)$/, pt = /^(inset_)?-?((\d+)?\.?(\d+)[a-z]+|0)_-?((\d+)?\.?(\d+)[a-z]+|0)/, mt = /^(url|image|image-set|cross-fade|element|(repeating-)?(linear|radial|conic)-gradient)\(.+\)$/, U = (e) => lt.test(e), W = (e) => !!e && !Number.isNaN(Number(e)), G = (e) => !!e && Number.isInteger(Number(e)), ht = (e) => e.endsWith("%") && W(e.slice(0, -1)), K = (e) => ut.test(e), gt = () => !0, _t = (e) => dt.test(e) && !ft.test(e), vt = () => !1, yt = (e) => pt.test(e), bt = (e) => mt.test(e), xt = (e) => !q(e) && !Y(e), St = (e) => Z(e, Lt, vt), q = (e) => st.test(e), J = (e) => Z(e, Rt, _t), Ct = (e) => Z(e, zt, W), wt = (e) => Z(e, Vt, gt), Tt = (e) => Z(e, Bt, vt), Et = (e) => Z(e, Ft, vt), Dt = (e) => Z(e, It, bt), Ot = (e) => Z(e, Ht, yt), Y = (e) => ct.test(e), X = (e) => Q(e, Rt), kt = (e) => Q(e, Bt), At = (e) => Q(e, Ft), jt = (e) => Q(e, Lt), Mt = (e) => Q(e, It), Nt = (e) => Q(e, Ht, !0), Pt = (e) => Q(e, Vt, !0), Z = (e, t, n) => {
	let r = st.exec(e);
	return r ? r[1] ? t(r[1]) : n(r[2]) : !1;
}, Q = (e, t, n = !1) => {
	let r = ct.exec(e);
	return r ? r[1] ? t(r[1]) : n : !1;
}, Ft = (e) => e === "position" || e === "percentage", It = (e) => e === "image" || e === "url", Lt = (e) => e === "length" || e === "size" || e === "bg-size", Rt = (e) => e === "length", zt = (e) => e === "number", Bt = (e) => e === "family-name", Vt = (e) => e === "number" || e === "weight", Ht = (e) => e === "shadow", Ut = /* @__PURE__ */ at(() => {
	let e = H("color"), t = H("font"), n = H("text"), r = H("font-weight"), i = H("tracking"), a = H("leading"), o = H("breakpoint"), s = H("container"), c = H("spacing"), l = H("radius"), u = H("shadow"), d = H("inset-shadow"), f = H("text-shadow"), ee = H("drop-shadow"), p = H("blur"), m = H("perspective"), h = H("aspect"), g = H("ease"), _ = H("animate"), v = () => [
		"auto",
		"avoid",
		"all",
		"avoid-page",
		"page",
		"left",
		"right",
		"column"
	], te = () => [
		"center",
		"top",
		"bottom",
		"left",
		"right",
		"top-left",
		"left-top",
		"top-right",
		"right-top",
		"bottom-right",
		"right-bottom",
		"bottom-left",
		"left-bottom"
	], y = () => [
		...te(),
		Y,
		q
	], ne = () => [
		"auto",
		"hidden",
		"clip",
		"visible",
		"scroll"
	], re = () => [
		"auto",
		"contain",
		"none"
	], b = () => [
		Y,
		q,
		c
	], x = () => [
		U,
		"full",
		"auto",
		...b()
	], ie = () => [
		G,
		"none",
		"subgrid",
		Y,
		q
	], ae = () => [
		"auto",
		{ span: [
			"full",
			G,
			Y,
			q
		] },
		G,
		Y,
		q
	], S = () => [
		G,
		"auto",
		Y,
		q
	], C = () => [
		"auto",
		"min",
		"max",
		"fr",
		Y,
		q
	], oe = () => [
		"start",
		"end",
		"center",
		"between",
		"around",
		"evenly",
		"stretch",
		"baseline",
		"center-safe",
		"end-safe"
	], w = () => [
		"start",
		"end",
		"center",
		"stretch",
		"center-safe",
		"end-safe"
	], T = () => ["auto", ...b()], E = () => [
		U,
		"auto",
		"full",
		"dvw",
		"dvh",
		"lvw",
		"lvh",
		"svw",
		"svh",
		"min",
		"max",
		"fit",
		...b()
	], D = () => [
		U,
		"screen",
		"full",
		"dvw",
		"lvw",
		"svw",
		"min",
		"max",
		"fit",
		...b()
	], se = () => [
		U,
		"screen",
		"full",
		"lh",
		"dvh",
		"lvh",
		"svh",
		"min",
		"max",
		"fit",
		...b()
	], O = () => [
		e,
		Y,
		q
	], ce = () => [
		...te(),
		At,
		Et,
		{ position: [Y, q] }
	], le = () => ["no-repeat", { repeat: [
		"",
		"x",
		"y",
		"space",
		"round"
	] }], k = () => [
		"auto",
		"cover",
		"contain",
		jt,
		St,
		{ size: [Y, q] }
	], A = () => [
		ht,
		X,
		J
	], j = () => [
		"",
		"none",
		"full",
		l,
		Y,
		q
	], M = () => [
		"",
		W,
		X,
		J
	], N = () => [
		"solid",
		"dashed",
		"dotted",
		"double"
	], ue = () => [
		"normal",
		"multiply",
		"screen",
		"overlay",
		"darken",
		"lighten",
		"color-dodge",
		"color-burn",
		"hard-light",
		"soft-light",
		"difference",
		"exclusion",
		"hue",
		"saturation",
		"color",
		"luminosity"
	], P = () => [
		W,
		ht,
		At,
		Et
	], de = () => [
		"",
		"none",
		p,
		Y,
		q
	], F = () => [
		"none",
		W,
		Y,
		q
	], I = () => [
		"none",
		W,
		Y,
		q
	], L = () => [
		W,
		Y,
		q
	], R = () => [
		U,
		"full",
		...b()
	];
	return {
		cacheSize: 500,
		theme: {
			animate: [
				"spin",
				"ping",
				"pulse",
				"bounce"
			],
			aspect: ["video"],
			blur: [K],
			breakpoint: [K],
			color: [gt],
			container: [K],
			"drop-shadow": [K],
			ease: [
				"in",
				"out",
				"in-out"
			],
			font: [xt],
			"font-weight": [
				"thin",
				"extralight",
				"light",
				"normal",
				"medium",
				"semibold",
				"bold",
				"extrabold",
				"black"
			],
			"inset-shadow": [K],
			leading: [
				"none",
				"tight",
				"snug",
				"normal",
				"relaxed",
				"loose"
			],
			perspective: [
				"dramatic",
				"near",
				"normal",
				"midrange",
				"distant",
				"none"
			],
			radius: [K],
			shadow: [K],
			spacing: ["px", W],
			text: [K],
			"text-shadow": [K],
			tracking: [
				"tighter",
				"tight",
				"normal",
				"wide",
				"wider",
				"widest"
			]
		},
		classGroups: {
			aspect: [{ aspect: [
				"auto",
				"square",
				U,
				q,
				Y,
				h
			] }],
			container: ["container"],
			columns: [{ columns: [
				W,
				q,
				Y,
				s
			] }],
			"break-after": [{ "break-after": v() }],
			"break-before": [{ "break-before": v() }],
			"break-inside": [{ "break-inside": [
				"auto",
				"avoid",
				"avoid-page",
				"avoid-column"
			] }],
			"box-decoration": [{ "box-decoration": ["slice", "clone"] }],
			box: [{ box: ["border", "content"] }],
			display: [
				"block",
				"inline-block",
				"inline",
				"flex",
				"inline-flex",
				"table",
				"inline-table",
				"table-caption",
				"table-cell",
				"table-column",
				"table-column-group",
				"table-footer-group",
				"table-header-group",
				"table-row-group",
				"table-row",
				"flow-root",
				"grid",
				"inline-grid",
				"contents",
				"list-item",
				"hidden"
			],
			sr: ["sr-only", "not-sr-only"],
			float: [{ float: [
				"right",
				"left",
				"none",
				"start",
				"end"
			] }],
			clear: [{ clear: [
				"left",
				"right",
				"both",
				"none",
				"start",
				"end"
			] }],
			isolation: ["isolate", "isolation-auto"],
			"object-fit": [{ object: [
				"contain",
				"cover",
				"fill",
				"none",
				"scale-down"
			] }],
			"object-position": [{ object: y() }],
			overflow: [{ overflow: ne() }],
			"overflow-x": [{ "overflow-x": ne() }],
			"overflow-y": [{ "overflow-y": ne() }],
			overscroll: [{ overscroll: re() }],
			"overscroll-x": [{ "overscroll-x": re() }],
			"overscroll-y": [{ "overscroll-y": re() }],
			position: [
				"static",
				"fixed",
				"absolute",
				"relative",
				"sticky"
			],
			inset: [{ inset: x() }],
			"inset-x": [{ "inset-x": x() }],
			"inset-y": [{ "inset-y": x() }],
			start: [{
				"inset-s": x(),
				start: x()
			}],
			end: [{
				"inset-e": x(),
				end: x()
			}],
			"inset-bs": [{ "inset-bs": x() }],
			"inset-be": [{ "inset-be": x() }],
			top: [{ top: x() }],
			right: [{ right: x() }],
			bottom: [{ bottom: x() }],
			left: [{ left: x() }],
			visibility: [
				"visible",
				"invisible",
				"collapse"
			],
			z: [{ z: [
				G,
				"auto",
				Y,
				q
			] }],
			basis: [{ basis: [
				U,
				"full",
				"auto",
				s,
				...b()
			] }],
			"flex-direction": [{ flex: [
				"row",
				"row-reverse",
				"col",
				"col-reverse"
			] }],
			"flex-wrap": [{ flex: [
				"nowrap",
				"wrap",
				"wrap-reverse"
			] }],
			flex: [{ flex: [
				W,
				U,
				"auto",
				"initial",
				"none",
				q
			] }],
			grow: [{ grow: [
				"",
				W,
				Y,
				q
			] }],
			shrink: [{ shrink: [
				"",
				W,
				Y,
				q
			] }],
			order: [{ order: [
				G,
				"first",
				"last",
				"none",
				Y,
				q
			] }],
			"grid-cols": [{ "grid-cols": ie() }],
			"col-start-end": [{ col: ae() }],
			"col-start": [{ "col-start": S() }],
			"col-end": [{ "col-end": S() }],
			"grid-rows": [{ "grid-rows": ie() }],
			"row-start-end": [{ row: ae() }],
			"row-start": [{ "row-start": S() }],
			"row-end": [{ "row-end": S() }],
			"grid-flow": [{ "grid-flow": [
				"row",
				"col",
				"dense",
				"row-dense",
				"col-dense"
			] }],
			"auto-cols": [{ "auto-cols": C() }],
			"auto-rows": [{ "auto-rows": C() }],
			gap: [{ gap: b() }],
			"gap-x": [{ "gap-x": b() }],
			"gap-y": [{ "gap-y": b() }],
			"justify-content": [{ justify: [...oe(), "normal"] }],
			"justify-items": [{ "justify-items": [...w(), "normal"] }],
			"justify-self": [{ "justify-self": ["auto", ...w()] }],
			"align-content": [{ content: ["normal", ...oe()] }],
			"align-items": [{ items: [...w(), { baseline: ["", "last"] }] }],
			"align-self": [{ self: [
				"auto",
				...w(),
				{ baseline: ["", "last"] }
			] }],
			"place-content": [{ "place-content": oe() }],
			"place-items": [{ "place-items": [...w(), "baseline"] }],
			"place-self": [{ "place-self": ["auto", ...w()] }],
			p: [{ p: b() }],
			px: [{ px: b() }],
			py: [{ py: b() }],
			ps: [{ ps: b() }],
			pe: [{ pe: b() }],
			pbs: [{ pbs: b() }],
			pbe: [{ pbe: b() }],
			pt: [{ pt: b() }],
			pr: [{ pr: b() }],
			pb: [{ pb: b() }],
			pl: [{ pl: b() }],
			m: [{ m: T() }],
			mx: [{ mx: T() }],
			my: [{ my: T() }],
			ms: [{ ms: T() }],
			me: [{ me: T() }],
			mbs: [{ mbs: T() }],
			mbe: [{ mbe: T() }],
			mt: [{ mt: T() }],
			mr: [{ mr: T() }],
			mb: [{ mb: T() }],
			ml: [{ ml: T() }],
			"space-x": [{ "space-x": b() }],
			"space-x-reverse": ["space-x-reverse"],
			"space-y": [{ "space-y": b() }],
			"space-y-reverse": ["space-y-reverse"],
			size: [{ size: E() }],
			"inline-size": [{ inline: ["auto", ...D()] }],
			"min-inline-size": [{ "min-inline": ["auto", ...D()] }],
			"max-inline-size": [{ "max-inline": ["none", ...D()] }],
			"block-size": [{ block: ["auto", ...se()] }],
			"min-block-size": [{ "min-block": ["auto", ...se()] }],
			"max-block-size": [{ "max-block": ["none", ...se()] }],
			w: [{ w: [
				s,
				"screen",
				...E()
			] }],
			"min-w": [{ "min-w": [
				s,
				"screen",
				"none",
				...E()
			] }],
			"max-w": [{ "max-w": [
				s,
				"screen",
				"none",
				"prose",
				{ screen: [o] },
				...E()
			] }],
			h: [{ h: [
				"screen",
				"lh",
				...E()
			] }],
			"min-h": [{ "min-h": [
				"screen",
				"lh",
				"none",
				...E()
			] }],
			"max-h": [{ "max-h": [
				"screen",
				"lh",
				...E()
			] }],
			"font-size": [{ text: [
				"base",
				n,
				X,
				J
			] }],
			"font-smoothing": ["antialiased", "subpixel-antialiased"],
			"font-style": ["italic", "not-italic"],
			"font-weight": [{ font: [
				r,
				Pt,
				wt
			] }],
			"font-stretch": [{ "font-stretch": [
				"ultra-condensed",
				"extra-condensed",
				"condensed",
				"semi-condensed",
				"normal",
				"semi-expanded",
				"expanded",
				"extra-expanded",
				"ultra-expanded",
				ht,
				q
			] }],
			"font-family": [{ font: [
				kt,
				Tt,
				t
			] }],
			"font-features": [{ "font-features": [q] }],
			"fvn-normal": ["normal-nums"],
			"fvn-ordinal": ["ordinal"],
			"fvn-slashed-zero": ["slashed-zero"],
			"fvn-figure": ["lining-nums", "oldstyle-nums"],
			"fvn-spacing": ["proportional-nums", "tabular-nums"],
			"fvn-fraction": ["diagonal-fractions", "stacked-fractions"],
			tracking: [{ tracking: [
				i,
				Y,
				q
			] }],
			"line-clamp": [{ "line-clamp": [
				W,
				"none",
				Y,
				Ct
			] }],
			leading: [{ leading: [a, ...b()] }],
			"list-image": [{ "list-image": [
				"none",
				Y,
				q
			] }],
			"list-style-position": [{ list: ["inside", "outside"] }],
			"list-style-type": [{ list: [
				"disc",
				"decimal",
				"none",
				Y,
				q
			] }],
			"text-alignment": [{ text: [
				"left",
				"center",
				"right",
				"justify",
				"start",
				"end"
			] }],
			"placeholder-color": [{ placeholder: O() }],
			"text-color": [{ text: O() }],
			"text-decoration": [
				"underline",
				"overline",
				"line-through",
				"no-underline"
			],
			"text-decoration-style": [{ decoration: [...N(), "wavy"] }],
			"text-decoration-thickness": [{ decoration: [
				W,
				"from-font",
				"auto",
				Y,
				J
			] }],
			"text-decoration-color": [{ decoration: O() }],
			"underline-offset": [{ "underline-offset": [
				W,
				"auto",
				Y,
				q
			] }],
			"text-transform": [
				"uppercase",
				"lowercase",
				"capitalize",
				"normal-case"
			],
			"text-overflow": [
				"truncate",
				"text-ellipsis",
				"text-clip"
			],
			"text-wrap": [{ text: [
				"wrap",
				"nowrap",
				"balance",
				"pretty"
			] }],
			indent: [{ indent: b() }],
			"vertical-align": [{ align: [
				"baseline",
				"top",
				"middle",
				"bottom",
				"text-top",
				"text-bottom",
				"sub",
				"super",
				Y,
				q
			] }],
			whitespace: [{ whitespace: [
				"normal",
				"nowrap",
				"pre",
				"pre-line",
				"pre-wrap",
				"break-spaces"
			] }],
			break: [{ break: [
				"normal",
				"words",
				"all",
				"keep"
			] }],
			wrap: [{ wrap: [
				"break-word",
				"anywhere",
				"normal"
			] }],
			hyphens: [{ hyphens: [
				"none",
				"manual",
				"auto"
			] }],
			content: [{ content: [
				"none",
				Y,
				q
			] }],
			"bg-attachment": [{ bg: [
				"fixed",
				"local",
				"scroll"
			] }],
			"bg-clip": [{ "bg-clip": [
				"border",
				"padding",
				"content",
				"text"
			] }],
			"bg-origin": [{ "bg-origin": [
				"border",
				"padding",
				"content"
			] }],
			"bg-position": [{ bg: ce() }],
			"bg-repeat": [{ bg: le() }],
			"bg-size": [{ bg: k() }],
			"bg-image": [{ bg: [
				"none",
				{
					linear: [
						{ to: [
							"t",
							"tr",
							"r",
							"br",
							"b",
							"bl",
							"l",
							"tl"
						] },
						G,
						Y,
						q
					],
					radial: [
						"",
						Y,
						q
					],
					conic: [
						G,
						Y,
						q
					]
				},
				Mt,
				Dt
			] }],
			"bg-color": [{ bg: O() }],
			"gradient-from-pos": [{ from: A() }],
			"gradient-via-pos": [{ via: A() }],
			"gradient-to-pos": [{ to: A() }],
			"gradient-from": [{ from: O() }],
			"gradient-via": [{ via: O() }],
			"gradient-to": [{ to: O() }],
			rounded: [{ rounded: j() }],
			"rounded-s": [{ "rounded-s": j() }],
			"rounded-e": [{ "rounded-e": j() }],
			"rounded-t": [{ "rounded-t": j() }],
			"rounded-r": [{ "rounded-r": j() }],
			"rounded-b": [{ "rounded-b": j() }],
			"rounded-l": [{ "rounded-l": j() }],
			"rounded-ss": [{ "rounded-ss": j() }],
			"rounded-se": [{ "rounded-se": j() }],
			"rounded-ee": [{ "rounded-ee": j() }],
			"rounded-es": [{ "rounded-es": j() }],
			"rounded-tl": [{ "rounded-tl": j() }],
			"rounded-tr": [{ "rounded-tr": j() }],
			"rounded-br": [{ "rounded-br": j() }],
			"rounded-bl": [{ "rounded-bl": j() }],
			"border-w": [{ border: M() }],
			"border-w-x": [{ "border-x": M() }],
			"border-w-y": [{ "border-y": M() }],
			"border-w-s": [{ "border-s": M() }],
			"border-w-e": [{ "border-e": M() }],
			"border-w-bs": [{ "border-bs": M() }],
			"border-w-be": [{ "border-be": M() }],
			"border-w-t": [{ "border-t": M() }],
			"border-w-r": [{ "border-r": M() }],
			"border-w-b": [{ "border-b": M() }],
			"border-w-l": [{ "border-l": M() }],
			"divide-x": [{ "divide-x": M() }],
			"divide-x-reverse": ["divide-x-reverse"],
			"divide-y": [{ "divide-y": M() }],
			"divide-y-reverse": ["divide-y-reverse"],
			"border-style": [{ border: [
				...N(),
				"hidden",
				"none"
			] }],
			"divide-style": [{ divide: [
				...N(),
				"hidden",
				"none"
			] }],
			"border-color": [{ border: O() }],
			"border-color-x": [{ "border-x": O() }],
			"border-color-y": [{ "border-y": O() }],
			"border-color-s": [{ "border-s": O() }],
			"border-color-e": [{ "border-e": O() }],
			"border-color-bs": [{ "border-bs": O() }],
			"border-color-be": [{ "border-be": O() }],
			"border-color-t": [{ "border-t": O() }],
			"border-color-r": [{ "border-r": O() }],
			"border-color-b": [{ "border-b": O() }],
			"border-color-l": [{ "border-l": O() }],
			"divide-color": [{ divide: O() }],
			"outline-style": [{ outline: [
				...N(),
				"none",
				"hidden"
			] }],
			"outline-offset": [{ "outline-offset": [
				W,
				Y,
				q
			] }],
			"outline-w": [{ outline: [
				"",
				W,
				X,
				J
			] }],
			"outline-color": [{ outline: O() }],
			shadow: [{ shadow: [
				"",
				"none",
				u,
				Nt,
				Ot
			] }],
			"shadow-color": [{ shadow: O() }],
			"inset-shadow": [{ "inset-shadow": [
				"none",
				d,
				Nt,
				Ot
			] }],
			"inset-shadow-color": [{ "inset-shadow": O() }],
			"ring-w": [{ ring: M() }],
			"ring-w-inset": ["ring-inset"],
			"ring-color": [{ ring: O() }],
			"ring-offset-w": [{ "ring-offset": [W, J] }],
			"ring-offset-color": [{ "ring-offset": O() }],
			"inset-ring-w": [{ "inset-ring": M() }],
			"inset-ring-color": [{ "inset-ring": O() }],
			"text-shadow": [{ "text-shadow": [
				"none",
				f,
				Nt,
				Ot
			] }],
			"text-shadow-color": [{ "text-shadow": O() }],
			opacity: [{ opacity: [
				W,
				Y,
				q
			] }],
			"mix-blend": [{ "mix-blend": [
				...ue(),
				"plus-darker",
				"plus-lighter"
			] }],
			"bg-blend": [{ "bg-blend": ue() }],
			"mask-clip": [{ "mask-clip": [
				"border",
				"padding",
				"content",
				"fill",
				"stroke",
				"view"
			] }, "mask-no-clip"],
			"mask-composite": [{ mask: [
				"add",
				"subtract",
				"intersect",
				"exclude"
			] }],
			"mask-image-linear-pos": [{ "mask-linear": [W] }],
			"mask-image-linear-from-pos": [{ "mask-linear-from": P() }],
			"mask-image-linear-to-pos": [{ "mask-linear-to": P() }],
			"mask-image-linear-from-color": [{ "mask-linear-from": O() }],
			"mask-image-linear-to-color": [{ "mask-linear-to": O() }],
			"mask-image-t-from-pos": [{ "mask-t-from": P() }],
			"mask-image-t-to-pos": [{ "mask-t-to": P() }],
			"mask-image-t-from-color": [{ "mask-t-from": O() }],
			"mask-image-t-to-color": [{ "mask-t-to": O() }],
			"mask-image-r-from-pos": [{ "mask-r-from": P() }],
			"mask-image-r-to-pos": [{ "mask-r-to": P() }],
			"mask-image-r-from-color": [{ "mask-r-from": O() }],
			"mask-image-r-to-color": [{ "mask-r-to": O() }],
			"mask-image-b-from-pos": [{ "mask-b-from": P() }],
			"mask-image-b-to-pos": [{ "mask-b-to": P() }],
			"mask-image-b-from-color": [{ "mask-b-from": O() }],
			"mask-image-b-to-color": [{ "mask-b-to": O() }],
			"mask-image-l-from-pos": [{ "mask-l-from": P() }],
			"mask-image-l-to-pos": [{ "mask-l-to": P() }],
			"mask-image-l-from-color": [{ "mask-l-from": O() }],
			"mask-image-l-to-color": [{ "mask-l-to": O() }],
			"mask-image-x-from-pos": [{ "mask-x-from": P() }],
			"mask-image-x-to-pos": [{ "mask-x-to": P() }],
			"mask-image-x-from-color": [{ "mask-x-from": O() }],
			"mask-image-x-to-color": [{ "mask-x-to": O() }],
			"mask-image-y-from-pos": [{ "mask-y-from": P() }],
			"mask-image-y-to-pos": [{ "mask-y-to": P() }],
			"mask-image-y-from-color": [{ "mask-y-from": O() }],
			"mask-image-y-to-color": [{ "mask-y-to": O() }],
			"mask-image-radial": [{ "mask-radial": [Y, q] }],
			"mask-image-radial-from-pos": [{ "mask-radial-from": P() }],
			"mask-image-radial-to-pos": [{ "mask-radial-to": P() }],
			"mask-image-radial-from-color": [{ "mask-radial-from": O() }],
			"mask-image-radial-to-color": [{ "mask-radial-to": O() }],
			"mask-image-radial-shape": [{ "mask-radial": ["circle", "ellipse"] }],
			"mask-image-radial-size": [{ "mask-radial": [{
				closest: ["side", "corner"],
				farthest: ["side", "corner"]
			}] }],
			"mask-image-radial-pos": [{ "mask-radial-at": te() }],
			"mask-image-conic-pos": [{ "mask-conic": [W] }],
			"mask-image-conic-from-pos": [{ "mask-conic-from": P() }],
			"mask-image-conic-to-pos": [{ "mask-conic-to": P() }],
			"mask-image-conic-from-color": [{ "mask-conic-from": O() }],
			"mask-image-conic-to-color": [{ "mask-conic-to": O() }],
			"mask-mode": [{ mask: [
				"alpha",
				"luminance",
				"match"
			] }],
			"mask-origin": [{ "mask-origin": [
				"border",
				"padding",
				"content",
				"fill",
				"stroke",
				"view"
			] }],
			"mask-position": [{ mask: ce() }],
			"mask-repeat": [{ mask: le() }],
			"mask-size": [{ mask: k() }],
			"mask-type": [{ "mask-type": ["alpha", "luminance"] }],
			"mask-image": [{ mask: [
				"none",
				Y,
				q
			] }],
			filter: [{ filter: [
				"",
				"none",
				Y,
				q
			] }],
			blur: [{ blur: de() }],
			brightness: [{ brightness: [
				W,
				Y,
				q
			] }],
			contrast: [{ contrast: [
				W,
				Y,
				q
			] }],
			"drop-shadow": [{ "drop-shadow": [
				"",
				"none",
				ee,
				Nt,
				Ot
			] }],
			"drop-shadow-color": [{ "drop-shadow": O() }],
			grayscale: [{ grayscale: [
				"",
				W,
				Y,
				q
			] }],
			"hue-rotate": [{ "hue-rotate": [
				W,
				Y,
				q
			] }],
			invert: [{ invert: [
				"",
				W,
				Y,
				q
			] }],
			saturate: [{ saturate: [
				W,
				Y,
				q
			] }],
			sepia: [{ sepia: [
				"",
				W,
				Y,
				q
			] }],
			"backdrop-filter": [{ "backdrop-filter": [
				"",
				"none",
				Y,
				q
			] }],
			"backdrop-blur": [{ "backdrop-blur": de() }],
			"backdrop-brightness": [{ "backdrop-brightness": [
				W,
				Y,
				q
			] }],
			"backdrop-contrast": [{ "backdrop-contrast": [
				W,
				Y,
				q
			] }],
			"backdrop-grayscale": [{ "backdrop-grayscale": [
				"",
				W,
				Y,
				q
			] }],
			"backdrop-hue-rotate": [{ "backdrop-hue-rotate": [
				W,
				Y,
				q
			] }],
			"backdrop-invert": [{ "backdrop-invert": [
				"",
				W,
				Y,
				q
			] }],
			"backdrop-opacity": [{ "backdrop-opacity": [
				W,
				Y,
				q
			] }],
			"backdrop-saturate": [{ "backdrop-saturate": [
				W,
				Y,
				q
			] }],
			"backdrop-sepia": [{ "backdrop-sepia": [
				"",
				W,
				Y,
				q
			] }],
			"border-collapse": [{ border: ["collapse", "separate"] }],
			"border-spacing": [{ "border-spacing": b() }],
			"border-spacing-x": [{ "border-spacing-x": b() }],
			"border-spacing-y": [{ "border-spacing-y": b() }],
			"table-layout": [{ table: ["auto", "fixed"] }],
			caption: [{ caption: ["top", "bottom"] }],
			transition: [{ transition: [
				"",
				"all",
				"colors",
				"opacity",
				"shadow",
				"transform",
				"none",
				Y,
				q
			] }],
			"transition-behavior": [{ transition: ["normal", "discrete"] }],
			duration: [{ duration: [
				W,
				"initial",
				Y,
				q
			] }],
			ease: [{ ease: [
				"linear",
				"initial",
				g,
				Y,
				q
			] }],
			delay: [{ delay: [
				W,
				Y,
				q
			] }],
			animate: [{ animate: [
				"none",
				_,
				Y,
				q
			] }],
			backface: [{ backface: ["hidden", "visible"] }],
			perspective: [{ perspective: [
				m,
				Y,
				q
			] }],
			"perspective-origin": [{ "perspective-origin": y() }],
			rotate: [{ rotate: F() }],
			"rotate-x": [{ "rotate-x": F() }],
			"rotate-y": [{ "rotate-y": F() }],
			"rotate-z": [{ "rotate-z": F() }],
			scale: [{ scale: I() }],
			"scale-x": [{ "scale-x": I() }],
			"scale-y": [{ "scale-y": I() }],
			"scale-z": [{ "scale-z": I() }],
			"scale-3d": ["scale-3d"],
			skew: [{ skew: L() }],
			"skew-x": [{ "skew-x": L() }],
			"skew-y": [{ "skew-y": L() }],
			transform: [{ transform: [
				Y,
				q,
				"",
				"none",
				"gpu",
				"cpu"
			] }],
			"transform-origin": [{ origin: y() }],
			"transform-style": [{ transform: ["3d", "flat"] }],
			translate: [{ translate: R() }],
			"translate-x": [{ "translate-x": R() }],
			"translate-y": [{ "translate-y": R() }],
			"translate-z": [{ "translate-z": R() }],
			"translate-none": ["translate-none"],
			accent: [{ accent: O() }],
			appearance: [{ appearance: ["none", "auto"] }],
			"caret-color": [{ caret: O() }],
			"color-scheme": [{ scheme: [
				"normal",
				"dark",
				"light",
				"light-dark",
				"only-dark",
				"only-light"
			] }],
			cursor: [{ cursor: [
				"auto",
				"default",
				"pointer",
				"wait",
				"text",
				"move",
				"help",
				"not-allowed",
				"none",
				"context-menu",
				"progress",
				"cell",
				"crosshair",
				"vertical-text",
				"alias",
				"copy",
				"no-drop",
				"grab",
				"grabbing",
				"all-scroll",
				"col-resize",
				"row-resize",
				"n-resize",
				"e-resize",
				"s-resize",
				"w-resize",
				"ne-resize",
				"nw-resize",
				"se-resize",
				"sw-resize",
				"ew-resize",
				"ns-resize",
				"nesw-resize",
				"nwse-resize",
				"zoom-in",
				"zoom-out",
				Y,
				q
			] }],
			"field-sizing": [{ "field-sizing": ["fixed", "content"] }],
			"pointer-events": [{ "pointer-events": ["auto", "none"] }],
			resize: [{ resize: [
				"none",
				"",
				"y",
				"x"
			] }],
			"scroll-behavior": [{ scroll: ["auto", "smooth"] }],
			"scroll-m": [{ "scroll-m": b() }],
			"scroll-mx": [{ "scroll-mx": b() }],
			"scroll-my": [{ "scroll-my": b() }],
			"scroll-ms": [{ "scroll-ms": b() }],
			"scroll-me": [{ "scroll-me": b() }],
			"scroll-mbs": [{ "scroll-mbs": b() }],
			"scroll-mbe": [{ "scroll-mbe": b() }],
			"scroll-mt": [{ "scroll-mt": b() }],
			"scroll-mr": [{ "scroll-mr": b() }],
			"scroll-mb": [{ "scroll-mb": b() }],
			"scroll-ml": [{ "scroll-ml": b() }],
			"scroll-p": [{ "scroll-p": b() }],
			"scroll-px": [{ "scroll-px": b() }],
			"scroll-py": [{ "scroll-py": b() }],
			"scroll-ps": [{ "scroll-ps": b() }],
			"scroll-pe": [{ "scroll-pe": b() }],
			"scroll-pbs": [{ "scroll-pbs": b() }],
			"scroll-pbe": [{ "scroll-pbe": b() }],
			"scroll-pt": [{ "scroll-pt": b() }],
			"scroll-pr": [{ "scroll-pr": b() }],
			"scroll-pb": [{ "scroll-pb": b() }],
			"scroll-pl": [{ "scroll-pl": b() }],
			"snap-align": [{ snap: [
				"start",
				"end",
				"center",
				"align-none"
			] }],
			"snap-stop": [{ snap: ["normal", "always"] }],
			"snap-type": [{ snap: [
				"none",
				"x",
				"y",
				"both"
			] }],
			"snap-strictness": [{ snap: ["mandatory", "proximity"] }],
			touch: [{ touch: [
				"auto",
				"none",
				"manipulation"
			] }],
			"touch-x": [{ "touch-pan": [
				"x",
				"left",
				"right"
			] }],
			"touch-y": [{ "touch-pan": [
				"y",
				"up",
				"down"
			] }],
			"touch-pz": ["touch-pinch-zoom"],
			select: [{ select: [
				"none",
				"text",
				"all",
				"auto"
			] }],
			"will-change": [{ "will-change": [
				"auto",
				"scroll",
				"contents",
				"transform",
				Y,
				q
			] }],
			fill: [{ fill: ["none", ...O()] }],
			"stroke-w": [{ stroke: [
				W,
				X,
				J,
				Ct
			] }],
			stroke: [{ stroke: ["none", ...O()] }],
			"forced-color-adjust": [{ "forced-color-adjust": ["auto", "none"] }]
		},
		conflictingClassGroups: {
			overflow: ["overflow-x", "overflow-y"],
			overscroll: ["overscroll-x", "overscroll-y"],
			inset: [
				"inset-x",
				"inset-y",
				"inset-bs",
				"inset-be",
				"start",
				"end",
				"top",
				"right",
				"bottom",
				"left"
			],
			"inset-x": ["right", "left"],
			"inset-y": ["top", "bottom"],
			flex: [
				"basis",
				"grow",
				"shrink"
			],
			gap: ["gap-x", "gap-y"],
			p: [
				"px",
				"py",
				"ps",
				"pe",
				"pbs",
				"pbe",
				"pt",
				"pr",
				"pb",
				"pl"
			],
			px: ["pr", "pl"],
			py: ["pt", "pb"],
			m: [
				"mx",
				"my",
				"ms",
				"me",
				"mbs",
				"mbe",
				"mt",
				"mr",
				"mb",
				"ml"
			],
			mx: ["mr", "ml"],
			my: ["mt", "mb"],
			size: ["w", "h"],
			"font-size": ["leading"],
			"fvn-normal": [
				"fvn-ordinal",
				"fvn-slashed-zero",
				"fvn-figure",
				"fvn-spacing",
				"fvn-fraction"
			],
			"fvn-ordinal": ["fvn-normal"],
			"fvn-slashed-zero": ["fvn-normal"],
			"fvn-figure": ["fvn-normal"],
			"fvn-spacing": ["fvn-normal"],
			"fvn-fraction": ["fvn-normal"],
			"line-clamp": ["display", "overflow"],
			rounded: [
				"rounded-s",
				"rounded-e",
				"rounded-t",
				"rounded-r",
				"rounded-b",
				"rounded-l",
				"rounded-ss",
				"rounded-se",
				"rounded-ee",
				"rounded-es",
				"rounded-tl",
				"rounded-tr",
				"rounded-br",
				"rounded-bl"
			],
			"rounded-s": ["rounded-ss", "rounded-es"],
			"rounded-e": ["rounded-se", "rounded-ee"],
			"rounded-t": ["rounded-tl", "rounded-tr"],
			"rounded-r": ["rounded-tr", "rounded-br"],
			"rounded-b": ["rounded-br", "rounded-bl"],
			"rounded-l": ["rounded-tl", "rounded-bl"],
			"border-spacing": ["border-spacing-x", "border-spacing-y"],
			"border-w": [
				"border-w-x",
				"border-w-y",
				"border-w-s",
				"border-w-e",
				"border-w-bs",
				"border-w-be",
				"border-w-t",
				"border-w-r",
				"border-w-b",
				"border-w-l"
			],
			"border-w-x": ["border-w-r", "border-w-l"],
			"border-w-y": ["border-w-t", "border-w-b"],
			"border-color": [
				"border-color-x",
				"border-color-y",
				"border-color-s",
				"border-color-e",
				"border-color-bs",
				"border-color-be",
				"border-color-t",
				"border-color-r",
				"border-color-b",
				"border-color-l"
			],
			"border-color-x": ["border-color-r", "border-color-l"],
			"border-color-y": ["border-color-t", "border-color-b"],
			translate: [
				"translate-x",
				"translate-y",
				"translate-none"
			],
			"translate-none": [
				"translate",
				"translate-x",
				"translate-y",
				"translate-z"
			],
			"scroll-m": [
				"scroll-mx",
				"scroll-my",
				"scroll-ms",
				"scroll-me",
				"scroll-mbs",
				"scroll-mbe",
				"scroll-mt",
				"scroll-mr",
				"scroll-mb",
				"scroll-ml"
			],
			"scroll-mx": ["scroll-mr", "scroll-ml"],
			"scroll-my": ["scroll-mt", "scroll-mb"],
			"scroll-p": [
				"scroll-px",
				"scroll-py",
				"scroll-ps",
				"scroll-pe",
				"scroll-pbs",
				"scroll-pbe",
				"scroll-pt",
				"scroll-pr",
				"scroll-pb",
				"scroll-pl"
			],
			"scroll-px": ["scroll-pr", "scroll-pl"],
			"scroll-py": ["scroll-pt", "scroll-pb"],
			touch: [
				"touch-x",
				"touch-y",
				"touch-pz"
			],
			"touch-x": ["touch"],
			"touch-y": ["touch"],
			"touch-pz": ["touch"]
		},
		conflictingClassGroupModifiers: { "font-size": ["leading"] },
		orderSensitiveModifiers: [
			"*",
			"**",
			"after",
			"backdrop",
			"before",
			"details-content",
			"file",
			"first-letter",
			"first-line",
			"marker",
			"placeholder",
			"selection"
		]
	};
});
//#endregion
//#region src/shared/lib/utils.ts
function $(...e) {
	return Ut(Te(e));
}
//#endregion
//#region src/shared/ui/badge.tsx
var Wt = /* @__PURE__ */ v("<span data-slot=badge>"), Gt = Oe("inline-flex w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-full border border-transparent px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-[color,box-shadow] aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 [&>svg]:pointer-events-none [&>svg]:size-3", {
	variants: { variant: {
		default: "bg-primary text-primary-foreground [a&]:hover:bg-primary/90",
		secondary: "bg-secondary text-secondary-foreground [a&]:hover:bg-secondary/90",
		destructive: "bg-destructive text-white dark:bg-destructive/60 dark: [a&]:hover:bg-destructive/90",
		outline: "border-border text-foreground [a&]:hover:bg-accent [a&]:hover:text-accent-foreground",
		ghost: "[a&]:hover:bg-accent [a&]:hover:text-accent-foreground",
		link: "text-primary underline-offset-4 [a&]:hover:underline"
	} },
	defaultVariants: { variant: "default" }
});
function Kt(e) {
	let [n, r] = t(e, ["class", "variant"]);
	return (() => {
		var e = Wt();
		return C(e, a({
			get "data-variant"() {
				return n.variant ?? "default";
			},
			get class() {
				return $(Gt({ variant: n.variant }), n.class);
			}
		}, r), !1, !1), e;
	})();
}
//#endregion
//#region src/shared/ui/button.tsx
var qt = /* @__PURE__ */ v("<button data-slot=button>"), Jt = Oe("inline-flex shrink-0 items-center justify-center gap-2 rounded-sm text-sm font-medium whitespace-nowrap transition-all outline-none disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4", {
	variants: {
		variant: {
			default: "bg-primary text-primary-foreground hover:bg-primary/90",
			destructive: "bg-destructive text-white hover:bg-destructive/90",
			outline: "border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:border-input dark:bg-input/30 dark:hover:bg-input/50",
			secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
			ghost: "hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50",
			link: "text-primary underline-offset-4 hover:underline"
		},
		size: {
			default: "h-9 px-4 py-2 has-[>svg]:px-3",
			xs: "h-6 gap-1 rounded-sm px-2 text-xs has-[>svg]:px-1.5 [&_svg:not([class*='size-'])]:size-3",
			sm: "h-8 gap-1.5 rounded-sm px-3 has-[>svg]:px-2.5",
			lg: "h-10 rounded-sm px-6 has-[>svg]:px-4",
			icon: "size-9",
			"icon-xs": "size-6 rounded-sm [&_svg:not([class*='size-'])]:size-3",
			"icon-sm": "size-8",
			"icon-lg": "size-10"
		}
	},
	defaultVariants: {
		variant: "default",
		size: "default"
	}
});
function Yt(e) {
	let [n, r] = t(e, [
		"class",
		"variant",
		"size",
		"type"
	]);
	return (() => {
		var e = qt();
		return C(e, a({
			get type() {
				return n.type ?? "button";
			},
			get "data-variant"() {
				return n.variant ?? "default";
			},
			get "data-size"() {
				return n.size ?? "default";
			},
			get class() {
				return $(Jt({
					variant: n.variant,
					size: n.size
				}), n.class);
			}
		}, r), !1, !1), e;
	})();
}
//#endregion
//#region src/shared/ui/input.tsx
var Xt = /* @__PURE__ */ v("<input data-slot=input>");
function Zt(e) {
	let [n, r] = t(e, ["class"]);
	return (() => {
		var e = Xt();
		return C(e, a({ get class() {
			return $("h-9 w-full min-w-0 rounded-sm border border-input bg-transparent px-3 py-1 text-base shadow-xs transition-[color,box-shadow] outline-none selection:bg-primary selection:text-primary-foreground file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm dark:bg-input/30", "", "aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40", n.class);
		} }, r), !1, !1), e;
	})();
}
//#endregion
//#region src/shared/ui/framed-panel.tsx
var Qt = /* @__PURE__ */ v("<div data-slot=framed-panel>");
function $t(e) {
	let [n, r] = t(e, [
		"class",
		"variant",
		"children"
	]), i = () => n.variant ?? "stone";
	return (() => {
		var e = Qt();
		return C(e, a({
			get "data-variant"() {
				return i();
			},
			get class() {
				return $("grimorio-frame p-5", i() === "stone" ? "grimorio-frame--stone text-foreground" : "grimorio-parchment-bg", n.class);
			}
		}, r), !1, !0), w(e, () => n.children), e;
	})();
}
//#endregion
//#region src/shared/ui/section-label.tsx
var en = Oe("uppercase", {
	variants: {
		papel: {
			titulo: "font-heading text-lg",
			secao: "text-2xs font-semibold tracking-[0.16em]",
			campo: "text-3xs tracking-widest"
		},
		contexto: {
			cena: "",
			painel: ""
		},
		tom: {
			gold: "text-grimorio-gold",
			muted: "text-muted-foreground",
			inherit: ""
		}
	},
	compoundVariants: [{
		papel: "titulo",
		contexto: "cena",
		class: "tracking-[0.16em]"
	}, {
		papel: "titulo",
		contexto: "painel",
		class: "tracking-wide"
	}],
	defaultVariants: {
		papel: "secao",
		tom: "muted",
		contexto: "cena"
	}
});
function tn(e) {
	let [n, r] = t(e, [
		"as",
		"tom",
		"contexto",
		"class",
		"papel",
		"padrao",
		"children"
	]);
	return c(F, a({
		get component() {
			return n.as ?? n.padrao;
		},
		get class() {
			return $(en({
				papel: n.papel,
				tom: n.tom,
				contexto: n.contexto
			}), n.class);
		}
	}, r, { get children() {
		return n.children;
	} }));
}
function nn(e) {
	return c(tn, a(e, {
		papel: "campo",
		padrao: "span"
	}));
}
//#endregion
//#region src/shared/ui/vital-bar.tsx
var rn = /* @__PURE__ */ v("<div><div role=progressbar aria-valuemin=0 class=\"h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-muted\"><div class=\"h-full rounded-full transition-[width]\"></div></div><span class=\"shrink-0 font-mono text-[13px] tabular-nums text-foreground\">/");
function an(e) {
	return e <= 25 ? "--hp-critical" : e <= 50 ? "--hp-hurt" : "--hp-full";
}
function on(e) {
	let t = an(e);
	return t === "--hp-critical" ? "--grimorio-crimson-bright" : t;
}
function sn(e) {
	let t = () => e.max > 0 ? Math.max(0, Math.min(100, e.current / e.max * 100)) : 0, n = () => e.kind === "hp" ? an(t()) : "--mp-arcane", r = () => e.kind === "hp" ? on(t()) : "--mp-arcane";
	return (() => {
		var i = rn(), a = i.firstChild, s = a.firstChild, l = a.nextSibling, u = l.firstChild;
		return w(i, c(nn, {
			tom: "inherit",
			class: "w-7 shrink-0 font-bold",
			get style() {
				return { color: `var(${r()})` };
			},
			get children() {
				return e.label;
			}
		}), a), w(l, () => e.current, u), w(l, () => e.max, null), o((r) => {
			var o = $("flex items-center gap-1.5", e.class), c = e.current, l = e.max, u = `${e.label} ${e.current} de ${e.max}`, d = `${t()}%`, f = `var(${n()})`;
			return o !== r.e && b(i, r.e = o), c !== r.t && y(a, "aria-valuenow", r.t = c), l !== r.a && y(a, "aria-valuemax", r.a = l), u !== r.o && y(a, "aria-label", r.o = u), d !== r.i && S(s, "width", r.i = d), f !== r.n && S(s, "background", r.n = f), r;
		}, {
			e: void 0,
			t: void 0,
			a: void 0,
			o: void 0,
			i: void 0,
			n: void 0
		}), i;
	})();
}
V("spa-botao", {
	variante: "default",
	tamanho: "default",
	texto: "Abrir"
}, (e) => (B(), c(Yt, {
	get variant() {
		return e.variante;
	},
	get size() {
		return e.tamanho;
	},
	get children() {
		return e.texto;
	}
}))), V("spa-chip", {
	variante: "default",
	texto: "ativo"
}, (e) => (B(), c(Kt, {
	get variant() {
		return e.variante;
	},
	get children() {
		return e.texto;
	}
}))), V("spa-campo", {
	placeholder: "",
	desabilitado: !1
}, (e) => (B(), c(Zt, {
	get placeholder() {
		return e.placeholder;
	},
	get disabled() {
		return e.desabilitado;
	}
}))), V("spa-painel", { titulo: "" }, (e) => (B(), c($t, {
	get title() {
		return e.titulo;
	},
	children: "conteúdo"
}))), V("spa-barra-vital", {
	tipo: "hp",
	rotulo: "Vida",
	atual: "42",
	max: "57"
}, (e) => (B(), c(sn, {
	get kind() {
		return e.tipo;
	},
	get label() {
		return e.rotulo;
	},
	get current() {
		return Number(e.atual);
	},
	get max() {
		return Number(e.max);
	}
})));
//#endregion
