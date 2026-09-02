package api

import "t20engine/aovivo"

import (
	"context"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// O patch que chega do socket é montado por uma LISTA de campos escrita à mão,
// e uma lista assim envelhece: ao entrar o `creatureId` (ALE-137) o cliente
// passou a mandá-lo e o servidor a descartá-lo em silêncio, com tudo
// compilando. Este teste percorre os campos em vez de conferir um.
func TestParseEntryPatchLosesNoField(t *testing.T) {
	patch := parseEntryPatch(map[string]any{
		"label":       "Chefe bandido",
		"initiative":  float64(17),
		"characterId": float64(3),
		"hpCurrent":   float64(20),
		"hpMax":       float64(30),
		"mpCurrent":   float64(5),
		"mpMax":       float64(10),
		"hpHidden":    true,
		"creatureId":  float64(7),
	})

	faltando := []string{}
	if patch.Label == nil {
		faltando = append(faltando, "label")
	}
	if patch.Initiative == nil {
		faltando = append(faltando, "initiative")
	}
	if patch.CharacterID == nil {
		faltando = append(faltando, "characterId")
	}
	if patch.HpCurrent == nil || patch.HpMax == nil {
		faltando = append(faltando, "hp")
	}
	if patch.MpCurrent == nil || patch.MpMax == nil {
		faltando = append(faltando, "mp")
	}
	if patch.HpHidden == nil {
		faltando = append(faltando, "hpHidden")
	}
	if patch.CreatureID == nil {
		faltando = append(faltando, "creatureId")
	}
	if len(faltando) > 0 {
		t.Fatalf("o parser descartou: %v", faltando)
	}
	if *patch.CreatureID != 7 {
		t.Errorf("creatureId chegou como %d, queria 7", *patch.CreatureID)
	}
}

// Condição em NPC (ALE-122, destravada pela ALE-137). A lista vem do CATÁLOGO
// e não de uma cópia escrita aqui: a cópia anterior desviou do livro — faltava
// `enfeitiçado`, e aplicá-la dava 400 para todo mundo.
func TestParseConditionsFiltersByTheCatalog(t *testing.T) {
	// Repare no ç: `enfeitiçado` é o ÚNICO dos 35 ids do catálogo com acento —
	// todos os outros são normalizados ("caido", não "caído"). É o mesmo id que
	// já derrubou a aplicação com 400 quando a API tinha a lista à mão, e a
	// grafia irregular é o que faz qualquer cópia errar de novo.
	list := parseConditions([]any{"caido", "inventada", "enfeitiçado", "atordoado"})

	if len(list) != 3 {
		t.Fatalf("passaram %v, queria as três do livro", list)
	}
	for _, id := range list {
		if id == "inventada" {
			t.Fatalf("id fora do catálogo passou: %v", list)
		}
	}
}

// Id desconhecido derruba um item, não a aplicação inteira: no meio do combate
// o mestre perderia as outras condições junto.
func TestParseConditionsNeitherDuplicatesNorBreaks(t *testing.T) {
	list := parseConditions([]any{"caido", "caido", 42, nil, ""})

	if len(list) != 1 || list[0] != "caido" {
		t.Fatalf("esperava só caido uma vez, veio %v", list)
	}
	if parseConditions(nil) == nil {
		t.Fatal("lista ausente tem de virar vazia, não nil — o cliente itera sem checar")
	}
}

// A condição é estado de COMBATE e mora na linha, como os PV atuais: o bloco de
// criatura descreve o vilão, e ele não volta na semana seguinte ainda caído.
func TestAConditionEntersAndLeavesTheEntry(t *testing.T) {
	st := aovivo.EmptyRuntimeState()
	id := idCounter()
	_ = aovivo.AddEntry(st, npc("Ogro", 12), id)

	aplicadas := []string{"caido", "atordoado"}
	if err := aovivo.UpdateEntry(st, "e1", aovivo.EntryPatch{Conditions: &aplicadas}); err != nil {
		t.Fatalf("aplicar: %v", err)
	}
	if len(st.Initiative[0].Conditions) != 2 {
		t.Fatalf("aplicou %v", st.Initiative[0].Conditions)
	}

	vazio := []string{}
	_ = aovivo.UpdateEntry(st, "e1", aovivo.EntryPatch{Conditions: &vazio})
	if len(st.Initiative[0].Conditions) != 0 {
		t.Fatalf("limpar deixou %v", st.Initiative[0].Conditions)
	}
}

// A iniciativa do jogador é somada pelo SERVIDOR (ALE-213).
//
// Antes o cliente mandava o total já pronto e quem decidia o bônus da perícia
// era o navegador — uma segunda implementação de regra do livro, livre para
// divergir do motor, que é exatamente o que a ALE-104 apagou. Agora ele manda o
// d20 e o Go pergunta à ficha COMPUTADA.
//
// O d20 continua vindo de fora, e de propósito: a mesa que rola dado FÍSICO
// digita o número, e nesse caminho não existe dado para o servidor rolar.
//
// O nível 8 é o que torna o teste honesto. Metade do nível entra em toda
// perícia — regra do motor, provada em `engine/` —, então o bônus é 4 e o 17
// NÃO pode ter vindo do cliente, que mandou 13.
func TestThePlayerInitiativeIsSummedByTheServer(t *testing.T) {
	f := newSelfInitiativeFixture(t)

	entry, err := f.srv.selfInitiativeEntry(f.player, f.campaignID, f.charID, 13)
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
		if _, err := f.srv.selfInitiativeEntry(f.player, f.campaignID, f.charID, d20); err == nil {
			t.Errorf("d20 %d passou", d20)
		}
	}
	// E a fronteira dos dois lados vale: 1 e 20 são dados de verdade.
	for _, d20 := range []int64{1, 20} {
		if _, err := f.srv.selfInitiativeEntry(f.player, f.campaignID, f.charID, d20); err != nil {
			t.Errorf("d20 %d recusado: %v", d20, err)
		}
	}
}

