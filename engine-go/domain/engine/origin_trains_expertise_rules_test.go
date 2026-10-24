package engine

import (
	"slices"
	"testing"
)

// ESCOLHER O BENEFÍCIO DE PERÍCIA DA ORIGEM TREINA A PERÍCIA (p85, ALE-490).
//
//	"Perícias. […] Você se torna treinado na perícia escolhida, representando
//	 aprendizado adquirido em sua vida pregressa."
//
// A escolha era gravada em `originChoices` e ninguém a lia para o treino: o
// `originModifiers` só colhia `modifiers` e `powerUid`, e o treino da ficha era
// a coluna `trained` de cada perícia, marcada à mão. Os 94 benefícios de perícia
// das 35 origens não treinavam nada.
//
// # O que se prende
//
// O PREDICADO: o benefício escolhido treina A perícia dele e nenhuma outra — o
// Acólito oferece Cura, Religião e Vontade, e escolher Cura não treina Religião.
// E a PROCEDÊNCIA: a ficha diz que quem treinou foi a origem, porque um treino
// que aparece sem a coluna marcada e sem dizer de onde veio é lido como defeito.
func TestTheChosenOriginSkillBenefitTrainsTheSkill(t *testing.T) {
	world := BookRuleset(primeBook(t))

	base := Character{
		Level: 1, Origin: "acolito", Wisdom: 2,
		Expertises: []CharacterExpertise{
			{Name: "Cura", Attribute: "wisdom"},
			{Name: "Religião", Attribute: "wisdom"},
		},
	}
	comCura := base
	comCura.OriginChoices = `["origin-acolito-pericia-cura"]`

	pericia := func(ch Character, nome string) ExpertiseBreakdown {
		for _, e := range world.ComputeSheet(ch, nil).Expertises {
			if e.Name == nome {
				return e
			}
		}
		t.Fatalf("a ficha não decompôs %s", nome)
		return ExpertiseBreakdown{}
	}

	// O controle: sem a escolha, Cura é só a Sabedoria. Se já viesse treinada,
	// o caso abaixo mediria um treino que não é da origem.
	if got := pericia(base, "Cura"); got.Total != 2 || got.Trained {
		t.Fatalf("o controle já estava errado: sem escolher o benefício, Cura vale %d "+
			"(treinada: %v), e devia valer 2 sem treino", got.Total, got.Trained)
	}

	cura := pericia(comCura, "Cura")
	// ½ nível 0 + Sabedoria 2 + treino 2 (p114: +2 do 1º ao 6º nível).
	if cura.Total != 4 || cura.Training != 2 || !cura.Trained {
		t.Errorf("Acólito que escolhe a perícia Cura ficou com Cura %d, treino %d, "+
			"treinada %v; a p85 diz que ele se torna treinado, e no 1º nível isso dá "+
			"4 (Sabedoria 2 + treino 2)", cura.Total, cura.Training, cura.Trained)
	}
	if !slices.Contains(cura.TrainedBy, "Origem: Acólito") {
		t.Errorf("o treino de Cura não diz de onde veio: TrainedBy = %q, e devia "+
			"nomear \"Origem: Acólito\"", cura.TrainedBy)
	}

	if got := pericia(comCura, "Religião"); got.Trained || got.Total != 2 {
		t.Errorf("escolher Cura treinou Religião (total %d, treinada %v): o benefício "+
			"treina a perícia DELE, e as outras da lista continuam de fora", got.Total, got.Trained)
	}
}

// A COLUNA MARCADA À MÃO CONTINUA VALENDO, e o treino não dobra.
//
// Treinar é um estado e não um bônus: a perícia treinada pela classe E pela
// origem tem +2, não +4. A coluna `trained` segue sendo a fonte das perícias da
// classe e das escolhas livres; a origem só se junta a ela.
func TestTrainingFromTheColumnAndFromTheOriginDoesNotStack(t *testing.T) {
	world := BookRuleset(primeBook(t))

	ch := Character{
		Level: 1, Origin: "acolito", Wisdom: 2,
		OriginChoices: `["origin-acolito-pericia-cura"]`,
		Expertises:    []CharacterExpertise{{Name: "Cura", Attribute: "wisdom", Trained: true}},
	}
	for _, e := range world.ComputeSheet(ch, nil).Expertises {
		if e.Name != "Cura" {
			continue
		}
		if e.Training != 2 || e.Total != 4 {
			t.Errorf("Cura treinada pela coluna E pela origem deu treino %d e total %d; "+
				"treinar não acumula, e o livro dá +2 no 1º nível (p114)", e.Training, e.Total)
		}
		return
	}
	t.Fatal("a ficha não decompôs Cura")
}
