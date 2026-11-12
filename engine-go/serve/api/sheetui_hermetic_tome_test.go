package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// O TOMO HERMÉTICO: "A CD para resistir a suas magias arcanas dessa escola aumenta em +2" (p160). Quem
// escolhe a escola é o jogador, na folha do tomo, e ele pode trocar (decisão do dono). Sem escola, o tomo
// não dá nada. A Bola de Fogo é evocação; o Sono, encantamento.

func spellCDAtTheTable(t *testing.T, f sceneFixture, spellName string) int {
	t.Helper()
	found := regexp.MustCompile(`Conjurar · ` + regexp.QuoteMeta(spellName) + `(?s:.*?)CD (\d+)`).FindStringSubmatch(acoesDaMesa(t, f))
	if found == nil {
		t.Fatalf("as Ações da mesa não escreveram a CD de %s", spellName)
	}
	cd, _ := strconv.Atoi(found[1])
	return cd
}

func choosesTheSchool(t *testing.T, f sceneFixture, item int64, school string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/itens/%d/escola/%s?tab=bag", f.charID, item, school)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, "").Body.String())
}

func TestTheHermeticTomeRaisesOnlyItsSchool(t *testing.T) {
	f := newSceneFixture(t)
	learnSpell(t, f, "bola-de-fogo")
	learnSpell(t, f, "sono")
	fire, sleep := spellCDAtTheTable(t, f, "Bola de Fogo"), spellCDAtTheTable(t, f, "Sono")

	tome := itemSemeia(t, f, f.charID, "tomo-hermetico", "Tomo hermético", "wielded")
	if got := spellCDAtTheTable(t, f, "Bola de Fogo"); got != fire {
		t.Errorf("sem escola escolhida, o tomo mudou a CD da Bola de Fogo de %d para %d", fire, got)
	}
	if !strings.Contains(bagScreen(t, f, f.charID), "Escolha a escola") {
		t.Error("a folha do tomo sem escola não pede a escolha")
	}

	if refusal := choosesTheSchool(t, f, tome, "evocacao"); refusal != "" {
		t.Fatalf("escolher evocação: %q", refusal)
	}
	if got := spellCDAtTheTable(t, f, "Bola de Fogo"); got != fire+2 {
		t.Errorf("com evocação, a CD da Bola de Fogo foi de %d para %d, e o livro dá +2", fire, got)
	}
	if got := spellCDAtTheTable(t, f, "Sono"); got != sleep {
		t.Errorf("o Sono é encantamento, e a CD dele foi de %d para %d", sleep, got)
	}

	if refusal := choosesTheSchool(t, f, tome, "encantamento"); refusal != "" {
		t.Fatalf("trocar para encantamento: %q", refusal)
	}
	if got := spellCDAtTheTable(t, f, "Sono"); got != sleep+2 {
		t.Errorf("trocada a escola, a CD do Sono foi de %d para %d, e o livro dá +2", sleep, got)
	}
	if got := spellCDAtTheTable(t, f, "Bola de Fogo"); got != fire {
		t.Errorf("trocada a escola, a Bola de Fogo devia voltar a %d, e está em %d", fire, got)
	}
	if refusal := choosesTheSchool(t, f, tome, "culinaria"); refusal == "" {
		t.Error("uma escola que não existe foi aceita")
	}
}
