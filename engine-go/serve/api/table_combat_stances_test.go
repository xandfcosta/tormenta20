package api

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS POSTURAS DE COMBATE DO CAVALEIRO (ALE-423, p54).
//
//	"Alguns poderes do cavaleiro são Posturas de Combate. Esses poderes
//	 compartilham as seguintes regras. • Assumir uma postura gasta uma ação de
//	 movimento e 2 PM. • Os efeitos de uma postura duram até o final da cena, a
//	 menos que sua descrição diga o contrário. • Você só pode manter uma
//	 postura por vez." (p54)
//
// As seis estavam no catálogo como `passive`: a ficha as listava, elas não
// custavam nada, não se assumia nenhuma, e nenhum número se mexia. O app
// modelava DUAS posturas no mundo inteiro — a Fúria e a Inspiração.
//
// INTEGRAÇÃO porque o que se prende são três regras que vivem em camadas
// diferentes: o custo do TURNO é da cena, o custo em PM e a exclusividade são
// do caso de uso, e o que cada postura FAZ é do motor. Um teste de unidade de
// qualquer uma delas não diria nada sobre as outras duas.

// knightOnTurn é a bancada: um cavaleiro de nível 6 com quatro Posturas de
// Combate escolhidas, NA VEZ, e um Goblin na fila.
//
// NA VEZ porque assumir postura custa ação de MOVIMENTO (p54), e fora da vez
// não há turno de onde tirá-la.
func knightOnTurn(t *testing.T) (sceneFixture, int64, string, string) {
	t.Helper()
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gm)
	sessionID := seedSession(t, s, campaignID)

	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: player, Name: "Postural", Origin: "guarda", Level: 6,
		Strength: 3, Dexterity: 1, Constitution: 4, Intelligence: 0, Wisdom: 1, Charisma: 2,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o cavaleiro: %v", err)
	}
	seedClasse(t, s, id, "Cavaleiro", 6)
	seedMember(t, s, campaignID, id)
	f := sceneFixture{s: s, gm: gm, player: player,
		campaignID: campaignID, sessionID: sessionID, charID: id}

	// O ESCUDO é pré-requisito da Muralha Intransponível (p54).
	if _, err := s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: id, Catalogid: sql.NullString{String: "escudo-pesado", Valid: true},
		Name: "Escudo pesado", Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: "wielded", Valid: true},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("empunhar o escudo: %v", err)
	}
	// A LINHA DE REFLEXOS existe para o bônus ter onde pousar: a ficha calcula
	// a resistência a partir da perícia, e sem a linha o ladrilho fica em +0 —
	// a Muralha somaria +1 em algo que não está na ficha.
	if _, err := s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: id, Name: "Reflexos", Attribute: "dexterity", Trained: 0, Custom: 0,
	}); err != nil {
		t.Fatalf("semear Reflexos: %v", err)
	}
	for _, power := range []string{
		"class.cavaleiro.postura-foco-de-batalha",
		"class.cavaleiro.postura-muralha-intransponivel",
		"class.cavaleiro.postura-torre-inabalavel",
	} {
		if refused := powerCommand(t, f, id, "escolhe/"+power, ""); refused != "" {
			t.Fatalf("escolher %s: %s", power, refused)
		}
	}
	startActionSceneWithTheKnightOnTurn(t, f)
	cavaleiro := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Postural")
	goblin := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Goblin")
	return f, id, cavaleiro, goblin
}

