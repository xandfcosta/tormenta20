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

// E A BANDEIRA ISENTA SAI RISCADA, com o motivo ao lado (p340, ALE-449).
//
// A Cota élfica é uma cota de malha — pesada —, então ela ACENDE a bandeira que
// zera a Destreza na Defesa e logo em seguida isenta o portador dela. Desenhar o
// crachá "Não soma Destreza na Defesa" num item cujo verbete existe para desfazer
// isso seria o cartão contradizendo a ficha, que é a fatura da ALE-319.
//
// O par riscado + motivo é o mesmo do deslocamento do anão, e o motivo é a NOTA
// que o verbete já escreve.
func TestTheExemptFlagIsStruckThroughWithItsReason(t *testing.T) {
	vested := "vested"
	cota := "cota-elfica"
	chips := thatGrantsItem(sheet.ItemDTO{
		Name: "Cota élfica", CatalogID: &cota, Equipped: &vested, Improvements: "[]",
	}, wearerExemptions{})

	var bandeira *itemChip
	for i := range chips {
		if chips[i].Text == "Não soma Destreza na Defesa" {
			bandeira = &chips[i]
		}
	}
	if bandeira == nil {
		t.Fatalf("o cartão não diz nada sobre a Destreza na Defesa. Veio %v",
			chipTexts(chips))
	}
	if !bandeira.Inactive {
		t.Error("o crachá da bandeira isenta não está riscado: ele promete uma " +
			"penalidade que este item existe para tirar")
	}
	if !temCracha(chipTexts(chips), "a Destreza se aplica como se fosse armadura leve") {
		t.Errorf("o motivo da isenção não aparece ao lado. Veio %v", chipTexts(chips))
	}
	// E A ISENÇÃO NÃO VIRA CRACHÁ PRÓPRIO de alvo: ela é o motivo, e um
	// "flagExempt" cru na tela seria identificador aparecendo para quem joga.
	for _, chip := range chipTexts(chips) {
		if strings.Contains(chip, "flagExempt") {
			t.Errorf("o cartão mostra %q, que é identificador", chip)
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
