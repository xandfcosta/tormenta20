package sheetui

import (
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O CRACHÁ QUE NÃO ALCANÇA O PORTADOR FICA RISCADO, E DIZ QUEM O ANULOU.
//
// O cartão da armadura anuncia "Deslocamento −3". Desde a ALE-383 o anão é
// isento disso (p20, "Devagar e Sempre"), e o crachá passou a MENTIR para ele.
//
// Sumir com o crachá seria a resposta errada, e a casa já decidiu isso uma vez:
// o `TestBlockedDexterityComesOutAsADimmedRow` mantém a Destreza bloqueada como
// linha zerada porque *"quem veste armadura pesada quer ver POR QUE a Defesa
// não subiu"*. Aqui é o mesmo: quem lê a ficha quer ver que a armadura CUSTARIA
// 3m e que a raça o poupa.
//
// O CONTROLE é o humano com a MESMA armadura: nele o crachá tem de continuar
// ativo e sozinho. Sem ele, uma isenção aplicada a todo mundo passaria por
// "o anão está certo".
func TestTheArmorChipIsStruckForWhoeverIsExemptAndNotForTheRest(t *testing.T) {
	plate := "armadura-completa"
	vested := "vested"
	item := sheet.ItemDTO{Name: "Armadura completa", CatalogID: &plate, Equipped: &vested}

	anao := wearerExemptions{engine.DisplacementIgnoresArmorAndLoad: "Devagar e Sempre"}

	for _, caso := range []struct {
		quem     string
		exempt   wearerExemptions
		riscado  bool
		companho string
	}{
		{"anão", anao, true, "isento por Devagar e Sempre"},
		{"humano", wearerExemptions{}, false, ""},
	} {
		t.Run(caso.quem, func(t *testing.T) {
			chips := thatGrantsItem(item, caso.exempt)

			achou := false
			for i, chip := range chips {
				if chip.Text != "Deslocamento -3" {
					continue
				}
				achou = true
				if chip.Inactive != caso.riscado {
					t.Fatalf("%s: o crachá %q veio Inactive=%v, esperava %v",
						caso.quem, chip.Text, chip.Inactive, caso.riscado)
				}
				if caso.companho == "" {
					continue
				}
				if i+1 >= len(chips) || chips[i+1].Text != caso.companho {
					t.Fatalf("%s: faltou o crachá %q logo depois do riscado.\nCrachás: %+v",
						caso.quem, caso.companho, chips)
				}
			}
			if !achou {
				t.Fatalf("%s: o crachá do deslocamento SUMIU do cartão — ele descreve o item e "+
					"tem de ficar.\nCrachás: %+v", caso.quem, chips)
			}
			// O controle de que o cartão não virou outra coisa: a Defesa da
			// armadura continua lá e ativa nos dois casos.
			for _, chip := range chips {
				if chip.Text == "Defesa +10" && chip.Inactive {
					t.Fatalf("%s: a Defesa da armadura foi riscada junto — a isenção vazou", caso.quem)
				}
			}
		})
	}
}

// exemptionsOf lê a habilidade do CATÁLOGO, e o nome que ela devolve é o do
// livro — não um rótulo inventado na tela.
func TestExemptionsComeFromTheRaceCatalogWithTheAbilityName(t *testing.T) {
	dto := sheet.CharacterDTO{Races: []sheet.RaceDTO{{Race: "Anão"}}}
	got := exemptionsOf(dto)

	if name := got[engine.DisplacementIgnoresArmorAndLoad]; name != "Devagar e Sempre" {
		t.Fatalf("a isenção do anão veio como %q, esperava %q — o nome sai do `race-defs.json`",
			name, "Devagar e Sempre")
	}
	if len(exemptionsOf(sheet.CharacterDTO{Races: []sheet.RaceDTO{{Race: "Humano"}}})) != 0 {
		t.Fatal("o humano veio com isenção: o varredor está pegando habilidade que não concede flag")
	}
}
