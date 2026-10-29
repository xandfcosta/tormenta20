package api

import (
	"regexp"
	"strings"
	"testing"
)

// O DIÁLOGO ESCOLHER DA FICHA PARALELA (ALE-473), pelo mockup aprovado: as
// pendências no topo, e as escolhas de raça, origem e classe com as peças do
// kit — opção para o valor curto, interruptor para a lista longa. Todo gesto
// leva a marca da ficha paralela, e nada do grimório entra no diálogo.
func TestTheParallelChoicesDialogOffersEveryChoiceWithTheKit(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Paladino", 3, 0, 0)
	page := parallelSheet(t, f, id, "abilities")

	dialog := regexp.MustCompile(`(?s)<dialog[^>]*id="power-choices".*?</dialog>`).FindString(page)
	if dialog == "" {
		t.Fatal("a aba Poderes paralela não desenhou o diálogo de escolhas")
	}
	if grimoire := regexp.MustCompile(`[\w:-]*grimorio[\w-]*`).FindAllString(dialog, 3); len(grimoire) > 0 {
		t.Errorf("o diálogo de escolhas desenhou classe do grimório: %v", grimoire)
	}
	if !strings.Contains(dialog, "Faltam") {
		t.Error("o diálogo não abre com as pendências do paladino")
	}
	for what, pattern := range map[string]string{
		"o devoto como opção":                    `<button[^>]*class="ds-option"[^>]*data-on:click="[^"]*/poderes/classe/Paladino/devoto/[^"]*tempestade=1`,
		"o benefício da origem como interruptor": `<input[^>]*role="switch"[^>]*data-on:change="[^"]*/poderes/origem/[^"]*tempestade=1`,
		"o poder da classe como interruptor":     `<input[^>]*role="switch"[^>]*data-on:change="[^"]*/poderes/escolhe/[^"]*tempestade=1`,
	} {
		if !regexp.MustCompile(pattern).MatchString(dialog) {
			t.Errorf("o diálogo não oferece %s", what)
		}
	}
}
