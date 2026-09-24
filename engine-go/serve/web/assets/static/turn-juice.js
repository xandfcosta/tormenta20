//#region serve/web/assets/src/lib/turn-juice.ts
function e(e) {
	return !!e && typeof e.animate == "function";
}
function t(t, n) {
	if (!e(t)) return;
	let r = t.getBoundingClientRect();
	if (r.width === 0 || r.height === 0) return;
	let i = document.createElement("div");
	i.setAttribute("aria-hidden", "true"), i.setAttribute("data-vital-blink", n.curou ? "curou" : "feriu"), i.style.cssText = [
		"position:fixed",
		`top:${r.top}px`,
		`left:${r.left}px`,
		`width:${r.width}px`,
		`height:${r.height}px`,
		`border-radius:${getComputedStyle(t).borderRadius}`,
		"pointer-events:none",
		"z-index:40",
		`background:var(${n.curou ? "--hp-full" : "--hp-critical"})`
	].join(";"), document.body.appendChild(i), i.animate([{ opacity: .45 }, { opacity: 0 }], {
		duration: 380,
		easing: "ease-out"
	}).finished.then(() => i.remove()).catch(() => i.remove());
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
function r(t) {
	e(t) && t.animate([{
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
export { n as i, t as n, e as r, r as t };
