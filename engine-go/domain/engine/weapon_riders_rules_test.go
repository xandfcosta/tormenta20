package engine

import (
	"path/filepath"
	"testing"
)

// O ENCANTO ATRAVESSA DO CATÁLOGO ATÉ A CARTA DA ARMA.
//
// Os casos do `attack_rules_test.go` prendem a ARITMÉTICA: dado extra não
// multiplica, bônus de crítico não entra no acerto normal. Este prende a
// COMPOSIÇÃO — que um modificador escrito no catálogo vira parcela na carta, e
// não some no caminho.
//
// É a metade que o oráculo não testemunha: nenhuma das 18 fichas empunha arma
// encantada, então o diff delas sai vazio por mais certo ou errado que o
// trajeto esteja.
func TestTheEnchantReachesTheWeaponCard(t *testing.T) {
	flamejante := Modifier{
		Target: ModifierTarget{K: "damage", DamageType: "fogo"},
		Dice:   "1d6", BonusType: "untyped",
	}
	dilacerante := Modifier{
		Target: ModifierTarget{K: "damage"}, Amount: 10, BonusType: "untyped",
		Condition: &ModifierCondition{C: "onCritical"},
	}
	empunhada := "wielded"
	efeitos := ComputeItemEffects([]ActiveItem{{
		SourceID: "espada", Source: "Espada longa", Equipped: &empunhada,
		Modifiers: []Modifier{flamejante, dilacerante},
	}})

	if len(efeitos.ExtraDamage) != 1 {
		t.Fatalf("esperava uma parcela extra e vieram %d", len(efeitos.ExtraDamage))
	}
	if got := efeitos.ExtraDamage[0]; got.Dice != "1d6" || got.Type != "fogo" {
		t.Errorf("a parcela veio %+v, e o catálogo escreveu 1d6 de fogo", got)
	}
	if got := efeitos.CriticalBonus[targetKey(ModifierTarget{K: "damage"})]; got != 10 {
		t.Errorf("o bônus de crítico veio %d e o catálogo escreveu 10", got)
	}

	// NENHUM DOS DOIS PODE TER VIRADO PARCELA DA PILHA. Se o dado extra somasse
	// como `amount`, ele entraria no dano de TODO ataque valendo zero — um
	// modificador que existe e não faz nada, que é a forma mais silenciosa de
	// defeito neste motor.
	if agg, achou := efeitos.ByTarget[targetKey(ModifierTarget{K: "damage"})]; achou {
		t.Errorf("o alvo `damage` recebeu %d na pilha, e as duas parcelas são da "+
			"CARTA: %+v", agg.Total, agg.Contributions)
	}

	// E NENHUM DOS DOIS PODE SER OFERECIDO COMO INTERRUPTOR: crítico não se
	// liga, e dado extra não é circunstância.
	if len(efeitos.Conditional) != 0 {
		t.Errorf("foram oferecidos %d interruptores, e nenhum dos dois é opt-in: %+v",
			len(efeitos.Conditional), efeitos.Conditional)
	}
}

