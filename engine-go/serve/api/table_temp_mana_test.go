package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A POÇA DE PM TEMPORÁRIO (ALE-423, p106).
//
//	"Certos efeitos fornecem PV ou PM temporários. Eles são somados a seus
//	 pontos atuais, mesmo que ultrapassem o máximo. Pontos temporários são
//	 sempre os primeiros a serem gastos. Caso não seja especificado o contrário,
//	 pontos temporários desaparecem no fim do dia." (p106)
//
// O `tempMp` existia no motor só como NOME DE ALVO: nenhum verbete o produzia,
// nada o gastava, e o `vitals.go` dizia por escrito que o app não o modelava.
// Seis verbetes do livro prometiam PM temporário e nenhum entregava.
//
// INTEGRAÇÃO porque o que se prende é um CANO que atravessa tudo: o gatilho na
// mesa escreve a poça, o funil de gasto a drena antes do poço, e a decisão de
// "cabe este gasto?" tem de contar com ela. Um teste de unidade do plano
// provaria a aritmética e nada sobre o bardo conseguir conjurar.

// manaBadge acha a fração do crachá pela ÂNCORA que a fileira tem para isso.
//
// A primeira versão era `PM</span>.*?(\d+)/(\d+)`, e ela casava com o "2 PM"
// do rótulo do botão de Inspiração — um número que não se mexe. Dois casos
// reprovaram dizendo que o motor não funcionava, e o motor estava certo: a
// sonda é que lia outro número.
var manaBadge = regexp.MustCompile(`data-vital="PM"[^>]*>(\d+)/(\d+)</span>`)

// manaReserve acha a parcela à parte, que vem logo depois da fração.
var manaReserve = regexp.MustCompile(`data-vital="PM"[^>]*>\d+/\d+</span>\s*<span[^>]*text-hp-temp[^>]*>\+(\d+)</span>`)

// bardOnTurn é a bancada: um bardo de nível 6 com Esgrima Mágica e Golpe
// Mágico, espada curta na mão, SOB INSPIRAÇÃO, e um Goblin na fila.
//
// A VEZ FICA COM O GOBLIN de propósito, pela razão que o arquivo do bônus
// cumulativo registra: confirmar o ataque de quem está na vez cobra a ação
// padrão, e o segundo golpe seria recusado — num sinal com status 200.
func bardOnTurn(t *testing.T) (sceneFixture, int64, string, string) {
	t.Helper()
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gm)
	sessionID := seedSession(t, s, campaignID)

	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: player, Name: "Trovador", Origin: "artista", Level: 6,
		Strength: 2, Dexterity: 3, Constitution: 2, Intelligence: 1, Wisdom: 0, Charisma: 4,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o bardo: %v", err)
	}
	seedClasse(t, s, id, "Bardo", 6)
	seedMember(t, s, campaignID, id)
	f := sceneFixture{s: s, gm: gm, player: player,
		campaignID: campaignID, sessionID: sessionID, charID: id}

	// A ARMA CORPO A CORPO é o que faz o gatilho existir: o Golpe Mágico pede
	// *"acertar um ataque corpo a corpo"* (p45), e com um arco na mão a poça
	// nunca encheria.
	if _, err := s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: id, Catalogid: sql.NullString{String: "espada-curta", Valid: true},
		Name: "Espada curta", Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: "wielded", Valid: true},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("empunhar a espada: %v", err)
	}
	// A CADEIA DE PRÉ-REQUISITO, pelos gestos da ficha: o Golpe Mágico exige a
	// Esgrima Mágica, e gravar a coluna à mão montaria uma ficha que o app não
	// sabe produzir.
	for _, power := range []string{"class.bardo.esgrima-magica", "class.bardo.golpe-magico"} {
		if refused := powerCommand(t, f, id, "escolhe/"+power, ""); refused != "" {
			t.Fatalf("escolher %s: %s", power, refused)
		}
	}
	// A INSPIRAÇÃO É AÇÃO PADRÃO (p44), então entrar nela PEDE A VEZ — e desde
	// que assumir postura cobra do turno, a ordem da bancada importa: entra na
	// postura na vez dele, e só DEPOIS a vez passa para o Goblin.
	startActionSceneWithTheBardOnTurn(t, f)
	if refused := powerCommand(t, f, id, "postura/inspiracao/entra", ""); refused != "" {
		t.Fatalf("entrar em Inspiração: %s", refused)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("passar a vez deu %d", rec.Code)
	}
	// O CONTROLE DA BANCADA: a Inspiração está MESMO acesa.
	if !strings.Contains(sheetTab(t, f, id, "conditionals"), "Inspiração") {
		t.Fatal("a bancada não entrou em Inspiração")
	}
	bardo := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Trovador")
	goblin := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Goblin")
	return f, id, bardo, goblin
}

