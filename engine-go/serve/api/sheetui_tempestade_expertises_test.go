package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// A ABA PERÍCIAS DA FICHA PARALELA (ALE-473): cada perícia é uma linha que abre
// a conta, e o treino é um interruptor — a não ser que outra coisa treine, e aí
// é um cadeado que diz quem (decisão do dono, no mockup aprovado).

// expertiseSum recorta a linha de UMA perícia: do `<details>` até o fim dele.
func expertiseSum(t *testing.T, page, name string) string {
	t.Helper()
	for _, block := range regexp.MustCompile(`(?s)<details[^>]*>.*?</details>`).FindAllString(page, -1) {
		if strings.Contains(block, `<div class="ds-row-title">`+name+`</div>`) {
			return block
		}
	}
	t.Fatalf("a aba Perícias paralela não desenhou a linha de %s", name)
	return ""
}

func TestTheParallelExpertiseTabLocksTheTrainingThatTheOriginGives(t *testing.T) {
	f, id := expertiseFixture(t)
	target := fmt.Sprintf("/personagens/%d/poderes/origem/origin-soldado-pericia-fortitude?tab=powers", id)
	if rec := f.requests(t, f.player, http.MethodPost, target, ""); rec.Code != http.StatusOK {
		t.Fatalf("escolher o benefício respondeu %d: %s", rec.Code, rec.Body.String())
	}
	page := parallelSheet(t, f, id, "expertises")

	fortitude := expertiseSum(t, page, "Fortitude")
	if !strings.Contains(fortitude, `<span class="ds-row-value">+6</span>`) {
		t.Errorf("a Fortitude do soldado com o benefício não fechou em +6 (½ nível 1 + CON 3 + treino 2):\n%s", fortitude)
	}
	if !strings.Contains(fortitude, "Treinada pela Origem: Soldado") || strings.Contains(fortitude, `role="switch"`) {
		t.Errorf("a Fortitude treinada pela origem tem de dizer quem treinou, e sem interruptor:\n%s", fortitude)
	}

	acrobatics := expertiseSum(t, page, "Acrobacia")
	if !regexp.MustCompile(`<input[^>]*role="switch"[^>]*data-on:change="[^"]*/pericias/treino/Acrobacia`).MatchString(acrobatics) {
		t.Errorf("a Acrobacia, de treino livre, não tem o interruptor que alterna o treino:\n%s", acrobatics)
	}

	// A "só treinada" sem treino diz POR QUE o número não vale (decisão do
	// dono): o número riscado sozinho não explicava nada.
	if acting := expertiseSum(t, page, "Atuação"); !strings.Contains(acting, "só treinada — você não tem treino") {
		t.Errorf("a Atuação sem treino não diz que precisa de treino:\n%s", acting)
	}
}