// O "self" do `initiative-self` é o que separa este caminho dos outros, que são
// todos do mestre: sem porta de papel, quem o guarda é o `resolveCombatant`, e
// ele recusa quem não é dono do personagem.
func TestRecordingSomeoneElsesInitiativeIsRefused(t *testing.T) {
	f := newSelfInitiativeFixture(t)

	_, err := f.srv.selfInitiativeEntry(f.intruder, f.campaignID, f.charID, 10)

	if err == nil {
		t.Fatal("um jogador registrou a iniciativa do personagem de outro")
	}
	if !strings.Contains(err.Error(), "neither the GM") {
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
	s.catalogs = catalogs

	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	intruder := seedUser(t, s, "intruso@t.com")
	campaignID := seedCampaign(t, s, gm)
	charID := seedCharacterAtLevel(t, s, player, "Arcanista", 8, 20, 30, 5, 10)
	seedMember(t, s, campaignID, charID, "player")
	if _, err := s.queries.CreateExpertise(ctx, sqlcgen.CreateExpertiseParams{
		Characterid: charID, Name: "Iniciativa", Attribute: "dexterity", Trained: 0, Custom: 0,
	}); err != nil {
		t.Fatalf("semear perícia: %v", err)
	}
	// Intruso na MESMA mesa: recusar alguém de fora seria recusar pela membresia,
	// e a regra que este teste mira é a POSSE do personagem.
	intruderChar := seedCharacter(t, s, intruder, "Colega", 20, 30, 5, 10)
	seedMember(t, s, campaignID, intruderChar, "player")

	return selfInitiativeFixture{
		srv: s, campaignID: campaignID,
		charID: charID, player: player, intruder: intruder,
	}
}

// Encerrar a cena EXPIRA os efeitos de duração "cena" do grupo (ALE-220).
//
// O livro não deixa margem: "Cena. A habilidade dura uma cena inteira,
// encerrando-se quando esse momento da história acaba" (p227), e o começo e o
// fim de uma cena "são determinadas pelo andamento da história" (p11) — que é
// exatamente o que o mestre declara ao clicar em Encerrar cena.
//
// O teste vai pelo `endSceneForTable` e não pelo socket porque é ele que faz o
// gesto inteiro; o `onSceneEnd` acima só carrega transporte e autorização. E
// afirma os DOIS lados: o de cena sai, o de dia FICA. Limpar demais aqui
// apagaria a bênção que o grupo comprou para o dia todo, e ninguém veria.
func TestEndingTheSceneExpiresThePartySceneEffects(t *testing.T) {
	f := newEndSceneFixture(t)

	state, err := f.srv.endSceneForTable(f.gm, f.campaignID, f.sessionID)
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
	seedMember(t, f.srv, f.campaignID, ausente, "player")
	seedEffect(t, f.srv, ausente, "bencao", "scene")

	if _, err := f.srv.endSceneForTable(f.gm, f.campaignID, f.sessionID); err != nil {
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
	seedMember(t, s, campaignID, charID, "player")
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
// ligada. Desligá-la assim mesmo devolveria o defeito da ALE-220 com o botão
// parecendo ter funcionado — fila zerada na tela e as bênçãos vivas na ficha.
func TestEndingTheSceneDoesNotTurnItOffIfItDidNotReachTheSheets(t *testing.T) {
	f := newEndSceneFixture(t)
	if _, err := f.srv.db.Exec("DROP TABLE campaign_members"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}

	if _, err := f.srv.endSceneForTable(f.gm, f.campaignID, f.sessionID); err == nil {
		t.Fatal("encerrou sem ter conseguido alcançar as fichas do grupo")
	}

	if !f.srv.sessions.GetState(f.sessionID).SceneActive {
		t.Error("a cena foi desligada mesmo assim")
	}
}