// startActionSceneWithTheBardOnTurn abre a cena de ação com o bardo NA VEZ e um
// Goblin esperando.
func startActionSceneWithTheBardOnTurn(t *testing.T, f sceneFixture) {
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
		sheetCombatant("Trovador", 5, f.charID)); err != nil {
		t.Fatalf("pôr o bardo na fila: %v", err)
	}
	// DUAS VEZES: a primeira é do Goblin (iniciativa 20), a segunda do bardo.
	for range 2 {
		if _, err := store.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
}

// sheetTab é a ficha aberta numa aba.
func sheetTab(t *testing.T, f sceneFixture, id int64, tab string) string {
	t.Helper()
	return f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=%s", id, tab), "").Body.String()
}

// manaOnTheSheet é o par (atual, máximo) do crachá de PM, e a RESERVA ao lado.
func manaOnTheSheet(t *testing.T, f sceneFixture, id int64) (current, reserve int) {
	t.Helper()
	screen := sheetTab(t, f, id, "conditionals")
	found := manaBadge.FindStringSubmatch(screen)
	if found == nil {
		t.Fatalf("não achei o crachá de PM na ficha")
	}
	current, err := strconv.Atoi(found[1])
	if err != nil {
		t.Fatalf("o PM atual não é número: %q", found[1])
	}
	// A RESERVA só é desenhada quando existe, então não achá-la é ZERO e não
	// falha — é o estado normal de quase toda ficha.
	if m := manaReserve.FindStringSubmatch(screen); m != nil {
		reserve, _ = strconv.Atoi(m[1])
	}
	return current, reserve
}

