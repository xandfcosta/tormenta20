package api

import (
	"context"
	"errors"
	"log"
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

// restParty aplica o descanso a cada personagem do grupo (encerrar cena, ou
// encerrar dia + curar + espelhar) e devolve quantos DERAM CERTO e o total.
//
// Best-effort por personagem de propósito — uma ficha que falha não pode impedir
// o descanso das outras quatro. Mas o resultado é CONTADO e volta na resposta:
// descartar a contagem faz o mestre ler "descansou" com duas de cinco fichas de
// fora. Best-effort é sobre continuar apesar da falha, não sobre escondê-la.
func (tr tableRules) restParty(user AuthUser, campaignID, sessionID int64, scope, condition string) (done, total int, err error) {
	if scope != "day" {
		return tr.expirePartyScene(user, campaignID, sessionID)
	}
	charIDs, err := tr.listMemberCharacterIds(context.Background(), campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, cid := range charIDs {
		if tr.restCharacterDay(user, sessionID, cid, condition) {
			done++
		}
	}
	return done, len(charIDs), nil
}

// expirePartyScene expira a duração "cena" de TODA ficha do grupo: os efeitos
// de escopo "scene", os usos "1/cena" e as posturas (o helper de domínio
// `EndScene` faz os três).
//
// É o caminho ÚNICO, e a unificação É a regra: o "Encerrar cena" do mestre e o
// "Expirar efeitos · cena" chamam ESTE helper. Com dois caminhos, encerrar a cena
// deixa a bênção de duração "cena" viva na ficha — a colisão C1 do glossário.
func (tr tableRules) expirePartyScene(user AuthUser, campaignID, sessionID int64) (done, total int, err error) {
	charIDs, err := tr.listMemberCharacterIds(context.Background(), campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, cid := range charIDs {
		if _, e := tr.EndScene(context.Background(), user, cid); e != nil {
			log.Printf("session %d: encerrar cena do personagem %d falhou (%v)", sessionID, cid, e)
			continue
		}
		done++
	}
	return done, len(charIDs), nil
}

// restCharacterDay encerra o dia de UMA ficha, cura e espelha os vitais no
// rastreador. Devolve se a ficha inteira deu certo — meia ficha descansada não
// conta, senão o ack diz "5 de 5" com dois PV que não foram gravados.
func (tr tableRules) restCharacterDay(user AuthUser, sessionID, characterID int64, condition string) bool {
	ctx := context.Background()
	if _, err := tr.endDay(ctx, user, characterID); err != nil {
		log.Printf("session %d: encerrar dia do personagem %d falhou (%v)", sessionID, characterID, err)
		return false
	}
	vitals, _, err := tr.restVitals(ctx, user, characterID, condition)
	if err != nil {
		log.Printf("session %d: descanso do personagem %d falhou (%v)", sessionID, characterID, err)
		return false
	}
	tr.mirrorVitalsToTracker(sessionID, characterID, vitals)
	return true
}

// mirrorVitalsToTracker copia os PV/PM recém-gravados para a linha viva do
// rastreador, quando o personagem está na iniciativa atual, para as barras
// mudarem sem recarga.
func (tr tableRules) mirrorVitalsToTracker(sessionID, characterID int64, vitals restedVitals) {
	for _, e := range tr.sessions.GetState(sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == characterID {
			hp, mp := vitals.hpCurrent, vitals.mpCurrent
			_, _ = tr.sessions.PatchVitals(sessionID, e.ID, &hp, &mp)
			return
		}
	}
}
