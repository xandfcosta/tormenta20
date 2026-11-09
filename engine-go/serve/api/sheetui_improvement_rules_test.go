package api

import (
	"strings"
	"testing"
)

// O QUE O LIVRO EXIGE DE CADA MELHORIA (p164-166). O gesto regrava a lista inteira, então o
// pré-requisito se confere contra o PEDIDO: tirar a Cruel e manter a Atroz também é recusado.

func TestAnImprovementNeedsWhatTheBookRequires(t *testing.T) {
	for _, c := range []struct {
		name, weapon, body, refusal string
	}{
		{"Atroz sem Cruel", "espada-longa", `{"item_improvements":["melhoria-atroz"]}`, "Cruel"},
		{"Pungente sem Certeira", "espada-longa", `{"item_improvements":["melhoria-pungente"]}`, "Certeira"},
		{"Sob Medida sem Ajustada", "couraca", `{"item_improvements":["melhoria-sob-medida"]}`, "Ajustada"},
		{"Harmonizada sozinha", "espada-longa", `{"item_improvements":["melhoria-harmonizada-arma"]}`, "outra melhoria"},
		{"Maciça e Precisa", "espada-longa", `{"item_improvements":["melhoria-macica","melhoria-precisa"]}`, "não pode"},
		{"Delicada e Reforçada", "armadura-completa", `{"item_improvements":["melhoria-delicada","melhoria-reforcada"]}`, "não pode"},
		{"Atroz com Cruel", "espada-longa", `{"item_improvements":["melhoria-cruel","melhoria-atroz"]}`, ""},
		{"Pungente com Certeira", "espada-longa", `{"item_improvements":["melhoria-certeira","melhoria-pungente"]}`, ""},
		{"Harmonizada com outra", "espada-longa", `{"item_improvements":["melhoria-cruel","melhoria-harmonizada-arma"]}`, ""},
		{"Harmonizada com o material", "espada-longa", `{"item_improvements":["melhoria-harmonizada-arma"],"item_material":"material-adamante"}`, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := fighterFixture(t)
			item := itemSemeia(t, f, id, c.weapon, c.weapon, "")
			refusal := improvements(t, f, id, item, c.body)
			if c.refusal == "" {
				if refusal != "" {
					t.Errorf("o pedido cabe, e foi recusado: %q", refusal)
				}
				return
			}
			if !strings.Contains(refusal, c.refusal) {
				t.Errorf("a recusa devia falar de %q, e veio %q", c.refusal, refusal)
			}
			if saved := itemImprovements(t, f, item); saved != "[]" {
				t.Errorf("a recusa gravou assim mesmo: %q", saved)
			}
		})
	}
}

func TestTakingOffThePrerequisiteIsRefusedWhileTheOtherStays(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	if refusal := improvements(t, f, id, sword, `{"item_improvements":["melhoria-cruel","melhoria-atroz"]}`); refusal != "" {
		t.Fatalf("o controle já estava errado: %q", refusal)
	}
	if refusal := improvements(t, f, id, sword, `{"item_improvements":["melhoria-atroz"]}`); !strings.Contains(refusal, "Cruel") {
		t.Errorf("tirar a Cruel e manter a Atroz devia ser recusado, e veio %q", refusal)
	}
}

