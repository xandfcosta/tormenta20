package engine

import "math"

// A CONTA DO ENCONTRO — Cap 7 p282 (ND de grupo e dificuldade) e Cap 8 p326
// (XP por desafio).
//
// Ela mora no `engine` e não no `api` porque é REGRA DO LIVRO, não desenho de
// tela: o construtor de encontros da Mesa e o construtor de dentro da sessão
// respondem a mesma pergunta, e uma segunda cópia é uma cópia que diverge.
//
// Portada da SPA na ALE-259, onde vivia em `shared/lib/encounter-math.ts` e
// `shared/rules/xp.ts`. A cópia de lá FICA enquanto a ferramenta da SPA
// existir — as duas convivem durante a migração, e some quando a `/gm` cair.

// NDDeGrupo é o ND efetivo de N criaturas iguais (p282).
//
//	ND < 1 → ND × quantidade      (quatro de ND 1/4 = ND 1)
//	ND ≥ 1 → ND + 2 a cada DOBRA  (dois de ND 1 = ND 3; quatro de ND 5 = ND 9)
//
// O `Log2` estende a regra para dobras não-inteiras, então um grupo de 3 cai
// entre 1× e 2×. Quem quiser ND inteiro arredonda o resultado — o livro dá a
// regra para dobras exatas e cala sobre o resto.
func NDDeGrupo(ndDaCriatura float64, quantidade int) float64 {
	if quantidade <= 0 {
		return 0
	}
	if ndDaCriatura < 1 {
		return ndDaCriatura * float64(quantidade)
	}
	if quantidade == 1 {
		return ndDaCriatura
	}
	return ndDaCriatura + 2*math.Log2(float64(quantidade))
}

// Dificuldade é a faixa em que o encontro cai, e o TOM é para a cor — nunca
// para o texto, que é o que a tela lê.
type Dificuldade struct {
	Rotulo string
	Tom    string // calmo | parelho | duro | mortal
}

// DificuldadeDoEncontro mapeia a diferença entre o ND do encontro e o nível do
// grupo numa faixa. ND igual ao nível é combate justo — "Médio" (p281).
//
// A diferença costuma ser FRACIONÁRIA — criatura abaixo de ND 1 dá ND de grupo
// fracionário, e a regra da dobra usa log2 —, enquanto as faixas são degraus
// inteiros. Por isso ela é ARREDONDADA primeiro. Sem esse arredondamento, uma
// diferença pequena e negativa como −0,75 (uma criatura de ND 1/4 contra um
// grupo de nível 1) escapava tanto do `<= -1` quanto do `== 0` e caía em
// "Difícil" (ALE-25).
func DificuldadeDoEncontro(diferenca float64) Dificuldade {
	degrau := math.Round(diferenca)
	switch {
	case degrau <= -3:
		return Dificuldade{"Trivial", "calmo"}
	case degrau <= -1:
		return Dificuldade{"Fácil", "calmo"}
	case degrau == 0:
		return Dificuldade{"Médio", "parelho"}
	case degrau <= 2:
		return Dificuldade{"Difícil", "duro"}
	default:
		return Dificuldade{"Mortal", "mortal"}
	}
}

// DesfechoDoEncontro é como o combate terminou (p326).
type DesfechoDoEncontro string

const (
	Vitoria DesfechoDoEncontro = "vitoria"
	Empate  DesfechoDoEncontro = "empate"
	Derrota DesfechoDoEncontro = "derrota"
)

var multiplicadorDoDesfecho = map[DesfechoDoEncontro]float64{
	Vitoria: 1,
	Empate:  0.5,
	Derrota: 0.25,
}

// DiferencaIrrelevante: cinco degraus de ND abaixo do nível do grupo e o
// desafio não vale XP nenhum ("Desafios Irrelevantes", p326).
const DiferencaIrrelevante = 5.0

// XPDoEncontro é o XP que CADA personagem leva.
//
//	base = ND × 1.000 × multiplicador do desfecho
//	cada = base / tamanho do grupo
//
// Zero quando não há grupo, quando o ND não é positivo, ou quando o desafio é
// irrelevante para o nível.
func XPDoEncontro(nd float64, nivelDoGrupo, tamanhoDoGrupo int, desfecho DesfechoDoEncontro) int {
	if tamanhoDoGrupo <= 0 || nd <= 0 {
		return 0
	}
	if nd <= float64(nivelDoGrupo)-DiferencaIrrelevante {
		return 0
	}
	mult, ok := multiplicadorDoDesfecho[desfecho]
	if !ok {
		mult = multiplicadorDoDesfecho[Vitoria]
	}
	return int(math.Floor(nd * 1000 * mult / float64(tamanhoDoGrupo)))
}
