package api

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"t20engine/app/combat"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A LANÇA BATE E VOA NA MESMA LINHA (p143-144, p148): a composição que se prova é a do botão até a mesa. Os números de
// cada uso são presos no motor; aqui se prova que o gesto chega no uso certo, e que o pedido refeito o leva.
//
// As cartas do bárbaro com a lança na outra mão: 0 o machado, 1 a lança e 2 o desarmado.
func barbarianWithASpear(t *testing.T) (sceneFixture, int64, string) {
	t.Helper()
	f, barbarian, goblin := barbarianOnTurn(t)
	if _, err := f.s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: barbarian, Catalogid: sql.NullString{String: "lanca", Valid: true},
		Name: "Lança", Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: "wielded2", Valid: true},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("empunhar a lança: %v", err)
	}
	// Com proficiência, para o −5 de cada arma não entrar na conta que se está lendo.
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET proficiencies = '["armas-simples","armas-marciais"]' WHERE id = ?`, barbarian); err != nil {
		t.Fatalf("dar as proficiências: %v", err)
	}
	return f, barbarian, goblin
}

func TestTheSpearOffersTheThrowInsideItsOwnRow(t *testing.T) {
	f, _, _ := barbarianWithASpear(t)
	standardActions := entre(t, acoesDaMesa(t, f), "Ação padrão", "Ação de movimento")
	// Nível 6 dá +3, e Pontaria soma a Destreza 2: +5. O machado não tem alcance: −5 (p143).
	for _, label := range []string{"Atacar · Lança", "Arremessar +5", "Arremessar +0"} {
		if !strings.Contains(standardActions, label) {
			t.Errorf("a ação padrão não oferece %q:\n%s", label, recorte(standardActions))
		}
	}
	if strings.Contains(standardActions, "(arremesso)") {
		t.Errorf("o arremesso virou uma linha própria:\n%s", recorte(standardActions))
	}
	if unarmed := entre(t, standardActions, "Atacar · Ataque desarmado", "Manobra"); strings.Contains(unarmed, "Arremessar") {
		t.Errorf("o desarmado oferece arremesso:\n%s", recorte(unarmed))
	}
}

func TestTheSpearAttacksWithFightingUpCloseAndAimWhenThrown(t *testing.T) {
	for _, c := range []struct {
		path, weapon string
		bonus        int
		otherUse     bool
	}{
		// Nível 6 dá +3: Luta soma a Força 4, e Pontaria a Destreza 2.
		{"atacar/1", "Lança", 7, false},
		{"atacar/1/outro-uso", "Lança (arremesso)", 5, true},
		// O machado de batalha não tem alcance: arremessá-lo custa −5 (p143).
		{"atacar/0/outro-uso", "Machado de batalha (arremesso)", 0, true},
	} {
		t.Run(c.weapon, func(t *testing.T) {
			f, barbarian, goblin := barbarianWithASpear(t)
			if refusal := rollsFromTheActions(t, f, barbarian, c.path, goblin); refusal != "" {
				t.Fatalf("o ataque foi recusado: %q", refusal)
			}
			pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
			if pending == nil {
				t.Fatal("o ataque não deixou provisório")
			}
			if pending.Weapon != c.weapon || pending.Total-pending.Roll != c.bonus {
				t.Errorf("o ataque saiu com %q e %+d; esperado %q e %+d", pending.Weapon, pending.Total-pending.Roll, c.weapon, c.bonus)
			}
			// A decisão do mestre e o dano refazem o pedido: o uso tem de ir junto.
			replay := combat.ReplayRequest(f.campaignID, f.sessionID, *pending, true, nil)
			if replay.OtherUse != c.otherUse {
				t.Errorf("o pedido refeito perdeu o uso: %v, esperado %v", replay.OtherUse, c.otherUse)
			}
		})
	}
}

// O desarmado não tem o que arremessar, e o servidor recusa mesmo sem o botão.
func TestTheFistIsNotThrown(t *testing.T) {
	f, barbarian, goblin := barbarianWithASpear(t)
	refusal := rollsFromTheActions(t, f, barbarian, "atacar/2/outro-uso", goblin)
	if !strings.Contains(refusal, "não tem outro uso") {
		t.Errorf("o arremesso do desarmado não foi recusado: %q", refusal)
	}
}

// "No fim do movimento, faz um ataque corpo a corpo" (p235): a investida é da lança na mão.
func TestTheSpearChargesUpClose(t *testing.T) {
	f, barbarian, goblin := barbarianWithASpear(t)
	if refusal := rollsFromTheActions(t, f, barbarian, "investida/1", goblin); refusal != "" {
		t.Errorf("a investida com a lança foi recusada: %q", refusal)
	}
}

// A azagaia é o caso inverso (p146): o botão dela é Bater, com −5. Luta com Força 4 e nível 6 dá +7, menos 5.
func TestTheJavelinOffersTheHitInsideItsOwnRow(t *testing.T) {
	f, barbarian, goblin := barbarianOnTurn(t)
	if _, err := f.s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: barbarian, Catalogid: sql.NullString{String: "azagaia", Valid: true},
		Name: "Azagaia", Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: "wielded2", Valid: true},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("empunhar a azagaia: %v", err)
	}
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET proficiencies = '["armas-simples","armas-marciais"]' WHERE id = ?`, barbarian); err != nil {
		t.Fatalf("dar as proficiências: %v", err)
	}
	if standardActions := entre(t, acoesDaMesa(t, f), "Ação padrão", "Ação de movimento"); !strings.Contains(standardActions, "Bater +2") {
		t.Errorf("a azagaia não oferece Bater +2:\n%s", recorte(standardActions))
	}
	if refusal := rollsFromTheActions(t, f, barbarian, "atacar/1/outro-uso", goblin); refusal != "" {
		t.Fatalf("bater com a azagaia foi recusado: %q", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil || pending.Weapon != "Azagaia (corpo a corpo)" || pending.Total-pending.Roll != 2 {
		t.Errorf("o golpe saiu assim: %+v", pending)
	}
}

// A conta da arma na ficha mostra os dois usos na mesma linha, e o −5 tem nome.
func TestTheWeaponSumShowsTheOtherUse(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	sword := sumLine(t, parallelSheet(t, f, id, "combat"), "Espada longa")
	for _, want := range []string{"Arremessar · Ataque · Pontaria", "Arremessar · Dano", "arma sem alcance (p143)"} {
		if !strings.Contains(sword, want) {
			t.Errorf("a conta da espada não diz %q:\n%s", want, sword)
		}
	}
}

// O Golpe Divino é "corpo a corpo" (p82): quem pede no arremesso ouve não, sem provisório e sem PM a cobrar.
func TestTheThrowRefusesAMeleeOnlyBoost(t *testing.T) {
	f, barbarian, goblin := barbarianWithASpear(t)
	seedClasse(t, f.s, barbarian, "Paladino", 5)
	request := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":1,"boost_power":"class.paladino.golpe-divino-1d8","boost_steps":0,"boost_split":""}`, goblin)
	path := fmt.Sprintf("/personagens/%d/acoes/atacar/1", barbarian)

	refusal := sceneRefusal(f.posts(t, f.player, path+"/outro-uso?embutida=1", request))
	if !strings.Contains(refusal, "Lança (arremesso)") || !strings.Contains(refusal, "não recebe") {
		t.Errorf("o arremesso com o Golpe Divino não foi recusado pelo nome: %q", refusal)
	}
	if pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack; pending != nil {
		t.Fatalf("a recusa deixou provisório: %+v", pending)
	}

	if refusal := sceneRefusal(f.posts(t, f.player, path+"?embutida=1", request)); refusal != "" {
		t.Fatalf("a lança corpo a corpo com o Golpe Divino foi recusada: %q", refusal)
	}
	if pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack; pending == nil || pending.BoostPM != 2 {
		t.Errorf("o golpe saiu assim: %+v; want o Golpe Divino com 2 PM (p82)", pending)
	}
}
