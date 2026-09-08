import { r as e, t } from "./turn-juice.js";
//#region api/piloto/src/grimorio.ts
function n(e) {
	return e.closest("figure, [data-par]")?.querySelector("[data-amostra]") ?? null;
}
function r(e) {
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
function i() {
	for (let e of document.querySelectorAll("[data-medir]")) {
		let t = n(e), r = e.dataset.medir;
		!t || !r || (e.textContent = getComputedStyle(t).getPropertyValue(r).trim() || "—");
	}
}
function a() {
	for (let e of document.querySelectorAll("[data-contraste]")) {
		let t = n(e);
		if (!t) continue;
		let i = r(getComputedStyle(t).backgroundColor);
		if (i === null) continue;
		let a = i >= 4.5;
		e.textContent = a ? `${i}:1 no painel` : `${i}:1 — só bloco, não texto`, e.classList.toggle("text-grimorio-gold", !a), e.classList.toggle("font-bold", !a), e.classList.toggle("text-muted-foreground", a);
	}
}
async function o() {
	let e = new Set([...document.querySelectorAll("[data-amostra-cela] *")].map((e) => e.tagName.toLowerCase()).filter((e) => e.includes("-")));
	await Promise.all([...e].map((e) => customElements.whenDefined(e))), await new Promise((e) => requestAnimationFrame(() => e(null))), s();
}
function s() {
	for (let e of document.querySelectorAll("[data-medir-cela]")) {
		let t = e.previousElementSibling?.querySelector("button, input, [role=\"progressbar\"]");
		if (!t) continue;
		let n = t.getBoundingClientRect(), r = getComputedStyle(t).borderRadius;
		e.textContent = `h ${Math.round(n.height)} · w ${Math.round(n.width)} · r ${r}`;
	}
}
function c() {
	let n = document.querySelector("[data-linha-iniciativa]"), r = window.matchMedia("(prefers-reduced-motion: reduce)"), i = () => {
		let e = document.querySelector("[data-movimento-reduzido]");
		e && (e.textContent = r.matches ? "LIGADO" : "desligado");
	};
	i(), r.addEventListener("change", i);
	for (let i of document.querySelectorAll("[data-disparar]")) i.addEventListener("click", () => {
		if (r.matches || !n) return;
		let a = i.dataset.disparar;
		a === "ferir" && t(n, { curou: !1 }), a === "curar" && t(n, { curou: !0 }), a === "vez" && e(n);
	});
}
function l() {
	i(), a(), o(), c();
}
l();
//#endregion
export { l as medeAFolha };
