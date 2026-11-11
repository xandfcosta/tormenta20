package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// OS ESTILOS DO ATAQUE A MAIS SE USAM, e não se ligam (p125, p127, p50, p65). "Se fizer isso, sofre -2
// em todos os testes de ataque até o seu próximo turno": é uma declaração feita antes de agredir, e
// declaração é o botão Usar do poder. O -2 dura até o próximo turno, e quem desfaz um uso por engano é o
// "encerrar" da aba Efeitos — não há chave para desligar.

func usesTheStyle(t *testing.T, f sceneFixture, id int64, power string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/poderes/usa/%s", id, power)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, `{}`).Body.String())
}

func TestTheAdditionalAttackStylesAreUsedAndNotSwitched(t *testing.T) {
	for _, c := range []struct {
		name, class, power, flag string
		weapons                  []string
	}{
		{"Estilo de Duas Armas", "", "estilo-de-duas-armas", "estilo-de-duas-armas", []string{"espada-longa", "adaga"}},
		{"Ambidestria do guerreiro", "Guerreiro", "class.guerreiro.ambidestria", "ataque-ambidestro", []string{"espada-longa", "adaga"}},
		{"Disparo Rápido", "", "disparo-rapido", "disparo-rapido", []string{"arco-curto"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id, goblin := styleOnTurn(t, "ataque-poderoso", false, c.weapons...)
			if c.class != "" {
				seedClasse(t, f.s, id, c.class, 1)
			}
			seedPowerChoices(t, f.s, id, `["`+c.power+`"]`, "{}")
			effects := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=conditionals", id), "").Body.String()
			if strings.Contains(effects, "flag:"+c.flag) {
				t.Error("a aba Efeitos ainda tem a chave do estilo")
			}
			if strings.Contains(acoesDaMesa(t, f), "flag:"+c.flag) {
				t.Error("as Ações da mesa ainda têm o chip do estilo")
			}
			if refusal := switchRefusal(t, f, id, c.flag); refusal == "" {
				t.Error("a chave do estilo ainda liga pela rota do interruptor")
			}
			if !strings.Contains(acoesDaMesa(t, f), "/poderes/usa/"+c.power) {
				t.Error("as Ações da mesa não têm o Usar do estilo")
			}

			if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
				t.Fatalf("o golpe sem usar: %q", refusal)
			}
			off := pendingOf(t, f)
			gmCancels(t, f)
			if refusal := usesTheStyle(t, f, id, c.power); refusal != "" {
				t.Fatalf("usar foi recusado: %q", refusal)
			}
			if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
				t.Fatalf("o primeiro golpe: %q", refusal)
			}
			if on := pendingOf(t, f); (on.Total-on.Roll)-(off.Total-off.Roll) != -2 {
				t.Errorf("usar mudou o ataque em %+d, e o livro dá -2", (on.Total-on.Roll)-(off.Total-off.Roll))
			}
			gmRules(t, f, "errou")
			if refusal := extraStrike(t, f, id, len(c.weapons)-1, goblin); refusal != "" {
				t.Errorf("o segundo ataque foi recusado: %q", refusal)
			}
		})
	}
}

// Usado por engano, ele se encerra na aba Efeitos, e o -2 sai junto.
func TestTheUsedStyleEndsInTheEffectsTab(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "estilo-de-duas-armas", false, "espada-longa", "adaga")
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o golpe sem usar: %q", refusal)
	}
	off := pendingOf(t, f)
	gmCancels(t, f)
	if refusal := usesTheStyle(t, f, id, "estilo-de-duas-armas"); refusal != "" {
		t.Fatalf("usar: %q", refusal)
	}
	end := fmt.Sprintf("/personagens/%d/efeitos/declaracao/estilo-de-duas-armas", id)
	effects := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=conditionals", id), "").Body.String()
	if !strings.Contains(effects, end) {
		t.Fatal("a aba Efeitos não tem o encerrar do estilo usado")
	}
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, end, `{}`).Body.String()); refusal != "" {
		t.Fatalf("encerrar foi recusado: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o golpe depois de encerrar: %q", refusal)
	}
	if after := pendingOf(t, f); after.Total-after.Roll != off.Total-off.Roll {
		t.Errorf("encerrado, o ataque devia voltar a %+d, e está em %+d", off.Total-off.Roll, after.Total-after.Roll)
	}
}

// Encerrar é só para o que se usa: uma postura ou um interruptor não saem por esta rota.
func TestOnlyADeclarationEndsByTheDeclarationRoute(t *testing.T) {
	f, id, _ := styleOnTurn(t, "ataque-poderoso", false, "espada-longa")
	end := fmt.Sprintf("/personagens/%d/efeitos/declaracao/ataque-poderoso", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, end, `{}`).Body.String()); refusal == "" {
		t.Error("o Ataque Poderoso, que é interruptor, foi aceito pela rota da declaração")
	}
}

// Com o Estilo de Duas Armas, a Ambidestria de classe só tira o -2 (p125): o Usar é um só, o do estilo.
func TestWithTheStyleTheAmbidextrousIsOnlyPassive(t *testing.T) {
	f, id, _ := styleOnTurn(t, "ataque-poderoso", false, "espada-longa", "adaga")
	seedClasse(t, f.s, id, "Guerreiro", 1)
	seedPowerChoices(t, f.s, id, `["estilo-de-duas-armas","class.guerreiro.ambidestria"]`, "{}")
	if strings.Contains(acoesDaMesa(t, f), "/poderes/usa/class.guerreiro.ambidestria") {
		t.Error("com o estilo, a Ambidestria não devia ter Usar")
	}
	if refusal := usesTheStyle(t, f, id, "class.guerreiro.ambidestria"); refusal == "" {
		t.Error("com o estilo, usar a Ambidestria devia ser recusado")
	}
}

// Nas duas classes, a Ambidestria é um Usar só: são dois poderes para o mesmo gesto.
func TestAmbidextrousInTwoClassesIsOneUse(t *testing.T) {
	f, id, _ := styleOnTurn(t, "ataque-poderoso", false, "espada-longa", "adaga")
	seedClasse(t, f.s, id, "Guerreiro", 1)
	seedClasse(t, f.s, id, "Caçador", 1)
	seedPowerChoices(t, f.s, id, `["class.guerreiro.ambidestria","class.cacador.ambidestria"]`, "{}")
	surface := acoesDaMesa(t, f)
	if got := strings.Count(surface, "/poderes/usa/class.guerreiro.ambidestria") + strings.Count(surface, "/poderes/usa/class.cacador.ambidestria"); got != 1 {
		t.Errorf("as Ações têm %d Usar de Ambidestria, e o gesto é um", got)
	}
}
