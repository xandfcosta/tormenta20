package api

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"testing"
)

// A HARMONIZADA (ARMA) NA FICHA E NA MESA (p165): "Escolha uma habilidade ativada ao se fazer um ataque […] e
// que custe pontos de mana. Esta habilidade tem seu custo em PM reduzido em –1 se utilizada com esta arma".
// Quem escolhe é o jogador, na folha da arma; o corte é do motor (`harmonized_weapon_rules_test.go`), e aqui
// se prende a LIGAÇÃO: a escolha grava, a linha da arma mostra o preço cortado, e a proposta o cobra.

const (
	harmonizedWeaponImprovement = "melhoria-harmonizada-arma"
	specialAttack               = "class.guerreiro.ataque-especial-4"
)

func harmonizedAbilityPath(item int64, power string) string {
	return fmt.Sprintf("itens/%d/melhorias/%s/habilidade/%s", item, harmonizedWeaponImprovement, power)
}

// harmonizedSword é uma espada longa empunhada com a Certeira e a Harmonizada, que pede "outra melhoria
// qualquer" (p165).
func harmonizedSword(t *testing.T, f sceneFixture, character int64) int64 {
	t.Helper()
	sword := itemSemeia(t, f, character, "espada-longa", "Espada longa", "wielded")
	body := `{"item_improvements":["melhoria-certeira","` + harmonizedWeaponImprovement + `"]}`
	if refusal := improvements(t, f, character, sword, body); refusal != "" {
		t.Fatalf("aplicar a Harmonizada foi recusado: %q", refusal)
	}
	return sword
}

// boostRowsOffering são os índices das linhas de ataque cuja tira oferece `power`.
func boostRowsOffering(t *testing.T, screen, power string) []string {
	t.Helper()
	found := regexp.MustCompile(`\$boost_weapon = (\d+); \$boost_power = "`+regexp.QuoteMeta(power)+`"`).
		FindAllStringSubmatch(screen, -1)
	rows := []string{}
	for _, f := range found {
		rows = append(rows, f[1])
	}
	return rows
}

func TestTheHarmonizedWeaponChargesItsAbilityOnePMLess(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, barbaro, "Guerreiro", 5)
	sword := harmonizedSword(t, f, barbaro)
	if refusal := bagCommand(t, f, barbaro, harmonizedAbilityPath(sword, specialAttack)); refusal != "" {
		t.Fatalf("escolher o Ataque Especial na Harmonizada foi recusado: %q", refusal)
	}
	if got := storedChoiceOf(t, f, sword, harmonizedWeaponImprovement); got != specialAttack {
		t.Fatalf("a habilidade gravada foi %q; want %s", got, specialAttack)
	}

	screen := html.UnescapeString(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), ""))
	rows := boostRowsOffering(t, screen, specialAttack)
	if len(rows) < 2 {
		t.Fatalf("o Ataque Especial está em %d linhas de ataque; want a espada e o desarmado, ao menos:\n%s", len(rows), recorte(screen))
	}
	// O PREÇO POR DEGRAU da linha da espada é [1, 1]: sem degrau a mais o Ataque Especial custa 1, que já é o
	// piso, e com um ele custa 2, menos o corte. A outra linha diz [1, 2].
	for _, row := range rows {
		cost := fmt.Sprintf(`$boost_weapon === %s && $boost_power === "%s" ? [1,1]`, row, specialAttack)
		if strings.Contains(screen, cost) {
			chosen := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":%s,"boost_power":%q,"boost_steps":1,"boost_split":"attack"}`,
				goblin, row, specialAttack)
			if refusal := sceneRefusal(f.posts(t, f.player,
				fmt.Sprintf("/personagens/%d/acoes/atacar/%s?embutida=1", barbaro, row), chosen)); refusal != "" {
				t.Fatalf("o golpe com o Ataque Especial na espada harmonizada foi recusado: %q", refusal)
			}
			pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
			if pending == nil || pending.BoostPowerID != specialAttack || pending.BoostPM != 1 {
				t.Errorf("o provisório levou %+v; want o Ataque Especial com 1 degrau por 1 PM, 2 menos a Harmonizada", pending)
			}
			return
		}
	}
	t.Fatalf("nenhuma linha mostra o Ataque Especial cortado pela Harmonizada:\n%s", recorte(screen))
}

