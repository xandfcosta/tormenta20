package ui

import (
	"fmt"
	"strings"
)

// A IDENTIDADE VISUAL derivada do NOME (ALE-234), portada de
// `shared/lib/hue-from-name.ts` e `shared/lib/initials.ts`.
//
// O app não tem campo de imagem para campanha nem para personagem: o que faz
// cada um ser reconhecível é um monograma sobre um gradiente cuja cor vem do
// NOME. Por isso a fórmula tem de ser a mesma nos dois lados até o último
// dígito — um matiz diferente faz a mesma campanha ter duas capas, e a pessoa
// que abre a tela nova acha que abriu outra mesa.
//
// # Por que no KIT, e não numa cena
//
// Isto morava no `api` e CINCO famílias liam de lá — campanhas, a crônica, a
// entrada na campanha, personagens e a mesa (ALE-278). Enquanto ficasse ali,
// cada cena que se mudasse levaria o hospedeiro junto.
//
// O kit é a casa certa porque nada aqui sabe do DOMÍNIO: são três funções de
// `string` para `string`. O que ficou para trás na mudança foi justamente a
// quarta função do arquivo original, o `papelNaCampanha` — "Mestrando",
// "Jogando", "Mesa de X" é regra de quem é o quê numa campanha, e o kit não
// pode saber disso.

// NameHue é o hash de 31 da SPA, e cada detalhe dele é para casar com o JS.
//
// O original itera por CODE POINT (`for...of`) mas lê `charCodeAt(0)`, que é a
// primeira unidade UTF-16. Para tudo no BMP — que inclui todo o pt-BR
// acentuado — os dois são o mesmo número, e o `range` do Go, que dá runas,
// concorda. Um nome com emoji divergiria; nenhum nome de campanha tem, e
// quando tiver isto aqui é o lugar de olhar.
//
// O `>>> 0` do JS é truncamento para 32 bits sem sinal. Em Go o `uint32` já
// envolve sozinho, e o resultado é o mesmo porque multiplicação e soma
// distribuem sobre o módulo.
func NameHue(nome string) int {
	var hash uint32
	for _, r := range nome {
		hash = hash*31 + uint32(r)
	}
	return int(hash % 360)
}

// NameGradient é a capa: o mesmo gradiente de 155° do `hueGradient`, com a
// claridade e o croma que a SPA usa para EMBLEMA (o retrato de personagem é um
// fio mais claro, e essa diferença é deliberada lá).
//
// Ele se chamava `gradienteDaCampanha` e o nome MENTIA: o cartão do herói o
// chama com o nome do personagem desde a ALE-239. O que ele deriva é um nome,
// qualquer nome — que é o que o nome novo diz.
func NameGradient(nome string) string {
	m := NameHue(nome)
	return fmt.Sprintf(
		"linear-gradient(155deg, oklch(0.5 0.14 %d) 0%%, oklch(0.30 0.09 %d) 70%%, oklch(0.22 0.06 %d) 100%%)",
		m, m, m,
	)
}

// Monogram é o monograma de até duas letras.
func Monogram(nome string) string {
	partes := strings.Fields(nome)
	if len(partes) == 0 {
		return "?"
	}
	if len(partes) > 2 {
		partes = partes[:2]
	}
	var b strings.Builder
	for _, parte := range partes {
		for _, r := range parte {
			b.WriteString(strings.ToUpper(string(r)))
			break
		}
	}
	return b.String()
}
