package engine

import (
	"path/filepath"
	"testing"
)

// OS MATERIAIS ESPECIAIS, na carta da arma (ALE-415).
//
// Cinco dos seis divergiam do livro, e nenhum teste os cobria: o
// `audit-equipment.py` confere PREÇO por tabela do capítulo 3, e os seis saem
// dele na lista de "NÃO MEDIDOS". A regra de um material nunca foi medida.
//
// O que este caso prende é o trajeto inteiro — catálogo, sobreposição, carta —
// para os três que o motor aplica, e a AUSÊNCIA para o que ele não deve
// aplicar.
func TestTheSpecialMaterialReachesTheWeaponCard(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	var oracle struct {
		Char Character `json:"char"`
	}
	readJSON(t, filepath.Join(dir, "bardo-versatil-nv7.json"), &oracle)

	card := func(material *string) WeaponCard {
		ch := oracle.Char
		ch.Items = []CharacterItem{{
			CatalogID: ptr("espada-longa"), Name: "Espada longa",
			Equipped: ptr("wielded"), Improvements: "[]", Material: material,
		}}
		cards := BookRuleset(catalogs).ComputeWeaponCards(ch, map[string]bool{})
		if len(cards) == 0 {
			t.Fatal("a espada longa empunhada não virou carta")
		}
		return cards[0]
	}

	nua := card(nil)

	// O LIVRO DÁ O NÚMERO: "uma espada longa de mitral tem margem de ameaça
	// 18-20" (p167). Não há conta a fazer — o valor esperado está impresso.
	if got := card(ptr("material-mitral")).CritRange; got != 18 {
		t.Errorf("a espada longa de mitral ameaça em %d e a p167 imprime o exemplo: 18-20", got)
	}
	// "Causa +2 pontos de dano por frio" (p166). Aqui o catálogo dizia `amount:
	// 1` com a nota "+1d6 frio" — o número e a nota erravam, e discordavam
	// entre si.
	if got := card(ptr("material-gelo-eterno")).DamageBonus; got != nua.DamageBonus+2 {
		t.Errorf("a espada longa de gelo eterno dá %d de bônus de dano e a nua dá %d — "+
			"a p166 diz +2", got, nua.DamageBonus)
	}
	// "Causa +1d6 de dano extra" (p167), e ele é PARCELA e não somando: o
	// catálogo não tinha nada, e a única forma antes da ALE-412 era mentir num
	// inteiro.
	vermelha := card(ptr("material-materia-vermelha"))
	if len(vermelha.ExtraDamage) != 1 || vermelha.ExtraDamage[0].Dice != "1d6" {
		t.Errorf("a espada de matéria vermelha veio com as parcelas %+v, e a p167 diz +1d6",
			vermelha.ExtraDamage)
	}
	if vermelha.DamageBonus != nua.DamageBonus {
		t.Errorf("o +1d6 da matéria vermelha virou %d de bônus somado — dado extra não soma "+
			"na pilha, ele rola", vermelha.DamageBonus-nua.DamageBonus)
	}
	// A AUSÊNCIA, e ela é o caso mais importante: o aço-rubi tinha um "+2 dano
	// vs criaturas vivas" que a p166 não escreve em lugar nenhum. A página dá
	// "ignora 10 pontos de redução de dano", que é propriedade do ATAQUE.
	if got := card(ptr("material-aco-rubi")).DamageBonus; got != nua.DamageBonus {
		t.Errorf("a espada de aço-rubi dá %d de bônus de dano e a nua dá %d — a p166 não "+
			"dá bônus de dano nenhum, e o +2 que estava no catálogo era inventado",
			got, nua.DamageBonus)
	}
}

// A MATÉRIA VERMELHA COBRA CARISMA, e a exceção é nomeada.
//
// "Estes itens assustadores impõem ao usuário penalidade de –2 em perícias
// baseadas em Carisma (exceto Intimidação)" (p167). A regra mora na ABERTURA do
// verbete e não numa das metades rotuladas — foi por isso que o auditor teve de
// ler o parágrafo de abertura junto.
//
// A exceção é escrita perícia a perícia porque o alvo que a expressaria de uma
// vez, o `expertiseRemovePenalty`, não tem leitor no motor: ele existe no
// `targetKey`, tem rótulo na aba Efeitos e ninguém o consome.
func TestTheRedMatterChargesCharismaExceptIntimidation(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }
	vermelha := "material-materia-vermelha"

	efeitos := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(Character{
		Items: []CharacterItem{{
			CatalogID: ptr("espada-longa"), Name: "Espada longa",
			Equipped: ptr("wielded"), Improvements: "[]", Material: &vermelha,
		}},
	}))

	for _, pericia := range []string{"Adestramento", "Atuação", "Diplomacia", "Enganação", "Jogatina"} {
		if got := StatFor(efeitos, ModifierTarget{K: "expertise", Name: pericia}).Total; got != -2 {
			t.Errorf("a matéria vermelha cobra %d em %s e a p167 diz –2", got, pericia)
		}
	}
	if got := StatFor(efeitos, ModifierTarget{K: "expertise", Name: "Intimidação"}).Total; got != 0 {
		t.Errorf("a matéria vermelha cobrou %d em Intimidação, e a p167 a EXCETUA — "+
			"assustar com um item assustador é o ponto da exceção", got)
	}
}
