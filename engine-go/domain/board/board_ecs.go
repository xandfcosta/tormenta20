package board

import "t20engine/domain/ecs"

// O MUNDO DO TABULEIRO: aqui a ENTIDADE é a PEÇA (ALE-413).
//
// Dois gestos rodam como sistemas sobre ele, porque a pergunta deles atravessa peças: a ocupação ("que
// quadrado está tomado, e quem já tem peça na mesa?", `board_occupancy_ecs.go`) e o terreno sob a peça
// ("que terreno ela pisa, e o que ele faz com o ataque?", `situations_ecs.go`). A entidade é a mesma nos
// dois; o que muda é o componente que cada pipeline escreve — o `body` para quem pergunta quadrado, o
// `boundTo` para quem pergunta fila.
//
// A REDAÇÃO já foi um terceiro mundo daqui, e voltou a ser um filtro (`board_redaction.go`): esconder o que
// o mestre escondeu olha uma peça de cada vez, e um mundo de índices não comprava nada sobre um laço que
// preserva a ordem.
//
// # O mundo é EFÊMERO, e o `BoardState` continua sendo a verdade
//
// Monta, roda, extrai — é a decisão da ALE-413 para todo o combate. O estado gravado e o que viaja no fio
// não mudam.

// tokenAt guarda o ÍNDICE da peça no `BoardState`, e não uma cópia: o mundo nasce e morre dentro de uma
// chamada, e a cópia duplicaria a peça inteira para devolvê-la igual.
type tokenAt struct{ Index int }

// worldOfTheBoard monta o mundo: uma entidade por peça, na ORDEM em que elas estão no tabuleiro.
//
// A ordem é a do fio, e o `ecs.World` varre por inserção justamente para isto — é a propriedade em volta da
// qual o núcleo foi desenhado, porque `map` em Go não a tem.
func worldOfTheBoard(b *BoardState) *ecs.World {
	w := ecs.NewWorld()
	for at := range b.Tokens {
		ecs.Set(w, w.Spawn(), tokenAt{Index: at})
	}
	return w
}
