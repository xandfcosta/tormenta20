package api

import (
	"fmt"
	"strings"
)

// A IDENTIDADE VISUAL derivada do nome (ALE-234), portada de
// `shared/lib/hue-from-name.ts` e `shared/lib/initials.ts`.
//
// O app não tem campo de imagem para campanha nem para personagem: o que faz
// cada um ser reconhecível é um monograma sobre um gradiente cuja cor vem do
// NOME. Por isso a fórmula tem de ser a mesma nos dois lados até o último
// dígito — um matiz diferente faz a mesma campanha ter duas capas, e a pessoa
// que abre a tela nova acha que abriu outra mesa.

// matizDoNome é o hash de 31 da SPA, e cada detalhe dele é para casar com o JS.
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
func matizDoNome(nome string) int {
	var hash uint32
	for _, r := range nome {
		hash = hash*31 + uint32(r)
	}
	return int(hash % 360)
}

// gradienteDaCampanha é a capa: o mesmo gradiente de 155° do `hueGradient`, com
// a claridade e o croma que a SPA usa para EMBLEMA (o retrato de personagem é
// um fio mais claro, e essa diferença é deliberada lá).
func gradienteDaCampanha(nome string) string {
	m := matizDoNome(nome)
	return fmt.Sprintf(
		"linear-gradient(155deg, oklch(0.5 0.14 %d) 0%%, oklch(0.30 0.09 %d) 70%%, oklch(0.22 0.06 %d) 100%%)",
		m, m, m,
	)
}

// iniciais é o monograma de até duas letras.
func iniciais(nome string) string {
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

// papelNaCampanha é o `roleLabel`: a POSTURA de quem olha.
//
// Uma mesa que é de OUTRA pessoa — que só um admin chega a ver listada — diz de
// quem ela é em vez da postura. O servidor entrega o papel `gm` ali, e escrever
// "Mestrando" faria parecer que a mesa é de quem está lendo (ALE-120).
func papelNaCampanha(papel string, dono *string) string {
	if dono != nil && *dono != "" {
		return "Mesa de " + *dono
	}
	if papel == "gm" {
		return "Mestrando"
	}
	return "Jogando"
}
