package ui

// O CRACHÁ da casa: a pílula pequena que LIGA E DESLIGA ou abre um detalhe.
//
// Treino da perícia, atributo com que ela rola, magia preparada, aprimoramento
// trocado, categoria da mochila, "3 mods" de um efeito, o atributo que a raça
// concede. SETE sítios na ficha, uma forma — mais o exemplo da folha de
// especificação, que é o oitavo chamador de propósito: peça sem par ali é peça
// que ninguém compara.
//
// # O piso de 24px é a razão desta receita existir (ALE-177)
//
// A issue dizia que 56% dos alvos da ficha reprovavam o WCAG 2.5.8 no telefone.
// Medido na ficha em Datastar, a 390px, nas sete abas e em dois heróis — um que
// conjura e um que não —, **reprovavam quatro**, e os quatro eram este crachá:
// os três de categoria na Mochila e o "preparada" nas Magias, todos com 21px de
// altura.
//
// Os outros 54 alvos pequenos passam, e por duas exceções que a norma escreve e
// que contar tamanho não enxerga:
//
//   - ESPAÇAMENTO — um alvo menor que 24px passa se um círculo de 24px de
//     diâmetro centrado nele não cruzar outro alvo. As linhas de perícia têm 62px
//     de passo, então sobra folga.
//   - EQUIVALENTE — o número da perícia é `size-11` (44×44) e dispara o MESMO
//     `$detalhe` que o nome de 139×20 ao lado. Um dos dois cumprindo os 24px
//     basta, e é o caso.
//
// Isso importa para quem for mexer: **este app troca alvo por densidade**, e a
// ficha é densa de propósito. O piso é 24 e não 44 porque 24 é o que a norma
// pede — os 44 do HIG da Apple custariam linha visível na tela de auditoria de
// números, e a medição não pediu isso.
//
// # Por que é uma receita e não cinco `class=` iguais
//
// Os sete escreviam a mesma pílula à mão, e um deles — o "Trocar" do
// aprimoramento — já tinha ganhado `min-h-7` sozinho. Alguém consertou um e não
// varreu os irmãos, que é como esta família anda, e por isso os outros seis
// ficaram a 21px sem ninguém notar. Com a receita,
// o `TestNoHandwrittenBadgeRecipe` cobra o crachá que nascer amanhã, e a
// cobertura volta a ser AMOSTRAGEM em vez de uma lista de sítios conhecidos.
//
// Ela carrega a GEOMETRIA e nada mais — piso, forma, respiro e anel de foco. A
// cor de ligado/desligado e a caixa alta ficam com quem chama, porque elas
// dizem o que aquele crachá específico significa e não são a mesma coisa nos
// cinco.
//
// O DISPLAY também fica de fora, e não é esquecimento: seis dos sítios são
// `<button>` e querem `inline-flex`, e o sétimo é o `<select>` do atributo, em
// que forçar `inline-flex` mexe na caixa que o navegador desenha sozinho. (Era
// `h-6` fixo antes, e virou o `min-h-6` da receita: passou a crescer se o texto
// da opção crescer, em vez de cortar.)
//
// # E por que ela mora num `.go` e não num `.templ`
//
// Porque ela é só uma função. Um `.templ` sem componente nenhum gera um import
// de `templ` que não se usa, e o pacote deixa de compilar. O Tailwind lê o `.go`
// pela auto-detecção — conferido com um canário posto no arquivo e presente na
// folha compilada, com e sem `@source` —, então nenhuma linha nova de CSS foi
// precisa.
//
// @example ui.BadgeClasses("inline-flex items-center gap-1 text-3xs uppercase")
func BadgeClasses(extra string) string {
	return Join(
		"min-h-6 rounded-full border px-2 outline-none transition-colors",
		"focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
		extra)
}
