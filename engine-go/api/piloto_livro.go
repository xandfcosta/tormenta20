package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"t20engine/plataforma"
	"t20engine/web/ui"
)

// O LIVRO servido pela mesa (ALE-264): o Tormenta 20 em PDF, entregue pelo
// próprio servidor, para o botão "abrir no livro" cair na página certa em
// QUALQUER navegador da mesa — e não só na máquina de quem tem o arquivo.
//
// Servir foi decisão do dono, e a forma é por CONFIGURAÇÃO (`LIVRO_PDF`): sem
// ela nada é servido e nenhum botão aparece. O PDF está fora do módulo Go e é
// ignorado pelo git, então `go:embed` não o alcança — e embutir 89 MB no
// binário seria o oposto do que a premissa de "um binário" quer dizer.
//
// DIVIDIR o livro em seções foi rejeitado antes de virar código: quebraria o
// `#page=N`, que é justamente o que faz o botão apontar para a página certa, e
// cobraria um mapa página→arquivo para sempre.

// enderecoDoLivro é o endereço público do livro e a ABERTURA do arquivo.
//
// Zero valor significa "não há livro", e é ele que as cenas recebem quando
// `LIVRO_PDF` não está configurado — por isso `naPagina` devolve string vazia
// em vez de um link quebrado.
type enderecoDoLivro struct {
	Base     string
	Abertura int
}

// naPagina devolve o endereço que abre o livro na página IMPRESSA pedida, com o
// verbete destacado.
//
//	v.Livro.naPagina(289, "Lobo") // → "/livro/ler?p=289&t=Lobo"
//
// Aponta para o LEITOR e não para o PDF cru, e a troca é medida: o visualizador
// do Chrome obedece `#page=N` e IGNORA `#search=` — não há como pedir destaque
// por URL —, e ainda transfere o arquivo inteiro para mostrar uma página (85 MiB
// contados no loopback). O leitor da casa (`frontend/src/piloto/leitor.ts`)
// resolve os dois: destaca o termo e pede faixas.
//
// A ABERTURA não entra aqui: quem soma é o leitor, que fala em página impressa
// com quem lê e em página de arquivo com o pdf.js. Ver
// `plataforma.Config.LivroAbertura` para a medição do 6.
func (l enderecoDoLivro) naPagina(pagina int, termo string) string {
	if l.Base == "" || pagina <= 0 {
		return ""
	}
	endereco := rotaDoLeitor + "?p=" + strconv.Itoa(pagina)
	if termo != "" {
		endereco += "&t=" + url.QueryEscape(termo)
	}
	return endereco
}

// livroServido é o que o servidor guarda: onde o arquivo está e como falar dele.
type livroServido struct {
	caminho  string
	digito   string
	endereco enderecoDoLivro
}

// rotaDoLivro é o caminho DENTRO do piloto; o público leva o `/` na frente
// porque é isso que o navegador pede (o `buildMux` monta com `StripPrefix`).
const rotaDoLivro = "/livro"

// rotaDoLeitor é o endereço PÚBLICO da cena que desenha o livro. Ela não é
// versionada porque é uma página HTML servida com `no-store`; quem carrega
// versão é o PDF que ela pede.
const rotaDoLeitor = "/livro/ler"

// abreOLivro lê a configuração UMA vez, no boot.
//
// Ausência de arquivo é degradação normal e não queda: a mesa inteira funciona
// sem o livro, e derrubar o servidor por causa de um botão seria trocar um
// problema pequeno por um grande. O aviso vai para o log com o caminho que
// falhou, porque configurar e não ver o botão é o sintoma sem explicação.
func abreOLivro(cfg plataforma.Config) livroServido {
	if cfg.LivroPDF == "" {
		return livroServido{}
	}
	info, err := os.Stat(cfg.LivroPDF)
	if err != nil || info.IsDir() {
		log.Printf("livro: %s não serve como PDF (%v) — o botão de abrir no livro não vai aparecer", cfg.LivroPDF, err)
		return livroServido{}
	}
	avisaSeNaoLinearizado(cfg.LivroPDF)
	digito := digitoDoLivro(info)
	return livroServido{
		caminho: cfg.LivroPDF,
		digito:  digito,
		endereco: enderecoDoLivro{
			Base:     rotaDoLivro + "?v=" + digito,
			Abertura: cfg.LivroAbertura,
		},
	}
}

// digitoDoLivro versiona o endereço a partir do TAMANHO e da data do arquivo,
// e não do conteúdo.
//
// A diferença é medida: somar os 89 MB custa uma leitura do arquivo inteiro em
// todo boot, para invalidar um cache que só muda quando alguém TROCA o arquivo
// — e trocar um arquivo muda o tamanho ou a data. É o mesmo par que qualquer
// servidor de arquivos usa para cunhar `ETag`.
func digitoDoLivro(info os.FileInfo) string {
	soma := sha256.Sum256(fmt.Appendf(nil, "%d-%d", info.Size(), info.ModTime().UnixNano()))
	return hex.EncodeToString(soma[:])[:12]
}

