package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"t20engine/app"
	"t20engine/app/rest"
	"t20engine/domain/live"
)

// AS ARMAS NATURAIS (quadro de Ataques Desarmados e Armas Naturais; Chifres, p25; Mordida,
// p31; Dentes Afiados e Membros Extras, p137; Presas Primordiais, p135).
//
// O ataque extra: "uma vez por rodada, quando usa a ação agredir para atacar com outra
// arma, pode gastar 1 PM para fazer um ataque corpo a corpo extra". Ele abre depois da
// resposta do mestre ao golpe principal, porque só existe uma proposta por vez (decisão do
// dono).

func minotaurOnTurn(t *testing.T) (sceneFixture, int64, string) {
	t.Helper()
	f, id, goblin := barbarianOnTurn(t)
	seedRaca(t, f.s, id, "minotauro")
	return f, id, goblin
}

// naturalIndex é o índice da arma na linha de Atacar, que é o que o caminho do gesto leva.
func naturalIndex(t *testing.T, f sceneFixture, weapon string) int {
	t.Helper()
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	at := strings.Index(surface, "Atacar · "+weapon)
	if at < 0 {
		t.Fatalf("as Ações não têm a linha de %s", weapon)
	}
	found := regexp.MustCompile(`/acoes/atacar/(\d+)`).FindStringSubmatch(surface[at:])
	if found == nil {
		t.Fatalf("a linha de %s não tem o gesto de rolar", weapon)
	}
	var index int
	fmt.Sscan(found[1], &index)
	return index
}

func extraStrike(t *testing.T, f sceneFixture, id int64, index int, goblin string) string {
	t.Helper()
	return rollsFromTheActions(t, f, id, fmt.Sprintf("ataque-extra/%d", index), goblin)
}

func gmChecks(t *testing.T, f sceneFixture) {
	t.Helper()
	gmRules(t, f, "acertou")
	if refusal := gmRefusal(t, f, "/ataque/dano"); refusal != "" {
		t.Fatalf("o dano foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/confirmar"); refusal != "" {
		t.Fatalf("o ✓ foi recusado: %q", refusal)
	}
}

func TestEachNaturalWeaponHasAnAttackLine(t *testing.T) {
	cases := []struct{ name, race, powers, weapon, detail string }{
		{"Chifres", "minotauro", "", "Chifres", "1d6"},
		{"Mordida do trog", "trog", "", "Mordida", "1d6"},
		{"Dentes Afiados", "", `["dentes-afiados"]`, "Mordida", "1d4"},
		{"Membros Extras", "", `["membros-extras"]`, "Patas insetoides", "1d4"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, id, _ := barbarianOnTurn(t)
			if c.race != "" {
				seedRaca(t, f.s, id, c.race)
			}
			if c.powers != "" {
				seedPowerChoices(t, f.s, id, c.powers, "{}")
			}
			surface := superficieDeAcoes(t, acoesDaMesa(t, f))
			at := strings.Index(surface, "Atacar · "+c.weapon)
			if at < 0 {
				t.Fatalf("as Ações não têm a linha de %s", c.weapon)
			}
			if !strings.Contains(surface[at:min(at+300, len(surface))], c.detail) {
				t.Errorf("a linha de %s não diz o dano %s", c.weapon, c.detail)
			}
			if !strings.Contains(sheetTab(t, f, id, "combat"), c.weapon) {
				t.Errorf("a aba Combate não mostra %s", c.weapon)
			}
		})
	}
}

// O gesto real substitui as ativações soltas, que cobravam 1 PM sem rolar nada.
func TestNoLooseGestureChargesTheNaturalWeapon(t *testing.T) {
	f, _, _ := minotaurOnTurn(t)
	if surface := superficieDeAcoes(t, acoesDaMesa(t, f)); strings.Contains(surface, "/poderes/usa/race.minotauro.chifres") {
		t.Error("as Ações ainda oferecem pagar os chifres sem rolar nada")
	}
}

