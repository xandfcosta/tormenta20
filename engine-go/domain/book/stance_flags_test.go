package book

import (
	"testing"

	"t20engine/domain/engine"
)

// A FLAG DECLARADA E A DOS MODIFICADORES SÃO A MESMA.
//
// A `Activation.Flag` passou a ser declarada porque três das seis Posturas de
// Combate (p54) não têm modificador de onde derivá-la. O preço disso é que
// agora existem DOIS lugares dizendo qual flag a postura acende — e duas
// grafias de um conceito é o que o GLOSSARY proíbe.
//
// O defeito que elas produzem é calado nos DOIS sentidos: com a declarada
// errada, a postura acende uma flag que modificador nenhum escuta e o
// personagem paga 2 PM por nada; com os modificadores errados, o número muda
// sozinho e ninguém consegue desligá-lo.
//
// DENOMINADOR: ele diz quantas olhou. Com zero posturas declaradas, "nenhuma
// divergiu" e "não mediu nada" são a mesma cor.
func TestEveryDeclaredStanceFlagMatchesItsModifiers(t *testing.T) {
	flags := ClassPowerFlags()
	measured := 0
	for _, a := range Activations() {
		if a.Kind != "stance" || a.Flag == "" {
			continue
		}
		measured++
		doPoder, temModificador := flags[a.ID]
		if !temModificador {
			// SEM MODIFICADOR NÃO HÁ O QUE CASAR, e é o caso que fez a
			// declaração existir: a Castigo de Ferro e a Provocação Petulante
			// não mexem em número nenhum — o que elas fazem é do mestre.
			continue
		}
		if doPoder != a.Flag {
			t.Errorf("%s declara a flag %q e os modificadores dele usam %q",
				a.ID, a.Flag, doPoder)
		}
	}
	if measured == 0 {
		t.Fatal("nenhuma postura declara flag — o guarda não mediu nada")
	}
}

// TODA POSTURA DE UM GRUPO CONHECE AS IRMÃS, e o grupo tem mais de uma.
//
// Um grupo com uma postura só é um grupo que não exclui ninguém — ele passaria
// por regra de exclusividade sem nunca excluir nada, que é a forma mais cara
// de um erro de digitação no nome do grupo aparecer.
func TestEveryStanceGroupHasMoreThanOneStance(t *testing.T) {
	size := map[string]int{}
	for _, s := range StancesFromCatalog() {
		if s.Group != "" {
			size[s.Group]++
		}
	}
	if len(size) == 0 {
		t.Fatal("nenhum grupo de posturas — o guarda não mediu nada")
	}
	for group, n := range size {
		if n < 2 {
			t.Errorf("o grupo %q tem %d postura: ou o nome está errado numa delas, "+
				"ou o grupo não precisa existir", group, n)
		}
	}
}

// O DEGRAU PAGO DE UMA POSTURA TEM DE SER REPRESENTÁVEL, e são duas condições.
//
// A `StanceDegreeModifiers` escreve o bônus TOTAL com o `bonusType` da base e
// conta com o não-empilhamento (p105) para o maior vencer. Isso dá a conta certa
// enquanto duas coisas valerem, e nenhuma das duas é garantida pelo catálogo:
//
//  1. O BÔNUS DA BASE DISPUTA. Um `untyped` SOMA (p105), então o total escrito
//     por cima dele viraria base+total: a Fúria de um degrau daria +5 em vez de
//     +3, e o número erraria para MAIS, que é o lado que ninguém reclama.
//  2. A POSTURA NÃO ACUMULA TAMBÉM. O efeito do degrau e o do bônus cumulativo
//     são gravados com a mesma chave — o id do poder da postura — e o
//     `active_effects` é único em (personagem, catálogo, escopo). A segunda
//     escrita apagaria a primeira, calada.
//
// DENOMINADOR: duas posturas escalam hoje (Fúria p41, Inspiração p44). Zero
// medidas quer dizer que a escala saiu do catálogo e este guarda ficou sem
// terreno — ver "Guarda vale o que vale o terreno" no CLAUDE.md.
func TestEveryStanceDegreeRidesANonStackingBonus(t *testing.T) {
	measured := 0
	for _, a := range Activations() {
		if a.Kind != "stance" || a.Scaling == nil {
			continue
		}
		measured++
		flag := StancesFromCatalog()
		stance, found := flagOfStance(flag, a.ID)
		if !found {
			t.Errorf("%s escala em degraus e não acende flag nenhuma — "+
				"o degrau pago não tem onde se aplicar", a.ID)
			continue
		}
		if a.Cumulative != nil {
			t.Errorf("%s tem degraus E bônus cumulativo, e os dois gravam o efeito "+
				"com a chave %q: um apagaria o outro", a.ID, a.ID)
		}
		base := StanceBase(stance)
		if len(base) == 0 {
			t.Errorf("%s escala em degraus e não move número nenhum — "+
				"o PM extra não compraria nada", a.ID)
			continue
		}
		for _, m := range base {
			if engine.BonusTypeAccumulates(m.BonusType) {
				t.Errorf("o bônus de %s em %q é %q, que SOMA (p105): o degrau escreve "+
					"o TOTAL por cima da base, e os dois somados dariam o dobro do degrau",
					a.ID, m.Target.K, m.BonusType)
			}
		}
	}
	if measured < 2 {
		t.Fatalf("o guarda mediu %d posturas que escalam, e o catálogo tem duas "+
			"(Fúria p41, Inspiração p44)", measured)
	}
}

