package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// O DADO EXTRA DE UM CONDICIONAL SÓ VALE COM A CONDIÇÃO (p338, ALE-457).
//
// O `harvestWeaponRiders` colhia o dado e saía com `return` ANTES de olhar a
// condição, então um dado condicional entrava na carta da arma sempre — e, por
// sair marcado `Suppressed`, nem sobrava para virar interruptor na aba Efeitos.
//
// É o MESMO defeito que o `Factor` deste pacote já teve, com o mesmo sintoma
// calado: o modificador existe, a tela não oferece nada, e o número vale de
// graça. A diferença é o sinal — o fator sumia e o dado sobrava.
//
// Quem pagou foi o MACHADO SILVESTRE (p338): *"quando você usa o machado
// silvestre em um ambiente ermo e ao ar livre, causa +1d8 de dano"*. O app o
// dava em masmorra, em cidade e em toda parte.
//
// UNITÁRIO pela razão do `conditional_factor_test`: isto é a REGRA de uma
// conversão — o que sobrevive à ida e volta entre a pilha e o opt-in.

// axeOfTheWilds é a linha do Machado silvestre: dado sob condição de TERRENO.
func axeOfTheWilds() []Modifier {
	return []Modifier{{
		Target:    ModifierTarget{K: "damage", Scope: "this"},
		Dice:      "1d8",
		BonusType: "untyped",
		Condition: &ModifierCondition{C: "terrain", Type: "ermo e ao ar livre"},
		Note:      "+1d8 de dano",
	}}
}

func TestTheDiceOfASwitchedOnConditionalOnlyCountsWhenOn(t *testing.T) {
	base := ComputeItemEffects(wornItemWith(axeOfTheWilds()))

	// DESLIGADO é o controle, e é a metade que estava quebrada: sem ela, um
	// dado que vale SEMPRE passaria no caso de baixo sem ninguém notar.
	if len(base.ExtraDamage) != 0 {
		t.Errorf("o dado desceu com o terreno DESLIGADO: %+v", base.ExtraDamage)
	}
	if len(base.Conditional) != 1 {
		t.Fatalf("o dado condicional não saiu para o opt-in: %d linhas", len(base.Conditional))
	}

	ligado := ApplyActiveConditionals(base, map[string]bool{base.Conditional[0].Term: true})
	if len(ligado.ExtraDamage) != 1 {
		t.Fatalf("o terreno ligado não trouxe o dado: %+v", ligado.ExtraDamage)
	}
	if got := ligado.ExtraDamage[0].Dice; got != "1d8" {
		t.Errorf("o dado ligado é %q, e o livro dá 1d8", got)
	}
}

// E O DADO INCONDICIONAL CONTINUA ATRAVESSANDO.
//
// É o controle do conserto, pela mesma razão do fator: adiar o dado condicional
// poderia ter adiado TODOS, e aí a Flamejante (p335) pararia de somar o +1d6 de
// fogo que ela dá sem condição nenhuma.
func TestTheUnconditionalDiceStillReachesTheCard(t *testing.T) {
	flaming := []Modifier{{
		Target:    ModifierTarget{K: "damage", Scope: "this", DamageType: "fogo"},
		Dice:      "1d6",
		BonusType: "enchantment",
		Condition: &ModifierCondition{C: "wielded"},
	}}
	effects := ComputeItemEffects(wornItemWith(flaming))
	if len(effects.ExtraDamage) != 1 {
		t.Fatalf("o dado SEM condição parou de descer: %+v", effects.ExtraDamage)
	}
	if effects.ExtraDamage[0].Dice != "1d6" || effects.ExtraDamage[0].Type != "fogo" {
		t.Errorf("o dado incondicional chegou torto: %+v", effects.ExtraDamage[0])
	}
}

