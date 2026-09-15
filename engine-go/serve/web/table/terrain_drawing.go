package table

import (
	"fmt"

	"t20engine/domain/board"
)

// COMO UMA CASA DE TERRENO SE PARECE: um ÍCONE num canto próprio de cada
// espécie, sobre uma tinta forte.
//
// ÍCONE e não uma lavagem de fundo, por duas razões que se somam. Lavagem de
// baixo contraste não se lê de longe, e uma mesa olha o tabuleiro de um metro;
// e lavagem não EMPILHA — folhagens são difícil E camuflagem (p267), e duas
// lavagens somadas viram uma terceira coisa que não é nenhuma das duas. Com
// quatro espécies em quatro cantos, duas na mesma casa continuam sendo duas
// coisas distintas.
//
// OS ÍCONES DIZEM A REGRA, não a aparência: `Shield` para cobertura porque ela é
// +5 na Defesa, `EyeOff` para camuflagem porque ela é 20% de chance de falha. O
// mestre não precisa lembrar que "névoa = camuflagem"; ele vê o que a casa FAZ.
//
// O ÍCONE SOME no zoom pequeno, porque abaixo de ~32px de lado ele vira um
// borrão que só suja. Quem o esconde é uma `@container` no CSS, medindo a casa —
// nesse regime a TINTA sozinha responde "tem alguma coisa aqui".

// speciesDrawing é como uma espécie de terreno aparece na casa e no trilho.
type speciesDrawing struct {
	// Icone é o nome do lucide, o mesmo que a casa e o botão do pincel usam. UM
	// desenho para os dois lugares: o mestre reconhece o pincel pelo que ele
	// PINTA, e não por uma legenda que ele teria de decorar.
	Icone string
	// Canto é o sufixo da classe que põe o ícone no lugar dele
	// (`terrain-corner-<Canto>`). Quatro espécies, quatro cantos.
	Canto string
}

// drawingBySpecies é a tabela, conferida contra a lista do domínio pelo
// `TestEveryKindHasADrawing`.
//
// Mapa e não campo no tipo do domínio, porque nome de ícone do lucide é
// APARÊNCIA e o domínio não tem por que conhecê-lo. O preço dessa separação é a
// espécie nova poder nascer sem desenho — e é por isso que o guarda existe e que
// o `drawing` recusa em vez de devolver um branco.
var drawingBySpecies = map[board.TerrainKind]speciesDrawing{
	board.TerrenoDificil:    {Icone: "Waves", Canto: "northwest"},
	board.TerrenoCobertura:  {Icone: "Shield", Canto: "northeast"},
	board.TerrenoCamuflagem: {Icone: "EyeOff", Canto: "southeast"},
	board.TerrenoElevado:    {Icone: "Mountain", Canto: "southwest"},
}

// drawing devolve o desenho da espécie, e ENTRA EM PÂNICO se não houver.
//
// Pânico e não um branco silencioso: uma espécie sem desenho pinta uma casa que
// não se distingue de nenhuma outra, e isso é indistinguível de "o pincel não
// funcionou". O `TestEveryKindHasADrawing` faz o pânico acontecer na suíte e não
// na mesa de alguém.
func drawing(especie board.TerrainKind) speciesDrawing {
	d, tem := drawingBySpecies[especie]
	if !tem {
		panic(fmt.Sprintf("a espécie de terreno %q não tem desenho: acrescente-a em oDesenhoDasEspecies", especie))
	}
	return d
}

// squareClass é o que a casa pintada veste: a espécie (que traz a tinta) e o
// canto do ícone.
func squareClass(especie string) string {
	d := drawing(board.TerrainKind(especie))
	return "board-terrain board-" + board.ClassOf(board.TerrainKind(especie)) + " terrain-corner-" + d.Canto
}