func TestTheNaturalWeaponIsAStandardAttackThatOpensNoExtra(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	horns := naturalIndex(t, f, "Chifres")
	if refusal := rollsFromTheActions(t, f, id, fmt.Sprintf("atacar/%d", horns), goblin); refusal != "" {
		t.Fatalf("o golpe com os chifres foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Weapon != "Chifres" {
		t.Fatalf("a proposta devia ser com os chifres: %+v", pa)
	}
	if sceneOf(t, f).StandardLeft {
		t.Error("atacar com os chifres é a ação padrão, e ela não foi paga")
	}
	gmChecks(t, f)
	if refusal := extraStrike(t, f, id, horns, goblin); !strings.Contains(refusal, "outra arma") {
		t.Errorf("o ataque com a própria arma natural não abre o extra (p25), e veio %q", refusal)
	}
}

func TestTheExtraAttackWaitsForTheGMAndCostsOnePM(t *testing.T) {
	for name, answers := range map[string]func(sceneFixture){
		"depois do erro": func(f sceneFixture) { gmRules(t, f, "errou") },
		"depois do ✓":    func(f sceneFixture) { gmChecks(t, f) },
	} {
		t.Run(name, func(t *testing.T) {
			f, id, goblin := minotaurOnTurn(t)
			horns := naturalIndex(t, f, "Chifres")
			if refusal := extraStrike(t, f, id, horns, goblin); !strings.Contains(refusal, "agredir") {
				t.Fatalf("sem agredir, o extra devia ser recusado, e veio %q", refusal)
			}
			if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
				t.Fatalf("o golpe principal: %q", refusal)
			}
			if refusal := extraStrike(t, f, id, horns, goblin); !strings.Contains(refusal, "mestre") {
				t.Fatalf("com o golpe esperando o mestre, o extra devia ser recusado, e veio %q", refusal)
			}
			answers(f)
			before, scene := mpOf(t, f.s, id), *sceneOf(t, f)
			if refusal := extraStrike(t, f, id, horns, goblin); refusal != "" {
				t.Fatalf("o extra foi recusado: %q", refusal)
			}
			pa := pendingOf(t, f)
			if pa == nil || pa.Weapon != "Chifres" || pa.Ruling != "" {
				t.Fatalf("o extra devia virar proposta com os chifres: %+v", pa)
			}
			if got := mpOf(t, f.s, id); got != before-1 {
				t.Errorf("o extra custa 1 PM: %d virou %d", before, got)
			}
			after := sceneOf(t, f)
			if after.StandardLeft != scene.StandardLeft || after.MovementLeft != scene.MovementLeft {
				t.Errorf("o extra não gasta ação: %+v virou %+v", scene, *after)
			}
		})
	}
}

// "Uma vez por rodada" (p25).
func TestTheExtraAttackHappensOncePerRound(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	horns := naturalIndex(t, f, "Chifres")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, horns, goblin); refusal != "" {
		t.Fatalf("o primeiro extra: %q", refusal)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, horns, goblin); !strings.Contains(refusal, "uma vez") {
		t.Errorf("o segundo extra devia ser recusado, e veio %q", refusal)
	}
}

// O golpe principal cancelado não aconteceu, e não abre nada.
func TestACancelledStrikeOpensNoExtra(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmCancels(t, f)
	if refusal := extraStrike(t, f, id, naturalIndex(t, f, "Chifres"), goblin); !strings.Contains(refusal, "agredir") {
		t.Errorf("sem golpe, o extra devia ser recusado, e veio %q", refusal)
	}
}

// O extra cancelado não aconteceu: o PM volta, e ele pode ser rolado de novo.
func TestACancelledExtraGivesItAllBack(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	horns := naturalIndex(t, f, "Chifres")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	before := mpOf(t, f.s, id)
	extraStrike(t, f, id, horns, goblin)
	gmCancels(t, f)
	if got := mpOf(t, f.s, id); got != before {
		t.Errorf("o extra cancelado não custa: %d virou %d", before, got)
	}
	if refusal := extraStrike(t, f, id, horns, goblin); refusal != "" {
		t.Errorf("o extra cancelado não conta na rodada, e veio %q", refusal)
	}
}

