package book

import (
	"strings"
	"testing"
)

// A ANTICRIATURA ESCOLHE UM TIPO (p335): "A arma é letal contra um tipo de criatura (ou uma raça de humanoides).
// […] role 1d6: 1) animal; 2) construto; 3) espírito; 4) monstro; 5) morto-vivo; 6) uma raça de humanoides." O
// mestre escolhe ao encantar, e o servidor recusa o encanto sem escolha, a escolha fora da lista e a escolha
// num encanto que não pede nenhuma.
func TestTheAnticreatureEnchantNeedsAKindFromTheBook(t *testing.T) {
	anti := []string{"encanto-anticriatura"}
	cases := []struct {
		name    string
		ids     []string
		choices map[string]string
		refusal string
	}{
		{"um tipo do livro", anti, map[string]string{"encanto-anticriatura": "morto-vivo"}, ""},
		{"uma raça de humanoides", anti, map[string]string{"encanto-anticriatura": "elfo"}, ""},
		{"sem escolha", anti, nil, "escolha o tipo de criatura"},
		{"humanoide sem raça", anti, map[string]string{"encanto-anticriatura": "humanoide"}, `"humanoide"`},
		{"planar não está no d6", anti, map[string]string{"encanto-anticriatura": "planar"}, `"planar"`},
		{"escolha num encanto que não pede", []string{"encanto-flamejante"},
			map[string]string{"encanto-flamejante": "animal"}, "não escolhe"},
		{"escolha sem o encanto", nil, map[string]string{"encanto-anticriatura": "animal"}, "não escolhe"},
	}
	for _, c := range cases {
		err := FitsEnchantChoices(c.ids, c.choices)
		switch {
		case c.refusal == "" && err != nil:
			t.Errorf("%s: recusou com %q", c.name, err)
		case c.refusal != "" && err == nil:
			t.Errorf("%s: aceitou, e devia recusar dizendo %q", c.name, c.refusal)
		case c.refusal != "" && !strings.Contains(err.Error(), c.refusal):
			t.Errorf("%s: recusou com %q, e a recusa devia dizer %q", c.name, err, c.refusal)
		}
	}
}

// O GASTO É OFERECIDO CONTRA QUALQUER ALVO, e a faixa do mestre diz se o alvo é do tipo escolhido (decisão do
// dono). A mesa sabe o TIPO do verbete do bestiário e do bloco do mestre, e a RAÇA do personagem; a linha
// digitada à mão não tem nenhum dos dois, e o personagem não tem tipo.
func TestTheTableSaysWhetherTheTargetIsOfTheChosenKind(t *testing.T) {
	cases := []struct {
		name   string
		chosen string
		target CreatureNature
		want   ChosenKindMatch
	}{
		{"morto-vivo contra morto-vivo", "morto-vivo", CreatureNature{Kind: "morto-vivo"}, TargetIsTheChosenKind},
		{"morto-vivo contra animal", "morto-vivo", CreatureNature{Kind: "animal"}, TargetIsNotTheChosenKind},
		{"morto-vivo contra a linha à mão", "morto-vivo", CreatureNature{}, TargetKindIsUnknown},
		{"morto-vivo contra um personagem", "morto-vivo", CreatureNature{Race: "humano"}, TargetKindIsUnknown},
		{"elfo contra o personagem elfo", "elfo", CreatureNature{Race: "elfo"}, TargetIsTheChosenKind},
		{"elfo contra o personagem humano", "elfo", CreatureNature{Race: "humano"}, TargetIsNotTheChosenKind},
		{"elfo contra o humanoide do bestiário", "elfo", CreatureNature{Kind: "humanoide"}, TargetKindIsUnknown},
		{"elfo contra o morto-vivo do bestiário", "elfo", CreatureNature{Kind: "morto-vivo"}, TargetIsNotTheChosenKind},
		{"nada escolhido", "", CreatureNature{Kind: "animal"}, TargetKindIsUnknown},
	}
	for _, c := range cases {
		if got := TargetOfTheChosenKind(c.chosen, c.target); got != c.want {
			t.Errorf("%s: a mesa leu %q, e devia ler %q", c.name, got, c.want)
		}
	}
}

// O NOME DA ESCOLHA na tela: o tipo como o bestiário o escreve, e a raça pelo nome do livro.
func TestTheChosenKindIsWrittenAsTheBookWritesIt(t *testing.T) {
	for chosen, want := range map[string]string{"morto-vivo": "morto-vivo", "espirito": "espírito", "elfo": "elfo"} {
		if got := ChosenKindName(chosen); got != want {
			t.Errorf("a escolha %q se escreve %q; quer %q", chosen, got, want)
		}
	}
}