func startActionSceneWithTheKnightOnTurn(t *testing.T, f sceneFixture) {
	t.Helper()
	store := f.s.sessions
	if _, err := store.State(t.Context(), f.sessionID); err != nil {
		t.Fatalf("carregar a sessão: %v", err)
	}
	if _, err := store.StartScene(context.Background(), f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("começar a cena: %v", err)
	}
	goblin := "goblin-salteador"
	if _, err := store.AddInitiativeEntry(context.Background(), f.sessionID, live.InitiativeEntry{
		Label: "Goblin", Initiative: 20, Type: "npc", MonsterID: &goblin,
	}); err != nil {
		t.Fatalf("pôr o Goblin na fila: %v", err)
	}
	if _, err := store.AddInitiativeEntry(context.Background(), f.sessionID,
		sheetCombatant("Postural", 5, f.charID)); err != nil {
		t.Fatalf("pôr o cavaleiro na fila: %v", err)
	}
	for range 2 { // a primeira vez é do Goblin, a segunda do cavaleiro
		if _, err := store.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
}

// assumes entra numa postura pela rota da ficha, e devolve a recusa.
func assumes(t *testing.T, f sceneFixture, id int64, flag string) string {
	t.Helper()
	return powerCommand(t, f, id, "postura/"+flag+"/entra", "")
}

// liveStances são as posturas em pé, lidas da aba Efeitos.
func liveStances(t *testing.T, f sceneFixture, id int64) []string {
	t.Helper()
	// "Posturas ativas" e não "POSTURAS ATIVAS": a caixa alta é do CSS, e quem
	// lê o HTML servido lê o que foi ESCRITO. A primeira versão desta sonda
	// procurava o que a TELA mostra e não achava nada — o mesmo erro que o
	// guia nomeia para tipografia.
	bloco := entre(t, sheetTab(t, f, id, "conditionals"), "Posturas ativas", "Efeitos ativos")
	live := []string{}
	for _, nome := range []string{
		"Foco de Batalha", "Muralha Intransponível", "Torre Inabalável",
		"Aríete Implacável", "Castigo de Ferro", "Provocação Petulante",
	} {
		if strings.Contains(bloco, nome) {
			live = append(live, nome)
		}
	}
	return live
}

// UMA POSTURA POR VEZ — *"você só pode manter uma postura por vez"* (p54).
//
// É a regra que o app não tinha como representar: ele modelava duas posturas
// no mundo inteiro, de classes diferentes, e elas nunca se encontraram.
func TestAssumingACombatStanceDropsThePreviousOne(t *testing.T) {
	f, cavaleiro, _, _ := knightOnTurn(t)

	if refused := assumes(t, f, cavaleiro, "postura-muralha"); refused != "" {
		t.Fatalf("assumir a Muralha: %s", refused)
	}
	if live := liveStances(t, f, cavaleiro); len(live) != 1 || live[0] != "Muralha Intransponível" {
		t.Fatalf("depois da primeira, as posturas em pé são %v", live)
	}
	// A SEGUNDA DERRUBA A PRIMEIRA, e é a fatia inteira numa asserção.
	if refused := assumes(t, f, cavaleiro, "postura-torre"); refused != "" {
		t.Fatalf("assumir a Torre: %s", refused)
	}
	if live := liveStances(t, f, cavaleiro); len(live) != 1 || live[0] != "Torre Inabalável" {
		t.Errorf("a troca de postura deixou %v em pé", live)
	}
}

// ASSUMIR CUSTA UMA AÇÃO DE MOVIMENTO (p54), e fora da vez não há de onde
// tirá-la.
//
// O custo do turno não era cobrado por NENHUMA postura antes desta fatia: as
// duas que existiam eram livre e padrão, e a livre não cobra nada — o buraco
// passou despercebido porque a Fúria não o expunha.
func TestAssumingACombatStanceCostsTheMovementAction(t *testing.T) {
	f, cavaleiro, _, _ := knightOnTurn(t)
	if refused := assumes(t, f, cavaleiro, "postura-muralha"); refused != "" {
		t.Fatalf("assumir a Muralha: %s", refused)
	}
	cena := stateOf(t, f.s.sessions, f.sessionID).Scene
	if cena.MovementLeft {
		t.Error("assumir a postura não gastou a ação de movimento")
	}
	// A PADRÃO FICA: a postura custa movimento, e o cavaleiro ainda ataca.
	if !cena.StandardLeft {
		t.Error("assumir a postura comeu a ação padrão junto")
	}
	// A SEGUNDA SAI, e sai da AÇÃO PADRÃO: *"você pode trocar sua ação padrão
	// por uma ação de movimento"* (p233), e a troca é de mão única. Esperar
	// recusa aqui seria afirmar uma regra que o livro não tem — foi o que a
	// primeira versão deste caso fez, e o motor estava certo.
	if refused := assumes(t, f, cavaleiro, "postura-torre"); refused != "" {
		t.Fatalf("a segunda postura tinha de sair pela troca da p233, e veio %q", refused)
	}
	if cena := stateOf(t, f.s.sessions, f.sessionID).Scene; cena.StandardLeft {
		t.Error("a segunda postura não cobrou a ação padrão pela troca da p233")
	}
	// A TERCEIRA não sai: o turno acabou.
	if refused := assumes(t, f, cavaleiro, "postura-foco-de-batalha"); !strings.Contains(refused, "não sobrou ação") {
		t.Errorf("a terceira postura tinha de ser recusada por falta de ação, e veio %q", refused)
	}
}

// A TORRE INABALÁVEL SOBE E NÃO PROMETE NÚMERO NENHUM.
//
// O livro dá a ela *"soma sua Constituição na Defesa"* e *"você não pode se
// deslocar"* (p55), e o motor não alcança nenhum dos dois: `scale` só é lido
// no caminho dos vitais, e `factor` sob condição é pulado pelo colhedor. Os
// dois guardas em `domain/catalog/modifier_reach_test.go` impedem que alguém
// os declare mesmo assim — um modificador que não é lido vira um verbete que
// cobra 2 PM e não muda nada, que é o defeito que esta família já produziu.
//
// O QUE ELA FAZ HOJE É INFORMAR, e isso não é pouco: a postura em pé aparece
// na ficha, o mestre a vê, e os dois efeitos são dele. O caso prende
// exatamente isso — ela sobe, custa, e NÃO mente sobre a Defesa.
func TestTheUnshakableTowerStandsWithoutPromisingNumbers(t *testing.T) {
	f, cavaleiro, _, _ := knightOnTurn(t)
	defesaAntes := defenceOnTheSheet(t, f, cavaleiro)

	if refused := assumes(t, f, cavaleiro, "postura-torre"); refused != "" {
		t.Fatalf("assumir a Torre: %s", refused)
	}
	if live := liveStances(t, f, cavaleiro); len(live) != 1 || live[0] != "Torre Inabalável" {
		t.Fatalf("a Torre não subiu: as posturas em pé são %v", live)
	}
	if depois := defenceOnTheSheet(t, f, cavaleiro); depois != defesaAntes {
		t.Errorf("a Torre mexeu na Defesa (%d → %d) sem o motor saber somar Constituição: "+
			"ou o motor aprendeu e o `scale` voltou, ou alguém declarou o que ele não lê",
			defesaAntes, depois)
	}
}

// A MURALHA DÁ +1 NA DEFESA E EM REFLEXOS (p54), e é a outra metade do que o
// motor consegue dizer das seis.
func TestTheImpassableWallRaisesDefenceAndReflexes(t *testing.T) {
	f, cavaleiro, _, _ := knightOnTurn(t)
	defesaAntes := defenceOnTheSheet(t, f, cavaleiro)
	reflexosAntes := reflexesOnTheSheet(t, f, cavaleiro)

	if refused := assumes(t, f, cavaleiro, "postura-muralha"); refused != "" {
		t.Fatalf("assumir a Muralha: %s", refused)
	}
	if defesaDepois := defenceOnTheSheet(t, f, cavaleiro); defesaDepois != defesaAntes+1 {
		t.Errorf("a Muralha tinha de dar +1 na Defesa: %d virou %d", defesaAntes, defesaDepois)
	}
	if depois := reflexesOnTheSheet(t, f, cavaleiro); depois != reflexosAntes+1 {
		t.Errorf("a Muralha tinha de dar +1 em Reflexos: %d virou %d", reflexosAntes, depois)
	}
}

// O FOCO DE BATALHA ENCHE A POÇA QUANDO O CAVALEIRO É ATACADO, e o livro NÃO
// pede acerto: *"sempre que um inimigo atacá-lo"* (p54).
//
// É o segundo produtor de PM temporário, e ele reusa inteiro o mecanismo
// cumulativo — o que esta fatia acrescentou foi o gatilho do lado de quem
// APANHA.
func TestTheBattleFocusFillsThePoolWhenTheKnightIsAttacked(t *testing.T) {
	f, cavaleiro, alvo, goblin := knightOnTurn(t)
	if refused := assumes(t, f, cavaleiro, "postura-foco-de-batalha"); refused != "" {
		t.Fatalf("assumir o Foco de Batalha: %s", refused)
	}
	_, antes := manaOnTheSheet(t, f, cavaleiro)

	// O GOLPE ERRA de propósito: o ponto é do ataque, não do acerto.
	theGameMasterConfirmsAMissAgainst(t, f, goblin, alvo)
	if _, depois := manaOnTheSheet(t, f, cavaleiro); depois != antes+1 {
		t.Errorf("ser atacado tinha de dar 1 PM temporário mesmo no erro: %d virou %d", antes, depois)
	}
}

// E QUEM ATACA NÃO GANHA NADA COM ELE — o controle do lado.
//
// Sem este caso, um gatilho disparado para o ATACANTE em vez de para o alvo
// passaria no caso acima sempre que os dois fossem o mesmo personagem, e
// passaria despercebido quando não fossem.
func TestNoBattleFocusPointGoesToTheAttacker(t *testing.T) {
	f, cavaleiro, alvo, goblin := knightOnTurn(t)
	if refused := assumes(t, f, cavaleiro, "postura-foco-de-batalha"); refused != "" {
		t.Fatalf("assumir o Foco de Batalha: %s", refused)
	}
	_, antes := manaOnTheSheet(t, f, cavaleiro)

	// Agora o CAVALEIRO é quem ataca.
	theGameMasterConfirmsAMissAgainst(t, f, alvo, goblin)
	if _, depois := manaOnTheSheet(t, f, cavaleiro); depois != antes {
		t.Errorf("atacar encheu a poça de quem atacou: %d virou %d", antes, depois)
	}
}

// theGameMasterConfirmsAMissAgainst semeia um golpe que ERRA e confirma.
func theGameMasterConfirmsAMissAgainst(t *testing.T, f sceneFixture, attacker, target string) {
	t.Helper()
	if _, err := f.s.sessions.ProposeAttack(context.Background(), f.sessionID, live.PendingAttack{
		AttackerEntryID: attacker, TargetEntryID: target,
		Weapon: "Espada longa", Roll: 3, Total: 6, Defense: 20,
		Hit: false, Critical: false, Melee: true,
	}); err != nil {
		t.Fatalf("semear o provisório: %v", err)
	}
	rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("confirmar o golpe deu %d", rec.Code)
	}
	if refusal := tableRefusal(t, rec.Body.String()); refusal != "" {
		t.Fatalf("confirmar o golpe foi recusado: %s", refusal)
	}
}

