package engine

import "testing"

// A FÓRMULA DO `ConditionalID` ESTÁ NO BANCO, E POR ISSO ELA NÃO PODE MUDAR.
//
// Ela é gravada em `character_conditionals.conditionalId` — é assim que a ficha
// lembra quais condicionais o jogador ligou. Mudar a fórmula não quebra
// compilação, não loga e não falha: o id gravado deixa de casar com o gerado, e
// **toda escolha de condicional volta a aparecer DESLIGADA**.
//
// # O esperado aqui é um LITERAL, e isso é o ponto
//
// Os cinco testes que já citavam o `ConditionalID` derivavam o esperado da
// própria função (`ConditionalID(base.Conditional[0])`). Isso prova que a
// linguagem devolve valores e nada sobre a FORMA — mudar a fórmula os mantém
// todos verdes. Este escreve a string à mão.
//
// # O que motivou escrevê-lo: a tentação é concreta (ALE-386)
//
// A fatia que dá `SourceID` às fontes torna óbvio "melhorar" esta chave pondo o
// id nela — que é exatamente o apagamento silencioso descrito acima. Se algum
// dia ela PRECISAR mudar, o caminho é uma migração que reescreva as linhas
// gravadas, e não uma edição na fórmula.
func TestTheConditionalIdKeepsItsExactShape(t *testing.T) {
	efeito := ConditionalEffect{
		Source:    "Armadura completa",
		BonusType: "untyped",
		Amount:    -3,
		Note:      "nota qualquer",
		Target:    ModifierTarget{K: "displacement", Scope: "armor"},
	}

	const gravado = "Armadura completa::displacement:armor::nota qualquer::-3::untyped"

	if got := ConditionalID(efeito); got != gravado {
		t.Fatalf("a fórmula do ConditionalID MUDOU.\n  gerado:  %s\n  gravado: %s\n\n"+
			"Esta string vive em `character_conditionals.conditionalId`. Mudá-la faz "+
			"toda escolha de condicional já gravada parar de casar — e a ficha as mostra "+
			"desligadas, sem erro em lugar nenhum.\n"+
			"Se a mudança é mesmo necessária, ela vem com uma migração que reescreve as "+
			"linhas, nunca sozinha.", got, gravado)
	}
}