// E LIGAR QUALQUER CONDICIONAL NÃO PODE APAGAR O DADO DE OUTRO ENCANTO.
//
// O `ApplyActiveConditionals` remonta o `ItemEffects` no `return` e não
// repassava `ExtraDamage`, `Announced` nem `CriticalBonus`. Com NENHUM
// condicional ligado ele devolve `effects` intacto, então o defeito só aparece
// quando a pessoa liga alguma coisa — e aí o +1d6 de fogo da Flamejante some da
// carta da arma sem nada acusar.
//
// É a terceira vez que esta remontagem perde um campo: o `Factors` já foi (a
// Torre Inabalável), e agora as três parcelas que não são somáveis.
func TestSwitchingSomethingOnDoesNotEraseAnotherEnchantsDice(t *testing.T) {
	wielded := "wielded"
	items := []ActiveItem{
		{SourceID: "espada", Source: "Espada flamejante", Equipped: &wielded, Modifiers: []Modifier{{
			Target: ModifierTarget{K: "damage", Scope: "this", DamageType: "fogo"},
			Dice:   "1d6", BonusType: "enchantment",
			Condition: &ModifierCondition{C: "wielded"},
		}}},
		{SourceID: "lacerante", Source: "Lacerante", Equipped: &wielded, Modifiers: []Modifier{{
			Target: ModifierTarget{K: "damage", Scope: "this"}, Amount: 10,
			BonusType: "untyped", Condition: &ModifierCondition{C: "onCritical"},
		}}},
		{SourceID: "excruciante", Source: "Excruciante", Equipped: &wielded, Modifiers: []Modifier{{
			Target:    ModifierTarget{K: "announcedCondition", Name: "sangrando"},
			BonusType: "untyped", Condition: &ModifierCondition{C: "wielded"},
		}}},
		{SourceID: "torre", Source: "Torre", Equipped: &wielded, Modifiers: stoppedByTheTower()},
	}
	base := ComputeItemEffects(items)
	// O DENOMINADOR do arranjo: sem as três parcelas presentes ANTES, as
	// asserções de baixo comparam zero com zero e passam sobre nada.
	if len(base.ExtraDamage) != 1 || len(base.Announced) != 1 || len(base.CriticalBonus) != 1 {
		t.Fatalf("o arranjo não produziu as três parcelas: dado=%+v anúncio=%+v crítico=%+v",
			base.ExtraDamage, base.Announced, base.CriticalBonus)
	}

	ligado := ApplyActiveConditionals(base, map[string]bool{FlagGroupID("postura-torre"): true})
	if len(ligado.ExtraDamage) != 1 {
		t.Errorf("ligar a postura apagou o dado da Flamejante: %+v", ligado.ExtraDamage)
	}
	// AS OUTRAS DUAS DO MESMO `return`. Elas não são caso à parte: as três são
	// os campos que a remontagem esqueceu, e prender só o dado deixaria as
	// outras caírem no próximo que mexer na função.
	if len(ligado.Announced) != len(base.Announced) {
		t.Errorf("ligar a postura apagou a condição anunciada: %d virou %d",
			len(base.Announced), len(ligado.Announced))
	}
	if len(ligado.CriticalBonus) != len(base.CriticalBonus) {
		t.Errorf("ligar a postura apagou o bônus de crítico: %d virou %d",
			len(base.CriticalBonus), len(ligado.CriticalBonus))
	}
}

// NENHUM DADO DO CATÁLOGO DISPARA NO CRÍTICO, e este guarda é o que mantém
// assim (ALE-457).
//
// O conserto acima manda todo dado CONDICIONAL para o `deferConditionals`, e lá
// ele vira interruptor na aba Efeitos. Isso é certo para o `terrain`, o
// `against` e o `context` — são circunstâncias que a pessoa declara —, e é
// ERRADO para o `onCritical`: crítico não se liga, e o `itemeffects.go` já diz
// isso por extenso duas vezes.
//
// Hoje não existe nenhum, então a linha de base é VAZIA e o guarda só precisa
// recusar o primeiro. Ele falha com o nome do verbete, que é a diferença entre
// "decida onde este dado mora" e "procure o que está errado" — e a decisão é
// real: um `+2d6 no crítico` precisa de um `riders` próprio, porque não é
// parcela somável nem interruptor.
func TestNoDiceInTheCatalogIsTriggeredByACritical(t *testing.T) {
	arquivos, err := filepath.Glob(filepath.Join("..", "catalog", "data", "*.json"))
	if err != nil {
		t.Fatalf("achar os catálogos: %v", err)
	}
	medidos := 0
	for _, arquivo := range arquivos {
		bruto, err := os.ReadFile(arquivo)
		if err != nil {
			t.Fatalf("ler %s: %v", arquivo, err)
		}
		var cru any
		if err := json.Unmarshal(bruto, &cru); err != nil {
			t.Fatalf("%s não é JSON: %v", arquivo, err)
		}
		medidos += dadosNoCritico(t, filepath.Base(arquivo), cru)
	}
	// O DENOMINADOR: um `Glob` que deixasse de casar, ou uma varredura que
	// parasse de descer, diria "nenhum dado no crítico" sobre nada.
	if medidos == 0 {
		t.Fatal("o guarda não achou um único modificador com dado: ele parou de medir")
	}
	t.Logf("modificadores com dado medidos: %d", medidos)
}

