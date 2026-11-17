package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// A SITUAÇÃO LIGADA ESTÁ EM VIGOR COMO OS OUTROS EFEITOS (decisão do dono). O número do
// ícone de Efeitos a contava, e a coluna da ficha não tinha crachá para ela: com uma
// situação ligada, o ícone dizia 1 e a coluna não mostrava nada. Agora ela é crachá, o X
// a desliga, e o ícone conta a mesma lista que a coluna desenha.
func TestTheSituationalSwitchedOnIsABadgeAndTheIconCountsTheSameList(t *testing.T) {
	f, id := fighterFixture(t)
	seedEfeitoCondicional(t, f.s, id, 3)
	key := offeredSituationalKey(t, f, id)
	count := regexp.MustCompile(`class="ds-nav-count" aria-label="(\d+) ativos"`)
	effectDialogs := regexp.MustCompile(`<dialog[^>]*id="effect-[^"]*"`)

	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, fmt.Sprintf(`{"conditional":%q}`, key)).Body.String()); refusal != "" {
		t.Fatalf("ligar a situação %q: %q", key, refusal)
	}
	page := sheetTabWith(t, f, id, "skills", `{}`)

	m := count.FindStringSubmatch(page)
	if m == nil {
		t.Fatal("o ícone de Efeitos não tem número com uma situação ligada")
	}
	var icon int
	fmt.Sscan(m[1], &icon)
	if badges := len(effectDialogs.FindAllString(page, -1)); icon != badges {
		t.Errorf("o ícone de Efeitos diz %d ativos, e a coluna desenha %d efeitos em vigor", icon, badges)
	}
	// O X do crachá desliga a mesma situação, pela mesma rota do interruptor.
	if !strings.Contains(page, `$conditional = &#39;`+key+`&#39;`) && !strings.Contains(page, `$conditional = '`+key+`'`) {
		t.Errorf("nenhum gesto da página desliga a situação %q", key)
	}
	if !regexp.MustCompile(`aria-label="Tirar [^"]*"[^>]*data-on:click="[^"]*` + regexp.QuoteMeta(key)).MatchString(page) {
		t.Errorf("a coluna não tem o crachá da situação %q com o X que a desliga", key)
	}
}
