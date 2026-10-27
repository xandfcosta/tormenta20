package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"t20engine/domain/sheet"
)

// O QUE OS PODERES DO CAÇADOR ACRESCENTAM À MARCA DA PRESA (p51, ALE-509).
//
//	"Espreitar. Quando usa a habilidade Marca da Presa, você recebe um bônus de
//	+1 em testes de perícia contra a criatura marcada. Esse bônus aumenta em +1
//	para cada PM adicional gasto na habilidade e também dobra com a habilidade
//	Inimigo."
//
// O Ponto Fraco dá +2 na margem de ameaça contra a marcada, e o Inimigo de
// (Criatura) dobra os dados da Marca quando a presa é do tipo escolhido. Tudo
// entra no mesmo efeito da Marca e no mesmo interruptor; o Inimigo é uma caixa
// no gesto, porque o app não guarda quem foi marcado (decisão do dono).

func hunterWith(t *testing.T, level int64, powers string) (sceneFixture, int64) {
	t.Helper()
	f, id := hunterFixture(t, level)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	seedPericia(t, f.s, id, "Percepção", "wisdom", false)
	choiceCom(t, f, id, powers, "[]")
	return f, id
}

// marksThePrey usa a Marca e liga o interruptor da criatura marcada.
func marksThePrey(t *testing.T, f sceneFixture, id int64, body string) {
	t.Helper()
	if refusal := powerCommand(t, f, id, "usa/class.cacador.marca-da-presa-1d4", body); refusal != "" {
		t.Fatalf("marcar com %s foi recusado: %q", body, refusal)
	}
	key := preyMarkSwitch(t, f, id)
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao?tab=abilities", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target,
		fmt.Sprintf(`{"conditional":%q}`, key)).Body.String()); refusal != "" {
		t.Fatalf("ligar %q foi recusado: %q", key, refusal)
	}
}

func perceptionOf(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler a ficha %d: %v", id, err)
	}
	computed, err := sheet.LoadAndCompute(context.Background(), f.s.queries, f.s.catalogs, row)
	if err != nil {
		t.Fatalf("computar a ficha %d: %v", id, err)
	}
	for _, ex := range computed.Expertises {
		if ex.Name == "Percepção" {
			return ex.Total
		}
	}
	t.Fatal("a perícia Percepção não apareceu na ficha computada")
	return 0
}

func critRangeOfTheFirstCard(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	cards := dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals))
	if len(cards) == 0 {
		t.Fatal("a ficha não tem carta de arma")
	}
	return cards[0].CritRange
}

// O ESPREITAR É DA MARCA: com 1 degrau, +2 em perícia contra a marcada, e o PM
// é só o da Marca.
func TestStalkingRidesThePreyMarkWithOnePlusTheSteps(t *testing.T) {
	f, id := hunterWith(t, 5, `["class.cacador.espreitar"]`)
	before := perceptionOf(t, f, id)
	mp := pm(t, f, id)
	marksThePrey(t, f, id, `{"stance_degrees":1}`)
	if got := perceptionOf(t, f, id) - before; got != 2 {
		t.Errorf("a Marca com 1 degrau e o Espreitar deram %+d em Percepção; a p51 diz +1 e +1 pelo PM adicional", got)
	}
	if spent := mp - pm(t, f, id); spent != 2 {
		t.Errorf("a Marca com 1 degrau cobrou %d PM; o Espreitar não cobra à parte", spent)
	}
	if refusal := powerCommand(t, f, id, "usa/class.cacador.espreitar", `{"stance_degrees":0}`); refusal == "" {
		t.Error("o Espreitar ainda é um gesto à parte, e o livro o faz parte da Marca")
	}
}

// O PONTO FRACO alarga a margem de ameaça em 2 contra a marcada.
func TestWeakSpotWidensTheThreatRangeAgainstTheMarkedPrey(t *testing.T) {
	f, id := hunterWith(t, 5, `["class.cacador.ponto-fraco"]`)
	before := critRangeOfTheFirstCard(t, f, id)
	marksThePrey(t, f, id, `{"stance_degrees":0}`)
	if got := critRangeOfTheFirstCard(t, f, id); got != before-2 {
		t.Errorf("a margem da espada longa com o Ponto Fraco é %d; antes era %d, e a p51 dá +2", got, before)
	}
}

// O INIMIGO DOBRA os dados da Marca, o Espreitar e o Ponto Fraco quando a
// caixa diz que a presa é do tipo escolhido.
func TestTheFavoredEnemyDoublesTheMarkWhenThePreyIsOfItsKind(t *testing.T) {
	f, id := hunterWith(t, 5,
		`["class.cacador.espreitar","class.cacador.ponto-fraco","class.cacador.inimigo-de-criatura"]`)
	perception, crit := perceptionOf(t, f, id), critRangeOfTheFirstCard(t, f, id)
	marksThePrey(t, f, id, `{"stance_degrees":1,"prey_favored":true}`)
	if extra := extraDiceOnTheFirstCard(t, f, id); len(extra) != 1 || extra[0].Dice != "2d8" {
		t.Errorf("a Marca com 1 degrau contra a presa do Inimigo pôs %+v; o 1d8 dobra para 2d8", extra)
	}
	if got := perceptionOf(t, f, id) - perception; got != 4 {
		t.Errorf("o Espreitar dobrado deu %+d em Percepção; want +4", got)
	}
	if got := critRangeOfTheFirstCard(t, f, id); got != crit-4 {
		t.Errorf("o Ponto Fraco dobrado deixou a margem em %d; antes %d, want 4 a mais", got, crit)
	}
}

// SEM O INIMIGO, a caixa não dobra nada: quem não tem o poder não o usa.
func TestTheFavoredEnemyBoxDoesNothingWithoutThePower(t *testing.T) {
	f, id := hunterWith(t, 5, `[]`)
	if refusal := powerCommand(t, f, id, "usa/class.cacador.marca-da-presa-1d4", `{"stance_degrees":1,"prey_favored":true}`); refusal == "" {
		t.Error("a caixa do Inimigo passou para quem não tem o poder")
	}
}
