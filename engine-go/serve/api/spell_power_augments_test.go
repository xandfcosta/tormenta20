package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// OS PODERES QUE SÃO APRIMORAMENTO (p131): "Eles seguem todas as regras para
// aprimoramentos", "desde que não ultrapasse seu limite de PM". Eles moram no
// diálogo de conjurar, somam no custo da magia e passam pelo mesmo teto; um
// botão Usar à parte cobrava o PM sem a magia nenhuma.

const allSpellPowers = `["magia-acelerada","magia-ampliada","magia-discreta"]`

// castWithPowers é a arcanista de 8º nível da bancada com os três poderes.
func castWithPowers(t *testing.T) sceneFixture {
	t.Helper()
	f := newSceneFixture(t)
	seedPowerChoices(t, f.s, f.charID, allSpellPowers, "{}")
	return f
}

// castWith conjura pela rota da aba Magias, com os sinais do diálogo no corpo, e
// devolve a recusa e o PM gasto. O PM é medido sempre: toda recusa tem de sair de
// graça.
func castWith(t *testing.T, f sceneFixture, spellID, signals string) (string, int64) {
	t.Helper()
	learnSpell(t, f, spellID)
	before := mpOf(t, f.s, f.charID)
	target := fmt.Sprintf("/personagens/%d/magias/conjura/%s?tab=spells", f.charID, spellID)
	refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, signals).Body.String())
	return refusal, before - mpOf(t, f.s, f.charID)
}

func TestSpellPowersAddTheirCostToTheCast(t *testing.T) {
	cases := []struct {
		name, spellID, signals string
		spent                  int64
	}{
		// 1º círculo: 1 PM (Tabela 4-1, p170), mais os 4 da Acelerada.
		{"Acelerada soma 4", "luz", `{"augment_quickened":1}`, 5},
		// 2º círculo: 3 PM, mais os 2 da Ampliada.
		{"Ampliada soma 2", "bola-de-fogo", `{"augment_amplified":1}`, 5},
		// "Aumenta" empilha (p171): curto para médio, médio para longo.
		{"Ampliada empilha", "luz", `{"augment_amplified":2}`, 5},
		{"Discreta soma 2", "luz", `{"augment_discreet":1}`, 3},
		{"Discreta com Ampliada", "luz", `{"augment_discreet":1,"augment_amplified":1}`, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := castWithPowers(t)
			refusal, spent := castWith(t, f, c.spellID, c.signals)
			if refusal != "" {
				t.Fatalf("a conjuração foi recusada: %q", refusal)
			}
			if spent != c.spent {
				t.Errorf("gastou %d PM; want %d", spent, c.spent)
			}
		})
	}
}

func TestSpellPowersAreRefusedForFree(t *testing.T) {
	cases := []struct {
		name, spellID, signals, says string
	}{
		// "desde que não ultrapasse seu limite de PM" (p131): nível 8, 3 + 2×3 = 9.
		{"acima do limite de PM", "bola-de-fogo", `{"augment_amplified":3}`, "limite de 8"},
		// "magias com execução de movimento, padrão ou completa" (p131).
		{"Acelerada numa reação", "queda-suave", `{"augment_quickened":1}`, "execução de movimento, padrão ou completa"},
		// Longo não tem próximo passo, e a magia não tem área.
		{"Ampliada sem passo nem área", "aviso", `{"augment_amplified":1}`, "alcance curto ou médio, ou com área"},
		// Só "aumenta" empilha (p171): a Discreta troca a magia e não se repete.
		{"Discreta repetida", "luz", `{"augment_discreet":2}`, "não se acumula"},
		// "Truques não podem ser usados em conjunto com outros aprimoramentos" (p171).
		{"truque acompanhado", "explosao-de-chamas", `{"augment0":1,"augment_discreet":1}`, "em conjunto"},
		// "Mudanças na mesma característica da magia nunca se acumulam" (p171), e o
		// índice 1 do Armamento da Natureza é o "Muda a execução para ação de movimento".
		{"duas mudanças de execução", "armamento-da-natureza", `{"augment1":1,"augment_quickened":1}`, "mudam a execução"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := castWithPowers(t)
			refusal, spent := castWith(t, f, c.spellID, c.signals)
			if !strings.Contains(refusal, c.says) {
				t.Errorf("a recusa diz %q; devia dizer %q", refusal, c.says)
			}
			if spent != 0 {
				t.Errorf("a recusa gastou %d PM", spent)
			}
		})
	}
}

