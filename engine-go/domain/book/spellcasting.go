package book

// A PROGRESSÃO DE CÍRCULO por classe: em que nível cada classe conjuradora
// destrava cada círculo.
//
// Ela mora no SERVIDOR porque é FRONTEIRA e não UX: sem ela a validação de
// aprimoramento confere índice, duplicata, `stacks ≥ 1` e "muda não empilha", e
// aceita qualquer `requiresCircle` — um pedido montado à mão conjura o que a
// regra não permite, em 126 dos 486 aprimoramentos do catálogo.
//
// A tabela vive no campo `spellcasting` das cinco classes conjuradoras em
// `classes.json`, ao lado da página de cada uma, e foi CONFERIDA contra o livro
// na ALE-338 — os vinte e cinco níveis, os cinco tetos, as listas e os atributos
// batem. Quem guarda a conferência é o `TestTheCircleProgressionMatchesTheBook`,
// com a página de cada classe; ele transcreve o livro à mão, do outro lado da
// tabela, e é a segunda opinião que morreu junto com a SPA.
//
// O que a auditoria corrigiu foi um círculo 0 que as cinco declaravam: o livro
// não tem esse círculo (Tabela 4-1, p170, vai do 1º ao 5º), e "truque" é um
// aprimoramento que zera o custo em PM — a magia continua no círculo dela. Nada
// o lia, e um número que ninguém lê é o que mais se parece com regra conferida.
//
// E mora no `book` porque **o destino de uma função é a DEPENDÊNCIA dela**: quem
// lê o catálogo é do livro, não a cena da ficha.

// SpellProgression é o que uma classe conjuradora destrava, e quando.
type SpellProgression struct {
	List      string `json:"list"`
	Attribute string `json:"attribute"`
	MaxCircle int    `json:"maxCircle"`
	// UnlockLevel é o nível em que cada círculo abre. NULO significa "esta
	// classe nunca chega lá" — o Bardo e o Druida param no 4º, o Paladino no 1º.
	// Ponteiro e não zero: nível 0 seria "abre de saída", que é outra coisa.
	UnlockLevel map[string]*int `json:"unlockLevel"`
}

// SpellProgressions é a tabela por nome de classe, só com quem conjura.
func SpellProgressions() map[string]SpellProgression {
	_, classes, _ := CharacterCatalogs()
	tabela := make(map[string]SpellProgression, len(classes))
	for _, c := range classes {
		if c.Spellcasting != nil {
			tabela[c.Name] = *c.Spellcasting
		}
	}
	return tabela
}
