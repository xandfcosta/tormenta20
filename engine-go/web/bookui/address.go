package bookui

import (
	"net/url"
	"strconv"

	"t20engine/web/routes"
)

// BookAddress é o endereço público do livro e a ABERTURA do arquivo.
//
// Zero valor significa "não há livro", e é ele que as cenas recebem quando
// `LIVRO_PDF` não está configurado — por isso `AtPage` devolve string vazia
// em vez de um link quebrado.
type BookAddress struct {
	Base     string
	Abertura int
}

// AtPage devolve o endereço que abre o livro na página IMPRESSA pedida, com o
// verbete destacado.
//
//	v.Livro.AtPage(289, "Lobo") // → "/livro/ler?p=289&t=Lobo"
//
// Aponta para o LEITOR e não para o PDF cru, e a troca é medida: o visualizador
// do Chrome obedece `#page=N` e IGNORA `#search=` — não há como pedir destaque
// por URL —, e ainda transfere o arquivo inteiro para mostrar uma página (85 MiB
// contados no loopback). O leitor da casa (`api/piloto/src/leitor.ts`)
// resolve os dois: destaca o termo e pede faixas.
//
// A ABERTURA não entra aqui: quem soma é o leitor, que fala em página impressa
// com quem lê e em página de arquivo com o pdf.js. Ver
// `plataforma.Config.LivroAbertura` para a medição do 6.
func (l BookAddress) AtPage(pagina int, termo string) string {
	if l.Base == "" || pagina <= 0 {
		return ""
	}
	endereco := routes.Reader + "?p=" + strconv.Itoa(pagina)
	if termo != "" {
		endereco += "&t=" + url.QueryEscape(termo)
	}
	return endereco
}