func TestASpellPowerTheSheetDoesNotHaveIsRefused(t *testing.T) {
	f := newSceneFixture(t)
	refusal, spent := castWith(t, f, "luz", `{"augment_discreet":1}`)
	if !strings.Contains(refusal, "Magia Discreta") {
		t.Errorf("a recusa diz %q; devia nomear o poder que falta", refusal)
	}
	if spent != 0 {
		t.Errorf("a recusa gastou %d PM", spent)
	}
}

// "Muda a execução da magia para ação livre" (p131): a conjuração não gasta a
// padrão do turno, e é isso que distingue este poder de pagar 4 PM à toa.
func TestTheQuickenedSpellLeavesTheStandardActionOnTheTurn(t *testing.T) {
	f := castWithPowers(t)
	startCombatWithSomeoneElseOnTurn(t, f)
	if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
		t.Fatalf("passar a vez ao personagem: %v", err)
	}
	if refusal, _ := castWith(t, f, "luz", `{"augment_quickened":1}`); refusal != "" {
		t.Fatalf("a magia acelerada foi recusada: %q", refusal)
	}
	if refusal, spent := castWith(t, f, "luz", ""); refusal != "" || spent != 1 {
		t.Errorf("a padrão do turno devia estar de pé depois da acelerada: recusa %q, %d PM", refusal, spent)
	}
}

func TestTheSpellSheetOffersTheSpellPowersTheCharacterHas(t *testing.T) {
	f := castWithPowers(t)
	learnSpell(t, f, "queda-suave")
	sheet := spellSheetOf(t, spellScreen(t, f, f.charID), "spell-queda-suave")

	for _, name := range []string{"Magia Acelerada", "Magia Ampliada", "Magia Discreta"} {
		if !strings.Contains(sheet, name) {
			t.Errorf("a folha da magia não oferece %s", name)
		}
	}
	// O custo da prévia soma o que se pode escolher: a Acelerada numa reação fica
	// trancada, e somá-la mostraria um custo que o servidor nunca cobra.
	if strings.Contains(sheet, "4 * $augment_quickened") {
		t.Error("a prévia soma a Acelerada, trancada nesta magia")
	}
	if !strings.Contains(sheet, "2 * $augment_amplified") {
		t.Error("a prévia não soma a Ampliada")
	}
	if !strings.Contains(sheet, "só em execução de movimento, padrão ou completa") {
		t.Error("a folha não diz por que a Acelerada está trancada")
	}

	// Quem não tem o poder não vê a linha.
	without := newSceneFixture(t)
	learnSpell(t, without, "queda-suave")
	if strings.Contains(spellSheetOf(t, spellScreen(t, without, without.charID), "spell-queda-suave"), "Magia Discreta") {
		t.Error("a folha oferece um poder que a ficha não tem")
	}
}

// Com o poder, a magia sem aprimoramento próprio também tem o que escolher, e o
// Conjurar da linha abre a folha em vez de conjurar direto.
func TestASpellWithoutItsOwnAugmentsOpensTheSheetForASpellPower(t *testing.T) {
	f := castWithPowers(t)
	learnSpell(t, f, "marionete")
	opens := regexp.MustCompile(`aria-label="Conjurar Marionete"[^>]*commandfor=`)
	if !opens.MatchString(spellScreen(t, f, f.charID)) {
		t.Error("o Conjurar da linha não abre a folha para escolher o poder")
	}
}

// O botão Usar cobrava o PM sem a magia: o gesto saiu da aba Poderes.
func TestSpellPowersHaveNoUseGestureOfTheirOwn(t *testing.T) {
	f := castWithPowers(t)
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities", f.charID), "").Body.String()
	for _, name := range []string{"Magia Acelerada", "Magia Ampliada", "Magia Discreta"} {
		if !strings.Contains(page, name) {
			t.Fatalf("o controle falhou: a aba Poderes não lista %s", name)
		}
		if strings.Contains(page, `aria-label="Usar `+name+`"`) {
			t.Errorf("%s ainda tem o botão Usar", name)
		}
	}
}
