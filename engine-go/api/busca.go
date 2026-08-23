package api

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// A REGRA DA BUSCA das listas (ALE-234).
//
// Ela vem do `fuzzy-filter.ts` da SPA, que embrulhava o
// `@tanstack/match-sorter-utils`, e as duas propriedades dele são as que
// importam para este app:
//
//   - INSENSÍVEL A ACENTO. O domínio é pt-BR e ninguém digita "Anão" com til
//     numa busca apressada no meio da sessão.
//   - TOLERANTE A TYPO. O comentário do arquivo original diz, com todas as
//     letras, que essa é "a parte que de fato importava" — então descartá-la
//     na migração seria regressão, não simplificação.
//
// O que NÃO veio junto é o RANQUEAMENTO. O `rankItem` devolve uma pontuação, e
// a SPA jogava fora: ela só lia `.passed`. Portar a pontuação seria portar
// código que ninguém lê.

// combina é a transformação que dobra acento: decompõe em base + marca e joga
// as marcas fora. Construída UMA vez porque ela é cara de montar e não tem
// estado.
var combina = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// dobra normaliza para comparação: minúsculas e sem acento.
//
// "Anão" e "anao" viram a mesma coisa, que é o ponto.
func dobra(s string) string {
	limpo, _, err := transform.String(combina, s)
	if err != nil {
		// A transformação só falha em entrada mal formada; comparar o original
		// é pior que nada, mas é melhor que a busca inteira parar de funcionar.
		return strings.ToLower(s)
	}
	return strings.ToLower(limpo)
}

// casaBusca responde se ALGUM dos campos casa com o que foi digitado.
//
// Busca vazia casa com tudo: não digitar não é filtrar.
func casaBusca(campos []string, busca string) bool {
	alvo := dobra(strings.TrimSpace(busca))
	if alvo == "" {
		return true
	}
	for _, campo := range campos {
		if casaUmCampo(dobra(campo), alvo) {
			return true
		}
	}
	return false
}

// casaUmCampo aplica as duas regras, da mais barata para a mais cara.
func casaUmCampo(campo, alvo string) bool {
	if strings.Contains(campo, alvo) {
		return true
	}
	return ehSubsequencia(campo, alvo)
}

// ehSubsequencia é a tolerância a typo, e ela é DELIBERADAMENTE frouxa numa
// direção só: aceita letras faltando ("ncromante" acha "Necromante"), não
// aceita letras trocadas nem sobrando.
//
// Isso cobre o typo que se comete de verdade digitando rápido — pular uma
// tecla — sem abrir a porta para "xyz" casar com tudo. Uma distância de edição
// completa aceitaria substituição e transposição, e numa lista de seis
// campanhas isso faria a busca devolver a lista inteira com frequência.
//
// A busca de UMA letra exige prefixo: com subsequência, "a" casaria com
// qualquer nome que tenha um "a" em qualquer lugar, que é a lista toda.
func ehSubsequencia(campo, alvo string) bool {
	if len([]rune(alvo)) < 2 {
		return strings.HasPrefix(campo, alvo)
	}
	restante := []rune(alvo)
	for _, r := range campo {
		if r == restante[0] {
			restante = restante[1:]
			if len(restante) == 0 {
				return true
			}
		}
	}
	return false
}
