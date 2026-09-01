package api

import (
	"t20engine/book"
)

// idDoPoder devolve o id do poder concedido pelo deus, ou vazio se ele não tem
// verbete no acervo.
func idDoPoder(nome string) string {
	for _, p := range book.Catalogs().Poderes {
		if p.Name == nome {
			return p.ID
		}
	}
	return ""
}
