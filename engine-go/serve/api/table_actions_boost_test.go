package api

import (
	"fmt"
	"strings"
	"testing"
)

// O IMPULSO ESCOLHIDO NA LINHA DE ATAQUE VIAJA NO GOLPE (ALE-487).
//
// A superfície Ações escreve a escolha em quatro sinais — a arma, o poder, os
// degraus e o lado —, e o "Rolar" daquela arma os leva até o provisório da mesa,
// que guarda o poder e o PM que a confirmação vai cobrar. O número com o impulso
// é do motor (`attack_boost_rules_test.go`); o que se prende aqui é a LIGAÇÃO.
func TestTheActionsSurfaceCarriesTheBoostIntoTheProposal(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, barbaro, "Guerreiro", 5)
	special := "class.guerreiro.ataque-especial-4"

	// A TIRA EXISTE na linha da arma: sem ela o sinal não teria quem o escreva.
	screen := f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), "")
	if !strings.Contains(screen, special) || !strings.Contains(screen, "Ataque Especial") {
		t.Fatalf("a linha de ataque não oferece o Ataque Especial:\n%s", recorte(screen))
	}

	// ESCOLHIDO NOUTRA ARMA, não viaja: o impulso é daquela linha.
	elsewhere := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":1,"boost_power":%q,"boost_steps":1,"boost_split":"attack"}`,
		goblin, special)
	if refusal := sceneRefusal(f.posts(t, f.player,
		fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), elsewhere)); refusal != "" {
		t.Fatalf("o golpe sem impulso foi recusado: %q", refusal)
	}
	if pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack; pending == nil || pending.BoostPowerID != "" {
		t.Fatalf("o impulso escolhido na arma 1 viajou no golpe da arma 0: %+v", pending)
	}
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/ataque/cancelar", ""); rec.Code != 200 {
		t.Fatalf("cancelar o provisório deu %d", rec.Code)
	}

	chosen := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":0,"boost_power":%q,"boost_steps":1,"boost_split":"attack"}`,
		goblin, special)
	if refusal := sceneRefusal(f.posts(t, f.player,
		fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), chosen)); refusal != "" {
		t.Fatalf("o golpe com o Ataque Especial foi recusado: %q", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil || pending.BoostPowerID != special || pending.BoostPM != 2 {
		t.Errorf("o provisório levou %+v; want o Ataque Especial com 1 degrau, 2 PM (p65)", pending)
	}
}
