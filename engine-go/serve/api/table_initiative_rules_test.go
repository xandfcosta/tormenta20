package api

import (
	"context"
	"errors"
	"t20engine/app"
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
)

func TestAConditionEntersAndLeavesTheEntry(t *testing.T) {
	st := live.EmptyRuntimeState()
	id := idCounter()
	_ = live.AddEntry(st, npc("Ogro", 12), id)

	aplicadas := []string{"caido", "atordoado"}
	if err := live.UpdateEntry(st, "e1", live.EntryPatch{Conditions: &aplicadas}); err != nil {
		t.Fatalf("aplicar: %v", err)
	}
	if len(st.Initiative[0].Conditions) != 2 {
		t.Fatalf("aplicou %v", st.Initiative[0].Conditions)
	}

	vazio := []string{}
	_ = live.UpdateEntry(st, "e1", live.EntryPatch{Conditions: &vazio})
	if len(st.Initiative[0].Conditions) != 0 {
		t.Fatalf("limpar deixou %v", st.Initiative[0].Conditions)
	}
}

// A iniciativa do jogador é somada pelo SERVIDOR: o cliente manda o d20 e o Go
// pergunta o bônus à ficha COMPUTADA. Quem decidisse o bônus no navegador seria
// uma segunda implementação de regra do livro, livre para divergir do motor.
//
// O d20 continua vindo de fora, e de propósito: a mesa que rola dado FÍSICO
// digita o número, e nesse caminho não existe dado para o servidor rolar.
//
// O nível 8 é o que torna o teste honesto. Metade do nível entra em toda
// perícia — regra do motor, provada em `engine/` —, então o bônus é 4 e o 17
// NÃO pode ter vindo do cliente, que mandou 13.
func TestThePlayerInitiativeIsSummedByTheServer(t *testing.T) {
	f := newSelfInitiativeFixture(t)

	entry, err := f.srv.initiativeQueue().Roster().SelfEntry(context.Background(), app.Caller{ID: f.player}, f.campaignID, f.charID, 13)
	if err != nil {
		t.Fatalf("registrar: %v", err)
	}

	if entry.Initiative != 17 {
		t.Errorf("iniciativa %d, queria 17 (d20 13 + ½ nível 8)", entry.Initiative)
	}
	if entry.CharacterID == nil || *entry.CharacterID != f.charID {
		t.Errorf("a linha não ficou ligada ao personagem: %+v", entry)
	}
}

// Um d20 é um d20. Fora de 1..20 o servidor recusa em vez de gravar: o campo é
// DIGITADO pelo jogador (é para isso que ele existe), e um dedo escorregando no
// teclado põe 133 na frente da fila inteira.
func TestAD20OutsideTheRangeIsRefused(t *testing.T) {
	f := newSelfInitiativeFixture(t)

	for _, d20 := range []int64{0, -3, 21, 100} {
		if _, err := f.srv.initiativeQueue().Roster().SelfEntry(context.Background(), app.Caller{ID: f.player}, f.campaignID, f.charID, d20); err == nil {
			t.Errorf("d20 %d passou", d20)
		}
	}
	// E a fronteira dos dois lados vale: 1 e 20 são dados de verdade.
	for _, d20 := range []int64{1, 20} {
		if _, err := f.srv.initiativeQueue().Roster().SelfEntry(context.Background(), app.Caller{ID: f.player}, f.campaignID, f.charID, d20); err != nil {
			t.Errorf("d20 %d recusado: %v", d20, err)
		}
	}
}

// O "self" deste caminho é o que o separa dos outros, que são todos do mestre:
// sem porta de papel, quem o guarda é o `Roster.Combatant`, e ele recusa quem
// não é dono do personagem.
//
// A asserção é sobre a RECUSA TIPADA e não sobre a frase: texto de erro é para
// quem lê, e prendê-lo faz o teste quebrar quando alguém melhora a mensagem.
func TestRecordingSomeoneElsesInitiativeIsRefused(t *testing.T) {
	f := newSelfInitiativeFixture(t)

	_, err := f.srv.initiativeQueue().Roster().SelfEntry(context.Background(), app.Caller{ID: f.intruder}, f.campaignID, f.charID, 10)

	if err == nil {
		t.Fatal("um jogador registrou a iniciativa do personagem de outro")
	}
	if !errors.Is(err, app.ErrForbidden) {
		t.Errorf("recusou pelo motivo errado: %v", err)
	}
}

type selfInitiativeFixture struct {
	srv        *Server
	campaignID int64
	charID     int64
	player     int64
	intruder   int64
}

// Personagem de NÍVEL 8 com a perícia Iniciativa na ficha: sem a linha da
// perícia o motor não computa nada para ela, e o bônus cairia em zero — o teste
// passaria verde sobre um servidor que não perguntou nada a ninguém.
func newSelfInitiativeFixture(t *testing.T) selfInitiativeFixture {
	t.Helper()
	s := newTestServer(t)
	ctx := context.Background()
	catalogs, err := engine.PrimeEngineCatalogs([]byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("preparar catálogo: %v", err)
	}
	s.primeCatalogs(catalogs)

	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	intruder := seedUser(t, s, "intruso@t.com")
	campaignID := seedCampaign(t, s, gm)
	charID := seedCharacterAtLevel(t, s, player, "Arcanista", 8, 20, 30, 5, 10)
	seedMember(t, s, campaignID, charID)
	if _, err := s.queries.CreateExpertise(ctx, sqlcgen.CreateExpertiseParams{
		Characterid: charID, Name: "Iniciativa", Attribute: "dexterity", Trained: 0, Custom: 0,
	}); err != nil {
		t.Fatalf("semear perícia: %v", err)
	}
	// Intruso na MESMA mesa: recusar alguém de fora seria recusar pela membresia,
	// e a regra que este teste mira é a POSSE do personagem.
	intruderChar := seedCharacter(t, s, intruder, "Colega", 20, 30, 5, 10)
	seedMember(t, s, campaignID, intruderChar)

	return selfInitiativeFixture{
		srv: s, campaignID: campaignID,
		charID: charID, player: player, intruder: intruder,
	}
}

