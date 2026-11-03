package api

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A LANÇA BATE E VOA (p143-144, p148): o mesmo item leva dois gestos, e a composição que se prova é a da carta
// até a mesa. Os números de cada uso são presos no motor; aqui se prova que o índice da Ações chega na carta certa.
//
// As cartas do bárbaro com a lança na outra mão: 0 o machado, 1 a lança, 2 a lança arremessada e 3 o desarmado.
func barbarianWithASpear(t *testing.T) (sceneFixture, int64, string) {
	t.Helper()
	f, barbaro, goblin := barbarianOnTurn(t)
	if _, err := f.s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: barbaro, Catalogid: sql.NullString{String: "lanca", Valid: true},
		Name: "Lança", Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: "wielded2", Valid: true},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("empunhar a lança: %v", err)
	}
	// Com proficiência, para o −5 de cada arma não entrar na conta que se está lendo.
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET proficiencies = '["armas-simples","armas-marciais"]' WHERE id = ?`, barbaro); err != nil {
		t.Fatalf("dar as proficiências: %v", err)
	}
	return f, barbaro, goblin
}

func TestTheActionsOfferBothUsesOfTheSpear(t *testing.T) {
	f, _, _ := barbarianWithASpear(t)
	padrao := entre(t, acoesDaMesa(t, f), "Ação padrão", "Ação de movimento")
	for _, use := range []string{"Lança", "Lança (arremesso)"} {
		if !strings.Contains(padrao, use) {
			t.Errorf("a ação padrão não oferece %q:\n%s", use, recorte(padrao))
		}
	}
}

func TestTheSpearAttacksWithFightingUpCloseAndAimWhenThrown(t *testing.T) {
	for _, c := range []struct {
		path, weapon string
		bonus        int
	}{
		// Nível 6 dá +3: Luta soma a Força 4, e Pontaria a Destreza 2.
		{"atacar/1", "Lança", 7},
		{"atacar/2", "Lança (arremesso)", 5},
	} {
		t.Run(c.weapon, func(t *testing.T) {
			f, barbaro, goblin := barbarianWithASpear(t)
			if refusal := rollsFromTheActions(t, f, barbaro, c.path, goblin); refusal != "" {
				t.Fatalf("o ataque foi recusado: %q", refusal)
			}
			pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
			if pending == nil {
				t.Fatal("o ataque não deixou provisório")
			}
			if pending.Weapon != c.weapon || pending.Total-pending.Roll != c.bonus {
				t.Errorf("o ataque saiu com %q e %+d; esperado %q e %+d", pending.Weapon, pending.Total-pending.Roll, c.weapon, c.bonus)
			}
		})
	}
}

// "No fim do movimento, faz um ataque corpo a corpo" (p235): a lança na mão investe, e a lança que voa não.
func TestOnlyTheSpearUpCloseCharges(t *testing.T) {
	f, barbaro, goblin := barbarianWithASpear(t)
	if refusal := rollsFromTheActions(t, f, barbaro, "investida/2", goblin); !strings.Contains(refusal, "Lança (arremesso)") {
		t.Errorf("a investida com a lança arremessada não foi recusada pelo nome: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, barbaro, "investida/1", goblin); refusal != "" {
		t.Errorf("a investida com a lança foi recusada: %q", refusal)
	}
}
