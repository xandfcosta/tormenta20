package reader

import (
	"net/http"
	"strconv"
	"strings"

	"t20engine/web/bookui"
	"t20engine/web/ui"
)

// O LEITOR (ALE-264): a cena que abre o PDF do livro na página do verbete e o
// DESTACA.
//
// Ela saiu do `api` na ALE-278, e a OUTRA metade do arquivo original ficou lá:
// ler `LIVRO_PDF` no boot, cunhar o dígito de cache e servir o arquivo com
// faixas. A divisão é por dependência — o que ficou toca `os.Stat` e o disco do
// dono da mesa; esta desenha uma página e não sabe onde o arquivo está, só o
// endereço dele.
//
// **Não é o visualizador do navegador**, que continua a um clique de distância.
// A distinção importa e está no GLOSSARIO: o leitor mostra uma página por vez
// com o termo marcado, o visualizador tem busca, miniaturas e impressão. Ele
// existe por medição — o Chrome ignora `#search=` e transfere o arquivo inteiro
// (85 MiB) para abrir uma página; o leitor destaca e custou 1 MiB.

// readerView é o que a cena precisa saber: onde está o PDF, em que página abrir,
// o que destacar e para onde voltar.
type readerView struct {
	PDF     string
	Page    int
	Opening int
	// Worker é o endereço VERSIONADO do worker do pdf.js, e ele vem no view em
	// vez de a cena chamá-lo direto: os estáticos são `go:embed` do hospedeiro, e
	// um componente `templ` não alcança a porta. É a mesma decisão do
	// `ui.Page.Asset` — a casca RECEBE o que ela não pode conhecer.
	Worker string
	Term   string
	Back   string
	// InDialog diz que esta cena está dentro do `<iframe>` do diálogo que a
	// casca desenha, e não numa aba própria. A diferença na tela é o link de
	// VOLTAR: dentro do diálogo ele levaria o iframe para a cena anterior, com a
	// mesa aparecendo miniaturizada dentro de uma caixa — quem fecha é o ✕ do
	// diálogo, que é do lado de fora.
	InDialog bool
}

// carregaOLeitor lê a URL e recusa o que não faz sentido.
//
// A página vem da URL porque este endereço é COMPARTILHÁVEL: o mestre manda
// "olha na p289" no chat da mesa, e o link tem de abrir lá. Página fora do livro
// cai na primeira em vez de derrubar a cena — endereço se digita à mão.
func readerFromRequest(r *http.Request, livro bookui.BookAddress) readerView {
	pagina, err := strconv.Atoi(r.URL.Query().Get("p"))
	if err != nil || pagina <= 0 {
		pagina = 1
	}
	voltar := r.Referer()
	if voltar == "" || !strings.HasPrefix(voltar, "/") {
		// Referer de outro site (ou nenhum) não vira link de voltar: seria um
		// endereço de terceiro na nossa barra. O Hub é o destino de quem chegou
		// por um link colado.
		voltar = "/"
	}
	return readerView{
		PDF:  livro.Base,
		Page: pagina,
		// `Opening` é nome NOVO e `Abertura` é o nome CHAMADO: a costura PT/EN
		// aparece nesta linha de propósito, e o GLOSSARIO diz por quê.
		Opening:  livro.Abertura,
		Term:     r.URL.Query().Get("t"),
		Back:     voltar,
		InDialog: r.URL.Query().Get("dialogo") != "",
	}
}

func (s Scene) handleReader(w http.ResponseWriter, r *http.Request) {
	livro := s.deps.BookAddress()
	// "Há livro configurado?" é respondida pelo próprio ENDEREÇO estar vazio, e
	// não por um método a mais na porta: sem `LIVRO_PDF` o hospedeiro não monta
	// endereço nenhum. Uma pergunta que o valor já responde não precisa de
	// assinatura.
	if livro.Base == "" {
		http.NotFound(w, r)
		return
	}
	v := readerFromRequest(r, livro)
	v.Worker = s.deps.Asset("pdf.worker.js")
	titulo := "Livro · Tormenta 20"
	if v.Term != "" {
		titulo = v.Term + " · Livro · Tormenta 20"
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: titulo,
		Forma:  ui.ShellBare,
		Voltar: v.Back,
		// O módulo do leitor só entra AQUI: são 540 KB de pdf.js, e mandá-los em
		// toda cena seria pôr um visualizador de PDF no caminho de quem abriu a
		// ficha de um personagem.
		Scripts: []string{s.deps.Asset("leitor.js")},
	}, readerScene(v))
}