// AS TRÊS SONDAS ancoram no que o HTML tem para isso, e não em texto de tela.

var (
	defenceBadge = regexp.MustCompile(`>DEF</span>\s*<span[^>]*>(\d+)</span>`)
	actionValue  = regexp.MustCompile(`tabular-nums text-foreground">([^<]+)</span>`)
	skillTotal   = regexp.MustCompile(`aria-label="Refl ([+-]\d+)"`)
)

// defenceOnTheSheet é o número do crachá — o que a mesa pergunta em voz alta.
func defenceOnTheSheet(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	found := defenceBadge.FindStringSubmatch(sheetTab(t, f, id, "combat"))
	if found == nil {
		t.Fatal("não achei o crachá de Defesa na ficha")
	}
	n, err := strconv.Atoi(found[1])
	if err != nil {
		t.Fatalf("a Defesa não é número: %q", found[1])
	}
	return n
}

// strideOnTheActionsSurface é o deslocamento como a pessoa o LÊ: a linha Mover
// da superfície Ações, em metros.
//
// Pela superfície e não pela coluna: o motor guarda o deslocamento em
// QUADRADOS, e o que a Torre Inabalável promete zerar é o que a mesa fala.
func strideOnTheActionsSurface(t *testing.T, f sceneFixture) string {
	t.Helper()
	linha := entre(t, superficieDeAcoes(t, acoesDaMesa(t, f)), "Mover", "</li>")
	found := actionValue.FindStringSubmatch(linha)
	if found == nil {
		t.Fatalf("não achei o deslocamento na linha de Mover:\n%s", recorte(linha))
	}
	return found[1]
}

// reflexesOnTheSheet é o total de Reflexos, lido do RÓTULO ACESSÍVEL do ladrilho
// de resistência na aba Combate.
//
// Pelo `aria-label` e não pelo texto: o ladrilho escreve "Refl" e o número em
// nós separados, e os testes de resistência não estão na aba Perícias — a
// primeira versão desta sonda os procurou lá e não achou nada.
func reflexesOnTheSheet(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	found := skillTotal.FindStringSubmatch(sheetTab(t, f, id, "combat"))
	if found == nil {
		t.Fatal("não achei o ladrilho de Reflexos na aba Combate")
	}
	n, err := strconv.Atoi(strings.TrimPrefix(found[1], "+"))
	if err != nil {
		t.Fatalf("o total de Reflexos não é número: %q", found[1])
	}
	return n
}
