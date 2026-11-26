package engine

import (
	"strings"
	"testing"
)

// QUEM PODE RECEBER A CONDIÇÃO (ver GLOSSARY, **condição**): cada recusa diz o porquê, e a condição que passa
// pelas quatro perguntas entra.
func TestTheConditionMarkSaysWhyItIsRefused(t *testing.T) {
	mind := 10
	osteon := ConditionTarget{Name: "Osso", Immunities: ImmunitiesOfCreature("morto-vivo", &mind)}
	dodger := ConditionTarget{Name: "Bárbara", Effects: ItemEffects{
		Flags:       map[string]bool{NeverSurprisedFlag: true},
		FlagSources: map[string]string{NeverSurprisedFlag: "Esquiva Sobrenatural"},
	}}
	plain := ConditionTarget{Name: "Ana"}
	for _, c := range []struct {
		name, condition string
		who             ConditionTarget
		says            string
	}{
		{"a condição que o livro não tem", "voando", plain, "p394-395"},
		{"a que só a carga dá", "sobrecarregado", plain, "carga"},
		{"a de quem é imune ao tipo dela", "envenenado", osteon, "imune a efeitos de veneno"},
		{"a que um poder barra", "surpreendido", dodger, "Esquiva Sobrenatural"},
	} {
		err := MarkRefusal(c.condition, c.who)
		if err == nil || !strings.Contains(err.Error(), c.says) {
			t.Errorf("%s (%s em %s): a recusa veio %v; want uma que diga %q", c.name, c.condition, c.who.Name, err, c.says)
		}
	}
	// O CONTROLE: as mesmas criaturas recebem o que nada barra, e sem ele "recusou" valeria numa regra que recusa tudo.
	for _, who := range []ConditionTarget{osteon, dodger, plain} {
		if err := MarkRefusal("abalado", who); err != nil {
			t.Errorf("%s recebe Abalado, e veio %v", who.Name, err)
		}
	}
}
