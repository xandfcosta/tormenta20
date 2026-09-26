package engine

import (
	"path/filepath"
	"testing"
)

// Os oito poderes gerais que declaravam bônus na descrição e não o aplicavam
// (ALE-399). Nenhum é incondicional, e eles se dividem em DUAS formas:
//
//   - três são MANOBRA, e aí o próprio alvo é o escopo — o livro diz "+2 em
//     testes de ataque para derrubar" (p125), e `maneuver:derrubar` já é isso.
//     Pendurar uma condição neles seria pedir à pessoa que ligue um interruptor
//     para uma circunstância que ela já escolheu ao usar a manobra.
//   - cinco são CIRCUNSTÂNCIA, e aí precisam do interruptor.
//
// A divisão é a regra, e é o que este caso prende: um bônus de manobra que
// exigisse opt-in valeria de menos, e um bônus de circunstância sem opt-in
// valeria sempre. Os dois erros são invisíveis num teste que só some números.
func TestTheManeuverPowersNeedNoSwitchAndTheSituationalOnesDo(t *testing.T) {
	world := BookRuleset(primeFromDump(t, filepath.Clean(
		filepath.Join(mustWd(t), "..", "..", "parity"))))

	efeitosDe := func(id string) ItemEffects {
		return ComputeItemEffects(world.ActiveItemsFor(
			Character{Level: 1, ClassPowers: `["` + id + `"]`}))
	}

	// ── manobra: vale sempre, e só na manobra que o livro nomeia ──
	for _, caso := range []struct{ id, manobra, outra string }{
		{"derrubar-aprimorado", "derrubar", "desarmar"},
		{"desarmar-aprimorado", "desarmar", "derrubar"},
		{"quebrar-aprimorado", "quebrar", "derrubar"},
	} {
		efeitos := efeitosDe(caso.id)
		if got := StatFor(efeitos, ModifierTarget{K: "maneuver", Name: caso.manobra}).Total; got != 2 {
			t.Errorf("%s: %s deu %d e o livro dá +2 em testes de ataque para %s (p125/p129)",
				caso.id, caso.manobra, got, caso.manobra)
		}
		// O CONTROLE: o bônus é da manobra NOMEADA, não de manobra em geral.
		if got := StatFor(efeitos, ModifierTarget{K: "maneuver", Name: caso.outra}).Total; got != 0 {
			t.Errorf("%s: subiu %s em %d, e o livro só dá para %s",
				caso.id, caso.outra, got, caso.manobra)
		}
	}

	// ── circunstância: desligado por padrão, ligado pelo termo ──
	for _, caso := range []struct {
		id     string
		alvo   ModifierTarget
		quero  int
		pagina string
	}{
		{"finta-aprimorada", ModifierTarget{K: "expertise", Name: "Enganação"}, 2, "p128"},
		{"estilo-de-arma-longa", ModifierTarget{K: "attack", Scope: "all"}, 2, "p125"},
		{"atraente", ModifierTarget{K: "expertiseByAttribute", Attribute: "charisma"}, 2, "p130"},
		{"lobo-solitario", ModifierTarget{K: "defense"}, 1, "p130"},
		{"torcida", ModifierTarget{K: "defense"}, 2, "p131"},
	} {
		desligado := efeitosDe(caso.id)
		if got := StatFor(desligado, caso.alvo).Total; got != 0 {
			t.Errorf("%s: %s vale %d DESLIGADO, e o livro o dá só na circunstância (%s)",
				caso.id, caso.alvo.K, got, caso.pagina)
		}
		ligados := map[string]bool{}
		for _, c := range desligado.Conditional {
			ligados[c.Term] = true
		}
		if len(ligados) == 0 {
			t.Errorf("%s: não ofereceu interruptor nenhum", caso.id)
			continue
		}
		if got := StatFor(ApplyActiveConditionals(desligado, ligados), caso.alvo).Total; got != caso.quero {
			t.Errorf("%s: %s deu %d ligado e o livro dá +%d (%s)",
				caso.id, caso.alvo.K, got, caso.quero, caso.pagina)
		}
	}
}