// O ENCANTO ESCOLHIDO NA SACOLA ATRAVESSA ATÉ A CARTA (ALE-416).
//
// O caso acima monta o `Modifier` em Go e prova a ARITMÉTICA da parcela. Este
// entra um passo antes, pelo campo que a cena escreve: `CharacterItem.Enchants`
// leva ids, o `ownItemMods` tem de ir buscá-los no catálogo, e a carta tem de
// sair com o efeito dos dois.
//
// Sem ele o campo podia existir na tabela, na cena e no DTO — e o coletor não
// olhar. Foi assim que os 28 encantos ficaram um ciclo inteiro no catálogo sem
// nenhuma arma poder carregá-los.
func TestTheChosenEnchantReachesTheWeaponCard(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	var oracle struct {
		Char Character `json:"char"`
	}
	readJSON(t, filepath.Join(dir, "bardo-versatil-nv7.json"), &oracle)

	forjada := func(improvements string, enchants ...string) WeaponCard {
		ch := oracle.Char
		ch.Items = []CharacterItem{{
			CatalogID: ptr("espada-longa"), Name: "Espada longa",
			Equipped: ptr("wielded"), Improvements: improvements, Enchants: enchants,
		}}
		cards := BookRuleset(catalogs).ComputeWeaponCards(ch, map[string]bool{})
		if len(cards) == 0 {
			t.Fatal("a espada longa empunhada não virou carta")
		}
		return cards[0]
	}
	card := func(enchants ...string) WeaponCard { return forjada("[]", enchants...) }

	nua := card()

	// O LIVRO IMPRIME O NÚMERO: "uma espada longa ameaçadora tem margem de
	// ameaça 17" (p335). Não há conta a conferir — está escrito no verbete.
	if got := card("encanto-ameacadora").CritRange; got != 17 {
		t.Errorf("a espada longa ameaçadora ameaça em %d e a p335 imprime o exemplo: 17", got)
	}
	// "A arma causa +1d6 de dano de fogo" (p335), e ele é PARCELA: dado extra
	// rola, não soma na pilha.
	flamejante := card("encanto-flamejante")
	if len(flamejante.ExtraDamage) != 1 || flamejante.ExtraDamage[0].Dice != "1d6" ||
		flamejante.ExtraDamage[0].Type != "fogo" {
		t.Errorf("a espada flamejante veio com as parcelas %+v, e a p335 diz +1d6 de fogo",
			flamejante.ExtraDamage)
	}
	// "+2 em testes de ataque e rolagens de dano" (p336), nos dois lados da
	// carta.
	formidavel := card("encanto-formidavel")
	if got := formidavel.Attack - nua.Attack; got != 2 {
		t.Errorf("a espada formidável soma %d ao ataque e a p336 diz +2", got)
	}
	if got := formidavel.DamageBonus - nua.DamageBonus; got != 2 {
		t.Errorf("a espada formidável soma %d ao dano e a p336 diz +2", got)
	}

	// OS DOIS JUNTOS DÃO +4, E NÃO +6: "bônus por encantos não se acumulam"
	// (p333). É a frase que explica o pré-requisito de um encanto que SUBSTITUI
	// o outro — a Magnífica exige a Formidável (p336) e depois a apaga.
	//
	// O balde é `enchantment`, PRÓPRIO do encanto, e não o `enhancement` que a
	// melhoria usa. Os dois não podem dividir balde: a mesma p333 manda SOMAR
	// melhoria com encanto, e um balde só faria a Certeira e a Formidável
	// disputarem.
	ambos := card("encanto-formidavel", "encanto-magnifica")
	if got := ambos.Attack - nua.Attack; got != 4 {
		t.Errorf("Formidável + Magnífica somam %d ao ataque, e a p333 diz que bônus de "+
			"encanto não acumula: o maior vence, e a p336 dá +4 à Magnífica", got)
	}
	if got := ambos.DamageBonus - nua.DamageBonus; got != 4 {
		t.Errorf("Formidável + Magnífica somam %d ao dano, e a p333 diz que bônus de "+
			"encanto não acumula: o maior vence, e a p336 dá +4 à Magnífica", got)
	}

	// E A MELHORIA SOMA COM O ENCANTO, que é a outra metade da p333: "some [...]
	// os bônus fornecidos por melhorias e encantos para determinar o bônus do
	// item". Sem este caso, pôr os dois no mesmo balde passaria despercebido —
	// o caso acima ficaria verde e a Certeira sumiria da conta.
	comMelhoria := forjada(`["melhoria-certeira"]`, "encanto-formidavel")
	if got := comMelhoria.Attack - nua.Attack; got != 3 {
		t.Errorf("a espada Certeira (melhoria, +1) e Formidável (encanto, +2) ataca com "+
			"%+d, e a p333 manda somar melhoria com encanto: +3", got)
	}
}
