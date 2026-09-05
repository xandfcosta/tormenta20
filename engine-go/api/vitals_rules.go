package api

import (
	"context"
	"errors"
	"log"
	"t20engine/aovivo"
)

// As regras dos vitais na mesa: quem pode editar e o espelho no tracker.
//
// Este arquivo é o que SOBROU de `realtime_vitals.go` quando o socket.io foi
// apagado (ALE-253). O corte foi pelo receptor: o que era `(g *realtimeGateway)`
// era transporte e morreu junto; o que está aqui é aplicação, e não mudou uma
// linha ao mudar de vizinho.
//
// Aqui morava "as rotas HTTP em `session_commands.go` e `board_commands.go`
// chamam exatamente as mesmas funções que os eventos chamavam". Os dois
// arquivos saíram na ALE-277 com os 36 manipuladores deles, que não tinham um
// chamador desde que as cenas em Datastar passaram a mutar o estado pela porta
// própria. A frase continua valendo com outro sujeito: quem chama estas funções
// hoje é a cena da Mesa, e elas continuam sem saber por onde o pedido entrou.

// assertVitalsEditableFor é a REGRA, e ela mudou de dono junto com o transporte
// (ALE-253): o mestre edita qualquer combatente, o jogador só o personagem
// dele, e NPC é do mestre porque não há ficha atrás para conferir dono.
func (tr tableRules) assertVitalsEditableFor(ctx context.Context, live liveCtx, entryID string) error {
	if live.Role == "gm" {
		return nil
	}
	state := tr.sessions.GetState(live.sessionID)
	idx := aovivo.FindEntryIndex(state, entryID)
	if idx < 0 {
		return errors.New("Entry " + entryID + " not found")
	}
	entry := state.Initiative[idx]
	if entry.CharacterID == nil {
		return errors.New("Only the GM can edit NPC vitals")
	}
	_, err := tr.assertCharacterOwner(ctx, live.UserID, *entry.CharacterID)
	return err
}

// restParty aplica o descanso a cada personagem do grupo (encerrar cena, ou
// encerrar dia + curar + espelhar) e devolve quantos DERAM CERTO e o total.
//
// Best-effort por personagem de propósito — uma ficha que falha não pode
// impedir o descanso das outras quatro. Mas o resultado é CONTADO e volta no
// ack (ALE-155): antes, o encerrar-cena era `_, _ =` e nem entrava na conta, de
// modo que o mestre lia "descansou" enquanto duas de cinco fichas não tinham
// descansado. Best-effort é sobre continuar apesar da falha, não sobre esconder
// que ela houve.
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
// É o caminho ÚNICO desde a ALE-220, e essa unificação É o conserto: o
// "Encerrar cena" do mestre e o "Expirar efeitos · cena" chamam ESTE helper.
// (Aquele segundo se chamava "Recuperar · cena" até a ALE-233, e o nome
// prometia PV e PM que ele nunca deu.)
// Antes só a Recuperação passava por aqui, e encerrar a cena deixava a bênção
// de duração "cena" viva na ficha — a colisão C1 do glossário.
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

// mirrorVitalsToTracker copies freshly-persisted PV/PM onto the matching live tracker entry
// (if the character is in the current initiative) so bars update without a reload.
func (tr tableRules) mirrorVitalsToTracker(sessionID, characterID int64, vitals restedVitals) {
	for _, e := range tr.sessions.GetState(sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == characterID {
			hp, mp := vitals.hpCurrent, vitals.mpCurrent
			_, _ = tr.sessions.PatchVitals(sessionID, e.ID, &hp, &mp)
			return
		}
	}
}
