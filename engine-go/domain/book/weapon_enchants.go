package book

import "fmt"

// O ENCANTO DA ARMA: quem cabe, quantos cabem, e o que exige o quê (p333-336).
//
// A regra mora no LIVRO e não numa cena, porque ela tem dois chamadores de
// naturezas diferentes: o jogador nunca encanta — quem concede é o mestre —, e
// a escrita dele vem de outra tela. Uma regra escrita dentro de um handler tem
// exatamente um chamador, e o segundo a reescreve de memória.

// slotsDeEncanto é o teto do livro, e ele é o MESMO número que a categoria do
// item mágico: "um item mágico menor possui um encanto, um médio possui dois e
// um item mágico maior possui três encantos" (p334), e a p334 ainda chama três
// de "o máximo possível".
//
// Por isso não há tamanho a guardar na arma: a CONTAGEM é a categoria.
const slotsDeEncanto = 2 + 1

// FitsWeaponEnchants é a recusa do servidor para um conjunto de encantos.
//
// Ela confere quatro coisas: que o item é ARMA, que cada id é encanto do livro,
// que o conjunto cabe no teto, e que cada pré-requisito está no MESMO conjunto.
//
// O TETO conta por PESO e não por quantidade: três encantos da Tabela 8-8
// "contam como dois", e o asterisco deles é regra — dois desses já enchem a
// arma. Contar linhas deixaria passar uma espada com três Magníficas.
//
// E o PRÉ-REQUISITO se confere contra o conjunto ESCOLHIDO, e não contra o que
// a arma já tinha: quem grava manda o conjunto inteiro, e conferir contra o
// estado anterior deixaria tirar o pré-requisito e manter o dependente no mesmo
// pedido.
//
// @example book.FitsWeaponEnchants(book.ItemByID("espada-longa"), []string{"encanto-formidavel"})
func FitsWeaponEnchants(catalog *Item, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if catalog == nil || catalog.Weapon == nil {
		return fmt.Errorf("só arma recebe encanto, e este item não é arma (p333)")
	}
	escolhidos := map[string]bool{}
	peso := 0
	for _, id := range ids {
		encanto := ItemByID(id)
		if encanto == nil || encanto.Category != "weapon-enchant" {
			return fmt.Errorf("%q não é um encanto do livro", id)
		}
		escolhidos[id] = true
		peso += max(encanto.CountsAs, 1)
	}
	if peso > slotsDeEncanto {
		return fmt.Errorf(
			"são %d encantos de peso em %q, e o livro dá no máximo %d (p334) — "+
				"a Energética, a Lancinante e a Magnífica contam como dois",
			peso, catalog.Name, slotsDeEncanto)
	}
	for id := range escolhidos {
		encanto := ItemByID(id)
		if encanto.Requires == "" || escolhidos[encanto.Requires] {
			continue
		}
		nome := encanto.Requires
		if pre := ItemByID(encanto.Requires); pre != nil {
			nome = pre.Name
		}
		return fmt.Errorf("%q exige %q na mesma arma (p335-336)", encanto.Name, nome)
	}
	return nil
}
