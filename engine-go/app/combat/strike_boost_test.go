package combat

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"t20engine/app"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O IMPULSO DE PM NO GOLPE (ALE-487). O que o motor soma é do
// `BoostedWeaponCards` e tem os casos do livro lá; aqui se prende o que ESTA
// camada decide: a carta do golpe é a impulsionada, o provisório leva o PM que o
// `session.Store.ProposeAttack` cobra, e o PM que a ficha não tem recusa a proposta.

// boostedSword devolve a espada com +3 no ataque por 2 PM, para qualquer pedido.
func boostedSword(choice engine.BoostChoice) ([]engine.WeaponCard, int, error) {
	card := aLongsword()
	card.Attack += 3
	return []engine.WeaponCard{card}, 2, nil
}

func TestTheBoostedCardRollsAndTheProvisionalCarriesTheCost(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()},
			Boost: boostedSword, ManaAvailable: 5},
		"ogro": {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))

	d20 := 10
	out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Boost: engine.BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"},
	})
	if err != nil {
		t.Fatalf("propor com impulso: %v", err)
	}
	if out.Total != 10+5+3 {
		t.Errorf("total %d, want 18 — o golpe rola com a carta IMPULSIONADA (+3)", out.Total)
	}
	if out.BoostPM != 2 || out.BoostPowerID != "class.paladino.golpe-divino-1d8" {
		t.Errorf("o provisório saiu com impulso %q por %d PM, want o Golpe Divino por 2 PM", out.BoostPowerID, out.BoostPM)
	}
}

// As posturas viajam na proposta porque os gatilhos do encerramento leem dela, e não da
// ficha daquele momento.
func TestTheProposalCarriesTheStancesOfBothSides(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}, Stances: []string{"furia"}},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10, Stances: []string{"postura-foco-de-batalha"}},
	}, table, semTabuleiro{}, fixedDie(4))

	for _, maneuver := range []string{"", "derrubar"} {
		d20 := 10
		out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
			SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true, Maneuver: maneuver,
		})
		if err != nil {
			t.Fatalf("propor %q: %v", maneuver, err)
		}
		if !slices.Equal(out.AttackerStances, []string{"furia"}) || !slices.Equal(out.TargetStances, []string{"postura-foco-de-batalha"}) {
			t.Errorf("%q saiu com as posturas %v e %v", maneuver, out.AttackerStances, out.TargetStances)
		}
		table.state.PendingAttack = nil
	}
}

func TestABoostTheSheetCannotPayIsRefused(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()},
			Boost: boostedSword, ManaAvailable: 1},
		"ogro": {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))

	d20 := 10
	_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Boost: engine.BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"},
	})
	if !errors.Is(err, app.ErrRefused) || !strings.Contains(err.Error(), "2 PM") || !strings.Contains(err.Error(), "1") {
		t.Errorf("2 PM com 1 na ficha: err = %v, want recusa que diz os dois números", err)
	}
	if table.stored != nil {
		t.Errorf("a proposta recusada foi guardada na mesa")
	}
}

func TestACombatantWithoutBoostsCannotAskForOne(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))

	d20 := 10
	_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Boost: engine.BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"},
	})
	if !errors.Is(err, app.ErrRefused) {
		t.Errorf("impulso pedido por quem não tem impulso nenhum: err = %v, want recusa", err)
	}
}

// O GASTO DE UM ENCANTO (p335-336) é da arma que tem o encanto: o golpe com a outra mão não o leva, e a
// recusa diz de que arma ele é. Sem isso, a adaga pagaria 2 PM e anunciaria o veneno da espada, porque o
// anúncio não tem escopo de arma.
func TestAnEnchantBoostIsRefusedOnAnotherWeapon(t *testing.T) {
	sword, dagger := aLongsword(), aLongsword()
	sword.ItemID, dagger.ItemID, dagger.Name = 7, 8, "Adaga"
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{sword, dagger},
			Boost: func(engine.BoostChoice) ([]engine.WeaponCard, int, error) {
				return []engine.WeaponCard{sword, dagger}, 2, nil
			}, ManaAvailable: 5},
		"ogro": {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))

	venomous := engine.BoostChoice{PowerID: engine.EnchantBoostID(7, "encanto-venenosa")}
	d20 := 10
	_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Weapon: 1, Boost: venomous,
	})
	if !errors.Is(err, app.ErrRefused) || !strings.Contains(err.Error(), "Adaga") {
		t.Errorf("o gasto da espada num golpe de adaga: err = %v, want recusa que diz a arma", err)
	}
	// O CONTROLE: com a espada, o mesmo pedido passa.
	table.state.PendingAttack = nil
	if _, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Weapon: 0, Boost: venomous,
	}); err != nil {
		t.Errorf("o gasto da espada num golpe de espada foi recusado: %v", err)
	}
}

