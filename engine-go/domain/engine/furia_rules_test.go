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

	// 10º nível, e a conta é +2: o degrau que a Tabela 1-6 abre no 6º é PAGO ao
	// entrar na postura, e o motor não sabe de pagamento nenhum — ele vê o livro
	// puro. O degrau chega como efeito ativo, e quem o prende é o
	// `TestThePaidDegreeRaisesTheStanceBonus`. Até a ALE-423 o catálogo o dava
	// de graça por nível, e este caso media +3 sem ninguém ter pagado.
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
	if got := subiu(ModifierTarget{K: "attack", Scope: "all"}); got != 2 {
		t.Fatalf("a Fúria subiu o ataque em %d e a p41 dá +2 de base — "+
			"o caso abaixo estaria medindo uma Fúria que não entrou", got)
	}
	if got := subiu(ModifierTarget{K: "damage", Scope: "all"}); got != 2 {
		t.Errorf("a Fúria subiu o dano em %d e a p41 dá o MESMO +2", got)
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
