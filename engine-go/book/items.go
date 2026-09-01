package book

import "strings"

// A BUSCA de item no catálogo, por id e por nome.
//
// Elas ficaram para trás quando o catálogo tipado saiu (ALE-278, segunda
// camada) porque moravam num arquivo da MOCHILA, e só apareceram quando a forja
// tentou sair: ela precisa achar o item que o kit inicial nomeia.

// itemDoLivroPorID é a busca por id no acervo já ordenado.
func ItemByID(id string) *Item {
	for i, entrada := range Catalogs().Itens {
		if entrada.ID == id {
			return &Catalogs().Itens[i]
		}
	}
	return nil
}

// bookItemByName é a busca por NOME, sem diferenciar maiúsculas.
//
// Existe porque nem toda procedência cita o item por id: a linha "Itens" de uma
// origem cita "Símbolo sagrado" por escrito (p85), e é o nome que tem de achar
// a entrada do livro para a linha nascer com o preço e os espaços certos.
func ItemByName(nome string) *Item {
	procurado := strings.ToLower(strings.TrimSpace(nome))
	for i, entrada := range Catalogs().Itens {
		if strings.ToLower(entrada.Name) == procurado {
			return &Catalogs().Itens[i]
		}
	}
	return nil
}