// Encerrar a cena EXPIRA os efeitos de duração "cena" do grupo.
//
// O livro não deixa margem: "Cena. A habilidade dura uma cena inteira,
// encerrando-se quando esse momento da história acaba" (p227), e o começo e o
// fim de uma cena "são determinadas pelo andamento da história" (p11) — que é
// exatamente o que o mestre declara ao clicar em Encerrar cena.
//
// O teste vai pelo caso de uso (`rest.Party.EndScene`) e não pelo socket porque é ele que faz o
// gesto inteiro; o `onSceneEnd` acima só carrega transporte e autorização. E
// afirma os DOIS lados: o de cena sai, o de dia FICA. Limpar demais aqui
// apagaria a bênção que o grupo comprou para o dia todo, e ninguém veria.
func TestEndingTheSceneExpiresThePartySceneEffects(t *testing.T) {
	f := newEndSceneFixture(t)

	state, err := f.srv.restParty().EndScene(context.Background(), app.Caller{ID: f.gm.ID}, f.campaignID, f.sessionID)
	if err != nil {
		t.Fatalf("encerrar a cena: %v", err)
	}

	if state.SceneActive {
		t.Error("a cena continuou ligada")
	}
	if got := effectScopes(t, f.srv, f.charID); len(got) != 1 || got[0] != "day" {
		t.Errorf("sobraram os escopos %v na ficha do grupo, queria só [day]", got)
	}
}

// E alcança TODA a ficha do grupo, não só quem está na fila: a bênção foi
// lançada na cena e a cena acabou para os cinco, inclusive para quem o mestre
// nunca chegou a pôr no rastreador.
func TestEndingTheSceneReachesWhoIsNotInTheTracker(t *testing.T) {
	f := newEndSceneFixture(t)
	ausente := seedCharacter(t, f.srv, f.player, "Ladino de fora", 10, 10, 2, 2)
	seedMember(t, f.srv, f.campaignID, ausente)
	seedEffect(t, f.srv, ausente, "bencao", "scene")

	if _, err := f.srv.restParty().EndScene(context.Background(), app.Caller{ID: f.gm.ID}, f.campaignID, f.sessionID); err != nil {
		t.Fatalf("encerrar a cena: %v", err)
	}

	if got := effectScopes(t, f.srv, ausente); len(got) != 0 {
		t.Errorf("a ficha fora da fila ficou com %v", got)
	}
}

type endSceneFixture struct {
	srv        *Server
	gm         AuthUser
	player     int64
	campaignID int64
	sessionID  int64
	charID     int64
}

func newEndSceneFixture(t *testing.T) endSceneFixture {
	t.Helper()
	s := newTestServer(t)
	gmID := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gmID)
	sessionID := seedSession(t, s, campaignID)
	charID := seedCharacter(t, s, player, "Clérigo", 10, 10, 5, 5)
	seedMember(t, s, campaignID, charID)
	seedEffect(t, s, charID, "bencao", "scene")
	seedEffect(t, s, charID, "heroismo", "day")

	srv := s
	if _, err := s.sessions.StartScene(sessionID); err != nil {
		t.Fatalf("iniciar a cena: %v", err)
	}
	// O Clérigo entra na FILA: sem ele lá, "quem não está na fila" seria todo
	// mundo e o segundo teste não separaria nada.
	if _, err := s.sessions.AddInitiativeEntry(sessionID, sheetCombatant("Clérigo", 14, charID)); err != nil {
		t.Fatalf("pôr o Clérigo na fila: %v", err)
	}
	return endSceneFixture{
		srv: srv, gm: AuthUser{ID: gmID, Email: "mestre@t.com"}, player: player,
		campaignID: campaignID, sessionID: sessionID, charID: charID,
	}
}

// Não alcançar as fichas do grupo ABORTA o gesto inteiro: a cena continua
// ligada. Desligá-la assim mesmo deixaria o botão parecendo ter funcionado —
// fila zerada na tela e as bênçãos vivas na ficha.
func TestEndingTheSceneDoesNotTurnItOffIfItDidNotReachTheSheets(t *testing.T) {
	f := newEndSceneFixture(t)
	if _, err := f.srv.db.Exec("DROP TABLE campaign_members"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}

	if _, err := f.srv.restParty().EndScene(context.Background(), app.Caller{ID: f.gm.ID}, f.campaignID, f.sessionID); err == nil {
		t.Fatal("encerrou sem ter conseguido alcançar as fichas do grupo")
	}

	if !f.srv.sessions.GetState(f.sessionID).SceneActive {
		t.Error("a cena foi desligada mesmo assim")
	}
}
