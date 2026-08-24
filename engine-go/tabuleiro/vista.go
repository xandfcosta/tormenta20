package tabuleiro

import (
	"strconv"
	"strings"

	"t20engine/aovivo"
)

// O QUE SE DESENHA DO TABULEIRO (ALE-263).
//
// O plano é INFINITO, e a SPA resolve isso desenhando só a JANELA que o mestre
// enquadrou — origem fracionária, recalculada a cada pixel arrastado. No piloto
// quem desenha é o SERVIDOR, e repetir aquele modelo custaria uma ida ao
// servidor por passo de arrasto, com o remendo do SSE disputando com o dedo.
//
// A escolha do dono foi outra e ela inverte a divisão: o servidor desenha o
// EXTENSO — o retângulo que contém tudo que existe, mais margem — e o
// enquadramento (deslizar e ampliar) vira `transform` no navegador. Arrastar
// deixa de custar pedido, e o remendo do SSE não perde o enquadramento porque
// ele não está no HTML.
//
// O que sustenta a troca é o TETO DA MESA: o servidor limita a sessão a 50
// combatentes, então "tudo que existe" é pequeno. O infinito não some — ele
// deixa de precisar ser desenhado.

// MargemDaMoldura são os quadrados vazios em volta do que existe.
//
// Três é o deslocamento de meio turno (9m = 6 quadrados, p106): o mestre vê para
// onde a peça da borda pode andar sem o servidor precisar ampliar nada. Zero
// deixaria a peça colada na borda, e um plano que acaba onde a peça está mente
// sobre um plano que não acaba.
const MargemDaMoldura = 3

// O piso da vista, em quadrados. São os mesmos números que a SPA usa como
// janela padrão, e o motivo de existirem aqui é o mesmo de lá: um tabuleiro
// recém-aberto não tem nada dentro, e uma moldura de 0×0 desenharia o nada.
const (
	MinimoDeColunas = 20
	MinimoDeLinhas  = 14
)

// Moldura é o retângulo que o servidor desenha, em quadrados do plano.
//
// X0 e Y0 podem ser NEGATIVOS, e isso é o plano infinito sobrevivendo à
// mudança: coordenada negativa é lugar legítimo, e o rótulo do eixo usa o número
// COM SINAL que o servidor guarda — num plano sem bordas, o "+1" de planilha
// mentiria sobre onde a peça está.
type Moldura struct {
	X0, Y0          int
	Colunas, Linhas int
}

// MolduraDe mede o que existe no tabuleiro e devolve o retângulo a desenhar.
//
// Entram as PEÇAS (com a pegada, que é o que ocupa mais de um quadrado), os
// MARCADORES e o terreno difícil — tudo que tem lugar. O que está escondido
// entra também: o mestre desenha a moldura a partir do estado dele, e a redação
// para o jogador acontece antes, no `StateForRole`.
func MolduraDe(b *BoardState) Moldura {
	if b == nil {
		return molduraCentradaEm(0, 0, 0, 0)
	}
	primeiro := true
	var minX, minY, maxX, maxY int
	cobre := func(x, y, tamanho int) {
		if primeiro {
			minX, minY, maxX, maxY = x, y, x+tamanho-1, y+tamanho-1
			primeiro = false
			return
		}
		minX, minY = min(minX, x), min(minY, y)
		maxX, maxY = max(maxX, x+tamanho-1), max(maxY, y+tamanho-1)
	}
	for i := range b.Tokens {
		// Pegada zero é peça de um quadrado: o campo é `omitempty` no fio, e
		// tratá-la como zero encolheria a moldura para fora da própria peça.
		pegada := b.Tokens[i].Footprint
		if pegada < 1 {
			pegada = 1
		}
		cobre(b.Tokens[i].X, b.Tokens[i].Y, pegada)
	}
	for i := range b.Markers {
		cobre(b.Markers[i].X, b.Markers[i].Y, 1)
	}
	for i := range b.Difficult {
		cobre(b.Difficult[i].X, b.Difficult[i].Y, 1)
	}
	if primeiro {
		return molduraCentradaEm(0, 0, 0, 0)
	}
	return molduraCentradaEm(minX-MargemDaMoldura, minY-MargemDaMoldura,
		maxX-minX+1+2*MargemDaMoldura, maxY-minY+1+2*MargemDaMoldura)
}

