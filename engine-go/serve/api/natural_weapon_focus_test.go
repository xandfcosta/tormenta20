package api

import (
	"strings"
	"testing"
)

// O FOCO EM ARMA ALCANÇA A ARMA NATURAL (p128, decisão do dono).

// p128: "Escolha uma arma". A arma natural é arma, e o +2 só vale nela.
func TestWeaponFocusReachesTheNaturalWeapon(t *testing.T) {
	f, id, _ := minotaurOnTurn(t)
	before := attackByWeapon(t, f, id)
	if !strings.Contains(sheetTab(t, f, id, "abilities"), "/poderes/escolha/foco-em-arma/minotauro-chifres") {
		t.Error("o Foco em Arma não oferece os chifres")
	}
	if refusal := powerCommand(t, f, id, "escolha/foco-em-arma/minotauro-chifres", ""); refusal != "" {
		t.Fatalf("escolher os chifres no Foco em Arma foi recusado: %q", refusal)
	}
	after := attackByWeapon(t, f, id)
	if after["Chifres"]-before["Chifres"] != 2 {
		t.Errorf("os chifres foram de %+d a %+d; o Foco em Arma dá +2", before["Chifres"], after["Chifres"])
	}
	if after["Ataque desarmado"] != before["Ataque desarmado"] {
		t.Errorf("o desarmado foi de %+d a %+d, e não foi escolhido", before["Ataque desarmado"], after["Ataque desarmado"])
	}
}

// O Foco em Arma só oferece a arma natural que a ficha tem.
func TestWeaponFocusOffersOnlyTheNaturalWeaponsOfTheSheet(t *testing.T) {
	f, id, _ := barbarianOnTurn(t)
	if strings.Contains(sheetTab(t, f, id, "abilities"), "/poderes/escolha/foco-em-arma/minotauro-chifres") {
		t.Error("quem não é minotauro vê os chifres no Foco em Arma")
	}
	if refusal := powerCommand(t, f, id, "escolha/foco-em-arma/minotauro-chifres", ""); !strings.Contains(refusal, "não tem a arma natural") {
		t.Errorf("escolher pelo endereço devia ser recusado, e veio %q", refusal)
	}
}
