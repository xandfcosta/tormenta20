package character

import (
	"testing"

	"t20engine/domain/engine"
)

// UMA CIRCUNSTÂNCIA, UM INTERRUPTOR (ALE-399).
//
// O agrupamento de situacionais só existia por FLAG: um condicional de
// `context` virava um grupo por MODIFICADOR. Isso nunca apareceu porque nenhuma
// entrada do catálogo tinha dois `context` com a mesma nota — até a Torcida,
// que dá "+2 em testes de perícia e Defesa" (p131) e portanto são dois.
//
// O resultado na tela eram DUAS linhas idênticas, "com torcida a seu favor /
// Torcida" repetida, sem nada que as distinguisse. Nenhum limiar pega isso: foi
// o olho, e é por isso que o guarda nasce aqui.
//
// O conserto não foi mexer no agrupamento e sim no DADO: circunstância com mais
// de um modificador usa `flagOn` com flag compartilhada, que é o mecanismo que
// o `SituationalGroupsOf` já tinha para desenhar uma linha só.
func TestOneCircumstanceOffersOneSwitch(t *testing.T) {
	daTorcida := func(condicao *engine.ModifierCondition) []engine.ConditionalEffect {
		fora := []engine.ConditionalEffect{}
		for _, alvo := range []engine.ModifierTarget{{K: "expertiseAll"}, {K: "defense"}} {
			c := engine.ConditionalEffect{
				Term: "torcida::" + alvo.K, SourceID: "torcida", Source: "Torcida",
				Amount: 2, Target: alvo, Note: "com torcida a seu favor",
			}
			if condicao.C == "flagOn" {
				c.Flag = condicao.Flag
			}
			fora = append(fora, c)
		}
		return fora
	}

	comFlag := SituationalGroupsOf(daTorcida(&engine.ModifierCondition{C: "flagOn", Flag: "torcida"}))
	if len(comFlag) != 1 {
		t.Errorf("a Torcida ofereceu %d interruptores e a circunstância é UMA — a tela "+
			"desenha uma linha por grupo, então %d grupos são %d linhas idênticas",
			len(comFlag), len(comFlag), len(comFlag))
	}
	if len(comFlag) == 1 && len(comFlag[0].Members) != 2 {
		t.Errorf("o interruptor da Torcida acende %d modificadores e o livro dá DOIS "+
			"(perícia e Defesa, p131) — meio bônus é pior que nenhum",
			len(comFlag[0].Members))
	}

	// O CONTROLE, e ele é o defeito original: sem flag, cada modificador vira um
	// grupo. Se algum dia o agrupamento passar a colapsar por nota, este caso
	// avisa que o dado pode voltar a ser `context`.
	semFlag := SituationalGroupsOf(daTorcida(&engine.ModifierCondition{C: "context"}))
	if len(semFlag) != 2 {
		t.Errorf("sem flag saíram %d grupos; o agrupamento mudou, e aí a escolha de "+
			"`flagOn` para circunstância de dois modificadores merece ser revista",
			len(semFlag))
	}
}
