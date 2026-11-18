package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A MESA AO VIVO PRECISA SABER QUE A FICHA MUDOU.
//
// O mestre aplica "Caído" num PC pela ficha do combatente, e o motor deriva
// Defesa e perícias da condição: sem o aviso, os dois veem números diferentes
// do mesmo personagem, sem nada na tela dizendo que discordam.
//
// O que se prende é O QUE A MESA RECEBE, e não um GANCHO opcional. Um caso
// afirmando que "nulo não derruba" não distingue desligado de QUEBRADO — foi
// assim que o gancho ficou sem quem o preenchesse com o Go inteiro VERDE, e
// quem acusou foi o e2e de dois clientes.

// O caminho inteiro contra o hub de verdade, o mesmo que o handler usa.
func TestTheSheetThatChangedReachesTheTable(t *testing.T) {
	s := newTestServer(t)
	table := sessionWithQueue(t, s, 1, true, live.InitiativeEntry{ID: "a", CharacterID: charIDOf(14)})
	conn := s.sse.Add(table, "c1", "player")

	s.sheetRules().characterChanged(context.Background(), 14)

	select {
	case frame := <-conn.Frames:
		if !strings.Contains(string(frame), "character-changed") ||
			!strings.Contains(string(frame), `"characterId":14`) {
			t.Fatalf("quadro = %q", frame)
		}
	default:
		t.Fatal("a mesa não recebeu aviso nenhum — é a ALE-245 desligada de novo")
	}
}

// Mesa que NÃO tem o personagem não recebe nada. O recorte é o que impede uma
// ficha salva de mandar toda a casa refazer busca.
func TestATableWithoutTheCharacterDoesNotReceiveIt(t *testing.T) {
	s := newTestServer(t)
	table := sessionWithQueue(t, s, 1, true, live.InitiativeEntry{ID: "a", CharacterID: charIDOf(99)})
	conn := s.sse.Add(table, "c1", "player")

	s.sheetRules().characterChanged(context.Background(), 14)

	select {
	case frame := <-conn.Frames:
		t.Fatalf("mesa sem o personagem recebeu %q", frame)
	default:
	}
}

// A busca é por SESSÃO VIVA, e só as que têm o personagem na fila.
//
// Avisar mesa que não tem aquele combatente seria mandar todo cliente da casa
// refazer busca a cada ficha salva — e a sala é o recorte natural de quem pode
// estar olhando.
func TestOnlyTheLiveSessionsHoldingTheCharacter(t *testing.T) {
	s := newTestServer(t)
	holding := sessionWithQueue(t, s, 1, true, live.InitiativeEntry{ID: "a", CharacterID: charIDOf(14)})
	sessionWithQueue(t, s, 2, true, live.InitiativeEntry{ID: "b", CharacterID: charIDOf(99)})
	// NPC na fila: `CharacterID` nulo não pode ser confundido com o 14.
	withNPC := sessionWithQueue(t, s, 3, true, live.InitiativeEntry{ID: "c"}, live.InitiativeEntry{ID: "d", CharacterID: charIDOf(14)})
	// A PLANEJADA conta: o mestre monta a fila e roda a cena nela antes de "começar".
	planned := sessionWithQueue(t, s, 4, false, live.InitiativeEntry{ID: "e", CharacterID: charIDOf(14)})
	// A ENCERRADA não: ninguém mais está jogando nela.
	ended := sessionWithQueue(t, s, 5, true, live.InitiativeEntry{ID: "f", CharacterID: charIDOf(14)})
	if _, err := s.queries.EndSession(context.Background(), sqlcgen.EndSessionParams{
		EndedAt: sql.NullString{String: dbvalue.NowISO(), Valid: true}, UpdatedAt: dbvalue.NowISO(), ID: ended,
	}); err != nil {
		t.Fatalf("encerrar a sessão: %v", err)
	}

	found := s.sessions.LiveSessionsWithCharacter(context.Background(), 14)
	if len(found) != 3 {
		t.Fatalf("sessões = %v, queria as três não encerradas que têm o 14", found)
	}
	for _, id := range found {
		if id != holding && id != withNPC && id != planned {
			t.Fatalf("sessões = %v — a %d não tem o personagem 14, ou foi encerrada", found, id)
		}
	}
}

