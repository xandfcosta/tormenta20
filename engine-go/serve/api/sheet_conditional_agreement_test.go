package api

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O CRACHÁ DA FICHA E A ABA COMBATE TÊM DE DIZER A MESMA DEFESA.
//
// # As duas contas, e a que passa o conjunto vazio
//
// O painel de Combate computa a ficha com os condicionais que o jogador LIGOU
// (`sheetForPanels`), e o cabeçalho dele diz por quê: *"com Fúria ligada, a base
// mostraria o ataque de quem não está em Fúria"*. O crachá do topo da MESMA tela
// sai do `HeroCardOf`, que passa pelo `sheet.Compute` — e esse passa
// `map[string]bool{}`.
//
// O mesmo cabeçalho já avisava do risco, uma função acima: *"duas contas que
// podem divergir no dia em que uma delas passar um conjunto de condicionais
// diferente do da outra"*. Esse dia é hoje, um arquivo adiante.
//
// # Por que nenhuma suíte via
//
// Porque é caso por DADO e não por navegação: ficha sem condicional ligado — a
// maioria — desenha o mesmo número pelos dois caminhos. Medido nos 18 oráculos,
// três têm condicional ligado e DOIS divergem na Defesa; a Lenda de nível 20
// erra por dez pontos (22 contra 32).
//
// # O CONTROLE vem primeiro
//
// Com o condicional DESLIGADO os dois já concordam, então um caso que só
// comparasse os dois números passaria sobre o repouso. Aqui a primeira metade
// afirma que ligar o condicional MOVE a aba Combate — sem isso, "os dois
// concordam" poderia ser "nenhum dos dois mudou".
func TestTheSheetBadgeAndTheCombatTabAgreeOnDefense(t *testing.T) {
	f, id := fighterFixture(t)
	seedConditionalDefenseEffect(t, f.s, id, 5)

	beforeEnabling := combatTabDefense(t, f, id)
	ligaOCondicional(t, f, id)
	ofPanel := combatTabDefense(t, f, id)

	if ofPanel == beforeEnabling {
		t.Fatalf("ligar o condicional não moveu a Defesa da aba Combate (%s): "+
			"o caso mediria duas telas paradas", ofPanel)
	}

	if ofBadge := badgeDefense(t, f, id); ofBadge != ofPanel {
		t.Errorf("o crachá do topo diz Defesa %s e a aba Combate diz %s, na MESMA ficha.\n"+
			"O crachá sai do `sheet.Compute`, que passa `map[string]bool{}` no lugar dos\n"+
			"condicionais que o jogador ligou; o painel passa os de verdade.",
			ofBadge, ofPanel)
	}
}

// seedConditionalDefenseEffect é o irmão do `seedEfeitoCondicional` que mexe na
// DEFESA — é ela que o crachá do topo mostra.
func seedConditionalDefenseEffect(t *testing.T, s *Server, id int64, howMuch int) {
	t.Helper()
	mods := fmt.Sprintf(
		`[{"target":{"k":"defense"},"amount":%d,"bonusType":"untyped",`+
			`"condition":{"c":"context","note":"enquanto estiver em Fúria"}}]`, howMuch)
	if _, err := s.sceneCore().Queries().CreateActiveEffect(context.Background(),
		sqlcgen.CreateActiveEffectParams{
			Characterid: id, Catalogid: "furia", Scope: "scene",
			Modifiers: mods, Createdat: dbvalue.NowISO(),
		}); err != nil {
		t.Fatalf("semear o efeito condicional de defesa: %v", err)
	}
}

// As duas telas escrevem a Defesa de formas diferentes, e cada uma tem o seu
// seletor: a aba Combate a põe no nome ACESSÍVEL da caixa, e o crachá do topo a
// põe no `<span>` depois do rótulo `DEF`.
var (
	panelDefensePattern = regexp.MustCompile(`aria-label="Defesa ([^"]+)"`)
	badgeDefensePattern = regexp.MustCompile(`>DEF</span>\s*<span[^>]*>([^<]+)</span>`)
)

func combatTabDefense(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	return firstMatchOf(t, panelDefensePattern, combatScreen(t, f, id), "a aba Combate")
}

func badgeDefense(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	body := f.requests(t, f.player, "GET",
		fmt.Sprintf("/personagens/%d?tab=expertises", id), "").Body.String()
	return firstMatchOf(t, badgeDefensePattern, body, "o crachá do topo")
}

func firstMatchOf(t *testing.T, re *regexp.Regexp, html, where string) string {
	t.Helper()
	found := re.FindStringSubmatch(html)
	if found == nil {
		t.Fatalf("%s não desenhou Defesa nenhuma — o seletor deixou de casar, "+
			"e sem ele este caso compararia duas strings vazias", where)
	}
	value := strings.TrimSpace(found[1])
	if value == "" {
		t.Fatalf("%s desenhou uma Defesa vazia", where)
	}
	return value
}
