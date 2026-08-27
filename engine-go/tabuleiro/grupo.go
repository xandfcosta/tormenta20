package tabuleiro

import (
	"fmt"

	"t20engine/engine"
)

// O GRUPO DE PEÇAS marcado por um retângulo (ALE-203, item 10 do dono).
//
// "Não temos ferramenta de seleção em área." Chegou uma horda de seis zumbis, e
// hoje reposicioná-los é seis arrastos — cada um com proposta, parada e
// confirmação. O gesto que falta é marcar e comandar.
//
// # O grupo NÃO propõe: ele REPOSICIONA
//
// O movimento de uma peça é uma PROPOSTA — trilha, paradas, custo, orçamento do
// turno — porque a mesa inteira decide sobre ela. O grupo é outra coisa: é o
// mestre arrumando a cena, e o mestre já move sem teto (`-1` de deslocamento).
// Fazer o grupo passar pela proposta seria seis propostas simultâneas sobre um
// orçamento que não existe.
//
// É por isso que marcar é gesto DE MESTRE e não de jogador: um jogador tem uma
// peça, e o que o grupo dispensa (a regra de deslocamento) é exatamente o que
// protege o turno dele.

// PecasNoRetangulo são os ids das peças cujo CORPO toca o retângulo.
//
// O corpo e não a âncora: uma Colossal ocupa 6×6 (p107), e marcá-la só quando o
// laço pega a quina dela faria o mestre desenhar em volta do dragão e não pegar
// o dragão.
//
// Exemplo:
//
//	PecasNoRetangulo(b, engine.Square{}, engine.Square{X: 5, Y: 5})
//	// → os ids das peças que aparecem no quadrado de (0,0) a (5,5)
func PecasNoRetangulo(b *BoardState, de, ate engine.Square) []string {
	if b == nil {
		return nil
	}
	x0, x1 := min(de.X, ate.X), max(de.X, ate.X)
	y0, y1 := min(de.Y, ate.Y), max(de.Y, ate.Y)
	var ids []string
	for i := range b.Tokens {
		t := &b.Tokens[i]
		pegada := max(t.Footprint, 1)
		if t.X <= x1 && t.X+pegada-1 >= x0 && t.Y <= y1 && t.Y+pegada-1 >= y0 {
			ids = append(ids, t.ID)
		}
	}
	return ids
}

// MoveOGrupo desloca as peças pelo MESMO delta, numa gravação só.
//
// Uma gravação porque o gesto é UM: seis `apply` fariam a mesa receber seis
// quadros, cada um com metade da horda no lugar novo e metade no velho.
//
// A peça que não existe mais é IGNORADA e não derruba o gesto: entre marcar e
// arrastar, o stream pode ter trazido a remoção de uma delas por outra pessoa —
// e recusar o movimento das outras cinco por causa disso seria punir o mestre
// por uma corrida que não é dele.
//
// O `DeOndeVeio` é gravado como no pouso de uma peça só: o "voltar para onde
// estava" do menu (ALE-206) tem de funcionar depois de um movimento de grupo.
func MoveOGrupo(b *BoardState, ids []string, dx, dy int) error {
	if b == nil {
		return fmt.Errorf("não há tabuleiro para mover o grupo")
	}
	if dx == 0 && dy == 0 {
		return nil
	}
	marcadas := map[string]bool{}
	for _, id := range ids {
		marcadas[id] = true
	}
	// DUAS passadas: a primeira confere que TODAS cabem, a segunda escreve. Sem
	// isso, uma coordenada absurda no meio da lista deixaria metade do grupo
	// movida — o pior estado possível, porque parece que o gesto funcionou.
	for i := range b.Tokens {
		if !marcadas[b.Tokens[i].ID] {
			continue
		}
		proposta := b.Tokens[i]
		proposta.X += dx
		proposta.Y += dy
		if err := assertSaneCoords(proposta); err != nil {
			return err
		}
	}
	for i := range b.Tokens {
		t := &b.Tokens[i]
		if !marcadas[t.ID] {
			continue
		}
		t.DeOndeVeio = &engine.Square{X: t.X, Y: t.Y}
		t.X += dx
		t.Y += dy
	}
	b.Version++
	return nil
}
