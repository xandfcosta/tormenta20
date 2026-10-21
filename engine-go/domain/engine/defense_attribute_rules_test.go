package engine

import (
	"path/filepath"
	"testing"
)

// A DEFESA PODE USAR OUTRO ATRIBUTO NO LUGAR DA DESTREZA (p69, p79, ALE-443).
//
// Quatro verbetes dizem a mesma coisa com palavras diferentes — "pode usar sua
// Inteligência na Defesa" (Blindagem), "pode usar seu Carisma em vez de
// Destreza" (Autoconfiança) —, e o que muda entre eles é QUANDO.
//
// # "Pode usar" é escolha, e a casa já decidiu como tratá-la
//
// A arma com acuidade faz o mesmo: a regra é opcional, então a ficha toma a
// MELHOR. Perguntar ao jogador qual atributo ele quer seria pedir uma decisão
// cuja resposta é sempre a mesma.
//
// # O CONTROLE é de outra CLASSE, e não do mesmo nobre sem o poder
//
// A Autoconfiança é concedida no 1º nível: não existe nobre sem ela, e a ficha
// que a `ClassPowers` não cita a tem do mesmo jeito. O primeiro caso deste
// guarda nasceu assim e mediu o poder contra o próprio poder.
func TestTheDefenseTakesTheBestAttributeThePowersAllow(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	// Des 1 contra Car 4 e Int 2: a troca MUDA o número em toda linha abaixo,
	// que é o que separa "o poder valeu" de "o poder não fez nada".
	defesa := func(class string, level int, armadura, poderes string) DefenseBreakdown {
		ch := Character{
			Level: level, Dexterity: 1, Charisma: 4, Intelligence: 2,
			Classes:     []CharacterClass{{ClassName: class, Level: level}},
			ClassPowers: poderes,
			// PROFICIENTE nas duas, senão a p152 tira a Destreza por outro
			// motivo e o caso mediria a falta de treino.
			Proficiencies: `["armaduras-leves","armaduras-pesadas"]`,
			Items: []CharacterItem{{
				CatalogID: ptr(armadura), Name: armadura, Equipped: ptr("vested"),
				Improvements: "[]",
			}},
		}
		return BookRuleset(catalogs).ComputeSheet(ch, map[string]bool{}).Defense
	}

	casos := []struct {
		ramo      string
		class     string
		level     int
		armadura  string
		poderes   string
		total     int
		atributo  string
		procedeu  string
		aritmetca string
	}{{
		ramo:  "O CONTROLE: o guerreiro não tem poder nenhum destes, e soma a Destreza",
		class: "Guerreiro", level: 5, armadura: "armadura-couro", poderes: `[]`,
		total: 13, atributo: "dexterity", aritmetca: "10 + DES 1 + 2 da armadura",
	}, {
		ramo:  "AUTOCONFIANÇA (p79): o Carisma entra no lugar da Destreza, de armadura leve",
		class: "Nobre", level: 1, armadura: "armadura-couro", poderes: `[]`,
		total: 16, atributo: "charisma", procedeu: "Autoconfiança",
		aritmetca: "10 + CAR 4 + 2 da armadura",
	}, {
		ramo: "AUTOCONFIANÇA de armadura PESADA: ela não vale, e a p79 diz por que — " +
			"'continua não podendo somar um atributo na Defesa quando usa armadura pesada'",
		class: "Nobre", level: 1, armadura: "armadura-completa", poderes: `[]`,
		total: 20, atributo: "", aritmetca: "10 + 10 da armadura, sem atributo nenhum",
	}, {
		ramo:  "ARMADURA BRILHANTE (p79) é o INVERSO: ela só vale COM a pesada",
		class: "Nobre", level: 8, armadura: "armadura-completa",
		poderes: `["class.nobre.armadura-brilhante"]`,
		total:   24, atributo: "charisma", procedeu: "Armadura Brilhante",
		aritmetca: "10 + CAR 4 + 10 da armadura",
	}, {
		ramo: "COURACEIRO (p69): a Inteligência entra 'quando usa armadura', e a " +
			"armadura leve é a que sobra do parêntese que exclui a pesada",
		class: "Inventor", level: 5, armadura: "armadura-couro",
		poderes: `["class.inventor.couraceiro"]`,
		total:   14, atributo: "intelligence", procedeu: "Couraceiro",
		aritmetca: "10 + INT 2 + 2 da armadura",
	}, {
		ramo:  "BLINDAGEM (p69): é ela, e não o Couraceiro, que atravessa a armadura pesada",
		class: "Inventor", level: 8, armadura: "armadura-completa",
		poderes: `["class.inventor.couraceiro","class.inventor.blindagem"]`,
		total:   22, atributo: "intelligence", procedeu: "Blindagem",
		aritmetca: "10 + INT 2 + 10 da armadura",
	}}

	for _, caso := range casos {
		got := defesa(caso.class, caso.level, caso.armadura, caso.poderes)
		if got.Total != caso.total {
			t.Errorf("%s\n  a Defesa deu %d, e é %d — %s", caso.ramo, got.Total,
				caso.total, caso.aritmetca)
		}
		// O ATRIBUTO e a PROCEDÊNCIA, porque o número certo pelo atributo
		// errado é a coincidência que uma ficha de Car 4 e Int 4 esconderia — e
		// porque é o que a linha da decomposição vai escrever na tela.
		if got.Attribute != caso.atributo {
			t.Errorf("%s\n  o atributo da Defesa foi %q, e é %q", caso.ramo,
				got.Attribute, caso.atributo)
		}
		if got.AttributeSource != caso.procedeu {
			t.Errorf("%s\n  a procedência da troca foi %q, e é %q", caso.ramo,
				got.AttributeSource, caso.procedeu)
		}
	}
}

// E A TROCA É TROCA: o atributo alternativo não SOMA com a Destreza.
//
// "Se fizer isso, não pode somar sua Destreza, mesmo que outras habilidades
// permitam" (Blindagem, p69). Um personagem com Destreza alta e o poder não
// pode ter os dois — e, pela mesma razão, ele fica com a MAIOR das duas.
func TestTheAlternativeAttributeReplacesDexterityInsteadOfAddingToIt(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	// O nobre AO CONTRÁRIO do caso de cima: Destreza 4 contra Carisma 2.
	ch := Character{
		Level: 1, Dexterity: 4, Charisma: 2,
		Classes:       []CharacterClass{{ClassName: "Nobre", Level: 1}},
		Proficiencies: `["armaduras-leves"]`,
		Items: []CharacterItem{{
			CatalogID: ptr("armadura-couro"), Name: "Armadura de couro",
			Equipped: ptr("vested"), Improvements: "[]",
		}},
	}
	defense := BookRuleset(catalogs).ComputeSheet(ch, map[string]bool{}).Defense
	// 10 + a MAIOR entre DES 4 e CAR 2 + 2 da armadura = 16, e nunca 10+4+2+2.
	if defense.Total != 16 {
		t.Errorf("a Defesa deu %d: a Destreza 4 vence o Carisma 2, e os dois juntos "+
			"seriam a soma que a p69 proíbe", defense.Total)
	}
	if defense.Attribute != "dexterity" || defense.AttributeSource != "" {
		t.Errorf("o atributo foi %q por %q: quando a Destreza ganha, ela entra SEM "+
			"procedência — não é licença de poder nenhum", defense.Attribute,
			defense.AttributeSource)
	}
}