// dadosNoCritico desce o JSON e conta os modificadores com `dice`, reprovando
// os que também trazem `onCritical`.
func dadosNoCritico(t *testing.T, arquivo string, no any) int {
	t.Helper()
	switch v := no.(type) {
	case map[string]any:
		achados := 0
		if dado, temDado := v["dice"].(string); temDado && dado != "" {
			achados++
			if c, ok := v["condition"].(map[string]any); ok && c["c"] == "onCritical" {
				t.Errorf("%s: um modificador soma %q no acerto crítico, e o motor não "+
					"tem casa para isso — o dado condicional vira INTERRUPTOR, e crítico "+
					"não se liga.\n    verbete: %v", arquivo, dado, v["note"])
			}
		}
		for _, filho := range v {
			achados += dadosNoCritico(t, arquivo, filho)
		}
		return achados
	case []any:
		achados := 0
		for _, filho := range v {
			achados += dadosNoCritico(t, arquivo, filho)
		}
		return achados
	}
	return 0
}

// A CARTA DA ARMA SÓ GANHA O DADO COM O TERRENO LIGADO (p338, ALE-457).
//
// O caso de cima prende a CONVERSÃO; este prende a LIGAÇÃO, que é a outra
// metade e a que o navegador não mostra: nada em `serve/` lê o `ExtraDamage` —
// quem o consome é a resolução do ataque, na mesa. Medido ao olhar a tela e não
// ver diferença nenhuma entre ligado e desligado.
//
// Integração de propósito: ele monta um personagem de verdade com o verbete de
// verdade e pergunta o que o chamador recebe, que é a faixa onde os defeitos de
// COMPOSIÇÃO moram.
func TestTheWeaponCardOnlyGetsTheDiceWithTheTerrainOn(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }

	ch := Character{
		Level: 10, Strength: 4,
		Proficiencies: `["armas-marciais"]`,
		Items: []CharacterItem{{
			CatalogID: ptr("machado-silvestre"), Name: "Machado silvestre",
			Equipped: ptr("wielded"), Improvements: "[]",
		}},
	}
	regra := BookRuleset(catalogs)

	desligado := regra.ComputeWeaponCards(ch, map[string]bool{})
	if len(desligado) != 1 {
		t.Fatalf("o Machado empunhado virou %d cartas", len(desligado))
	}
	if n := len(desligado[0].ExtraDamage); n != 0 {
		t.Errorf("o +1d8 desceu FORA do ermo: %+v", desligado[0].ExtraDamage)
	}

	// A CHAVE vem do próprio opt-in, e não escrita à mão: o `TermID` é um
	// endereço composto, e transcrevê-lo aqui seria derivar o esperado da
	// implementação com um passo a mais.
	efeitos := ComputeItemEffects(regra.ActiveItemsFor(ch))
	if len(efeitos.Conditional) != 1 {
		t.Fatalf("o Machado não ofereceu o opt-in do terreno: %d", len(efeitos.Conditional))
	}
	ligado := regra.ComputeWeaponCards(ch, map[string]bool{efeitos.Conditional[0].Term: true})
	if len(ligado[0].ExtraDamage) != 1 {
		t.Fatalf("o terreno ligado não pôs o dado na carta: %+v", ligado[0].ExtraDamage)
	}
	if got := ligado[0].ExtraDamage[0].Dice; got != "1d8" {
		t.Errorf("a carta ligada traz %q, e a p338 dá 1d8", got)
	}
}