// molduraCentradaEm cresce o retângulo até o piso, mantendo o conteúdo no MEIO.
//
// Crescer só para a direita e para baixo empurraria o combate para a quina de um
// tabuleiro vazio, e o mestre teria de deslizar para achar as próprias peças.
func molduraCentradaEm(x0, y0, colunas, linhas int) Moldura {
	if colunas < MinimoDeColunas {
		x0 -= (MinimoDeColunas - colunas) / 2
		colunas = MinimoDeColunas
	}
	if linhas < MinimoDeLinhas {
		y0 -= (MinimoDeLinhas - linhas) / 2
		linhas = MinimoDeLinhas
	}
	return Moldura{X0: x0, Y0: y0, Colunas: colunas, Linhas: linhas}
}

// ── como uma peça se PARECE ─────────────────────────────────────────────────

// AparenciaDaPeca é o que o desenho precisa saber sobre uma peça.
type AparenciaDaPeca struct {
	// Monograma tem SEMPRE duas letras. Não é o `initials` da casa, que devolve
	// uma letra para nome de uma palavra: no retrato do herói uma letra grande
	// funciona, mas no tabuleiro a peça é um disco cheio de vizinhos e um "O"
	// solto tem metade da massa que ela precisa para ser achada num relance.
	Monograma string
	// Instancia é o número do selo, vazio quando não há.
	Instancia string
	// Matiz é 0..359, derivado da ESPÉCIE.
	Matiz int
}

// AparenciaDe traduz o rótulo da peça em como ela se desenha (ALE-179).
//
// A REGRA que isto carrega: a cor é da ESPÉCIE e o número é da INSTÂNCIA. Antes
// o matiz vinha do rótulo inteiro, então "Zumbi 1" e "Zumbi 2" — a mesma
// criatura — saíam em cores sem relação nenhuma, e "Zumbi 3" podia calhar na cor
// do paladino: a cor dizia "coisas diferentes" sobre coisas iguais.
//
// "Eu ataco o Zumbi 3" é a frase mais dita da noite, e ela passa a ter resposta
// num relance — inclusive para quem não distingue matiz, porque o selo é TEXTO.
func AparenciaDe(rotulo string) AparenciaDaPeca {
	especie, numero := aovivo.Especie(rotulo)
	a := AparenciaDaPeca{Monograma: monogramaDe(especie), Matiz: matizDe(especie)}
	if numero > 0 {
		a.Instancia = strconv.Itoa(numero)
	}
	return a
}

func monogramaDe(especie string) string {
	palavras := strings.Fields(especie)
	if len(palavras) == 0 {
		return "?"
	}
	if len(palavras) == 1 {
		return strings.ToUpper(duasPrimeirasRunas(palavras[0]))
	}
	return strings.ToUpper(primeiraRuna(palavras[0]) + primeiraRuna(palavras[1]))
}

// As letras saem por RUNA e não por byte: "Ácido" começa com dois bytes, e
// cortar por índice devolveria meia letra.
func duasPrimeirasRunas(s string) string {
	r := []rune(s)
	if len(r) > 2 {
		r = r[:2]
	}
	return string(r)
}

func primeiraRuna(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

// matizDe é o mesmo hash de 31 do `hueFromName` da SPA, e ele tem de continuar
// sendo O MESMO: a peça do tabuleiro e o retrato do herói mostram a mesma
// criatura, e duas fórmulas dariam duas cores para ela em duas telas.
//
// Percorre por RUNA e usa o ponto de código, como o `for ch of name` do
// JavaScript — iterar bytes daria outro número em todo nome acentuado.
func matizDe(nome string) int {
	var hash uint32
	for _, r := range nome {
		hash = hash*31 + uint32(r)
	}
	return int(hash % 360)
}
