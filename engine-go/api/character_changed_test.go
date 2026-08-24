package api

import "t20engine/aovivo"

import (
	"strings"
	"testing"
)

// A MESA AO VIVO PRECISA SABER QUE A FICHA MUDOU (ALE-245).
//
// O mestre aplica "Caído" num PC pela ficha do combatente e a tela do jogador
// não ficava sabendo. Pior que o chip faltando: o motor deriva Defesa e
// perícias da condição (ALE-28), então os dois viam números diferentes do mesmo
// personagem, sem nada na tela dizendo que discordavam. É a família da ALE-122.
//
// Este teste mudou de FORMA na ALE-253, e o motivo vale mais que ele. Antes ele
// prendia um GANCHO: `characterChanged` chamava `s.notifyCharacterChanged` se
// não fosse nulo, e havia um caso afirmando que NULO NÃO DERRUBA. Aquilo estava
// certo para o socket — o gateway nascia noutro arquivo e o `Server` existia
// sem ele. Só que apagar o gateway deixou o gancho sem quem o preenchesse, o Go
// inteiro seguiu VERDE, e quem acusou foi o e2e de dois clientes.
//
// Um teste que afirma "desligado é caminho normal" não distingue desligado de
// QUEBRADO. Agora o hub é campo do `Server`, não há nulo a tolerar, e o que se
// prende é o que a mesa recebe.

// TestAFichaQueMudouChegaNaMesa exercita o caminho inteiro contra o hub de
// verdade — o mesmo que o handler usa.
func TestAFichaQueMudouChegaNaMesa(t *testing.T) {
	umPersonagem := func(id int64) *int64 { return &id }
	s := &Server{
		sse: aovivo.NewSSEHub(),
		sessions: &aovivo.SessionStore{States: map[int64]*aovivo.SessionRuntimeState{
			7: {Initiative: []aovivo.InitiativeEntry{{ID: "a", CharacterID: umPersonagem(14)}}},
		}},
	}
	conn := s.sse.Add(7, "c1", "player")

	s.characterChanged(14)

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
func TestMesaSemOPersonagemNaoRecebe(t *testing.T) {
	umPersonagem := func(id int64) *int64 { return &id }
	s := &Server{
		sse: aovivo.NewSSEHub(),
		sessions: &aovivo.SessionStore{States: map[int64]*aovivo.SessionRuntimeState{
			7: {Initiative: []aovivo.InitiativeEntry{{ID: "a", CharacterID: umPersonagem(99)}}},
		}},
	}
	conn := s.sse.Add(7, "c1", "player")

	s.characterChanged(14)

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
func TestSoAsSessoesVivasComOPersonagem(t *testing.T) {
	umPersonagem := func(id int64) *int64 { return &id }
	st := &aovivo.SessionStore{States: map[int64]*aovivo.SessionRuntimeState{
		1: {Initiative: []aovivo.InitiativeEntry{{ID: "a", CharacterID: umPersonagem(14)}}},
		2: {Initiative: []aovivo.InitiativeEntry{{ID: "b", CharacterID: umPersonagem(99)}}},
		// NPC na fila: `CharacterID` nulo não pode ser confundido com o 14.
		3: {Initiative: []aovivo.InitiativeEntry{{ID: "c"}, {ID: "d", CharacterID: umPersonagem(14)}}},
	}}

	achadas := st.LiveSessionsWithCharacter(14)

	if len(achadas) != 2 {
		t.Fatalf("sessões = %v, queria as duas que têm o 14", achadas)
	}
	for _, id := range achadas {
		if id != 1 && id != 3 {
			t.Fatalf("sessões = %v — a %d não tem o personagem 14", achadas, id)
		}
	}
}

// O mesmo personagem DUAS vezes na fila (o mestre pôs a ficha e o jogador
// entrou sozinho) avisa a sessão UMA vez. Avisar duas faria o cliente refazer a
// mesma busca duas vezes por escrita.
func TestSessaoRepetidaAvisaUmaVez(t *testing.T) {
	umPersonagem := func(id int64) *int64 { return &id }
	st := &aovivo.SessionStore{States: map[int64]*aovivo.SessionRuntimeState{
		1: {Initiative: []aovivo.InitiativeEntry{
			{ID: "a", CharacterID: umPersonagem(14)},
			{ID: "b", CharacterID: umPersonagem(14)},
		}},
	}}

	if achadas := st.LiveSessionsWithCharacter(14); len(achadas) != 1 {
		t.Fatalf("sessões = %v, queria uma só", achadas)
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
func TestForaDeMesaNaoAvisaNinguem(t *testing.T) {
	st := &aovivo.SessionStore{States: map[int64]*aovivo.SessionRuntimeState{
		1: {Initiative: []aovivo.InitiativeEntry{{ID: "a"}}},
	}}

	if achadas := st.LiveSessionsWithCharacter(14); len(achadas) != 0 {
		t.Fatalf("sessões = %v, queria nenhuma", achadas)
	}
}
