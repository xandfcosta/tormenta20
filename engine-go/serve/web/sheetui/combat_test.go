package sheetui

import (
	"fmt"
	"strings"
	"testing"

	"t20engine/domain/engine"
)

// Os guardas do painel de COMBATE.
//
// O que eles prendem é a REGRA e a DECISÃO: que os condicionais ligados entram
// na conta, que as linhas de um diálogo somam o total que ele mostra, e quem vê
// cada bloco. Os NÚMEROS em si são do motor e já têm o oráculo de paridade —
// repetir aqui a ficha inteira seria a mesma tabela escrita duas vezes.

// Uma ficha com atributos e perícias de VERDADE.
//
// O `seedCharacterAtLevel` deixa todo atributo em zero, e sobre zeros um painel
// de combate inteiro fica indistinguível de um painel vazio: "+0" em toda caixa
// passaria igual se a conta não acontecesse. Os valores abaixo são escolhidos
// para que cada número da tela seja DIFERENTE dos outros.
//
// Nível 3, e daí saem à mão: ½ nível = 1, e o treino de uma perícia treinada
// vale +2 até o 6º (`trainingBonusForLevel`).
// AS LINHAS DE UM DIÁLOGO SOMAM O TOTAL QUE ELE MOSTRA.
//
// É a promessa que um diálogo de decomposição faz, e a mais fácil de quebrar sem
// perceber: o `Defense.Base` do motor é 10 + o atributo num campo só, enquanto a
// tela mostra o 10 numa linha e o atributo na seguinte. Usar o campo cru
// contaria o atributo DUAS vezes — a Defesa da caixa continuaria certa e só o
// diálogo mentiria, que é o defeito que ninguém vê até alguém somar.
//
// Os três ramos importam, e o terceiro é a p69/p79: a Destreza aplicada, a
// Destreza bloqueada por armadura pesada, e OUTRO atributo no lugar dela.
func TestTheDefenseRowsAddUpToTheTotal(t *testing.T) {
	cases := []struct {
		name      string
		attribute string
		amount    int
		source    string
	}{
		{name: "com a Destreza aplicada", attribute: "dexterity", amount: 3},
		{name: "com a Destreza bloqueada pela armadura pesada"},
		{name: "com o Carisma no lugar da Destreza", attribute: "charisma",
			amount: 4, source: "Autoconfiança"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// O motor embute o atributo no `Base`, e o total sobe junto.
			sheet := engine.ComputedSheet{
				Defense: engine.DefenseBreakdown{
					Base: 10 + tc.amount, Total: 15 + tc.amount,
					VsMelee: 15 + tc.amount, VsRanged: 15 + tc.amount,
					Attribute: tc.attribute, AttributeAmount: tc.amount,
					AttributeSource: tc.source,
					Contributions:   []engine.BreakdownContribution{{Source: "Armadura", Amount: 5}},
				},
			}

			soma := 0
			for _, row := range defenseRows(sheet) {
				soma += rowValue(t, row.Value)
			}
			if soma != sheet.Defense.Total {
				t.Errorf("as linhas somam %d e a caixa mostra %d: o diálogo mente sobre de onde vem a Defesa",
					soma, sheet.Defense.Total)
			}
		})
	}
}

// E A LINHA DIZ DE ONDE SAIU A LICENÇA (p69, p79, ALE-443).
//
// Um "Carisma +4" solto na decomposição de um nobre de armadura pesada se lê
// como defeito justamente para quem conhece a p152 — "se usar uma armadura
// pesada, você não aplica sua Destreza na Defesa". O nome do poder é o que
// explica a exceção, e sem ele a tela dá o número certo com cara de erro.
func TestTheSubstitutedAttributeRowNamesThePowerThatAllowedIt(t *testing.T) {
	sheet := engine.ComputedSheet{
		Defense: engine.DefenseBreakdown{
			Base: 14, Total: 24, VsMelee: 24, VsRanged: 24,
			Attribute: "charisma", AttributeAmount: 4,
			AttributeSource: "Armadura Brilhante",
			Contributions:   []engine.BreakdownContribution{{Source: "Armadura completa", Amount: 10}},
		},
	}
	linha := defenseRows(sheet)[1]
	if linha.Label != "Carisma (Armadura Brilhante)" {
		t.Errorf("a linha do atributo é %q, e ela tem de dizer os dois: o atributo que "+
			"entrou e o poder que o autorizou", linha.Label)
	}
	if linha.Value != "+4" {
		t.Errorf("a linha do atributo vale %q e o Carisma trocado vale +4", linha.Value)
	}
	if linha.Muted {
		t.Error("a linha está apagada: só a Destreza BLOQUEADA se apaga, e esta entrou na conta")
	}
}

