package engine

import "testing"

// UM FATOR SÓ VALE SE A CONDIÇÃO DELE SE CUMPRIR (ALE-378, fatia 3).
//
// A resolução é uma sequência de sistemas, e a ORDEM é a regra: quem colhe os
// fatores roda DEPOIS de quem cala o que a condição recusa. Trocar os dois
// aplicaria o fator de um item que a pessoa nem está empunhando.
//
// # Medido: sem este caso, a troca passa VERDE
//
// Sabotando a ordem dos dois sistemas, a suíte inteira — oráculo incluído —
// seguiu verde. A razão é que os dois únicos fatores de hoje vêm da tabela de
// CONDIÇÕES, e ela os declara sem condição nenhuma: eles sempre valem, então a
// ordem nunca é exercida.
//
// O que torna isso alcançável é a emenda de campanha (ALE-387): um mestre pode
// conceder um item cujo fator só vale empunhado. A fatia que abriu a porta não
// foi esta, mas é esta que tem a rede.
func TestAFactorWhoseConditionFailsDoesNotApply(t *testing.T) {
	vested := "vested"
	// O fator vale EMPUNHANDO, e o item está VESTIDO: a condição não se cumpre.
	meia := &Ratio{Num: 1, Den: 2}
	items := []ActiveItem{{
		SourceID: "amuleto-do-mestre", Source: "Amuleto", Equipped: &vested,
		Modifiers: []Modifier{
			{Target: ModifierTarget{K: "displacement"}, Amount: 2, BonusType: "untyped"},
			{
				Target: ModifierTarget{K: "displacement"}, BonusType: "untyped",
				Factor: meia, Condition: &ModifierCondition{C: "wielded"},
			},
		},
	}}

	effects := ComputeItemEffects(items)

	if got := StatFor(effects, ModifierTarget{K: "displacement"}).Total; got != 2 {
		t.Fatalf("a parcela somou %d e esperava 2 — o controle já estava errado", got)
	}
	if fator, tem := effects.Factors[targetKey(ModifierTarget{K: "displacement"})]; tem && fator.isSet() {
		t.Errorf("o fator %d/%d foi colhido, e a condição dele (empunhar) NÃO se cumpre.\n"+
			"Quem colhe fator roda depois de quem cala o que a condição recusa.",
			fator.Num, fator.Den)
	}

	// O CONTROLE POSITIVO: com a condição cumprida, o mesmo fator TEM de valer.
	// Sem ele, um coletor que nunca colhe fator nenhum passaria no caso acima.
	wielded := "wielded"
	items[0].Equipped = &wielded
	comFator := ComputeItemEffects(items)
	fator := comFator.Factors[targetKey(ModifierTarget{K: "displacement"})]
	if !fator.isSet() || fator.Num != 1 || fator.Den != 2 {
		t.Errorf("empunhando, o fator devia ser 1/2 e veio %+v — o caso acima estaria "+
			"medindo um colhedor que não colhe nada", fator)
	}
}

// O TERMO QUE PERDE O EMPILHAMENTO CONTINUA NO MUNDO, com a razão escrita.
//
// É o que esta fatia compra além do refactor. O `ComputeItemEffects` antigo
// DESCARTAVA o perdedor, e a ficha — que existe para responder "por que 12?" —
// não tinha como dizer "−2 de X, não aplicado por não empilhar".
//
// O caso mede o mundo, e não a saída: a saída é byte a byte igual à de antes, e
// é o oráculo que responde por ela.
func TestTheTermThatLosesStackingStaysInTheWorldWithItsReason(t *testing.T) {
	vested := "vested"
	items := []ActiveItem{{
		SourceID: "armadura", Source: "Armadura", Equipped: &vested,
		Modifiers: []Modifier{
			{Target: ModifierTarget{K: "defense"}, Amount: 2, BonusType: "armor"},
			{Target: ModifierTarget{K: "defense"}, Amount: 5, BonusType: "armor"},
		},
	}}

	// A conta: `armor` não empilha, vence o +5.
	if got := StatFor(ComputeItemEffects(items), ModifierTarget{K: "defense"}).Total; got != 5 {
		t.Fatalf("a Defesa somou %d e o empilhamento manda valer o maior, 5", got)
	}

	mundo, perdedores := worldAfterResolving(items)
	if mundo == 0 {
		t.Fatal("nenhum termo no mundo — o caso mediria o vazio")
	}
	if len(perdedores) != 1 {
		t.Fatalf("esperava 1 termo calado pelo empilhamento e achei %d: %v", len(perdedores), perdedores)
	}
	if perdedores[0] == "" {
		t.Error("o termo calado não diz POR QUE — a razão é o que a tela vai mostrar")
	}
}
