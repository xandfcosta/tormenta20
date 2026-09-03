package reader

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"t20engine/web/bookui"
	"t20engine/web/routes"
	"t20engine/web/ui"
)

// A PORTA do LEITOR (ALE-278). Três métodos, e nenhum deles toca o disco.
//
// A metade que toca o disco ficou no `api`: ler `LIVRO_PDF`, `os.Stat`, cunhar o
// dígito de cache, avisar sobre linearização e servir o arquivo com faixas. Uma
// cena que recebesse a `plataforma.Config` para descobrir onde o PDF está teria
// o hospedeiro dentro dela.
//
// # A pergunta que o VALOR já responde
//
// "Há livro configurado?" não entrou na porta. Sem `LIVRO_PDF` o hospedeiro não
// monta endereço nenhum, então `BookAddress().Base == ""` diz isso — e um método
// a mais para uma pergunta que o valor já responde é assinatura sem informação.
// É a regra da menor pergunta chegando ao limite dela: às vezes a menor pergunta
// é nenhuma.
type Deps interface {
	// BookAddress é onde o PDF está, e por quantas páginas ele começa antes da
	// página impressa 1 (a `abertura`).
	BookAddress() bookui.BookAddress
	// Asset é o endereço VERSIONADO de um estático. O leitor precisa de dois — o
	// módulo do pdf.js e o worker dele —, e os dois são `go:embed` do hospedeiro.
	//
	// Ele pede o resolvedor e não os dois endereços prontos porque o que varia é
	// só o nome do arquivo: dois métodos seriam a mesma pergunta duas vezes.
	Asset(arquivo string) string
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// Scene é a cena montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }

// Routes registra a rota do leitor.
//
// Ela é IRMÃ do `/livro`, que serve o arquivo e continua no hospedeiro: quem
// quiser o PDF cru — imprimir, buscar no visualizador do navegador — tem o
// endereço de sempre. Rotas irmãs de propósito, e agora em pacotes diferentes
// pela mesma razão que a porta acima descreve.
func Routes(r chi.Router, s Scene) {
	r.Get(routes.Reader, s.handleReader)
}
