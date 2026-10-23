package book

import "strings"
import "testing"

// O ENCANTO CABE NO QUE O LIVRO MANDA, e são duas tabelas (ALE-437).
//
// A Tabela 8-8 é de ARMAS (p335) e a 8-10 é de ARMADURAS & ESCUDOS (p338): um
// encanto de armadura numa espada não é "um encanto a mais", é outro verbete
// do livro. Enquanto só havia a primeira, a recusa podia perguntar "é arma?";
// agora ela pergunta se a FAMÍLIA do item é uma das que o encanto aceita.
func TestAnEnchantOnlyFitsTheFamilyItsTableNames(t *testing.T) {
	for _, caso := range []struct {
		nome, item, encanto, recusa string
	}{
		{nome: "arma recebe encanto de arma", item: "espada-longa", encanto: "encanto-formidavel"},
		{nome: "armadura recebe encanto de armadura", item: "armadura-completa", encanto: "encanto-sombrio"},
		{nome: "escudo recebe encanto de armadura", item: "escudo-pesado", encanto: "encanto-acrobatico"},
		{
			nome: "armadura NÃO recebe encanto de arma", item: "armadura-completa",
			encanto: "encanto-formidavel", recusa: "não cabe",
		},
		{
			nome: "arma NÃO recebe encanto de armadura", item: "espada-longa",
			encanto: "encanto-sombrio", recusa: "não cabe",
		},
		{
			nome: "o que não é nenhum dos dois não recebe encanto", item: "mochila",
			encanto: "encanto-sombrio", recusa: "não cabe",
		},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			err := FitsItemEnchants(ItemByID(caso.item), []string{caso.encanto})
			if caso.recusa == "" {
				if err != nil {
					t.Errorf("o livro aceita e a recusa veio: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("%s recebeu %s em silêncio", caso.item, caso.encanto)
			}
			if !strings.Contains(err.Error(), caso.recusa) {
				t.Errorf("a recusa não diz o que está errado: %v", err)
			}
		})
	}
}

// O TETO DE TRÊS NÃO ALCANÇA O ARTEFATO, e alcança todo o resto (p334, p346,
// ALE-455).
//
// Os três casos são o desenho inteiro, e o segundo é o que importa: a exceção é
// do ARTEFATO e de mais nada. Sem ele, "o teto deixou de valer para um item" e
// "o teto deixou de valer" têm a mesma cor — e a segunda leitura abriria a porta
// para uma espada longa com quatro encantos de peso, que é o que a escada de
// preço da p334 existe para fechar.
func TestTheCeilingOfThreeDoesNotReachAnArtifact(t *testing.T) {
	// Os três que a Espada-Deus traz, e eles somam PESO 4 porque a Magnífica
	// conta como dois. O número está escrito na mão de propósito: derivá-lo do
	// `CountsAs` do catálogo faria a asserção andar junto com o defeito.
	osTresDaEspadaDeus := []string{"encanto-ameacadora", "encanto-magnifica", "encanto-veloz"}

	espadaDeus := ItemByID("espada-deus")
	if espadaDeus == nil {
		t.Fatal("a Espada-Deus não está no catálogo: o caso não mediria nada")
	}
	if !espadaDeus.Artifact {
		t.Fatal("a Espada-Deus não está marcada como artefato, e é dela que a exceção trata")
	}
	if err := FitsFoundItemEnchants(espadaDeus, osTresDaEspadaDeus); err != nil {
		t.Errorf("a p346 escreve a Espada-Deus com esses encantos, e a recusa veio: %v", err)
	}

	// O CONTROLE: o MESMO conjunto, numa espada longa comum, continua recusado.
	// É o que prova que a exceção é do artefato e não do conjunto.
	espadaLonga := ItemByID("espada-longa")
	if espadaLonga.Artifact {
		t.Fatal("a espada longa está marcada como artefato: o controle mediria o mesmo ramo")
	}
	err := FitsFoundItemEnchants(espadaLonga, osTresDaEspadaDeus)
	if err == nil {
		t.Error("uma espada longa comum recebeu 4 encantos de peso em silêncio")
	} else if !strings.Contains(err.Error(), "no máximo 3") {
		t.Errorf("a recusa da espada comum não fala do teto: %v", err)
	}

	// E a FAMÍLIA continua valendo para o artefato: a exceção é do TETO só.
	err = FitsFoundItemEnchants(espadaDeus, []string{"encanto-sombrio"})
	if err == nil {
		t.Error("a Espada-Deus recebeu um encanto de ARMADURA: ser artefato dispensa o " +
			"teto da escada de fabricação, não a tabela de onde o encanto sai")
	} else if !strings.Contains(err.Error(), "não cabe") {
		t.Errorf("a recusa de família não diz o que está errado: %v", err)
	}
}
