package board

import (
	"strconv"
	"strings"

	"t20engine/domain/live"
)

// ── como uma peça se PARECE ─────────────────────────────────────────────────

// TokenAppearance é o que o desenho precisa saber sobre uma peça.
type TokenAppearance struct {
	// Monogram tem SEMPRE duas letras. Não é o `initials` da casa, que devolve
	// uma letra para nome de uma palavra: no retrato do herói uma letra grande
	// funciona, mas no tabuleiro a peça é um disco cheio de vizinhos e um "O"
	// solto tem metade da massa que ela precisa para ser achada num relance.
	Monogram string
	// Instance é o número do selo, vazio quando não há.
	Instance string
	// Hue é 0..359, derivado da ESPÉCIE.
	Hue int
}

// AppearanceOf traduz o rótulo da peça em como ela se desenha.
//
// A REGRA que isto carrega: a cor é da ESPÉCIE e o número é da INSTÂNCIA. Com o
// matiz vindo do rótulo inteiro, "Zumbi 1" e "Zumbi 2" — a mesma criatura —
// saem em cores sem relação nenhuma, e "Zumbi 3" pode calhar na cor do
// paladino: a cor diria "coisas diferentes" sobre coisas iguais.
//
// "Eu ataco o Zumbi 3" é a frase mais dita da noite, e ela tem resposta num
// relance — inclusive para quem não distingue matiz, porque o selo é TEXTO.
func AppearanceOf(label string) TokenAppearance {
	species, number := live.Species(label)
	a := TokenAppearance{Monogram: monogramOf(species), Hue: hueOf(species)}
	if number > 0 {
		a.Instance = strconv.Itoa(number)
	}
	return a
}

func monogramOf(species string) string {
	words := strings.Fields(species)
	if len(words) == 0 {
		return "?"
	}
	if len(words) == 1 {
		return strings.ToUpper(firstTwoRunes(words[0]))
	}
	return strings.ToUpper(firstRune(words[0]) + firstRune(words[1]))
}

// As letras saem por RUNA e não por byte: "Ácido" começa com dois bytes, e
// cortar por índice devolveria meia letra.
func firstTwoRunes(s string) string {
	r := []rune(s)
	if len(r) > 2 {
		r = r[:2]
	}
	return string(r)
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

// hueOf é um hash de 31, e ele tem de continuar sendo O MESMO do retrato do
// herói: as duas telas mostram a mesma criatura, e duas fórmulas dariam duas
// cores para ela.
//
// Percorre por RUNA e usa o ponto de código, como o `for ch of name` do
// JavaScript — iterar bytes daria outro número em todo nome acentuado.
func hueOf(name string) int {
	var hash uint32
	for _, r := range name {
		hash = hash*31 + uint32(r)
	}
	return int(hash % 360)
}

// Place é uma cena guardada da crônica.
//
// O que a mesa chama de "lugar" é o tabuleiro CONGELADO: a taverna com as nove
// peças onde ficaram, para reabrir na semana seguinte sem remontar nada.
type Place struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// Tokens é só a CONTAGEM: a lista serve para escolher onde jogar, e mandar
	// a cena inteira de cada lugar seria mandar o acervo do mestre a cada
	// abertura de menu. A cena chega ao reabrir.
	Tokens    int    `json:"tokens"`
	UpdatedAt string `json:"updatedAt"`
}
