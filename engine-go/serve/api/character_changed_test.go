package api

import (
	"strings"
	"t20engine/app/session"
	"t20engine/domain/live"
	"t20engine/infra/events"
	"testing"
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
	charIDOf := func(id int64) *int64 { return &id }
	s := &Server{
		// O barramento é OBRIGATÓRIO, e um `Server` montado à mão sem ele explode
		// no primeiro `characterChanged`. É o desenho: nulo que EXPLODE é melhor
		// que nulo tolerado, que é como o gancho acima nasceu desligado.
		bus: &events.Bus{},
		sse: live.NewSSEHub(),
		sessions: &session.Store{States: map[int64]*live.SessionRuntimeState{
			7: {Initiative: []live.InitiativeEntry{{ID: "a", CharacterID: charIDOf(14)}}},
		}},
	}
	conn := s.sse.Add(7, "c1", "player")

	s.sheetRules().characterChanged(14)

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
	charIDOf := func(id int64) *int64 { return &id }
	s := &Server{
		bus: &events.Bus{},
		sse: live.NewSSEHub(),
		sessions: &session.Store{States: map[int64]*live.SessionRuntimeState{
			7: {Initiative: []live.InitiativeEntry{{ID: "a", CharacterID: charIDOf(99)}}},
		}},
	}
	conn := s.sse.Add(7, "c1", "player")

	s.sheetRules().characterChanged(14)

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
	charIDOf := func(id int64) *int64 { return &id }
	st := &session.Store{States: map[int64]*live.SessionRuntimeState{
		1: {Initiative: []live.InitiativeEntry{{ID: "a", CharacterID: charIDOf(14)}}},
		2: {Initiative: []live.InitiativeEntry{{ID: "b", CharacterID: charIDOf(99)}}},
		// NPC na fila: `CharacterID` nulo não pode ser confundido com o 14.
		3: {Initiative: []live.InitiativeEntry{{ID: "c"}, {ID: "d", CharacterID: charIDOf(14)}}},
	}}

	found := st.LiveSessionsWithCharacter(14)

	if len(found) != 2 {
		t.Fatalf("sessões = %v, queria as duas que têm o 14", found)
	}
	for _, id := range found {
		if id != 1 && id != 3 {
			t.Fatalf("sessões = %v — a %d não tem o personagem 14", found, id)
		}
	}
}

// O mesmo personagem DUAS vezes na fila (o mestre pôs a ficha e o jogador
// entrou sozinho) avisa a sessão UMA vez. Avisar duas faria o cliente refazer a
// mesma busca duas vezes por escrita.
func TestARepeatedSessionIsAnnouncedOnce(t *testing.T) {
	charIDOf := func(id int64) *int64 { return &id }
	st := &session.Store{States: map[int64]*live.SessionRuntimeState{
		1: {Initiative: []live.InitiativeEntry{
			{ID: "a", CharacterID: charIDOf(14)},
			{ID: "b", CharacterID: charIDOf(14)},
		}},
	}}

	if found := st.LiveSessionsWithCharacter(14); len(found) != 1 {
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
	st := &session.Store{States: map[int64]*live.SessionRuntimeState{
		1: {Initiative: []live.InitiativeEntry{{ID: "a"}}},
	}}

	if found := st.LiveSessionsWithCharacter(14); len(found) != 0 {
		t.Fatalf("sessões = %v, queria nenhuma", found)
	}
}
