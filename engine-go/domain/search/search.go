// Package search é o casamento e a pontuação de busca do app.
//
// Pacote próprio e não uma função dentro de quem busca: o `Fold` — que
// desacentua — é chamado de vários lados, e uma CÓPIA dele que só faça
// `ToLower` devolve "atuação" com acento e desliga a chave que o nome vira. Um
// elo para um endereço que não existe, sem erro, sem panic, sem log.
package search

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// A REGRA DA BUSCA das listas, com as duas propriedades que importam a este app:
//
//   - INSENSÍVEL A ACENTO. O domínio é pt-BR e ninguém digita "Anão" com til
//     numa busca apressada no meio da sessão.
//   - TOLERANTE A TYPO, que é a parte que de fato pega o erro de digitação.
//
// O `Matches` das listas não RANQUEIA: elas perguntam "passou?" e desenham em
// ordem alfabética. Quem precisa de ordem de relevância é o buscador do livro,
// mais abaixo.

// combining é a transformação que dobra acento: decompõe em base + marca e joga
// as marcas fora. Construída UMA vez porque ela é cara de montar e não tem
// estado.
var combining = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// Fold normaliza para comparação: minúsculas e sem acento.
//
// "Anão" e "anao" viram a mesma coisa, que é o ponto.
func Fold(s string) string {
	limpo, _, err := transform.String(combining, s)
	if err != nil {
		// A transformação só falha em entrada mal formada; comparar o original
		// é pior que nada, mas é melhor que a busca inteira parar de funcionar.
		return strings.ToLower(s)
	}
	return strings.ToLower(limpo)
}

// Matches responde se ALGUM dos campos casa com o que foi digitado.
//
// Busca vazia casa com tudo: não digitar não é filtrar.
func Matches(campos []string, busca string) bool {
	alvo := Fold(strings.TrimSpace(busca))
	if alvo == "" {
		return true
	}
	for _, campo := range campos {
		if matchesField(Fold(campo), alvo) {
			return true
		}
	}
	return false
}

// matchesField aplica as duas regras, da mais barata para a mais cara.
func matchesField(campo, alvo string) bool {
	if strings.Contains(campo, alvo) {
		return true
	}
	return isSubsequence(campo, alvo)
}

