package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// OS ESTILOS PELO GESTO. Os testes de motor passam o interruptor ligado direto, e foi assim que o
// Estilo de Uma Arma deixou de ser oferecido com a suíte inteira verde: o que importa é o toque.

func switchRefusal(t *testing.T, f sceneFixture, id int64, flag string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", id)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, fmt.Sprintf(`{"conditional":"flag:%s"}`, flag)).Body.String())
}

// "Em uma das mãos e nada na outra" (p128). O interruptor fica, e avisa: quem decide é o mestre, como
// no Estilo de Duas Armas. O número é que não mexe enquanto a mão não estiver livre.
func TestTheOneWeaponSwitchStaysAndWarnsWithAFullHand(t *testing.T) {
	const warning = "sem a mão livre"
	for _, c := range []struct {
		name    string
		weapons []string
		warns   bool
		moves   bool
	}{
		{"com a outra mão livre", []string{"espada-longa"}, false, true},
		{"com o escudo na outra", []string{"espada-longa", "escudo-leve"}, true, false},
		{"com duas armas", []string{"espada-longa", "adaga"}, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id, _ := styleOnTurn(t, "estilo-de-uma-arma", false, c.weapons...)
			chips := acoesDaMesa(t, f)
			if !strings.Contains(chips, "com arma em uma das mãos") {
				t.Fatal("o chip do Estilo de Uma Arma sumiu")
			}
			if got := strings.Contains(chips, warning); got != c.warns {
				t.Errorf("avisou = %v, want %v", got, c.warns)
			}
			before := defenceOnTheSheet(t, f, id)
			if refusal := switchRefusal(t, f, id, "estilo-de-uma-arma"); refusal != "" {
				t.Fatalf("ligar foi recusado: %q", refusal)
			}
			if moved := defenceOnTheSheet(t, f, id) != before; moved != c.moves {
				t.Errorf("a Defesa mudou = %v, want %v", moved, c.moves)
			}
			if refusal := switchRefusal(t, f, id, "estilo-de-uma-arma"); refusal != "" {
				t.Errorf("desligar foi recusado: %q", refusal)
			}
		})
	}
}

// AMBIDESTRIA. Com o poder geral: "em vez disso não sofre penalidade" (p125), e os dois ataques ficam.
// Sozinha, na classe: os mesmos dois ataques, com -2 (p50, p65).
func TestAmbidextrousGivesTheSecondAttack(t *testing.T) {
	for _, c := range []struct {
		name, class, powers, flag, label string
		penalty                          int
	}{
		{"guerreiro com o estilo", "Guerreiro", `["estilo-de-duas-armas","class.guerreiro.ambidestria"]`, "estilo-de-duas-armas", "Duas Armas: um ataque com cada, sem penalidade", 0},
		{"guerreiro sem o estilo", "Guerreiro", `["class.guerreiro.ambidestria"]`, "ataque-ambidestro", "Ambidestria: um ataque com cada", -2},
		{"caçador sem o estilo", "Caçador", `["class.cacador.ambidestria"]`, "ataque-ambidestro", "Ambidestria: um ataque com cada", -2},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id, goblin := styleOnTurn(t, "ataque-poderoso", false, "espada-longa", "adaga")
			seedClasse(t, f.s, id, c.class, 1)
			seedPowerChoices(t, f.s, id, c.powers, "{}")
			if chips := acoesDaMesa(t, f); !strings.Contains(chips, c.label) {
				t.Fatalf("as Ações não têm o chip %q", c.label)
			}
			if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
				t.Fatalf("o golpe sem o interruptor: %q", refusal)
			}
			off := pendingOf(t, f)
			gmCancels(t, f)
			if refusal := switchRefusal(t, f, id, c.flag); refusal != "" {
				t.Fatalf("ligar foi recusado: %q", refusal)
			}
			if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
				t.Fatalf("o primeiro golpe: %q", refusal)
			}
			on := pendingOf(t, f)
			if got := (on.Total - on.Roll) - (off.Total - off.Roll); got != c.penalty {
				t.Errorf("o interruptor mudou o ataque em %+d, want %+d", got, c.penalty)
			}
			gmRules(t, f, "errou")
			if refusal := extraStrike(t, f, id, 1, goblin); refusal != "" {
				t.Errorf("o segundo ataque foi recusado: %q", refusal)
			}
		})
	}
}

// Com os dois poderes há um interruptor só: dois dariam dois -2, ou um que não faz nada.
func TestAmbidextrousAndTheStyleShareOneSwitch(t *testing.T) {
	f, id, _ := styleOnTurn(t, "ataque-poderoso", false, "espada-longa", "adaga")
	seedClasse(t, f.s, id, "Guerreiro", 1)
	seedPowerChoices(t, f.s, id, `["estilo-de-duas-armas","class.guerreiro.ambidestria"]`, "{}")
	if strings.Contains(acoesDaMesa(t, f), "Ambidestria: um ataque com cada") {
		t.Error("com o poder geral, o interruptor da classe não devia existir")
	}
}

// Quem tem a Ambidestria nas duas classes paga o -2 uma vez: são dois poderes para o mesmo gesto.
func TestAmbidextrousInTwoClassesPaysOnce(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "ataque-poderoso", false, "espada-longa", "adaga")
	seedClasse(t, f.s, id, "Guerreiro", 1)
	seedClasse(t, f.s, id, "Caçador", 1)
	seedPowerChoices(t, f.s, id, `["class.guerreiro.ambidestria","class.cacador.ambidestria"]`, "{}")
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o golpe: %q", refusal)
	}
	off := pendingOf(t, f)
	gmCancels(t, f)
	if refusal := switchRefusal(t, f, id, "ataque-ambidestro"); refusal != "" {
		t.Fatalf("ligar: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o golpe: %q", refusal)
	}
	on := pendingOf(t, f)
	if got := (on.Total - on.Roll) - (off.Total - off.Roll); got != -2 {
		t.Errorf("o interruptor mudou o ataque em %+d, e o livro dá -2", got)
	}
}