// As que só valem em certas peças: a Mira em arma de disparo menos a funda, e a Delicada e a Selada
// em armadura pesada. A lista não as oferece, e o servidor recusa o pedido à mão.
func TestSomeImprovementsOnlyFitSomeItems(t *testing.T) {
	for _, c := range []struct {
		name, item, improvement string
		fits                    bool
	}{
		{"Mira no arco", "arco-curto", "melhoria-mira-telescopica", true},
		{"Mira na besta", "besta-leve", "melhoria-mira-telescopica", true},
		{"Mira na funda", "funda", "melhoria-mira-telescopica", false},
		{"Mira na espada", "espada-longa", "melhoria-mira-telescopica", false},
		{"Mira na azagaia", "azagaia", "melhoria-mira-telescopica", false},
		{"Delicada em armadura pesada", "armadura-completa", "melhoria-delicada", true},
		{"Delicada em armadura leve", "couraca", "melhoria-delicada", false},
		{"Selada em armadura pesada", "armadura-completa", "melhoria-selada", true},
		{"Selada em armadura leve", "couraca", "melhoria-selada", false},
		{"Aprimorado na luva de pelica", "luva-pelica", "melhoria-aprimorado", true},
		{"Aprimorado na maleta", "maleta-medicamentos", "melhoria-aprimorado", true},
		{"Aprimorado na gazua", "gazua", "melhoria-aprimorado", true},
		{"Aprimorado no estojo", "estojo-de-disfarces", "melhoria-aprimorado", true},
		{"Aprimorado no equipamento de viagem", "equipamento-de-viagem", "melhoria-aprimorado", true},
		{"Aprimorado nos instrumentos de ofício", "instrumentos-de-oficio", "melhoria-aprimorado", true},
		{"Aprimorado no instrumento musical", "instrumento-musical", "melhoria-aprimorado", true},
		{"Aprimorado na sela", "sela", "melhoria-aprimorado", true},
		{"Aprimorado no traje da corte", "traje-corte", "melhoria-aprimorado", true},
		{"Aprimorado na coleção de livros", "coleccao-de-livros", "melhoria-aprimorado", true},
		{"Aprimorado no manto élfico", "manto-elfico", "melhoria-aprimorado", true},
		{"Aprimorado no traje de viajante, que não mexe em perícia", "traje-viajante", "melhoria-aprimorado", false},
		{"Aprimorado nas botas reforçadas", "botas-reforcadas", "melhoria-aprimorado", false},
		{"Aprimorado na corda", "corda", "melhoria-aprimorado", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := fighterFixture(t)
			item := itemSemeia(t, f, id, c.item, c.item, "")
			refusal := improvements(t, f, id, item, `{"item_improvements":["`+c.improvement+`"]}`)
			if (refusal == "") != c.fits {
				t.Errorf("cabe = %v, e a resposta foi %q", c.fits, refusal)
			}
			name := map[string]string{
				"melhoria-mira-telescopica": "Mira telescópica", "melhoria-delicada": "Delicada", "melhoria-selada": "Selada",
				"melhoria-aprimorado": "Aprimorado",
			}[c.improvement]
			dialog := improvementDialog(t, f, id, item)
			if got := strings.Contains(dialog, `aria-label="`+name+`"`); got != c.fits {
				t.Errorf("a lista oferece %s = %v, want %v", name, got, c.fits)
			}
		})
	}
}

// A que já está no item continua na lista mesmo sem caber, para poder sair.
func TestAnImprovementThatNoLongerFitsStaysInTheListToComeOff(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	if _, err := f.s.db.ExecContext(t.Context(), "UPDATE character_items SET improvements = ? WHERE id = ?", `["melhoria-mira-telescopica"]`, sword); err != nil {
		t.Fatal(err)
	}
	if dialog := improvementDialog(t, f, id, sword); !strings.Contains(dialog, `aria-label="Mira telescópica"`) {
		t.Error("a melhoria que já está no item sumiu da lista")
	}
	if refusal := improvements(t, f, id, sword, `{"item_improvements":[]}`); refusal != "" {
		t.Errorf("tirá-la foi recusado: %q", refusal)
	}
}

// A lista diz em que perícia o item soma, e não "sem efeito mecânico".
func TestTheImprovedLineSaysWhichSkill(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, "maleta-medicamentos", "Maleta de medicamentos", "")
	if dialog := improvementDialog(t, f, id, item); !strings.Contains(dialog, "+1 em Cura") {
		t.Error("a Aprimorado da maleta não diz +1 em Cura")
	}
}