// A DESTREZA BLOQUEADA APARECE ZERADA, E A LINHA NOMEIA O ITEM QUE A CALOU
// (ALE-444).
//
// Sumir seria a resposta errada para a pergunta que o diálogo existe para
// responder: quem veste armadura pesada quer ver POR QUE a Defesa não subiu com
// a Destreza dele, e uma linha ausente não diz nada.
//
// E o motivo sai do MOTOR. Escrito à mão aqui, ele dizia "bloqueada por
// armadura pesada" — uma categoria que a tela não tinha como conferir, e que a
// ficha de um ladino de armadura de COURO desmentia.
func TestBlockedDexterityComesOutAsADimmedRow(t *testing.T) {
	sheet := engine.ComputedSheet{
		Defense: engine.DefenseBreakdown{
			Base: 10, Total: 10, VsMelee: 10, VsRanged: 10,
			AttributeBlockedBy: "Cota de malha",
		},
	}
	rows := defenseRows(sheet)
	if len(rows) < 2 {
		t.Fatalf("a Defesa saiu com %d linhas: nem a base e a Destreza estão lá", len(rows))
	}
	dex := rows[1]
	if !strings.Contains(dex.Label, "bloqueada por Cota de malha") {
		t.Errorf("a linha da Destreza é %q, e ela tem de nomear a peça que a calou", dex.Label)
	}
	if dex.Value != "+0" {
		t.Errorf("a Destreza bloqueada vale %q e devia valer +0: ela não entra na Defesa", dex.Value)
	}
	if !dex.Muted {
		t.Error("a linha da Destreza bloqueada não está apagada: ela se lê como um bônus que existe")
	}
}

// QUEM VÊ O BLOCO DE ARMA, e a regra não é "quem tem arma".
//
// É DECISÃO com três ramos, e o do meio é o que se perde num porte: o marcial de
// mãos livres vê o texto de vazio, para a caixa não parecer quebrada, enquanto o
// conjurador de mãos livres não vê o bloco — para ele o assunto é a tripla
// mágica, e um "nenhuma arma empunhada" seria ruído sobre o que ele nunca teve.
func TestTheWeaponBlockFollowsWhoWieldsAndWhoCasts(t *testing.T) {
	oneWeapon := []engine.WeaponCard{{Name: "Machado", Skill: "Luta", Damage: "1d12", CritRange: 20, CritMult: 3}}
	cases := []struct {
		name      string
		cards     []engine.WeaponCard
		caster    bool
		wantBlock bool
		wantCards int
	}{
		{"o marcial que empunha vê o cartão", oneWeapon, false, true, 1},
		{"o marcial de mãos livres vê o texto de vazio", nil, false, true, 0},
		{"o conjurador que empunha vê o cartão", oneWeapon, true, true, 1},
		{"o conjurador de mãos livres não vê o bloco", nil, true, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			panel := panelForCombat(engine.ComputedSheet{}, tc.cards, tc.caster)
			if panel.ShowWeapons != tc.wantBlock {
				t.Errorf("ShowWeapons = %v, quer %v", panel.ShowWeapons, tc.wantBlock)
			}
			if len(panel.Weapons) != tc.wantCards {
				t.Errorf("saíram %d cartões, quer %d", len(panel.Weapons), tc.wantCards)
			}
		})
	}
}

// A TRIPLA MÁGICA SÓ SAI PARA QUEM CONJURA POR CLASSE.
func TestTheSpellTripletOnlyShowsForWhoCasts(t *testing.T) {
	if tiles := panelForCombat(engine.ComputedSheet{}, nil, false).MagicTiles; len(tiles) != 0 {
		t.Errorf("quem não conjura recebeu %d caixas mágicas", len(tiles))
	}
	tiles := panelForCombat(engine.ComputedSheet{}, nil, true).MagicTiles
	if len(tiles) != 3 {
		t.Fatalf("o conjurador recebeu %d caixas mágicas, quer 3", len(tiles))
	}
	for _, tile := range tiles {
		if !tile.Magic {
			t.Errorf("a caixa %q saiu com a paleta de combate em vez da arcana", tile.Label)
		}
	}
}

// rowValue lê o "+5" de uma linha de volta para inteiro.
func rowValue(t *testing.T, text string) int {
	t.Helper()
	var n int
	if _, err := fmt.Sscanf(strings.Replace(text, "+", "", 1), "%d", &n); err != nil {
		t.Fatalf("a linha tem o valor %q, que não é um número com sinal", text)
	}
	return n
}

// COM A ACUIDADE COM ARMA, A PARCELA DO DANO É A DESTREZA, e o rótulo a segue:
// o número já era o da Destreza, e a linha dizia "FOR".
func TestTheFinesseDamageRowSaysDexterity(t *testing.T) {
	_, damage := weaponRows(engine.WeaponCard{Skill: "Luta", StrDamage: 3, DexDamage: true})
	if len(damage) == 0 || damage[0].Label != "DES" || damage[0].Value != "+3" {
		t.Errorf("a primeira parcela do dano com Acuidade não é DES +3: %+v", damage)
	}
}
