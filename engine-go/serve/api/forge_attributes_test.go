package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// OS ATRIBUTOS NA FOLHA DA FORJA. Eles são parte do formulário, e o herói nasce com eles: não há
// segunda tela, nem pontos a gastar. Quem rolou na mesa escreve o que saiu (p17).

// A escolha é livre, e o Forjar leva direto à ficha.
func TestTheHeroIsBornWithTheWrittenAttributes(t *testing.T) {
	f := newSceneFixture(t)
	answers := aFolhaPreenchida()
	for key, value := range map[string]string{
		"strength": "7", "dexterity": "-1", "constitution": "2", "intelligence": "−3", "wisdom": "", "charisma": "+4",
	} {
		answers.Set(key, value)
	}
	rec := postaAForja(t, f, f.player, "/personagens/nova", answers)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("a forja recusou (%d):\n%s", rec.Code, rec.Body.String())
	}
	id := oIDDoDestino(t, rec.Header().Get("Location"))
	if want := "/personagens/" + strconv.FormatInt(id, 10); rec.Header().Get("Location") != want {
		t.Errorf("o Forjar levou a %q, e não à ficha", rec.Header().Get("Location"))
	}
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o herói: %v", err)
	}
	if row.Strength != 7 || row.Dexterity != -1 || row.Constitution != 2 || row.Intelligence != -3 || row.Wisdom != 0 || row.Charisma != 4 {
		t.Errorf("For %d, Des %d, Con %d, Int %d, Sab %d, Car %d: esperava 7, -1, 2, -3, 0 e 4",
			row.Strength, row.Dexterity, row.Constitution, row.Intelligence, row.Wisdom, row.Charisma)
	}
}

// O que não é número recusa citando o que veio, e a folha mostra o que foi escrito.
func TestTheForgeRefusesAnAttributeThatIsNotAValue(t *testing.T) {
	f := newSceneFixture(t)
	for _, written := range []string{"abc", "150", "-100", "1.5"} {
		answers := aFolhaPreenchida()
		answers.Set("strength", written)
		rec := postaAForja(t, f, f.player, "/personagens/nova", answers)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("a forja aceitou Força %q: %d", written, rec.Code)
			continue
		}
		if !strings.Contains(rec.Body.String(), `Força: `) || !strings.Contains(rec.Body.String(), written) {
			t.Errorf("a recusa de %q não cita o atributo e o valor:\n%s", written, rec.Body.String())
		}
	}
	for _, limit := range []string{"99", "-99"} {
		answers := aFolhaPreenchida()
		answers.Set("strength", limit)
		if rec := postaAForja(t, f, f.player, "/personagens/nova", answers); rec.Code != http.StatusSeeOther {
			t.Errorf("a forja recusou o limite %s: %d", limit, rec.Code)
		}
	}
}

// A linha mostra o total e de onde ele vem, antes de o herói existir: o elfo dá Inteligência +2 (p22).
func TestTheForgeRowShowsTheSumWithTheRace(t *testing.T) {
	f := newSceneFixture(t)
	body := postaAForja(t, f, f.player, "/personagens/nova/esboco",
		url.Values{"race": {"elfo"}, "intelligence": {"1"}}).Body.String()
	if !strings.Contains(body, "+1 livre +2 Raça: Elfo") {
		t.Errorf("a linha de Inteligência não mostrou a soma:\n%s", body)
	}
	if !strings.Contains(body, ">+3<") {
		t.Errorf("a linha de Inteligência não mostrou o total +3:\n%s", body)
	}
	if !strings.Contains(body, `name="intelligence"`) || !strings.Contains(body, `value="1"`) {
		t.Errorf("o campo não guardou o valor escrito:\n%s", body)
	}
}

// O `−` e o `+` só redesenham: o valor mora no formulário.
func TestTheForgeStepperMovesTheWrittenValue(t *testing.T) {
	f := newSceneFixture(t)
	rec := postaAForja(t, f, f.player, "/personagens/nova/esboco/atributo/strength/-1", url.Values{"strength": {"0"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("o degrau respondeu %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `name="strength"`) || !strings.Contains(rec.Body.String(), `value="-1"`) {
		t.Errorf("o `−` não baixou a Força para -1:\n%s", rec.Body.String())
	}
	if code := postaAForja(t, f, f.player, "/personagens/nova/esboco/atributo/sorte/1", nil).Code; code != http.StatusBadRequest {
		t.Errorf("o degrau aceitou um atributo que não existe: %d", code)
	}
}

// Abaixo de −5 o livro prevê um efeito (p17). A folha avisa e não trava: quem decide é o mestre.
func TestBelowMinusFiveTheForgeWarnsWithoutRefusing(t *testing.T) {
	f := newSceneFixture(t)
	answers := aFolhaPreenchida()
	answers.Set("race", "humano")
	answers.Set("constitution", "-6")
	draft := postaAForja(t, f, f.player, "/personagens/nova/esboco", answers).Body.String()
	if !strings.Contains(draft, "morre (p17)") {
		t.Errorf("a linha de Constituição −6 não avisou:\n%s", draft)
	}
	if rec := postaAForja(t, f, f.player, "/personagens/nova", answers); rec.Code != http.StatusSeeOther {
		t.Errorf("o aviso virou recusa: %d", rec.Code)
	}
	answers.Set("constitution", "-5")
	if draft := postaAForja(t, f, f.player, "/personagens/nova/esboco", answers).Body.String(); strings.Contains(draft, "morre (p17)") {
		t.Error("−5 ainda não tem efeito: é só MENOR que −5")
	}
}