// A investida é outra ação (p235), e a manobra substitui o ataque dentro do agredir (p233).
func TestWhatOpensTheExtraAttack(t *testing.T) {
	t.Run("a investida não abre", func(t *testing.T) {
		f, id, goblin := minotaurOnTurn(t)
		if refusal := charges(t, f, id, "investida", goblin); refusal != "" {
			t.Fatalf("a investida: %q", refusal)
		}
		gmRules(t, f, "errou")
		if refusal := extraStrike(t, f, id, naturalIndex(t, f, "Chifres"), goblin); !strings.Contains(refusal, "agredir") {
			t.Errorf("a investida não é a ação agredir, e veio %q", refusal)
		}
	})
	t.Run("a manobra abre", func(t *testing.T) {
		f, id, goblin := minotaurOnTurn(t)
		if refusal := rollsFromTheActions(t, f, id, "manobra/derrubar", goblin); refusal != "" {
			t.Fatalf("a manobra: %q", refusal)
		}
		if refusal := gmRefusal(t, f, "/ataque/confirmar"); refusal != "" {
			t.Fatalf("o ✓ da manobra: %q", refusal)
		}
		if refusal := extraStrike(t, f, id, naturalIndex(t, f, "Chifres"), goblin); refusal != "" {
			t.Errorf("a manobra substitui o ataque do agredir, e veio %q", refusal)
		}
	})
	t.Run("o ataque desarmado abre", func(t *testing.T) {
		f, id, goblin := minotaurOnTurn(t)
		if refusal := rollsFromTheActions(t, f, id, fmt.Sprintf("atacar/%d", naturalIndex(t, f, "Ataque desarmado")), goblin); refusal != "" {
			t.Fatalf("o golpe desarmado: %q", refusal)
		}
		gmRules(t, f, "errou")
		if refusal := extraStrike(t, f, id, naturalIndex(t, f, "Chifres"), goblin); refusal != "" {
			t.Errorf("o desarmado é outra arma, e veio %q", refusal)
		}
	})
}

func TestTheExtraAttackEndsWithTheTurn(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	horns := naturalIndex(t, f, "Chifres")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	for range 2 { // o Goblin e a vez de novo
		if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
	if refusal := extraStrike(t, f, id, horns, goblin); !strings.Contains(refusal, "agredir") {
		t.Errorf("a vez passou, e o extra devia ser recusado; veio %q", refusal)
	}
}

func TestTheExtraAttackNeedsTheMana(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	horns := naturalIndex(t, f, "Chifres")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if _, err := f.s.db.ExecContext(context.Background(),
		"INSERT OR REPLACE INTO character_damage (characterId, hpDamage, mpSpent) VALUES (?, 0, ?)", id, poolsOf(t, f.s, id).MpMax); err != nil {
		t.Fatalf("gastar o PM: %v", err)
	}
	if refusal := extraStrike(t, f, id, horns, goblin); !strings.Contains(refusal, "PM") {
		t.Errorf("sem PM, o extra devia ser recusado, e veio %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Ruling != live.RulingMiss {
		t.Errorf("a recusa mexeu na mesa: %+v", pa)
	}
}

// "Gastar 2 PM para fazer um ataque corpo a corpo extra com cada uma" (p137): dois ataques,
// um de cada vez, pelo mesmo pagamento.
func TestTheExtraLimbsPayOnceForTwoAttacks(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedPowerChoices(t, f.s, id, `["membros-extras"]`, "{}")
	limbs := naturalIndex(t, f, "Patas insetoides")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	before := mpOf(t, f.s, id)

	if refusal := extraStrike(t, f, id, limbs, goblin); refusal != "" {
		t.Fatalf("a primeira pata: %q", refusal)
	}
	if got := mpOf(t, f.s, id); got != before-2 {
		t.Errorf("a primeira pata paga os 2 PM: %d virou %d", before, got)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, limbs, goblin); refusal != "" {
		t.Fatalf("a segunda pata: %q", refusal)
	}
	if got := mpOf(t, f.s, id); got != before-2 {
		t.Errorf("a segunda pata não paga de novo: %d virou %d", before, got)
	}
	gmCancels(t, f)
	if got := mpOf(t, f.s, id); got != before-2 {
		t.Errorf("cancelar a segunda pata não devolve o que a primeira pagou: %d", got)
	}
	if refusal := extraStrike(t, f, id, limbs, goblin); refusal != "" {
		t.Fatalf("a segunda pata de novo: %q", refusal)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, limbs, goblin); !strings.Contains(refusal, "uma vez") {
		t.Errorf("a terceira pata devia ser recusada, e veio %q", refusal)
	}
}

// A arma natural do golpe não é "outra arma" para ela mesma, mas é para as outras (p25).
func TestANaturalWeaponOpensTheOtherOnes(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	seedPowerChoices(t, f.s, id, `["dentes-afiados"]`, "{}")
	horns := naturalIndex(t, f, "Chifres")
	rollsFromTheActions(t, f, id, fmt.Sprintf("atacar/%d", horns), goblin)
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, horns, goblin); !strings.Contains(refusal, "outra arma") {
		t.Errorf("os chifres não são outra arma para os chifres, e veio %q", refusal)
	}
	if !strings.Contains(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(), "minotauro-chifres") {
		t.Error("a mesa não diz às Ações qual fonte o golpe barrou")
	}
	if refusal := extraStrike(t, f, id, naturalIndex(t, f, "Mordida"), goblin); refusal != "" {
		t.Errorf("os chifres são outra arma para a mordida, e veio %q", refusal)
	}
}

// As duas fontes são independentes: cada uma tem o próprio "uma vez por rodada".
func TestTwoNaturalWeaponsHaveTheirOwnExtra(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	seedPowerChoices(t, f.s, id, `["dentes-afiados"]`, "{}")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, naturalIndex(t, f, "Chifres"), goblin); refusal != "" {
		t.Fatalf("o extra dos chifres: %q", refusal)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, naturalIndex(t, f, "Mordida"), goblin); refusal != "" {
		t.Errorf("o extra da mordida: %q", refusal)
	}
}

