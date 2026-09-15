package reader

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"t20engine/serve/web/bookui"
	"t20engine/serve/web/routes"
	"t20engine/serve/web/ui"
)

// A PORTA do LEITOR, e nenhum dos três métodos toca o disco: ler `LIVRO_PDF`,
// cunhar o dígito de cache e servir o arquivo com faixas ficou no hospedeiro.
//
// "Há livro configurado?" NÃO entrou na porta: sem `LIVRO_PDF` não há endereço,
// então `BookAddress().Base == ""` já responde. Método para uma pergunta que o
// valor responde é assinatura sem informação.
type Deps interface {
	// BookAddress é onde o PDF está, e por quantas páginas ele começa antes da
	// página impressa 1 (a `abertura`).
	BookAddress() bookui.BookAddress
	// Asset é o resolvedor e não os dois endereços prontos (o pdf.js e o worker
	// dele): o que varia é só o nome do arquivo.
	Asset(arquivo string) string
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// Scene é a cena montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }

// Routes registra a rota do leitor. Ela é IRMÃ do `/livro`, que serve o arquivo
// cru e continua no hospedeiro — quem quiser imprimir tem o endereço de sempre.
func Routes(r chi.Router, s Scene) {
	r.Get(routes.Reader, s.handleReader)
}
