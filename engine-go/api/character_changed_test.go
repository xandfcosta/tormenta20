package api

import "testing"

// A MESA AO VIVO PRECISA SABER QUE A FICHA MUDOU (ALE-245).
//
// O mestre aplica "Caído" num PC pela ficha do combatente e a tela do jogador
// não ficava sabendo. Pior que o chip faltando: o motor deriva Defesa e
// perícias da condição (ALE-28), então os dois viam números diferentes do mesmo
// personagem, sem nada na tela dizendo que discordavam. É a família da ALE-122.
//
// O que impedia era estrutural: o gateway guarda `s *Server`, e o ponteiro
// nunca vai na direção contrária, então NENHUM handler HTTP conseguia falar com
// a sala. O `Server` ganhou um gancho, e o `SocketHandler()` o preenche.

// TestOGanchoNuloNaoDerruba prende a escolha de nulo ser caminho NORMAL.
//
// O `Server` existe sem gateway em dois lugares de verdade: em todo teste de
// handler, e num binário que sirva só HTTP. Se `characterChanged` exigisse o
// gancho, a suíte inteira de handlers quebraria — e o que é pior, quebraria
// longe daqui.
func TestOGanchoNuloNaoDerruba(t *testing.T) {
	s := &Server{}

	s.characterChanged(7) // não deve entrar em pânico
}

func TestOGanchoRecebeOPersonagemQueMudou(t *testing.T) {
	var avisados []int64
	s := &Server{notifyCharacterChanged: func(id int64) { avisados = append(avisados, id) }}

	s.characterChanged(14)

	if len(avisados) != 1 || avisados[0] != 14 {
		t.Fatalf("avisados = %v, queria [14]", avisados)
	}
}

// A busca é por SESSÃO VIVA, e só as que têm o personagem na fila.
//
// Avisar mesa que não tem aquele combatente seria mandar todo cliente da casa
// refazer busca a cada ficha salva — e a sala é o recorte natural de quem pode
// estar olhando.
func TestSoAsSessoesVivasComOPersonagem(t *testing.T) {
	umPersonagem := func(id int64) *int64 { return &id }
	st := &sessionStore{states: map[int64]*SessionRuntimeState{
		1: {Initiative: []InitiativeEntry{{ID: "a", CharacterID: umPersonagem(14)}}},
		2: {Initiative: []InitiativeEntry{{ID: "b", CharacterID: umPersonagem(99)}}},
		// NPC na fila: `CharacterID` nulo não pode ser confundido com o 14.
		3: {Initiative: []InitiativeEntry{{ID: "c"}, {ID: "d", CharacterID: umPersonagem(14)}}},
	}}

	achadas := st.liveSessionsWithCharacter(14)

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
	st := &sessionStore{states: map[int64]*SessionRuntimeState{
		1: {Initiative: []InitiativeEntry{
			{ID: "a", CharacterID: umPersonagem(14)},
			{ID: "b", CharacterID: umPersonagem(14)},
		}},
	}}

	if achadas := st.liveSessionsWithCharacter(14); len(achadas) != 1 {
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
	st := &sessionStore{states: map[int64]*SessionRuntimeState{
		1: {Initiative: []InitiativeEntry{{ID: "a"}}},
	}}

	if achadas := st.liveSessionsWithCharacter(14); len(achadas) != 0 {
		t.Fatalf("sessões = %v, queria nenhuma", achadas)
	}
}