// isSubsequence é a tolerância a typo, e ela é DELIBERADAMENTE frouxa numa
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
func isSubsequence(campo, alvo string) bool {
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

// ── o RANQUEAMENTO, que só o buscador do livro precisa ───────────────────────
//
// Ele varre o acervo inteiro e mostra SEIS por grupo. Sem ordem de relevância,
// "abal" mostraria as seis primeiras em ordem alfabética entre as que casam, e
// "Abalado" poderia não estar entre elas: cortar sem ranquear é escolher ao
// acaso o que a pessoa vê.

// pontuaBusca mede o quanto o NOME casa com o que foi digitado. Zero é não
// casar; maior é melhor.
//
//	pontuaBusca("Abalado", "abal")      // → 80, prefixo
//	pontuaBusca("Bola de Fogo", "fogo") // → 60, começa uma palavra
//
// Só o nome, e de propósito: casar por descrição não distingue o verbete que a
// pessoa procura do que apenas menciona a palavra. A descrição entra por fora,
// com peso baixo — ver `Score`.
//
// VÁRIOS TERMOS são exigidos TODOS, e a nota é a média. Sem isso "bola fogo"
// não acha "Bola de Fogo": o nome não começa com a frase, não a contém, e pular
// o "de " estoura a folga do quase-igual. Digitar duas palavras de um nome é
// como se procura o que se lembra pela metade.
func Score(nome, busca string) int {
	alvo := Fold(strings.TrimSpace(busca))
	if alvo == "" {
		return 0
	}
	campo := Fold(nome)
	termos := strings.Fields(alvo)
	if len(termos) == 1 {
		return scoreTerm(campo, alvo)
	}
	soma := 0
	for _, termo := range termos {
		ponto := scoreTerm(campo, termo)
		if ponto == 0 {
			return 0
		}
		soma += ponto
	}
	return soma / len(termos)
}

// scoreTerm é a escada, do casamento mais forte para o mais fraco.
func scoreTerm(campo, termo string) int {
	switch {
	case campo == termo:
		return 100
	case strings.HasPrefix(campo, termo):
		return 80
	case startsAWord(campo, termo):
		return 60
	case strings.Contains(campo, termo):
		return 40
	case isNearlyEqual(campo, termo):
		return 20
	}
	return 0
}

// isNearlyEqual é a tolerância a typo APERTADA, e ela existe porque a frouxa não
// serve aqui.
//
// O `isSubsequence` aceita letras faltando em qualquer lugar, e num acervo de
// milhares isso empurra o resultado de verdade para fora do corte de seis:
// "abal" traz "Capitão-Baluarte" e "Suporte Ambiental".
//
// A regra: as letras podem faltar, mas o buraco todo cabe em DUAS. "ncromante"
// continua achando "Necromante" (uma letra pulada), e "abal" para de achar
// "Capitão-Baluarte" (seis). É a diferença entre corrigir um dedo torto e
// aceitar qualquer coisa.
func isNearlyEqual(campo, alvo string) bool {
	letras := []rune(campo)
	procurado := []rune(alvo)
	if len(procurado) < 2 {
		return strings.HasPrefix(campo, alvo)
	}
	for inicio := range letras {
		if letras[inicio] != procurado[0] {
			continue
		}
		if gapUntil(letras[inicio:], procurado) <= 2 {
			return true
		}
	}
	return false
}

// noMatch é o "não casou" que o `gapUntil` devolve, e é uma CONSTANTE e não
// `len(letras)+1`: num resto de UMA letra o calculado dá 2, que está DENTRO da
// folga — "abal" passaria a casar com "Naja" pelo último "a". Sentinela
// calculado a partir da entrada é sentinela que a entrada alcança.
const noMatch = 1 << 30

func gapUntil(letras, procurado []rune) int {
	buraco, i := 0, 0
	for _, r := range letras {
		if r == procurado[i] {
			i++
			if i == len(procurado) {
				return buraco
			}
			continue
		}
		buraco++
		if buraco > 2 {
			break
		}
	}
	return noMatch
}

// startsAWord procura o termo no COMEÇO de qualquer palavra.
//
// É o que põe "Bola de Fogo" acima de "Explosão de Fogo Congelante" quando se
// digita "fogo": as duas contêm o termo, mas numa ele abre a palavra. E é o que
// tira "trabalho" de uma busca por "abal".
//
// Por LIMITE e não por `strings.Fields`: no corpo de uma regra a palavra vem
// colada em pontuação ("(abalado", "abalado,"), e cortar só no espaço deixaria
// esses casos de fora. A letra anterior é decodificada como RUNA porque o
// domínio é pt-BR — um byte solto no meio de "ção" não é letra nenhuma.
func startsAWord(campo, alvo string) bool {
	de := 0
	for {
		onde := strings.Index(campo[de:], alvo)
		if onde < 0 {
			return false
		}
		onde += de
		anterior, _ := utf8.DecodeLastRuneInString(campo[:onde])
		if onde == 0 || !(unicode.IsLetter(anterior) || unicode.IsDigit(anterior)) {
			return true
		}
		de = onde + 1
	}
}

// ScoreText é o último recurso: o termo aparece no CORPO da regra.
//
// Dez é deliberadamente baixo e sem graus: achar "concealment" no efeito de uma
// magia é um acerto de verdade, e ainda assim vale menos que qualquer casamento
// de nome.
//
// TRÊS letras para entrar: com duas, "ab" aparece no corpo de centenas de
// regras. O nome continua buscável desde a primeira letra — é lá que a pessoa
// sabe o que procura.
//
// Todos os termos, e cada um abrindo uma PALAVRA. "Contém" cru aqui é o que
// fazia "abal" achar "trabalho".
func ScoreText(textos []string, busca string) int {
	alvo := Fold(strings.TrimSpace(busca))
	if len([]rune(alvo)) < 3 {
		return 0
	}
	junto := Fold(strings.Join(textos, " "))
	for _, termo := range strings.Fields(alvo) {
		if !startsAWord(junto, termo) {
			return 0
		}
	}
	return 10
}
