package tabuleiro

import (
	"strconv"
	"strings"

	"t20engine/aovivo"
)

// COMO UMA PEÇA SE PARECE.
//
// Este arquivo teve uma segunda metade — a MOLDURA (ALE-263), o retângulo que o
// servidor desenhava porque ele continha tudo o que existe, mais margem. Ela
// saiu inteira na ALE-203, com os testes dela.
//
// A razão de guardar isto escrito: a moldura era uma resposta razoável a uma
// pergunta real ("como desenhar um plano infinito de um servidor?"), e ela
// FALHOU de um jeito que só aparece no uso. Ela CRESCIA — pintar perto da borda
// mexia no `X0`, e o mesmo ponto da tela virava outro quadrado entre dois
// cliques (medido na bancada: de -11 para -12). E ela era uma caixa: fora dela
// não havia onde clicar, então pintar longe do grupo exigia primeiro que ela
// crescesse até lá.
//
// O que entrou no lugar é a divisão que o Excalidraw usa: o servidor manda o que
// EXISTE em coordenada absoluta, e o NAVEGADOR recorta com uma janela que nunca
// vai ao servidor (`api/piloto_mesa_janela.go`). O infinito parou de precisar de
// um retângulo que o contivesse.

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
