package catalog

import (
	"regexp"
	"testing"
)

// spellForSustainsSweep é o que a varredura do aprimoramento que sustenta precisa: o nome, para a falha dizer
// QUAL magia, a duração, e os aprimoramentos crus.
type spellForSustainsSweep struct {
	Name     string `json:"name"`
	Duration string `json:"duration"`
	Augments []struct {
		Description string `json:"description"`
		Sustains    *bool  `json:"sustains"`
	} `json:"augments"`
}

var (
	becomesSustained = regexp.MustCompile(`duração para sustentada`)
	durationChanges  = regexp.MustCompile(`duração para`)
)

// TestEveryAugmentThatChangesWhetherTheSpellIsSustainedSaysSo amarra a marca `sustains` ao texto do livro.
//
// O Conjurar abre a manutenção quando a magia conjurada é sustentada (p227), e quem diz se ela é, além da
// duração do verbete, é o aprimoramento escolhido: "Muda a duração para sustentada" põe a Invisibilidade na
// manutenção, e o "Muda a duração para cena" da Oração a tira. A marca é a ÚNICA leitura disso, e o texto é
// prosa: sem esta varredura, um aprimoramento transcrito sem a marca conjuraria a magia sem cobrar a
// manutenção, e nada reprovaria.
//
// A marca fala SÓ de entrar e sair da sustentada. As outras mudanças de duração ("para um dia", "para 1
// semana") ficam sem ela, porque nenhuma conta as lê.
func TestEveryAugmentThatChangesWhetherTheSpellIsSustainedSaysSo(t *testing.T) {
	spells := decodeResource[map[string]spellForSustainsSweep](t, "spells")
	if len(spells) == 0 {
		t.Fatal("nenhuma magia no catálogo: não há o que varrer, e verde aqui não valeria nada")
	}
	entering, leaving := 0, 0
	for id, m := range spells {
		for i, a := range m.Augments {
			var want *bool
			switch {
			case becomesSustained.MatchString(a.Description):
				want = new(true)
				entering++
			case m.Duration == "sustentada" && durationChanges.MatchString(a.Description):
				want = new(false)
				leaving++
			}
			if !samePointerBool(a.Sustains, want) {
				t.Errorf("%s (%s) aprimoramento %d: o texto pede `sustains` %s e o campo diz %s: %q",
					m.Name, id, i, pointerBoolText(want), pointerBoolText(a.Sustains), a.Description)
			}
		}
	}
	// O CONTROLE: a regex casa os dois sentidos. Uma regex que nunca casasse deixaria a varredura verde com
	// todas as marcas ausentes.
	if entering == 0 || leaving == 0 {
		t.Fatalf("a varredura achou %d aprimoramentos que tornam a magia sustentada e %d que a tiram: a regex parou de casar o texto",
			entering, leaving)
	}
}

func samePointerBool(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func pointerBoolText(b *bool) string {
	if b == nil {
		return "ausente"
	}
	if *b {
		return "true"
	}
	return "false"
}
