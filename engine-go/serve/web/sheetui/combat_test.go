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
// perceber: o `Defense.Base` do motor é 10 + Destreza num campo só, enquanto a
// tela mostra o 10 numa linha e a Destreza na seguinte. Usar o campo cru
// contaria a Destreza DUAS vezes — a Defesa da caixa continuaria certa e só o
// diálogo mentiria, que é o defeito que ninguém vê até alguém somar.
//
// Os dois ramos importam: com Destreza aplicada e com ela bloqueada.
func TestTheDefenseRowsAddUpToTheTotal(t *testing.T) {
	cases := []struct {
		name    string
		applied bool
	}{
		{"com a Destreza aplicada", true},
		{"com a Destreza bloqueada por armadura pesada", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sheet := engine.ComputedSheet{
				Defense: engine.DefenseBreakdown{
					Base: 10, Total: 15, VsMelee: 15, VsRanged: 15, DexApplied: tc.applied,
					Contributions: []engine.BreakdownContribution{{Source: "Armadura", Amount: 5}},
				},
				Attributes: map[string]engine.AttributeBreakdown{"dexterity": {Total: 3}},
			}
			if tc.applied {
				// O motor embute a Destreza no `Base`; o total sobe junto.
				sheet.Defense.Base = 13
				sheet.Defense.Total = 18
				sheet.Defense.VsMelee, sheet.Defense.VsRanged = 18, 18
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

// A DESTREZA BLOQUEADA APARECE ZERADA, e não some.
//
// Sumir seria a resposta errada para a pergunta que o diálogo existe para
// responder: quem veste armadura pesada quer ver POR QUE a Defesa não subiu com
// a Destreza dele, e uma linha ausente não diz nada.
func TestBlockedDexterityComesOutAsADimmedRow(t *testing.T) {
	sheet := engine.ComputedSheet{
		Defense:    engine.DefenseBreakdown{Base: 10, Total: 10, VsMelee: 10, VsRanged: 10, DexApplied: false},
		Attributes: map[string]engine.AttributeBreakdown{"dexterity": {Total: 3}},
	}
	rows := defenseRows(sheet)
	if len(rows) < 2 {
		t.Fatalf("a Defesa saiu com %d linhas: nem a base e a Destreza estão lá", len(rows))
	}
	dex := rows[1]
	if !strings.Contains(dex.Label, "bloqueada por armadura pesada") {
		t.Errorf("a linha da Destreza é %q e não diz que ela está bloqueada", dex.Label)
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
