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