// flagOfStance acha a flag que aquela ativação acende, pelo id dela.
func flagOfStance(byFlag map[string]Stance, activationID string) (string, bool) {
	for flag, s := range byFlag {
		if s.ID == activationID {
			return flag, true
		}
	}
	return "", false
}

// TODA CONCESSÃO DE POSTURA PENDE DE UM PODER DE CLASSE, porque é por ele que a
// posse se pergunta.
//
// Quem aplica a concessão filtra com o `sheet.OwnsClassPower`, e ele só responde
// por poder de CLASSE. Uma concessão de `race.*`, `origin.*` ou `deus.*` que
// exigisse flag voltaria `false` para todo mundo e nasceria INERTE — o poder
// estaria na ficha, a postura subiria, e a concessão nunca sairia, sem erro em
// lugar nenhum. Aqui a chegada dele reprova com o id, e quem a trouxer estende a
// pergunta da posse em vez de descobrir isso na mesa.
//
// DENOMINADOR: uma concessão de flag hoje, a Alma de Bronze (p41). Zero medidas
// quer dizer que o terreno sumiu — ver "Guarda vale o que vale o terreno" no
// `CLAUDE.md`.
func TestEveryFlagGrantHangsOnAClassPower(t *testing.T) {
	measured := 0
	for _, a := range Activations() {
		if a.RequiresFlag == "" || a.Grant == nil {
			continue
		}
		measured++
		if _, found := ClassPowers()[a.ID]; found {
			continue
		}
		t.Errorf("%s exige a flag %q e CONCEDE algo, e não é poder de classe: a posse "+
			"dele não tem como ser perguntada, então a concessão não sairia para "+
			"ninguém", a.ID, a.RequiresFlag)
	}
	if measured == 0 {
		t.Fatal("nenhuma ativação concede por flag — o guarda não mediu nada")
	}
}

// TODA CONCESSÃO DO CATÁLOGO TEM QUEM A APLIQUE, e as espécies são TRÊS.
//
// Cada uma espera um gatilho diferente, e é o par (gatilho, espécie) que diz
// quem aplica:
//
//   - a POSTURA (`requiresFlag` + `temp-hp`) espera a flag acender, e sai no
//     `applyStanceGrants`;
//   - o PODER DE USAR (`instant` + `active-effect`) espera o uso, e sai no
//     `writeGrantedEffect`;
//   - o INÍCIO DE CENA (`passive` + `temp-hp`) não espera gesto nenhum, e sai
//     na varredura do `Party.StartScene`.
//
// É uma lista de PERMITIDOS e não de proibidos, pela razão que o `CLAUDE.md` já
// dá: uma lista de proibidos subconta em silêncio. Um par novo — um `instant`
// que conceda PV temporários, um `passive` que ligue modificadores — reprova
// aqui com o id, em vez de virar dado inerte no catálogo.
//
// Era isso que faltava: CINCO concessões ficaram anos no arquivo sem ninguém
// que as aplicasse, e nada reclamou (ALE-426).
func TestEveryGrantHasSomeoneToApplyIt(t *testing.T) {
	applied := map[string]string{
		"flag+temp-hp":          "applyStanceGrants, ao entrar na postura",
		"instant+active-effect": "writeGrantedEffect, ao usar o poder",
		"passive+temp-hp":       "Party.StartScene, na varredura do início da cena",
	}
	measured := 0
	for _, spec := range Activations() {
		if spec.Grant == nil {
			continue
		}
		measured++
		trigger := spec.Kind
		if spec.RequiresFlag != "" {
			trigger = "flag"
		}
		if _, found := applied[trigger+"+"+spec.Grant.Kind]; found {
			continue
		}
		t.Errorf("%s concede %q com gatilho %q, e ninguém aplica essa combinação: "+
			"a concessão fica no catálogo e não sai para a mesa, sem erro nenhum. "+
			"Escreva quem a aplica e acrescente o par aqui", spec.ID, spec.Grant.Kind, trigger)
	}
	if measured == 0 {
		t.Fatal("nenhuma ativação concede nada — o guarda não mediu nada")
	}
}

// E TODA CONCESSÃO DE EFEITO ATIVO CARREGA O QUE LIGAR.
//
// O `PowerGrant` não tinha o campo `Modifiers` e o catálogo o trazia: a lista
// era DESCARTADA na leitura, e a concessão aplicada escreveria um efeito vazio
// — uma linha na aba Efeitos que não move número nenhum, que é pior que a
// ausência, porque parece que funcionou (ALE-426).
func TestEveryActiveEffectGrantCarriesItsModifiers(t *testing.T) {
	measured := 0
	for _, spec := range Activations() {
		if spec.Grant == nil || spec.Grant.Kind != "active-effect" {
			continue
		}
		measured++
		if len(spec.Grant.Modifiers) == 0 {
			t.Errorf("%s concede um efeito ativo e não diz o que ele liga", spec.ID)
		}
	}
	if measured == 0 {
		t.Fatal("nenhuma concessão de efeito ativo — o guarda não mediu nada")
	}
}
