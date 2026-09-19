package api

import (
	"context"
	"errors"
	"t20engine/domain/live"
)

// As regras dos vitais na mesa: quem pode editar e o espelho no rastreador.
// Quem as chama é a cena da Mesa, e elas não sabem por onde o pedido entrou.

// assertVitalsEditableFor é a REGRA: o mestre edita qualquer combatente, o
// jogador só o personagem dele, e NPC é do mestre porque não há ficha atrás para
// conferir dono.
func (tr tableRules) assertVitalsEditableFor(ctx context.Context, asked liveCtx, entryID string) error {
	if asked.Role == "gm" {
		return nil
	}
	state := tr.sessions.GetState(asked.sessionID)
	idx := live.FindEntryIndex(state, entryID)
	if idx < 0 {
		return errors.New("Entry " + entryID + " not found")
	}
	entry := state.Initiative[idx]
	if entry.CharacterID == nil {
		return errors.New("Only the GM can edit NPC vitals")
	}
	_, err := tr.assertCharacterOwner(ctx, asked.UserID, *entry.CharacterID)
	return err
}
