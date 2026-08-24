//#region src/features/session-tracker/turn-juice.ts
function e(e) {
	return !!e && typeof e.animate == "function";
}
function t(t, n) {
	if (!e(t)) return;
	let r = document.createElement("div");
	r.setAttribute("aria-hidden", "true"), r.style.cssText = [
		"position:absolute",
		"inset:0",
		"border-radius:inherit",
		"pointer-events:none",
		`background:var(${n.curou ? "--hp-full" : "--hp-critical"})`
	].join(";"), t.appendChild(r), r.animate([{ opacity: .45 }, { opacity: 0 }], {
		duration: 380,
		easing: "ease-out"
	}).finished.then(() => r.remove()).catch(() => r.remove());
}
function n(t) {
	e(t) && t.animate([
		{
			transform: "scale(1)",
			boxShadow: "0 0 0 0 transparent"
		},
		{
			transform: "scale(1.015)",
			boxShadow: "0 0 14px 2px color-mix(in oklch, var(--grimorio-gold) 45%, transparent)",
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
//#endregion
//#region src/piloto/grimorio.ts
function r(e) {
	return e.closest("figure, [data-par]")?.querySelector("[data-amostra]") ?? null;
}
function i(e) {
	let t = document.createElement("canvas");
	t.width = 1, t.height = 1;
	let n = t.getContext("2d"), r = document.querySelector(".scene-grimorio");
	if (!n || !r) return null;
	let i = (e) => {
		n.clearRect(0, 0, 1, 1), n.fillStyle = e, n.fillRect(0, 0, 1, 1);
		let [t, r, i] = n.getImageData(0, 0, 1, 1).data;
		return [
			t ?? 0,
			r ?? 0,
			i ?? 0
		];
	}, a = ([e, t, n]) => {
		let [r, i, a] = [
			e,
			t,
			n
		].map((e) => {
			let t = e / 255;
			return t <= .04045 ? t / 12.92 : ((t + .055) / 1.055) ** 2.4;
		});
		return .2126 * (r ?? 0) + .7152 * (i ?? 0) + .0722 * (a ?? 0);
	}, o = getComputedStyle(r).getPropertyValue("--grimorio-panel").trim(), [s, c] = [a(i(e)), a(i(o))].sort((e, t) => t - e);
	return Number((((s ?? 0) + .05) / ((c ?? 0) + .05)).toFixed(2));
}
function a() {
	for (let e of document.querySelectorAll("[data-medir]")) {
		let t = r(e), n = e.dataset.medir;
		!t || !n || (e.textContent = getComputedStyle(t).getPropertyValue(n).trim() || "—");
	}
}
function o() {
	for (let e of document.querySelectorAll("[data-contraste]")) {
		let t = r(e);
		if (!t) continue;
		let n = i(getComputedStyle(t).backgroundColor);
		if (n === null) continue;
		let a = n >= 4.5;
		e.textContent = a ? `${n}:1 no painel` : `${n}:1 — só bloco, não texto`, e.classList.toggle("text-grimorio-gold", !a), e.classList.toggle("font-bold", !a), e.classList.toggle("text-muted-foreground", a);
	}
}
async function s() {
	let e = new Set([...document.querySelectorAll("[data-amostra-cela] *")].map((e) => e.tagName.toLowerCase()).filter((e) => e.includes("-")));
	await Promise.all([...e].map((e) => customElements.whenDefined(e))), await new Promise((e) => requestAnimationFrame(() => e(null))), c();
}
function c() {
	for (let e of document.querySelectorAll("[data-medir-cela]")) {
		let t = e.previousElementSibling?.querySelector("button, input, [role=\"progressbar\"]");
		if (!t) continue;
		let n = t.getBoundingClientRect(), r = getComputedStyle(t).borderRadius;
		e.textContent = `h ${Math.round(n.height)} · w ${Math.round(n.width)} · r ${r}`;
	}
}
function l() {
	let e = document.querySelector("[data-linha-iniciativa]"), r = window.matchMedia("(prefers-reduced-motion: reduce)"), i = () => {
		let e = document.querySelector("[data-movimento-reduzido]");
		e && (e.textContent = r.matches ? "LIGADO" : "desligado");
	};
	i(), r.addEventListener("change", i);
	for (let i of document.querySelectorAll("[data-disparar]")) i.addEventListener("click", () => {
		if (r.matches || !e) return;
		let a = i.dataset.disparar;
		a === "ferir" && t(e, { curou: !1 }), a === "curar" && t(e, { curou: !0 }), a === "vez" && n(e);
	});
}
function u() {
	a(), o(), s(), l();
}
u();
//#endregion
export { u as medeAFolha };
