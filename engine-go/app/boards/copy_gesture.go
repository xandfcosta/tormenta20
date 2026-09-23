package boards

import (
	"context"
	"fmt"

	"t20engine/app/session"
	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// DUPLICAR E COLAR: a linha da fila e a peça do mapa, na mesma transação
// (ALE-376).
//
// Os dois escrevem nos DOIS donos de dado, e na ordem INVERSA à do movimento: a
// linha entra na fila primeiro e a peça nasce depois. Uma falha na segunda
// deixava **uma linha órfã na fila, sem peça no mapa** — o mestre vê um
// combatente que não está em lugar nenhum, e tirá-lo é um gesto que ele não
// sabe que precisa fazer.

// CopyBond é o que a cópia leva junto, traduzido do modo que a cena guardou.
type CopyBond int

const (
	// BondNone é "só a peça": nada é escrito na fila.
	BondNone CopyBond = iota
	// BondShared é "sangrando junto": a cópia aponta para a MESMA linha, então
	// as duas peças dividem PV. Nada é escrito na fila.
	BondShared
	// BondOwnLine é "sozinha": linha nova na fila, com os PV cheios.
	BondOwnLine
)

// CopyRequest é um duplicar ou um colar, já resolvido do que veio do pedido.
//
// O BLOCO DE CRIATURA já vem clonado quando o modo pedia — o `ClonedCreature`
// carrega o id novo. Ele é clonado FORA desta transação de propósito, e a razão
// é o impasse da ALE-371: o `Cast` escreve pelo caderno dele, que é outra
// conexão do pool, e chamá-lo com a transação aberta bateria na própria trava.
//
// O preço está dito: se o gesto for recusado depois, o bloco clonado fica no
// acervo da campanha sem ninguém apontando para ele. É um órfão que o mestre VÊ
// e apaga, e não um combatente invisível na fila — que era o estrago de antes.
type CopyRequest struct {
	SessionID      int64
	BoardID        string
	Bond           CopyBond
	ClonedCreature *int64
	Template       board.BoardToken
	// Destination só existe no COLAR: o duplicar põe a cópia ao lado sozinho.
	Destination *engine.Square
	// OriginBoard é de onde a peça foi copiada, e pode não ser o tabuleiro em
	// que ela vai pousar — é justamente quando o colar mais serve.
	OriginBoard string
}

// Copy duplica ou cola a peça, com a linha da fila que ela pedir.
func (g Gestures) Copy(ctx context.Context, req CopyRequest) (*board.BoardState, error) {
	var out *board.BoardState
	err := g.units.Do(ctx, func(u session.Unit) error {
		inUnit := g.inUnit(ctx, u)
		bond, err := g.lineForCopy(inUnit, req)
		if err != nil {
			return err
		}
		if req.Destination != nil {
			out, err = g.boards.PasteToken(inUnit, req.SessionID, req.BoardID,
				req.Template, bond, req.Destination.X, req.Destination.Y)
			return err
		}
		out, err = g.boards.DuplicateToken(inUnit, req.SessionID, req.BoardID, req.Template.ID, bond)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// lineForCopy devolve a linha que a cópia vai apontar — a mesma da original, uma
// nova, ou nenhuma.
func (g Gestures) lineForCopy(ctx context.Context, req CopyRequest) (*live.InitiativeEntry, error) {
	if req.Bond == BondNone {
		return nil, nil
	}
	row, err := g.queueLineOf(ctx, req.SessionID, req.Template)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf(
			"%s não é um combatente da fila, e sem PV não há o que dividir nem o que copiar",
			req.Template.Label)
	}
	if req.Bond == BondShared {
		return row, nil
	}
	ownLine := *row
	if req.ClonedCreature != nil {
		ownLine.CreatureID = req.ClonedCreature
	}
	return g.addsACopyOfTheLine(ctx, req.SessionID, ownLine)
}

// queueLineOf é a linha da fila por trás de uma peça, ou nulo.
func (g Gestures) queueLineOf(
	ctx context.Context, sessionID int64, token board.BoardToken,
) (*live.InitiativeEntry, error) {
	if token.EntryID == nil {
		return nil, nil
	}
	state, err := g.sessions.State(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	for i := range state.Initiative {
		if state.Initiative[i].ID == *token.EntryID {
			return &state.Initiative[i], nil
		}
	}
	return nil, nil
}

// addsACopyOfTheLine põe na fila uma linha igual à do original, de pé.
//
// O PV ATUAL vira o MÁXIMO da nova, e não o máximo da original: o segundo zumbi
// chega inteiro, não com os 12 de 130 que o primeiro levou de porrada. As
// CONDIÇÕES ficam para trás pela mesma razão — caído e sangrando são estado de
// combate do primeiro, e o que entra agora entra de pé.
//
// A linha nova é achada por DIFERENÇA e nunca pelo último da lista: o `AddEntry`
// ORDENA a fila por iniciativa depois de inserir, então a recém-chegada pode
// pousar em qualquer posição. Pegar `Initiative[len-1]` daria a de menor
// iniciativa da mesa, e daria certo por acaso sempre que o zumbi fosse lento.
func (g Gestures) addsACopyOfTheLine(
	ctx context.Context, sessionID int64, template live.InitiativeEntry,
) (*live.InitiativeEntry, error) {
	before := map[string]bool{}
	state, err := g.sessions.State(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	for i := range state.Initiative {
		before[state.Initiative[i].ID] = true
	}
	nova := template
	nova.ID = ""
	nova.Conditions = nil
	if template.HpMax != nil {
		full := live.DerefOr(template.HpMax, 0)
		nova.HpCurrent, nova.HpMax = &full, &full
	}
	after, err := g.sessions.AddInitiativeEntry(ctx, sessionID, nova)
	if err != nil {
		return nil, err
	}
	for i := range after.Initiative {
		if !before[after.Initiative[i].ID] {
			return &after.Initiative[i], nil
		}
	}
	return nil, fmt.Errorf("a linha de %s não entrou na fila", template.Label)
}
