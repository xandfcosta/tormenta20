package api

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"testing"

	"t20engine/domain/engine"
)

// O GASTO DE 2 PM DO ENCANTO NA LINHA DE ATAQUE (p335-336). A tira "Com" da superfície Ações oferece o
// gasto só na linha da arma que tem o encanto — a outra mão não o leva —, e o "Rolar" daquela linha o leva
// até o provisório da mesa, com o PM que a proposta cobra. O que o golpe pago soma é do motor
// (`enchant_boost_rules_test.go`); aqui se prende a LIGAÇÃO.
func TestTheEnchantBoostIsOfferedOnlyOnItsWeaponAndTravelsInTheStrike(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	sword := itemSemeia(t, f, barbaro, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(context.Background(), sword, []string{"encanto-venenosa"}); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}
	venomous := engine.EnchantBoostID(sword, "encanto-venenosa")

	screen := html.UnescapeString(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), ""))
	offered := regexp.MustCompile(`\$boost_weapon = (\d+); \$boost_power = "`+regexp.QuoteMeta(venomous)+`"`).
		FindAllStringSubmatch(screen, -1)
	rows := map[string]bool{}
	for _, found := range offered {
		rows[found[1]] = true
	}
	if len(rows) != 1 {
		t.Fatalf("o gasto da Venenosa está em %d linhas de ataque (%v); quer só na da espada:\n%s", len(rows), rows, recorte(screen))
	}
	var weapon int
	for row := range rows {
		weapon, _ = strconv.Atoi(row)
	}

	chosen := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":%d,"boost_power":%q,"boost_steps":0,"boost_split":""}`,
		goblin, weapon, venomous)
	if refusal := sceneRefusal(f.posts(t, f.player,
		fmt.Sprintf("/personagens/%d/acoes/atacar/%d?embutida=1", barbaro, weapon), chosen)); refusal != "" {
		t.Fatalf("o golpe com o gasto da Venenosa foi recusado: %q", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil || pending.BoostPowerID != venomous || pending.BoostPM != 2 {
		t.Errorf("o provisório levou %+v; quer o gasto da Venenosa por 2 PM (p336)", pending)
	}
}