// avisaSeNaoLinearizado diz, no boot, que o arquivo configurado não passou pelo
// `qpdf --linearize`.
//
// O QUE FOI MEDIDO, e ele desmente a suposição com que isto começou. A aposta
// era que a linearização faria o visualizador pedir só as FAIXAS da página, e
// não faz — pelo menos não no Chrome, abrindo o PDF como navegação de topo.
// Contando os bytes da interface de loopback (ruído aferido antes: 0 KiB em 4s,
// duas vezes), abrir `#page=295` transferiu o ARQUIVO INTEIRO nos dois casos:
// 85 MiB com o PDF cru de 89.489.751 bytes, 75 MiB com o linearizado de
// 78.622.788. Os dois números batem com o tamanho do arquivo, que é o controle
// da leitura.
//
// Então o ganho MEDIDO do `qpdf` é de tamanho — 12% —, e não de faixas. A
// renderização progressiva que a linearização promete não foi medida daqui: no
// localhost a transferência termina antes de haver o que ver, e medi-la exigiria
// estrangular o link. Quem for afirmá-la, meça antes.
//
// Aviso e não conserto: linearizar aqui obrigaria o servidor a depender do
// `qpdf` instalado e a gravar um segundo arquivo de 78 MB no boot. O que o
// servidor pode fazer barato é NOMEAR o comando.
func avisaSeNaoLinearizado(caminho string) {
	f, err := os.Open(caminho)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	// O dicionário de linearização é o PRIMEIRO objeto do arquivo, por
	// definição: se ele não está no começo, ele não existe.
	cabeca := make([]byte, 2048)
	n, _ := io.ReadFull(f, cabeca)
	if ehLinearizado(cabeca[:n]) {
		return
	}
	log.Printf("livro: %s não está linearizado — medido, o navegador transfere o arquivo inteiro para abrir uma página, e o qpdf ainda o encolhe 12%%. Conserto: qpdf --linearize %s %s", caminho, caminho, caminho+".linear")
}

// ehLinearizado procura a marca do PDF linearizado no começo do arquivo.
func ehLinearizado(cabeca []byte) bool {
	return strings.Contains(string(cabeca), "/Linearized")
}

// ── a cena do leitor ─────────────────────────────────────────────────────────

// leitorView é o que a cena precisa saber: onde está o PDF, em que página abrir,
// o que destacar e para onde voltar.
type leitorView struct {
	PDF      string
	Pagina   int
	Abertura int
	Termo    string
	Voltar   string
	// EmDialogo diz que esta cena está dentro do `<iframe>` do diálogo que a
	// casca desenha, e não numa aba própria. A diferença na tela é o link de
	// VOLTAR: dentro do diálogo ele levaria o iframe para a cena anterior, com a
	// mesa aparecendo miniaturizada dentro de uma caixa — quem fecha é o ✕ do
	// diálogo, que é do lado de fora.
	EmDialogo bool
}

// carregaOLeitor lê a URL e recusa o que não faz sentido.
//
// A página vem da URL porque este endereço é COMPARTILHÁVEL: o mestre manda
// "olha na p289" no chat da mesa, e o link tem de abrir lá. Página fora do livro
// cai na primeira em vez de derrubar a cena — endereço se digita à mão.
func (s *Server) carregaOLeitor(r *http.Request) leitorView {
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
	return leitorView{
		PDF:       s.livro.endereco.Base,
		Pagina:    pagina,
		Abertura:  s.livro.endereco.Abertura,
		Termo:     r.URL.Query().Get("t"),
		Voltar:    voltar,
		EmDialogo: r.URL.Query().Get("dialogo") != "",
	}
}

func (s *Server) handleLeitorDoLivro(w http.ResponseWriter, r *http.Request) {
	if s.livro.caminho == "" {
		http.NotFound(w, r)
		return
	}
	v := s.carregaOLeitor(r)
	titulo := "Livro · Tormenta 20"
	if v.Termo != "" {
		titulo = v.Termo + " · Livro · Tormenta 20"
	}
	s.escrevePagina(w, r, http.StatusOK, ui.Page{
		Titulo: titulo,
		Forma:  ui.ShellBare,
		Voltar: v.Voltar,
		// O módulo do leitor só entra AQUI: são 540 KB de pdf.js, e mandá-los em
		// toda cena seria pôr um visualizador de PDF no caminho de quem abriu a
		// ficha de um personagem.
		Scripts: []string{EstaticoDoPiloto("leitor.js")},
	}, leitorDoLivro(v))
}

// LivroDoPiloto serve o PDF configurado, com faixas.
//
// `http.ServeFile` responde `Range` sozinho — é isso que faz o visualizador de
// PDF pedir só os pedaços da página quando o arquivo é linearizado.
//
// O alcance do cache é `private` porque esta rota sai DEPOIS do `requirePagina`:
// `public` autorizaria um cache compartilhado a guardar e reentregar o livro de
// alguém que entrou para quem não entrou.
func (s *Server) LivroDoPiloto() http.Handler {
	// Sem livro a rota é 404 e PRONTO — o 404 não passa pela política de cache,
	// e essa ordem é conserto de um vermelho: um dígito VAZIO fazia
	// `strings.Contains(ifNoneMatch, "")` responder verdadeiro, e a rota
	// devolvia 304 para todo mundo em vez de 404. Visto no
	// `TestSemConfiguracaoARotaDoLivroDa404`, que nasceu vermelho por isso.
	if s.livro.caminho == "" {
		return http.NotFoundHandler()
	}
	return comCacheVersionado(s.livro.digito, "private", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.livro.caminho)
	}))
}