// O mesmo personagem DUAS vezes na fila (o mestre pôs a ficha e o jogador
// entrou sozinho) avisa a sessão UMA vez. Avisar duas faria o cliente refazer a
// mesma busca duas vezes por escrita.
func TestARepeatedSessionIsAnnouncedOnce(t *testing.T) {
	s := newTestServer(t)
	sessionWithQueue(t, s, 1, true,
		live.InitiativeEntry{ID: "a", CharacterID: charIDOf(14)},
		live.InitiativeEntry{ID: "b", CharacterID: charIDOf(14)})
	if found := s.sessions.LiveSessionsWithCharacter(context.Background(), 14); len(found) != 1 {
		t.Fatalf("sessões = %v, queria uma só", found)
	}
}

// O LIMITE do aviso, prendido de propósito: quem acha a mesa é a FILA DA
// INICIATIVA, então um personagem que não está nela não propaga.
//
// No caso comum isso é o que se quer — a ficha editada fora de sessão é a
// maioria das escritas, e ela não pode custar transmissão nenhuma. Mas há um
// caso descoberto e vale saber qual é: o jogador está na sala com a ficha
// aberta, o personagem dele NÃO está na fila (fora de combate), e o mestre
// aplica uma condição. Esse aviso não chega, e a tela dele só se corrige no
// próximo refetch — que o `refetchOnWindowFocus` faz ao trocar de aba.
//
// Cobrir esse caso exigiria achar a mesa pela CAMPANHA do personagem, o que é
// uma leitura de banco a cada escrita de ficha. Não foi feito, e a troca está
// escrita aqui para quem for decidir de novo.
func TestOffTableNobodyIsAnnouncedTo(t *testing.T) {
	s := newTestServer(t)
	sessionWithQueue(t, s, 1, true, live.InitiativeEntry{ID: "a"})
	if found := s.sessions.LiveSessionsWithCharacter(context.Background(), 14); len(found) != 0 {
		t.Fatalf("sessões = %v, queria nenhuma", found)
	}
}

func charIDOf(id int64) *int64 { return &id }

// sessionWithQueue grava uma sessão com esta fila no retrato — em andamento, ou só planejada — numa campanha
// nova. A fila mora no banco, e é lá que a busca das sessões vivas a lê.
func sessionWithQueue(t *testing.T, s *Server, number int64, active bool, queue ...live.InitiativeEntry) int64 {
	t.Helper()
	ctx := context.Background()
	campaign := seedCampaign(t, s, seedUser(t, s, fmt.Sprintf("mestre%d@t.com", number)))
	sess, err := s.queries.CreateSession(ctx, sqlcgen.CreateSessionParams{
		Campaignid: campaign, Sessionnumber: number, Createdat: dbvalue.NowISO(), Updatedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("criar a sessão %d: %v", number, err)
	}
	if active {
		if _, err := s.queries.StartSessionFresh(ctx, sqlcgen.StartSessionFreshParams{
			StartedAt: sql.NullString{String: dbvalue.NowISO(), Valid: true}, UpdatedAt: dbvalue.NowISO(), ID: sess.ID,
		}); err != nil {
			t.Fatalf("começar a sessão %d: %v", number, err)
		}
	}
	state := live.EmptyRuntimeState()
	state.Initiative = queue
	blob, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("serializar a fila: %v", err)
	}
	if err := s.queries.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: string(blob), UpdatedAt: dbvalue.NowISO(), ID: sess.ID,
	}); err != nil {
		t.Fatalf("gravar a fila da sessão %d: %v", number, err)
	}
	return sess.ID
}
