package ui

// BadgeClasses é a GEOMETRIA do crachá da casa: a pílula pequena que liga e
// desliga ou abre um detalhe.
//
// Ela carrega piso, forma, respiro e anel de foco, e nada mais. A cor de
// ligado/desligado e a caixa alta ficam com quem chama, porque dizem o que
// aquele crachá significa e não são a mesma coisa nos sete sítios.
//
// O piso é 24px e não os 44 do HIG da Apple: 24 é o que o WCAG 2.5.8 pede, e
// este app troca alvo por densidade de propósito — a ficha é densa (ALE-177).
//
// O DISPLAY fica de fora porque um dos sítios é um `<select>`, e forçar
// `inline-flex` nele mexe na caixa que o navegador desenha sozinho.
//
// E ela mora num `.go` e não num `.templ` porque é só uma função: um `.templ`
// sem componente nenhum gera um import de `templ` que não se usa, e o pacote
// deixa de compilar.
//
// @example ui.BadgeClasses("inline-flex items-center gap-1 text-3xs uppercase")
func BadgeClasses(extra string) string {
	return Join(
		"min-h-6 rounded-full border px-2 outline-none transition-colors",
		extra)
}
