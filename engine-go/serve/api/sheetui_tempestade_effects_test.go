package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// A ABA EFEITOS DA FICHA PARALELA (ALE-473), pelo mockup aprovado: as
// condições como crachás removíveis, o que está em vigor com o Encerrar, a
// situação com interruptor, e os sempre ativos dos itens recolhidos.

// effectsInPlay é o bárbaro com uma condição, uma postura em curso e um
// situacional oferecido: os três blocos com conteúdo na mesma tela.
func effectsInPlay(t *testing.T) (sceneFixture, int64, string) {
	t.Helper()
	f, id := barbaro(t, 10)
	seedEfeitoCondicional(t, f.s, id, 3)
	if rec := effect(t, f, id, "condicao/caido"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Caído: status %d", rec.Code)
	}
	rage := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/poderes/postura/furia/entra?tab=abilities", id), "")
	if refusal := sceneRefusal(rage.Body.String()); rage.Code != http.StatusOK || refusal != "" {
		t.Fatalf("entrar na Fúria: status %d, recusa %q", rage.Code, refusal)
	}
	return f, id, parallelSheet(t, f, id, "conditionals")
}

func effectsTab(t *testing.T, page string) string {
	t.Helper()
	tab := regexp.MustCompile(`(?s)<section[^>]*aria-label="Efeitos".*?</section>`).FindString(page)
	if tab == "" {
		t.Fatal("a ficha paralela não desenhou a aba Efeitos")
	}
	return tab
}

func TestTheParallelEffectsTabShowsWhatIsInPlay(t *testing.T) {
	_, _, page := effectsInPlay(t)
	if strings.Contains(page, "ainda está na ficha de hoje") {
		t.Fatal("a aba Efeitos paralela ainda leva à ficha de hoje")
	}
	tab := effectsTab(t, page)

	prone := "effect-condition-caido"
	for what, pattern := range map[string]string{
		"o Caído como crachá que abre a descrição": `commandfor="` + prone + `"`,
		"o X do Caído":                 `data-on:click="[^"]*/efeitos/condicao/caido[^"]*tempestade=1`,
		"o gesto de aplicar condição":  `commandfor="condition-picker"`,
		"o Encerrar da Fúria":          `aria-label="Encerrar Fúria"[^>]*data-on:click="[^"]*/efeitos/postura/furia`,
		"o interruptor da situação":    `<input[^>]*role="switch"[^>]*data-on:change="[^"]*/efeitos/situacao[^"]*tempestade=1`,
		"o gesto de aplicar uma magia": `commandfor="buff-picker"`,
	} {
		if !regexp.MustCompile(pattern).MatchString(tab) {
			t.Errorf("a aba não tem %s", what)
		}
	}

	// A DESCRIÇÃO É UMA SÓ, e fora da coluna do desktop: dentro do `<aside>`,
	// que o celular esconde, o toque no crachá da aba não abriria nada.
	if n := strings.Count(page, `id="`+prone+`"`); n != 1 {
		t.Errorf("a descrição do Caído aparece %d vezes; tem de ser uma", n)
	}
	aside := regexp.MustCompile(`(?s)<aside.*?</aside>`).FindString(page)
	if strings.Contains(aside, "<dialog") {
		t.Error("as descrições dos efeitos ainda moram dentro da coluna do desktop")
	}
}

// O ESCOLHEDOR DE CONDIÇÃO é um diálogo do kit, e o toque aplica e fecha.
func TestTheParallelConditionPickerAppliesAndCloses(t *testing.T) {
	_, _, page := effectsInPlay(t)
	picker := regexp.MustCompile(`(?s)<dialog[^>]*id="condition-picker".*?</dialog>`).FindString(page)
	if picker == "" {
		t.Fatal("a aba não desenhou o diálogo de condições")
	}
	if !regexp.MustCompile(`command="close"[^>]*data-on:click="[^"]*/efeitos/condicao/abalado[^"]*tempestade=1`).MatchString(picker) {
		t.Error("o diálogo não aplica o Abalado e fecha no mesmo toque")
	}
	// O Caído já está aplicado: ele não volta como opção.
	if strings.Contains(picker, "/efeitos/condicao/caido") {
		t.Error("o diálogo oferece o Caído, que já está aplicado")
	}
}