func devotedToTheDragon(t *testing.T, f sceneFixture, id int64) {
	t.Helper()
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET god = 'kallyadranoch', godPowers = '["Presas Primordiais"]' WHERE id = ?`, id); err != nil {
		t.Fatalf("semear a devoção: %v", err)
	}
}

func awakensTheFangs(t *testing.T, f sceneFixture, id int64) {
	t.Helper()
	if refusal := powerCommand(t, f, id, "usa/deus.kallyadranoch.presas-primordiais", ""); refusal != "" {
		t.Fatalf("ativar as Presas: %q", refusal)
	}
}

// As Presas Primordiais só existem depois de ativadas, até o fim da cena (p135).
func TestThePrimordialFangsNeedToBeAwakened(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	devotedToTheDragon(t, f, id)
	if strings.Contains(superficieDeAcoes(t, acoesDaMesa(t, f)), "Atacar · Mordida") {
		t.Fatal("sem ativar, as Ações já mostram a mordida")
	}
	awakensTheFangs(t, f, id)
	if !strings.Contains(sheetTab(t, f, id, "combat"), "Mordida") {
		t.Error("depois de ativar, a aba Combate não mostra a mordida")
	}
	fangs := naturalIndex(t, f, "Mordida")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, fangs, goblin); refusal != "" {
		t.Errorf("as Presas têm o ataque extra: %q", refusal)
	}
	if err := rest.NewScopes(f.s.queries, f.s.catalogs).EndScene(context.Background(), app.Caller{ID: f.player}, id); err != nil {
		t.Fatalf("encerrar a cena: %v", err)
	}
	if strings.Contains(sheetTab(t, f, id, "combat"), "Mordida") {
		t.Error("a cena acabou, e a mordida continua na ficha")
	}
}

// "Se já possuir outro ataque natural de mordida, em vez disso, o dano desse ataque aumenta em
// dois passos" (p135).
func TestTheFangsRaiseTheBiteThatIsAlreadyThere(t *testing.T) {
	f, id, _ := barbarianOnTurn(t)
	seedRaca(t, f.s, id, "trog")
	devotedToTheDragon(t, f, id)
	awakensTheFangs(t, f, id)
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	if lines := strings.Count(surface, "Atacar · Mordida"); lines != 1 {
		t.Fatalf("as Presas não dão outra mordida, e há %d linhas", lines)
	}
	at := strings.Index(surface, "Atacar · Mordida")
	if !strings.Contains(surface[at:min(at+300, len(surface))], "1d10") {
		t.Errorf("1d6 mais dois passos é 1d10, e a linha diz %q", surface[at:min(at+160, len(surface))])
	}
}

// A habilidade tomada na Memória Póstuma (p29) é da ficha, com a arma natural dela.
func TestABorrowedAbilityBringsItsNaturalWeapon(t *testing.T) {
	f, id, _ := barbarianOnTurn(t)
	seedRaca(t, f.s, id, "osteon")
	if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", "minotauro-chifres"); refusal != "" {
		t.Fatalf("tomar os chifres: %q", refusal)
	}
	if !strings.Contains(superficieDeAcoes(t, acoesDaMesa(t, f)), "Atacar · Chifres") {
		t.Error("a habilidade tomada não deu a linha de ataque")
	}
}

// O poder escolhido pela origem também é da ficha: ela já paga o Carisma dele (p137).
func TestAnOriginPowerBringsItsNaturalWeapon(t *testing.T) {
	f, id, _ := barbarianOnTurn(t)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET originChoices = '["origin-assistente-poder-poder-da-tormenta-escolha"]' WHERE id = ?`, id); err != nil {
		t.Fatalf("semear a origem: %v", err)
	}
	seedPowerChoices(t, f.s, id, "[]", `{"origin-assistente-poder-poder-da-tormenta-escolha":["dentes-afiados"]}`)
	if !strings.Contains(superficieDeAcoes(t, acoesDaMesa(t, f)), "Atacar · Mordida") {
		t.Error("o poder da origem não deu a linha de ataque")
	}
}

