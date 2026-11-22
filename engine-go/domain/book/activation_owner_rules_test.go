package book

import (
	"testing"

	"t20engine/domain/engine"
)

// O DONO DE UMA ATIVAÇÃO se diz num lugar só (decisão do dono): pelo id, ou pelo NOME dentro da mesma procedência.
// O nome é a ponte entre catálogos que grafam o mesmo poder de jeitos diferentes, mas ele não atravessa classes:
// o Esgrimista do guerreiro e o do bucaneiro são habilidades diferentes com o mesmo nome (p48, p68).
func TestTheOwnerLinkStaysInsideTheClass(t *testing.T) {
	for _, c := range []struct {
		name         string
		power        PowerRef
		activationID string
		owns         bool
	}{
		{"o Esgrimista do guerreiro é dele", PowerRef{"class.guerreiro.esgrimista", "Esgrimista"}, "class.guerreiro.esgrimista", true},
		{"o Esgrimista do guerreiro não é o do bucaneiro", PowerRef{"class.guerreiro.esgrimista", "Esgrimista"}, "class.bucaneiro.esgrimista", false},
		{"o Ambidestria do caçador não é o do guerreiro", PowerRef{"class.cacador.ambidestria", "Ambidestria"}, "class.guerreiro.ambidestria", false},
		{"a Coragem se liga ao poder pelo nome, na mesma classe", PowerRef{"class.paladino.julgamento-divino-coragem", "Julgamento Divino: Coragem"}, "class.paladino.julgamento-coragem", true},
		{"a Coragem Total divina é a de cada deus que a concede", PowerRef{"divino.coragem-total", "Coragem Total"}, "deus.khalmyr.coragem-total", true},
		{"a Coragem Total divina, de outro deus", PowerRef{"divino.coragem-total", "Coragem Total"}, "deus.arsenal.coragem-total", true},
		{"a Engenhosidade do kliren é a da raça", PowerRef{"kliren-engenhosidade", "Engenhosidade"}, "race.kliren.engenhosidade", true},
		{"a Engenhosidade do kliren não é a do inventor", PowerRef{"kliren-engenhosidade", "Engenhosidade"}, "class.inventor.engenhosidade", false},
		{"a Engenhosidade do inventor não é a do kliren", PowerRef{"class.inventor.engenhosidade", "Engenhosidade"}, "race.kliren.engenhosidade", false},
		{"a linha de degrau é da postura", PowerRef{"class.bardo.inspiracao-2", "Inspiração +2"}, "class.bardo.inspiracao", true},
	} {
		spec := ActivationOf(c.activationID, "")
		if spec == nil {
			t.Fatalf("%s: %s não está no registro", c.name, c.activationID)
		}
		if got := OwnerIndexOf([]PowerRef{c.power}).Owns(*spec); got != c.owns {
			t.Errorf("%s: dono = %v; want %v", c.name, got, c.owns)
		}
	}
}

// A IDA respeita a mesma regra: o poder de uma classe que só se liga pelo nome acha a ativação DA CLASSE dele, e
// não a de outra com o mesmo nome.
func TestTheActivationOfAPowerStaysInsideTheClass(t *testing.T) {
	for _, c := range []struct{ powerID, want string }{
		{"class.guerreiro.outra-grafia", "class.guerreiro.valentao"},
		{"class.lutador.outra-grafia", "class.lutador.valentao"},
	} {
		if spec := ActivationOf(c.powerID, "Valentão"); spec == nil || spec.ID != c.want {
			t.Errorf("o Valentão de %s achou %v; want %s", c.powerID, spec, c.want)
		}
	}
}

// A CLASSE DA ATIVAÇÃO segue a mesma ligação: a Engenhosidade do kliren é de raça, e o fator de uma classe não a
// alcança — antes ela herdava "Inventor" pelo nome.
func TestARaceAbilityIsNotPricedAsAClassOne(t *testing.T) {
	kliren := *ActivationOf("race.kliren.engenhosidade", "")
	halfForInventors := engine.AbilityCostRules{Factors: map[string]engine.Ratio{"Inventor": {Num: 1, Den: 2}}}
	if got, catalog := UsePrice(kliren, halfForInventors, Use{}), ActivationPm(kliren); got != catalog {
		t.Errorf("a Engenhosidade do kliren custa %d sob a metade do inventor; want o do catálogo, %d", got, catalog)
	}
}
