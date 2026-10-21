package sheetui

import (
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// O CARTÃO DIZ O QUE O ITEM COMPOSTO CONCEDE (p337, p340, ALE-445).
//
// O item específico dobra base, melhoria, material e encanto num cartão só, e é
// nele que a pessoa procura "o que esta coisa faz". Os crachás dele passaram a
// mostrar coisas que um item mundano nunca mostrou, e três mentiam:
//
//   - o passo de dano do adamante aparecia numa ARMADURA, onde ele nunca vale;
//   - o dado do encanto saía como um crachá "Dano" sem número;
//   - a redução de fogo saía como "Redução de dano", sem o tipo — que é o que
//     decide contra o que ela vale (p229).
func TestTheComposedItemCardSaysWhatItGrants(t *testing.T) {
	wielded, vested := "wielded", "vested"
	avalanche, baluarte := "avalanche", "baluarte-anao"

	armaChips := chipTexts(thatGrantsItem(sheet.ItemDTO{
		Name: "Avalanche", CatalogID: &avalanche, Equipped: &wielded, Improvements: "[]",
	}, wearerExemptions{}))
	for _, esperado := range []string{
		"Dano 1d12",                  // o dado da BASE, herdado
		"Ataque +2",                  // Formidável (p336)
		"Dano +2",                    // Gelo eterno (p166)
		"Dano (frio) +1d6",           // Congelante (p335), que é DADO e não número
		"Redução de dano (fogo) +10", // a regra própria do verbete (p337)
	} {
		if !temCracha(armaChips, esperado) {
			t.Errorf("o cartão da Avalanche não diz %q. Veio %v", esperado, armaChips)
		}
	}

	armaduraChips := chipTexts(thatGrantsItem(sheet.ItemDTO{
		Name: "Baluarte anão", CatalogID: &baluarte, Equipped: &vested, Improvements: "[]",
	}, wearerExemptions{}))
	for _, esperado := range []string{
		"Defesa +10",         // armadura completa (p154)
		"Defesa +1",          // Reforçada (p166)
		"Defesa +2",          // Defensor (p338)
		"Redução de dano +5", // Adamante em armadura pesada (p166)
	} {
		if !temCracha(armaduraChips, esperado) {
			t.Errorf("o cartão do Baluarte anão não diz %q. Veio %v", esperado, armaduraChips)
		}
	}
	// E NÃO DIZ o que o EIXO DE USO desmente: o passo de dano do adamante é da
	// metade "Arma" do verbete (p166), e esta peça se veste.
	for _, chip := range armaduraChips {
		// Os DOIS textos: o rótulo em português e a chave crua, que é como ele
		// aparecia antes de o alvo ganhar nome na tela.
		if strings.Contains(chip, "passo de dano") || strings.Contains(chip, "damageStep") {
			t.Errorf("o cartão do Baluarte anão promete %q, e ninguém empunha uma "+
				"armadura completa", chip)
		}
	}
}

func chipTexts(chips []itemChip) []string {
	texts := []string{}
	for _, c := range chips {
		texts = append(texts, c.Text)
	}
	return texts
}

func temCracha(chips []string, text string) bool {
	for _, c := range chips {
		if c == text {
			return true
		}
	}
	return false
}