// Voltar a vez não renova o turno, e o extra aberto não passa para quem entrou nela.
func TestRewindingTheTurnClosesTheExtra(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	store := f.s.sessions
	other := seedCharacterAtLevel(t, f.s, f.player, "Outro", "Bárbaro", 1, 0, 0)
	seedRaca(t, f.s, other, "minotauro")
	seedMember(t, f.s, f.campaignID, other)
	if _, err := store.AddInitiativeEntry(context.Background(), f.sessionID, sheetCombatant("Outro", 30, other)); err != nil {
		t.Fatalf("pôr o outro na fila: %v", err)
	}
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	page := func() string { return f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String() }
	if !strings.Contains(page(), "extra_open: true") {
		t.Fatal("o extra não abriu")
	}
	for range 4 {
		state := stateOf(t, store, f.sessionID)
		if state.Initiative[state.TurnIndex].Label == "Outro" {
			break
		}
		if _, err := store.PreviousTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("voltar a vez: %v", err)
		}
	}
	state := stateOf(t, store, f.sessionID)
	if state.Initiative[state.TurnIndex].Label != "Outro" {
		t.Fatalf("a vez devia ter voltado para o outro, e está com %s", state.Initiative[state.TurnIndex].Label)
	}
	if strings.Contains(page(), "extra_open: true") {
		t.Error("a vez voltou, e a mesa segue com o extra aberto")
	}
	body := fmt.Sprintf(`{"turn_target":%q}`, goblin)
	refusal := sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/ataque-extra/%d?embutida=1", other, 1), body))
	if !strings.Contains(refusal, "agredir") {
		t.Errorf("quem entrou na vez não agrediu, e veio %q", refusal)
	}
}

func TestTheTableTellsTheActionsAboutTheExtra(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	if !strings.Contains(surface, "Ataque extra · Chifres") {
		t.Fatal("as Ações não oferecem o ataque extra")
	}
	if !strings.Contains(surface, "$extra_open") {
		t.Error("o botão do extra não espera o golpe principal")
	}
	page := func() string { return f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String() }
	if strings.Contains(page(), "extra_open: true") {
		t.Fatal("sem golpe, a mesa já abre o extra")
	}
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if !strings.Contains(page(), "extra_open: true") {
		t.Error("depois do golpe, a mesa não abre o extra")
	}
	extraStrike(t, f, id, naturalIndex(t, f, "Chifres"), goblin)
	if !strings.Contains(page(), "Ataque extra · 1 PM") {
		t.Error("o cartão não diz que é o ataque extra, nem o PM dele")
	}
}

// O ataque extra é da arma natural, e o endereço escrito à mão com outra arma não passa.
func TestAStrangeExtraIsRefused(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, 0, goblin); !strings.Contains(refusal, "arma natural") {
		t.Errorf("o extra com a arma de mão devia ser recusado, e veio %q", refusal)
	}
}