// theGameMasterConfirmsAMeleeHit semeia um acerto CORPO A CORPO e confirma.
func theGameMasterConfirmsAMeleeHit(t *testing.T, f sceneFixture, attacker, target string) {
	t.Helper()
	if _, err := f.s.sessions.ProposeAttack(context.Background(), f.sessionID, live.PendingAttack{
		AttackerEntryID: attacker, TargetEntryID: target, TargetLabel: "Goblin",
		Weapon: "Espada curta", Roll: 14, Total: 19, Defense: 13,
		Hit: true, Critical: false, Melee: true, RawDamage: 1, Damage: 1,
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

// O GOLPE MÁGICO ENCHE A POÇA, e a ficha a mostra ao lado do poço.
//
// É a fatia inteira numa frase: um verbete que prometia PM temporário e não
// entregava passa a entregar, e o número aparece onde a pessoa lê o mana.
func TestTheBardsMeleeHitFillsTheTemporaryManaPool(t *testing.T) {
	f, bardo, atacante, goblin := bardOnTurn(t)
	antesPM, antesReserva := manaOnTheSheet(t, f, bardo)
	if antesReserva != 0 {
		t.Fatalf("a bancada já começou com reserva de %d", antesReserva)
	}

	theGameMasterConfirmsAMeleeHit(t, f, atacante, goblin)
	depoisPM, depoisReserva := manaOnTheSheet(t, f, bardo)
	if depoisReserva != 2 {
		t.Errorf("o golpe tinha de dar 2 PM temporários, e a reserva é %d", depoisReserva)
	}
	// A POÇA É PARCELA À PARTE: ela não entra no poço, porque *"são somados a
	// seus pontos atuais, mesmo que ultrapassem o máximo"* e o poço tem teto.
	if depoisPM != antesPM {
		t.Errorf("a poça mexeu no poço: %d virou %d", antesPM, depoisPM)
	}
}

// O ATAQUE À DISTÂNCIA NÃO ENCHE NADA — *"ao acertar um ataque corpo a
// corpo"* (p45).
//
// O controle da família: sem ele, uma implementação que acumulasse em todo
// acerto passaria no caso acima.
func TestNoRangedHitFillsTheBardsPool(t *testing.T) {
	f, bardo, atacante, goblin := bardOnTurn(t)

	if _, err := f.s.sessions.ProposeAttack(context.Background(), f.sessionID, live.PendingAttack{
		AttackerEntryID: atacante, TargetEntryID: goblin, TargetLabel: "Goblin",
		Weapon: "Arco curto", Roll: 14, Total: 19, Defense: 13,
		Hit: true, Critical: false, Melee: false, RawDamage: 1, Damage: 1,
	}); err != nil {
		t.Fatalf("semear o provisório: %v", err)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if _, reserva := manaOnTheSheet(t, f, bardo); reserva != 0 {
		t.Errorf("um acerto à distância encheu a poça em %d", reserva)
	}
}

// A POÇA É GASTA PRIMEIRO — *"pontos temporários são sempre os primeiros a
// serem gastos"* (p106).
//
// Conjurar é o gasto medido porque ele é o que a mesa faz o tempo todo, e
// porque o caminho dele é OUTRO que o da cobrança de poder: se a regra morasse
// no gesto em vez de no funil, um dos dois ficaria para trás.
func TestTheTemporaryPoolIsSpentBeforeTheRealOne(t *testing.T) {
	f, bardo, atacante, goblin := bardOnTurn(t)
	theGameMasterConfirmsAMeleeHit(t, f, atacante, goblin)
	antesPM, antesReserva := manaOnTheSheet(t, f, bardo)
	if antesReserva != 2 {
		t.Fatalf("a bancada não encheu a poça: reserva %d", antesReserva)
	}

	// A VEZ VOLTA PARA O BARDO: conjurar é ação padrão, e *"só a reação
	// acontece fora dela"* (p233). O golpe foi confirmado fora da vez de
	// propósito — ver o cabeçalho da bancada —, e é só agora que ela importa.
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("girar a vez deu %d", rec.Code)
	}
	// LUZ custa 1 PM, que é menos que a poça: ela sai inteira da reserva.
	castOnTheSheet(t, f, bardo, "luz")
	depoisPM, depoisReserva := manaOnTheSheet(t, f, bardo)
	if depoisReserva != antesReserva-1 {
		t.Errorf("a magia tinha de tirar 1 da reserva: %d virou %d", antesReserva, depoisReserva)
	}
	if depoisPM != antesPM {
		t.Errorf("a magia comeu o poço de verdade com a reserva cheia: %d virou %d", antesPM, depoisPM)
	}
}

// E O ±PM NÃO TOCA NA POÇA (decisão do dono).
//
// O ± é CORREÇÃO e não gasto: o mestre o usa para acertar um número que ficou
// errado, e drenar a poça com ele faria "−1" não fazer o que ele quer.
func TestTheManualStepDoesNotDrainTheTemporaryPool(t *testing.T) {
	f, bardo, atacante, goblin := bardOnTurn(t)
	theGameMasterConfirmsAMeleeHit(t, f, atacante, goblin)
	antesPM, antesReserva := manaOnTheSheet(t, f, bardo)

	rec := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/vitais/pm/-1?embutida=1", bardo), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o passo de PM deu %d", rec.Code)
	}
	depoisPM, depoisReserva := manaOnTheSheet(t, f, bardo)
	if depoisReserva != antesReserva {
		t.Errorf("o ±PM drenou a poça: %d virou %d", antesReserva, depoisReserva)
	}
	if depoisPM != antesPM-1 {
		t.Errorf("o ±PM tinha de tirar 1 do poço: %d virou %d", antesPM, depoisPM)
	}
}

// castOnTheSheet aprende e conjura pela aba Magias, e FALHA na recusa.
func castOnTheSheet(t *testing.T, f sceneFixture, id int64, spell string) {
	t.Helper()
	learn := fmt.Sprintf("/personagens/%d/magias/aprende/%s?tab=spells", id, spell)
	if rec := f.requests(t, f.player, http.MethodPost, learn, ""); rec.Code != http.StatusOK {
		t.Fatalf("aprender %q deu %d", spell, rec.Code)
	}
	cast := fmt.Sprintf("/personagens/%d/magias/conjura/%s?tab=spells", id, spell)
	rec := f.requests(t, f.player, http.MethodPost, cast, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("conjurar %q deu %d", spell, rec.Code)
	}
	if refusal := sceneRefusal(rec.Body.String()); refusal != "" {
		t.Fatalf("conjurar %q foi recusado: %s", spell, refusal)
	}
}

// O TETO É DO QUE SE GANHOU NA CENA, e GASTAR NÃO O DEVOLVE.
//
//	"Você pode ganhar um máximo de PM temporários por cena igual ao seu
//	 nível." (p45)
//
// É a regra que o bônus de ataque não tinha e esta precisa: lá nada consome o
// número, então "quanto vale" e "quanto foi ganho" andam iguais para sempre.
// Aqui a poça é GASTA, e um teto lido da poça deixaria o bardo acumulando dois
// PM por golpe a cena inteira — bastaria gastar entre um golpe e outro.
func TestSpendingTheTemporaryPoolDoesNotReopenTheSceneCap(t *testing.T) {
	f, bardo, atacante, goblin := bardOnTurn(t)

	// O bardo é de nível 6, e o golpe dá 2: três golpes enchem o teto.
	for range 3 {
		theGameMasterConfirmsAMeleeHit(t, f, atacante, goblin)
	}
	_, noTeto := manaOnTheSheet(t, f, bardo)
	if noTeto != 6 {
		t.Fatalf("três golpes tinham de dar 6 de reserva, e deram %d", noTeto)
	}
	// O CONTROLE DO TETO: o quarto golpe não move nada.
	theGameMasterConfirmsAMeleeHit(t, f, atacante, goblin)
	if _, depois := manaOnTheSheet(t, f, bardo); depois != noTeto {
		t.Fatalf("o quarto golpe passou do teto de nível: %d virou %d", noTeto, depois)
	}

	// GASTA UM, e volta a bater: o teto já foi alcançado, então a poça não sobe.
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("girar a vez deu %d", rec.Code)
	}
	castOnTheSheet(t, f, bardo, "luz")
	_, depoisDoGasto := manaOnTheSheet(t, f, bardo)
	if depoisDoGasto != noTeto-1 {
		t.Fatalf("a magia tinha de tirar 1 da reserva: %d virou %d", noTeto, depoisDoGasto)
	}
	// A VEZ SAI DO BARDO de novo: a magia gastou a ação padrão dele, e
	// confirmar na vez dele cobraria uma segunda — o golpe seria recusado, num
	// sinal com status 200, e o caso mediria o nada.
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("girar a vez deu %d", rec.Code)
	}
	theGameMasterConfirmsAMeleeHit(t, f, atacante, goblin)
	if _, depois := manaOnTheSheet(t, f, bardo); depois != depoisDoGasto {
		t.Errorf("gastar devolveu o teto: a reserva foi de %d para %d depois de um golpe novo",
			depoisDoGasto, depois)
	}
}

