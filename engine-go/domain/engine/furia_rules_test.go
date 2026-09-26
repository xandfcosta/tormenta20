package engine

import (
	"path/filepath"
	"testing"
)

// A FÚRIA DÁ ATAQUE E DANO, E NÃO RESISTÊNCIA (p41, ALE-388).
//
//	"Fúria. Você pode gastar 2 PM para invocar uma fúria selvagem. Você recebe
//	+2 em testes de ataque e rolagens de dano corpo a corpo, mas não pode fazer
//	nenhuma ação que exija calma e concentração […]. A cada cinco níveis, pode
//	gastar +1 PM para aumentar os bônus em +1."
//
// Dois alvos. O catálogo dava QUATRO: cada um dos quatro degraus concedia
// também `expertise:Fortitude` e `expertise:Vontade`, num total de oito
// modificadores que o livro não imprime em lugar nenhum.
//
// # De onde o erro veio, e por que o schema nunca o pegaria
//
// A p41 tem, a poucas linhas do texto da Fúria, a linha "Perícias. Fortitude
// (Con) e Luta (For) […] Sobrevivência (Sab) e Vontade (Sab)" — que é a lista
// de perícias TREINADAS do bárbaro. A transcrição leu uma como a outra.
//
// Schema valida FORMA: dois modificadores a mais, bem formados, são um schema
// válido. Só conferir contra a página pega isto — e quem o pegou foi o
// `scripts/audit-classes.py`, que exige o ALVO de cada modificador ser nomeado
// no texto do livro.
//
// # O controle está no mesmo caso
//
// Ele afirma que ataque e dano CONTINUAM subindo. Sem isso, apagar a Fúria
// inteira passaria verde.
func TestFuryGivesAttackAndDamageAndNotResistances(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	world := BookRuleset(primeFromDump(t, dir))

	// 10º nível: +2 de base mais um degrau a cada cinco níveis = +3, e os dois
	// são `morale`, que não empilha.
	ch := Character{
		Level:   10,
		Classes: []CharacterClass{{ClassName: "Bárbaro", Level: 10}},
		Expertises: []CharacterExpertise{
			{Name: "Fortitude", Attribute: "constitution"},
			{Name: "Vontade", Attribute: "wisdom"},
		},
	}
	efeitos := ComputeItemEffects(world.ActiveItemsFor(ch))
	emFuria := ApplyActiveConditionals(efeitos, map[string]bool{FlagGroupID("furia"): true})

	subiu := func(alvo ModifierTarget) int {
		return StatFor(emFuria, alvo).Total - StatFor(efeitos, alvo).Total
	}

	// O CONTROLE: a Fúria tem de estar fazendo alguma coisa.
	if got := subiu(ModifierTarget{K: "attack", Scope: "all"}); got != 3 {
		t.Fatalf("a Fúria subiu o ataque em %d e a p41 dá +3 no 10º nível — "+
			"o caso abaixo estaria medindo uma Fúria que não entrou", got)
	}
	if got := subiu(ModifierTarget{K: "damage", Scope: "all"}); got != 3 {
		t.Errorf("a Fúria subiu o dano em %d e a p41 dá o MESMO +3", got)
	}

	for _, pericia := range []string{"Fortitude", "Vontade"} {
		if got := subiu(ModifierTarget{K: "expertise", Name: pericia}); got != 0 {
			t.Errorf("a Fúria subiu %s em %d, e a p41 não dá resistência nenhuma.\n"+
				"A lista de perícias TREINADAS do bárbaro está a poucas linhas do texto "+
				"da Fúria na mesma página — foi de lá que os oito modificadores vieram.",
				pericia, got)
		}
	}
}
