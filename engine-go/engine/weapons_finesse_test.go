package engine

import (
	"path/filepath"
	"testing"
)

// TestWeaponFinesse covers ALE-31: a weapon may use Destreza instead of Força on
// the attack (and, with Acuidade com Arma, damage). Cross-language parity is
// proven by the weaponCards oracle (5 seed chars wield an adaga); this asserts
// the branches directly Go-side, incl. the Acuidade path no fixture exercises.
func TestWeaponFinesse(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)
	none := map[string]bool{}

	ptr := func(s string) *string { return &s }
	// bardo-versatil-nv7 has the 29 expertises + STR 1 / DEX 3 (DES beats FOR).
	var oracle struct {
		Char Character `json:"char"`
	}
	readJSON(t, filepath.Join(dir, "bardo-versatil-nv7.json"), &oracle)
	base := oracle.Char

	card := func(catalogID, powers string) WeaponCard {
		ch := base
		ch.ClassPowers = powers
		ch.Items = []CharacterItem{{
			CatalogID: ptr(catalogID), Name: catalogID, Equipped: ptr("wielded"), Improvements: "[]",
		}}
		cards := catalogs.ComputeWeaponCards(ch, none)
		if len(cards) == 0 {
			t.Fatalf("%s: no weapon card", catalogID)
		}
		return cards[0]
	}

	// Adaga: finesse inerente → ataque em Destreza.
	if got := card("adaga", "[]").Attribute; got != "dexterity" {
		t.Errorf("adaga attribute = %q, quer dexterity", got)
	}
	// Espada curta (leve, sem finesse): Força sem Acuidade, Destreza com Acuidade.
	if got := card("espada-curta", "[]").Attribute; got != "strength" {
		t.Errorf("espada-curta sem Acuidade = %q, quer strength", got)
	}
	if got := card("espada-curta", `["acuidade-com-arma"]`).Attribute; got != "dexterity" {
		t.Errorf("espada-curta com Acuidade = %q, quer dexterity", got)
	}
	// A Acuidade leva a Destreza pro DANO da adaga (o finesse inerente é só ataque).
	semAcu := card("adaga", "[]").StrDamage
	comAcu := card("adaga", `["acuidade-com-arma"]`).StrDamage
	if comAcu <= semAcu {
		t.Errorf("Acuidade deveria aumentar o dano da adaga: sem=%d com=%d", semAcu, comAcu)
	}
}
