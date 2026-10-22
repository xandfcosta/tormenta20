package sheetui

import (
	"strings"
	"testing"

	"t20engine/domain/book"
)

// A FICHA DO ITEM DIZ QUE MAGIA ESTÁ NO FRASCO (p341, ALE-451).
//
// Uma poção não TEM regra: ela aponta para uma — *"contêm o efeito de uma
// magia. Quando são ativados, geram o efeito dessa magia e então desaparecem"*.
// Sem dizer QUAL, o cartão dela é um nome e um preço, e quem bebe não sabe o
// que aconteceu.
//
// E o app INFORMA, não resolve: a área da Bola de Fogo, o alvo do Sono e a
// duração da Invisibilidade são do mestre. O que a ficha deve é a referência —
// a magia, o círculo dela e o que o aprimoramento daquele frasco comprou.
func TestThePotionSheetNamesTheSpellItHolds(t *testing.T) {
	casos := []struct {
		id       string
		contem   string
		aprimora string
	}{{
		id:     "granada-de-bola-de-fogo",
		contem: "contém Bola de Fogo · 2º círculo · granada",
	}, {
		id:       "pocao-de-invisibilidade-duracao-cena",
		contem:   "contém Invisibilidade · 2º círculo · poção",
		aprimora: "aprimoramento para duração cena",
	}, {
		id:     "oleo-de-arma-magica",
		contem: "contém Arma Mágica · 1º círculo · óleo",
	}}

	for _, caso := range casos {
		entrada := book.ItemByID(caso.id)
		if entrada == nil {
			t.Fatalf("o catálogo não tem %q", caso.id)
		}
		linhas := thatSaysBook(*entrada).Rows
		if !temCracha(linhas, caso.contem) {
			t.Errorf("a ficha de %s não diz %q. Veio %v", caso.id, caso.contem, linhas)
		}
		if caso.aprimora == "" {
			continue
		}
		achou := false
		for _, linha := range linhas {
			if strings.Contains(linha, caso.aprimora) {
				achou = true
			}
		}
		if !achou {
			t.Errorf("a ficha de %s não diz o aprimoramento %q. Veio %v",
				caso.id, caso.aprimora, linhas)
		}
	}
}

// O ATRIBUTO APARECE EM PORTUGUÊS NO CRACHÁ (ALE-452).
//
// O alvo `attribute` guarda a chave do MOTOR — `charisma` —, e o crachá a
// mostrava crua: "Atributo (charisma) +2" na Coroa majestosa. Identificador na
// tela é o mesmo defeito do `damageStep` cru, e ele só apareceu quando um item
// passou a somar atributo (a Couraça do comando e os quinze acessórios que
// somam +1 ou +2).
func TestTheAttributeBadgeSpeaksPortuguese(t *testing.T) {
	coroa := book.ItemByID("coroa-majestosa")
	if coroa == nil {
		t.Fatal("o catálogo não tem a Coroa majestosa")
	}
	linhas := thatSaysBook(*coroa).Rows
	if !temCracha(linhas, "Atributo (Carisma) +2") {
		t.Errorf("a ficha da Coroa majestosa não diz \"Atributo (Carisma) +2\". "+
			"Veio %v", linhas)
	}
	for _, linha := range linhas {
		if strings.Contains(linha, "charisma") {
			t.Errorf("a ficha mostra %q, que é a chave do motor", linha)
		}
	}
}
