//#region api/piloto/src/lib/turn-juice.ts
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
export { e as n, n as r, t };