// A POÇA CONTA PARA DECIDIR SE O GASTO CABE.
//
// É a segunda metade da p106, e a que é fácil de esquecer: se o gasto drena a
// poça primeiro, então a poça também autoriza o gasto. Sem isto, o bardo de
// poço vazio e poça cheia seria recusado ao conjurar — a poça existiria só
// para ser ignorada, e a tela mostraria um número que não serve para nada.
func TestAnEmptyManaPoolStillCastsOnTheTemporaryReserve(t *testing.T) {
	f, bardo, atacante, goblin := bardOnTurn(t)
	theGameMasterConfirmsAMeleeHit(t, f, atacante, goblin)

	// O POÇO A ZERO, pelo funil de verdade.
	arrangePools(t, f.s, bardo, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.MpCurrent = 0
		return pools, nil
	})
	semPoco, reserva := manaOnTheSheet(t, f, bardo)
	if semPoco != 0 || reserva != 2 {
		t.Fatalf("a bancada queria poço 0 e reserva 2, e tem %d e %d", semPoco, reserva)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("girar a vez deu %d", rec.Code)
	}

	castOnTheSheet(t, f, bardo, "luz")
	depoisPoco, depoisReserva := manaOnTheSheet(t, f, bardo)
	if depoisReserva != 1 {
		t.Errorf("a magia tinha de sair da reserva, que iria de 2 para 1, e ela está em %d", depoisReserva)
	}
	// O POÇO NÃO VAI A NEGATIVO: PM que falta é gesto recusado, e um poço
	// negativo gravado desenharia uma barra para trás.
	if depoisPoco != 0 {
		t.Errorf("o poço vazio virou %d", depoisPoco)
	}
}

// A RESERVA DIZ DE QUE ELA É, e em cada fileira a frase é a sua.
//
// O `title` era fixo em "PV temporários — o dano gasta estes primeiro": com o
// PM ganhando poça, ele passaria a mentir na fileira de baixo, afirmando que
// aquele número é de vida e que o DANO o consome. O verbo é a diferença — o PV
// é gasto pelo que chega, o PM pelo que a pessoa escolhe fazer.
func TestEachTemporaryReserveSaysWhichVitalItIs(t *testing.T) {
	f, bardo, atacante, goblin := bardOnTurn(t)
	theGameMasterConfirmsAMeleeHit(t, f, atacante, goblin)

	screen := sheetTab(t, f, bardo, "conditionals")
	if !strings.Contains(screen, "PM temporários — conjurar e ativar poder gastam estes primeiro") {
		t.Errorf("a reserva de PM não diz que é de PM")
	}
	// O CONTROLE: a frase do PV continua existindo no arquivo servido para
	// quem a tem. Sem ele, apagar as duas passaria verde aqui.
	if strings.Contains(screen, "PV temporários — o dano gasta estes primeiro") {
		t.Errorf("a reserva de PM saiu com a frase do PV")
	}
}
