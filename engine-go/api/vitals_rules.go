package api

import "t20engine/aovivo"

import (
	"context"
	"errors"
	"log"
)

// As regras dos vitais na mesa: quem pode editar e o espelho no tracker.
//
// Este arquivo é o que SOBROU de `realtime_vitals.go` quando o socket.io foi
// apagado (ALE-253). O corte foi pelo receptor: o que era `(g *realtimeGateway)`
// era transporte e morreu junto; o que está aqui é aplicação, e não mudou uma
// linha ao mudar de vizinho. As rotas HTTP em `session_commands.go` e
// `board_commands.go` chamam exatamente as mesmas funções que os eventos
// chamavam.

// assertVitalsEditableFor é a REGRA, e ela mudou de dono junto com o transporte
// (ALE-253): o mestre edita qualquer combatente, o jogador só o personagem
// dele, e NPC é do mestre porque não há ficha atrás para conferir dono.
func (s *Server) assertVitalsEditableFor(ctx context.Context, live liveCtx, entryID string) error {
	if live.role == "gm" {
		return nil
	}
	state := s.sessions.GetState(live.sessionID)
	idx := aovivo.FindEntryIndex(state, entryID)
	if idx < 0 {
		return errors.New("Entry " + entryID + " not found")
	}
	entry := state.Initiative[idx]
	if entry.CharacterID == nil {
		return errors.New("Only the GM can edit NPC vitals")
	}
	_, err := s.assertCharacterOwner(ctx, live.userID, *entry.CharacterID)
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
func (s *Server) restParty(user AuthUser, campaignID, sessionID int64, scope, condition string) (done, total int, err error) {
	if scope != "day" {
		return s.expirePartyScene(user, campaignID, sessionID)
	}
	charIDs, err := s.listMemberCharacterIds(context.Background(), campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, cid := range charIDs {
		if s.restCharacterDay(user, sessionID, cid, condition) {
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
// "Encerrar cena" do mestre e a "Recuperar · cena" agora chamam ESTE helper.
// Antes só a Recuperação passava por aqui, e encerrar a cena deixava a bênção
// de duração "cena" viva na ficha — a colisão C1 do glossário.
func (s *Server) expirePartyScene(user AuthUser, campaignID, sessionID int64) (done, total int, err error) {
	charIDs, err := s.listMemberCharacterIds(context.Background(), campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, cid := range charIDs {
		if _, e := s.EndScene(context.Background(), user, cid); e != nil {
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
func (s *Server) restCharacterDay(user AuthUser, sessionID, characterID int64, condition string) bool {
	ctx := context.Background()
	if _, err := s.endDay(ctx, user, characterID); err != nil {
		log.Printf("session %d: encerrar dia do personagem %d falhou (%v)", sessionID, characterID, err)
		return false
	}
	vitals, _, err := s.restVitals(ctx, user, characterID, condition)
	if err != nil {
		log.Printf("session %d: descanso do personagem %d falhou (%v)", sessionID, characterID, err)
		return false
	}
	s.mirrorVitalsToTracker(sessionID, characterID, vitals)
	return true
}

// mirrorVitalsToTracker copies freshly-persisted PV/PM onto the matching live tracker entry
// (if the character is in the current initiative) so bars update without a reload.
func (s *Server) mirrorVitalsToTracker(sessionID, characterID int64, vitals restedVitals) {
	for _, e := range s.sessions.GetState(sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == characterID {
			hp, mp := vitals.hpCurrent, vitals.mpCurrent
			_, _ = s.sessions.PatchVitals(sessionID, e.ID, &hp, &mp)
			return
		}
	}
}