func TestTheHarmonizedWeaponSheetChoosesTheAbility(t *testing.T) {
	f := newSceneFixture(t)
	sword := harmonizedSword(t, f, f.charID)
	sheet := itemSheetSearching(t, f, sword, "")
	if !strings.Contains(sheet, "Habilidade da Harmonizada") || !strings.Contains(sheet, "p165") {
		t.Fatal("a folha da espada harmonizada não pede a habilidade da Harmonizada, com a p165")
	}
	for _, power := range []string{specialAttack, "class.paladino.golpe-divino-1d8"} {
		if !strings.Contains(sheet, harmonizedAbilityPath(sword, power)) {
			t.Errorf("a folha não oferece %s, que se paga no ataque", power)
		}
	}
	if refusal := bagCommand(t, f, f.charID, harmonizedAbilityPath(sword, "class.paladino.golpe-divino-1d8")); refusal != "" {
		t.Fatalf("escolher o Golpe Divino foi recusado: %q", refusal)
	}
	if !strings.Contains(itemSheetSearching(t, f, sword, ""), "Golpe Divino custa 1 PM a menos") {
		t.Error("com a habilidade escolhida, a folha não diz qual custa menos")
	}
}

func TestTheHarmonizedAbilityRefusesWhatDoesNotFit(t *testing.T) {
	f := newSceneFixture(t)
	sword := harmonizedSword(t, f, f.charID)
	// Um poder que não se paga no ataque não tem custo de golpe a cortar.
	if refusal := bagCommand(t, f, f.charID, harmonizedAbilityPath(sword, "class.paladino.codigo-do-heroi")); !strings.Contains(refusal, "ataque") {
		t.Errorf("escolher o Código do Herói deu %q; want a recusa", refusal)
	}
	bare := itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "")
	if refusal := bagCommand(t, f, f.charID, harmonizedAbilityPath(bare, specialAttack)); !strings.Contains(refusal, "Harmonizada") {
		t.Errorf("escolher a habilidade numa espada sem a Harmonizada deu %q; want a recusa", refusal)
	}
	path := fmt.Sprintf("itens/%d/melhorias/melhoria-certeira/habilidade/%s", sword, specialAttack)
	if refusal := bagCommand(t, f, f.charID, path); !strings.Contains(refusal, "não pede habilidade") {
		t.Errorf("escolher habilidade para uma melhoria que não pede deu %q; want a recusa", refusal)
	}
}

// O GASTO DO ENCANTO DESTA ARMA entra na lista (decisão do dono), e só o dela: o gasto de outra arma nunca
// sai com esta.
func TestTheHarmonizedWeaponOffersItsOwnEnchantSpend(t *testing.T) {
	f := newSceneFixture(t)
	sword := harmonizedSword(t, f, f.charID)
	if strings.Contains(itemSheetSearching(t, f, sword, ""), harmonizedAbilityPath(sword, "encanto-")) {
		t.Error("a espada sem encanto oferece o gasto de um encanto")
	}
	if err := f.s.characterPlays().EnchantItem(context.Background(), sword, []string{"encanto-tumular"}, nil); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}
	sheet := itemSheetSearching(t, f, sword, "")
	if !strings.Contains(sheet, harmonizedAbilityPath(sword, "encanto-tumular")) || !strings.Contains(sheet, "Tumular") {
		t.Fatal("a espada tumular harmonizada não oferece o gasto da Tumular")
	}
	if refusal := bagCommand(t, f, f.charID, harmonizedAbilityPath(sword, "encanto-venenosa")); !strings.Contains(refusal, "ataque") {
		t.Errorf("escolher a Venenosa, que a espada não tem, deu %q; want a recusa", refusal)
	}
	if refusal := bagCommand(t, f, f.charID, harmonizedAbilityPath(sword, "encanto-tumular")); refusal != "" {
		t.Fatalf("escolher a Tumular foi recusado: %q", refusal)
	}
	if got := storedChoiceOf(t, f, sword, harmonizedWeaponImprovement); got != "encanto-tumular" {
		t.Errorf("a escolha gravada foi %q; want encanto-tumular", got)
	}
	if !strings.Contains(itemSheetSearching(t, f, sword, ""), "Tumular custa 1 PM a menos") {
		t.Error("com a Tumular escolhida, a folha não diz qual custa menos")
	}
}