// O PV DA TUMULAR (p336): "você perde 1d8 pontos de vida". O custo vem do gasto, acerte ou não, e a
// proposta o rola com o mesmo rolador do golpe e o leva no provisório, para a sessão cobrar junto do PM
// (decisão do dono). O gasto sem custo de PV não rola nada.
func TestTheTomblikeBoostRollsItsHitPointCostIntoTheProposal(t *testing.T) {
	for _, c := range []struct {
		name, cost string
		want       int
	}{
		{"com o 1d8 da Tumular", "1d8", 4},
		{"sem custo de PV", "", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			table := aTable(t)
			strike := NewStrike(sheetDouble{
				"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()},
					Boost: boostedSword, ManaAvailable: 5,
					BoostHitPointCost: func(engine.BoostChoice) string { return c.cost }},
				"ogro": {EntryID: "ogro", Label: "Ogro", Defense: 10},
			}, table, semTabuleiro{}, fixedDie(4))
			d20 := 10
			out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
				SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
				Boost: engine.BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"},
			})
			if err != nil {
				t.Fatalf("propor: %v", err)
			}
			if out.BoostHitPoints != c.want {
				t.Errorf("o provisório leva %d de PV do gasto, want %d", out.BoostHitPoints, c.want)
			}
		})
	}
}

// O GASTO DA ANTICRIATURA vale contra o tipo escolhido no encanto (p335), e é oferecido contra qualquer alvo: a
// proposta guarda o tipo e o que a mesa sabia do alvo no instante do d20, para a faixa do mestre dizer se ele é
// desse tipo (decisão do dono). O gasto que vale contra qualquer um não guarda nada.
func TestTheAnticreatureBoostCarriesWhatTheTableKnowsOfTheTarget(t *testing.T) {
	sword := aLongsword()
	sword.ItemID = 7
	anticreature := engine.BoostChoice{PowerID: engine.EnchantBoostID(7, "encanto-anticriatura")}
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{sword},
			Boost: func(engine.BoostChoice) ([]engine.WeaponCard, int, error) {
				return []engine.WeaponCard{sword}, 2, nil
			},
			BoostAgainst: func(choice engine.BoostChoice) string {
				if choice == anticreature {
					return "morto-vivo"
				}
				return ""
			}, ManaAvailable: 5},
		"ogro": {EntryID: "ogro", Label: "Ogro zumbi", Defense: 10, Nature: book.CreatureNature{Kind: "morto-vivo"}},
	}, table, semTabuleiro{}, fixedDie(4))

	d20 := 10
	propose := func(boost engine.BoostChoice) *live.ChosenKindRead {
		t.Helper()
		table.state.PendingAttack = nil
		out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
			SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
			Boost: boost,
		})
		if err != nil {
			t.Fatalf("propor com %q: %v", boost.PowerID, err)
		}
		return out.BoostAgainst
	}
	got := propose(anticreature)
	if got == nil || *got != (live.ChosenKindRead{Chosen: "morto-vivo", TargetKind: "morto-vivo"}) {
		t.Errorf("a proposta da Anticriatura guardou %+v; quer morto-vivo contra um alvo morto-vivo", got)
	}
	if got := propose(engine.BoostChoice{PowerID: engine.EnchantBoostID(7, "encanto-venenosa")}); got != nil {
		t.Errorf("o gasto da Venenosa, que vale contra qualquer alvo, guardou %+v", got)
	}
}
