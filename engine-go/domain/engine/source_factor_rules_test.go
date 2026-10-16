package engine

import (
	"path/filepath"
	"testing"
)

// O FATOR QUE MULTIPLICA UMA FONTE, e a Fúria Titânica é quem o pede
// (ALE-429, p42).
//
//	"Fúria Titânica. No 20º nível, o bônus que você recebe nos testes de ataque
//	 e rolagens de dano quando usa Fúria é dobrado. Por exemplo, se gastar 5 PM,
//	 em vez de um bônus de +5, recebe um bônus de +10." (p42)
//
// O EXEMPLO TRABALHADO é o caso de baixo, e ele separa duas leituras da mesma
// frase: "dobra o bônus" podia ser o total do ALVO — e aí Luta, Força e arma
// dobrariam junto — ou a PARCELA da Fúria. O livro responde com números: 5 PM
// gastos dão +10, e não o dobro do ataque inteiro.
func TestTheTitanicFuryDoublesTheFuryParcelAndNothingElse(t *testing.T) {
	world := BookRuleset(primeFromDump(t, filepath.Clean(
		filepath.Join(mustWd(t), "..", "..", "parity"))))
	furiaLigada := map[string]bool{FlagGroupID("furia"): true}
	ataqueDe := func(ch Character, ligados map[string]bool) int {
		efeitos := ApplyActiveConditionals(ComputeItemEffects(world.ActiveItemsFor(ch)), ligados)
		return StatFor(efeitos, ModifierTarget{K: "attack", Scope: "all"}).Total
	}
	barbaro := func(level int) Character {
		return Character{
			Level: level, Strength: 4,
			Classes: []CharacterClass{{ClassName: "Bárbaro", Level: level}},
		}
	}

	// SEM DEGRAU: o +2 da base (p41) dobrado é +4.
	semTitanica := ataqueDe(barbaro(19), furiaLigada)
	comTitanica := ataqueDe(barbaro(20), furiaLigada)
	if semTitanica != 2 {
		t.Fatalf("o bárbaro de 19º tem +%d de ataque em fúria, e a p41 dá +2 — "+
			"a bancada mediu outra coisa", semTitanica)
	}
	if comTitanica != 4 {
		t.Errorf("o bárbaro de 20º tem +%d de ataque em fúria, e a Fúria Titânica "+
			"dobra o +2 da base: +4", comTitanica)
	}

	// O EXEMPLO DO LIVRO: 5 PM gastos são a base mais três degraus, e o degrau
	// pago mora num efeito ativo com a chave do PODER — é a parcela que a
	// Titânica dobra. +5 vira +10, e não o dobro do ataque inteiro.
	comDegrau := barbaro(20)
	comDegrau.ActiveEffects = []ActiveEffectRow{{
		CatalogID: "class.barbaro.furia", Scope: "scene",
		Modifiers: `[{"target":{"k":"attack","scope":"all"},"amount":5,"bonusType":"morale"}]`,
	}}
	if got := ataqueDe(comDegrau, furiaLigada); got != 10 {
		t.Errorf("com 5 PM gastos o ataque ficou em +%d, e o exemplo da p42 dá +10", got)
	}

	// E NADA ALÉM DA FÚRIA: a mesma ficha com a fúria DESLIGADA não muda, e o
	// que a Titânica acrescenta é exatamente a parcela dobrada. Sem este
	// controle, um fator que multiplicasse o total do alvo passaria igual.
	semFuria := ataqueDe(barbaro(20), map[string]bool{})
	if semFuria != 0 {
		t.Errorf("fora da fúria o bárbaro de 20º tem +%d de bônus de ataque, e a "+
			"Titânica só vale \"quando usa Fúria\" (p42)", semFuria)
	}
}
