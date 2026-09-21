package bookui

import (
	"t20engine/domain/book"
)

// PowerID devolve o id do poder concedido pelo deus, ou vazio se ele não tem
// verbete no acervo.
func PowerID(name string) string {
	for _, p := range book.Catalogs().Powers {
		if p.Name == name {
			return p.ID
		}
	}
	return ""
}
