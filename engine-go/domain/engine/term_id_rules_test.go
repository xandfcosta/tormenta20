package engine

import (
	"strings"
	"testing"
)

// A FORMA DO ENDEREÇO DE UM TERMO ESTÁ NO BANCO, E POR ISSO ELA NÃO MUDA SOZINHA.
//
// Ela é gravada em DOIS lugares: `character_conditionals.conditionalId`, que é
// o opt-in do jogador, e `campaign_silences.term`, que é o que o mestre
// desligou. Mudar a fórmula não quebra compilação, não loga e não falha — o id
// gravado deixa de casar com o gerado, e o que o jogador ligou volta DESLIGADO
// enquanto o que o mestre calou volta a valer.
//
// # O esperado aqui é um LITERAL, e isso é o ponto
//
// Um caso que derivasse o esperado da própria função prova que a linguagem
// devolve valores e nada sobre a FORMA: mudar a fórmula o manteria verde. Este
// escreve a string à mão.
//
// > Aqui morava o `TestTheConditionalIdKeepsItsExactShape`, que prendia a
// > fórmula do `ConditionalID` (`fonte::alvo::nota::valor::tipo`). Ele dizia,
// > com razão, que trocá-la exigiria uma migração — e a troca aconteceu na
// > ALE-387, com a migração 00019 e com a MEDIÇÃO que a autorizou: a tabela
// > tinha ZERO linhas. O que sobrou dos dois ids é um só, e ele não carrega o
// > valor, que era a fragilidade do antigo.
func TestTheTermAddressKeepsItsExactShape(t *testing.T) {
	const gravado = "armadura-completa::displacement:armor::::"

	got := TermID("armadura-completa", Modifier{
		Target:    ModifierTarget{K: "displacement", Scope: "armor"},
		Amount:    -3,
		BonusType: "untyped",
		Note:      "nota qualquer",
	})
	if got != gravado {
		t.Fatalf("a fórmula do TermID MUDOU.\n  gerado:  %s\n  gravado: %s\n\n"+
			"Esta string vive em `character_conditionals.conditionalId` e em "+
			"`campaign_silences.term`. Mudá-la faz o opt-in do jogador voltar desligado "+
			"e o silêncio do mestre parar de valer — sem erro em lugar nenhum.\n"+
			"Se a mudança é mesmo necessária, ela vem com uma migração que reescreve as "+
			"linhas, nunca sozinha.", got, gravado)
	}
}

// AS DUAS FAMÍLIAS DE CHAVE NÃO PODEM COLIDIR.
//
// A mesma coluna guarda o endereço de um TERMO e o de um GRUPO de flag, e a
// separação é estrutural: todo endereço de termo tem `::`, e o de grupo nunca
// tem. Se ela deixasse de valer, ligar um grupo poderia ligar um termo
// qualquer — e o contrário.
func TestTheTermAndGroupKeysCannotCollide(t *testing.T) {
	const gravado = "flag:furia"
	if got := FlagGroupID("furia"); got != gravado {
		t.Fatalf("a fórmula do FlagGroupID mudou: %q, gravado %q", got, gravado)
	}

	termo := TermID("qualquer-fonte", Modifier{Target: ModifierTarget{K: "attack"}})
	if !strings.Contains(termo, "::") {
		t.Errorf("o endereço de termo %q não tem `::` — é ele que o separa do grupo", termo)
	}
	if strings.Contains(FlagGroupID("furia"), "::") {
		t.Error("o endereço de grupo ganhou `::` e passou a poder colidir com um termo")
	}
	// E o controle: uma FONTE cujo nome contivesse o prefixo do grupo não pode
	// produzir um endereço que se pareça com um.
	if FlagGroupID("furia") == TermID("flag:furia", Modifier{}) {
		t.Error("uma fonte chamada `flag:furia` produziu o endereço do grupo dela")
	}
}

// UM INTERRUPTOR DE GRUPO LIGA A REGRA INTEIRA (ALE-387).
//
// A tela oferece UM interruptor para os condicionais que dividem uma flag — "um
// item caseiro com três modificadores é uma coisa só na mesa" —, e o crachá ao
// lado dele diz quantos são. Enquanto a chave era o endereço do PRIMEIRO membro,
// ligar o grupo dobrava um modificador e deixava os outros fora: a ficha somava
// um terço de uma regra, com a tela dizendo "3 mods".
//
// # Por que isto não era um defeito VIVO, e passou a poder ser
//
// As únicas flags do livro são `furia` e `inspiracao`, e as duas são POSTURAS —
// que a tela de situacionais exclui de propósito, porque o interruptor delas
// mora nos Poderes. Medido no catálogo: nenhum outro grupo de flag existe.
//
// O que mudou é a emenda de campanha: um mestre que conceda um item com dois
// modificadores `flagOn` cai exatamente neste caminho. A fatia que abriu a porta
// é a que tem de fechá-la.
func TestAGroupSwitchFoldsEveryModifierOfTheFlag(t *testing.T) {
	naFlag := func(target ModifierTarget, amount int) Modifier {
		return Modifier{
			Target: target, Amount: amount, BonusType: "untyped",
			Condition: &ModifierCondition{C: "flagOn", Flag: "bencao-caseira", Label: "Bênção caseira"},
		}
	}
	base := ComputeItemEffects([]ActiveItem{
		vested("amuleto-do-mestre",
			naFlag(ModifierTarget{K: "attack", Scope: "all"}, 2),
			naFlag(ModifierTarget{K: "damage", Scope: "all"}, 3),
		),
	})
	if len(base.Conditional) != 2 {
		t.Fatalf("o arranjo devia oferecer 2 condicionais e ofereceu %d — sem dois membros, "+
			"ligar o grupo e ligar um termo dariam o mesmo resultado", len(base.Conditional))
	}

	ligado := ApplyActiveConditionals(base, map[string]bool{FlagGroupID("bencao-caseira"): true})

	ataque := StatFor(ligado, ModifierTarget{K: "attack", Scope: "all"}).Total
	dano := StatFor(ligado, ModifierTarget{K: "damage", Scope: "all"}).Total
	if ataque != 2 || dano != 3 {
		t.Errorf("o grupo ligou ataque=%d dano=%d, esperava 2 e 3.\n"+
			"Um dos dois zerado quer dizer que o interruptor dobrou só um membro — "+
			"a ficha soma metade da regra com a tela dizendo que são duas.", ataque, dano)
	}
	if len(ligado.Conditional) != 0 {
		t.Errorf("sobraram %d condicionais na oferta depois de o grupo ser ligado", len(ligado.Conditional))
	}
}
