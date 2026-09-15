package book

// A PROGRESSÃO DE CÍRCULO por classe (ALE-272, fatia 6; movida para cá na
// ALE-278).
//
// # Ela vivia SÓ no TypeScript, e por isso a regra era só de interface
//
// O `SPELL_PROGRESSION` do front dizia em que nível cada classe destrava cada
// círculo, e era com ele que a SPA TRANCAVA um aprimoramento de círculo alto. O
// servidor nunca soube dessa tabela: o `validateAugments` conferia índice,
// duplicata, `stacks ≥ 1` e "muda não empilha", e aceitava qualquer
// `requiresCircle` — um pedido montado à mão conjurava o que a regra não
// permite. São 126 dos 486 aprimoramentos do catálogo, um quarto deles.
//
// Travar na UI é UX; a fronteira é o servidor. A tabela veio para o campo
// `spellcasting` das cinco classes conjuradoras em `classes.json`, ao lado da
// página de cada uma.
//
// # Ela foi MOVIDA, e não retranscrita
//
// Nenhum número aqui foi lido do livro naquela fatia: o que se garantiu é que a
// mudança é FIEL. Uma auditoria contra o livro é outro trabalho, e fingir que
// ela aconteceu seria pior que não fazê-la.
//
// # Por que ela mora no `book`
//
// A cena da ficha lia `catalog.Resource("classes")` DIRETO e desempacotava a
// tabela por conta. É o quarto caso desta família — depois do `items.go` da
// forja, do improviso do trilho do mestre e do `race-defs` de personagens —, e
// a regra que aqueles achados deixaram é sempre a mesma: **o destino de uma
// função é a DEPENDÊNCIA dela.** Quem lê o catálogo é do livro.

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
