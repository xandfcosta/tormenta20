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
	catalogs := primeBook(t)
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

// O ADAMANTE ANDA O DADO DA ARMA (p166, pela Tabela 3-2 da p143).
//
// "Arma. Aumenta o dano em um passo" — a metade que o verbete escrevia e o
// motor não tinha como aplicar: até a ALE-435 a falta estava declarada no
// `unmodeled` do material, por não existir a tabela de passos.
//
// A espada longa é o caso que o livro ilustra melhor: 1d8 é uma célula da
// coluna Normal, e a linha dela dá 1d10 no +1 passo.
func TestTheAdamantWeaponWalksItsDamageDie(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }

	card := func(material *string, equipped string) WeaponCard {
		cards := BookRuleset(catalogs).ComputeWeaponCards(Character{
			Items: []CharacterItem{{
				CatalogID: ptr("espada-longa"), Name: "Espada longa",
				Equipped: ptr(equipped), Improvements: "[]", Material: material,
			}},
		}, map[string]bool{})
		if len(cards) == 0 {
			t.Fatalf("a espada longa %s não virou carta", equipped)
		}
		return cards[0]
	}

	// O CONTROLE primeiro: sem o material, a carta diz o dado do catálogo.
	if got := card(nil, "wielded").Damage; got != "1d8" {
		t.Fatalf("a espada longa nua causa %q, e o catálogo diz 1d8 — o resto deste "+
			"caso não mediria nada", got)
	}
	if got := card(ptr("material-adamante"), "wielded").Damage; got != "1d10" {
		t.Errorf("a espada longa de adamante causa %q, e a p166 manda andar um passo: "+
			"1d8 vira 1d10 pela Tabela 3-2 (p143)", got)
	}
	// E O PASSO É DA ARMA NA MÃO: o material também vale para armadura, e um
	// item que não está empunhado não tem carta nenhuma para andar.
	if got := card(ptr("material-mitral"), "wielded").Damage; got != "1d8" {
		t.Errorf("a espada longa de mitral causa %q: o mitral não anda passo nenhum "+
			"(p167), e um passo que valesse para todo material seria regra inventada", got)
	}
}

// A MATÉRIA VERMELHA COBRA CARISMA, e a exceção é nomeada.
//
// "Estes itens assustadores impõem ao usuário penalidade de –2 em perícias
// baseadas em Carisma (exceto Intimidação)" (p167). A regra mora na ABERTURA do
// verbete e não numa das metades rotuladas — foi por isso que o auditor teve de
// ler o parágrafo de abertura junto.
//
// A exceção é escrita perícia a perícia porque não há alvo que a expresse de
// uma vez. Havia um candidato, o `expertiseRemovePenalty`, e ele não existe mais
// — a ALE-418 o cortou justamente por isto: nenhum leitor no motor e nenhuma
// entrada no catálogo. Ter rótulo na aba Efeitos não é ter leitor.
func TestTheRedMatterChargesCharismaExceptIntimidation(t *testing.T) {
	catalogs := primeBook(t)
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

// O GELO ETERNO PROTEGE DE FOGO, e a redução segue a peça (p166, ALE-440).
//
// "Armadura e Escudo. Fornece redução de fogo: armaduras leves e escudos,
// redução 5; armaduras pesadas, redução 10."
//
// A metade estava declarada como não modelada por duas razões que ENVELHECERAM:
// a redução por TIPO entrou na ALE-432 e a ramificação pela peça na ALE-438.
// Uma nota que diz "o motor não tem" depois de o motor ter é defeito entregue —
// ela esconde o número de quem forjou o item.
func TestTheEternalIceReducesFireByThePieceItIsOn(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }
	gelo := "material-gelo-eterno"

	fogo := func(item, equipped string) int {
		ch := Character{Items: []CharacterItem{{
			CatalogID: ptr(item), Name: item, Equipped: ptr(equipped),
			Improvements: "[]", Material: &gelo,
		}}}
		effects := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(ch))
		return StatFor(effects, ModifierTarget{K: "damageReduction", DamageType: "fogo"}).Total
	}

	if got := fogo("armadura-completa", "vested"); got != 10 {
		t.Errorf("a armadura pesada de gelo eterno reduz %d de fogo e a p166 diz 10", got)
	}
	if got := fogo("armadura-couro", "vested"); got != 5 {
		t.Errorf("a armadura leve de gelo eterno reduz %d de fogo e a p166 diz 5", got)
	}
	if got := fogo("escudo-pesado", "wielded"); got != 5 {
		t.Errorf("o escudo de gelo eterno reduz %d de fogo e a p166 diz 5", got)
	}
	// E A ARMA NÃO REDUZ NADA: a metade de arma do verbete causa +2 de dano por
	// frio, e nada ali protege de fogo.
	if got := fogo("espada-longa", "wielded"); got != 0 {
		t.Errorf("a espada de gelo eterno reduz %d de fogo, e a p166 não dá redução a arma", got)
	}
}

// A MADEIRA TOLLON DÁ RESISTÊNCIA A MAGIA no escudo e no esotérico (p167).
//
// "Escudo e Esotérico. Fornece resistência a magia +2." A nota dizia que
// "resistência a magia não existe no motor", e ela existe desde a ALE-418: é o
// alvo `resistance` com a condição `against`, que a p229 define como bônus em
// TESTE — "uma criatura com resistência a magia +2 recebe +2 em testes de
// Fortitude, Reflexos ou Vontade contra habilidades mágicas".
func TestTheTollonWoodGivesMagicResistanceOnShieldAndCatalyst(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }
	tollon := "material-madeira-tollon"

	// ELA É CIRCUNSTANCIAL, e por isso chega como OPT-IN e não somada: a p229
	// define resistência a magia como bônus em teste "contra habilidades
	// mágicas", e a condição `against` é o interruptor que a pessoa liga quando
	// a circunstância acontece. Somá-la sempre daria +2 em toda Fortitude.
	oferecida := func(item, equipped string) (ConditionalEffect, bool) {
		ch := Character{Items: []CharacterItem{{
			CatalogID: ptr(item), Name: item, Equipped: ptr(equipped),
			Improvements: "[]", Material: &tollon,
		}}}
		effects := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(ch))
		for _, c := range effects.Conditional {
			if c.Target.K == "resistance" {
				return c, true
			}
		}
		return ConditionalEffect{}, false
	}

	escudo, achou := oferecida("escudo-leve", "wielded")
	if !achou {
		t.Fatalf("o escudo leve de madeira tollon não oferece resistência a magia nenhuma")
	}
	if escudo.Amount != 2 {
		t.Errorf("o escudo leve de madeira tollon oferece %d de resistência e a p167 diz 2",
			escudo.Amount)
	}
	// E A ARMA NÃO DÁ: a metade de arma do verbete fala de vencer RD e de custo
	// em PM, e nenhuma das duas é resistência.
	if _, achou := oferecida("espada-longa", "wielded"); achou {
		t.Error("a espada de madeira tollon oferece resistência, e a p167 só a dá a " +
			"escudo e esotérico")
	}
}
