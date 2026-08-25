package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"t20engine/plataforma"
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

// naPagina devolve o endereço que abre o livro na página IMPRESSA pedida.
//
//	v.Livro.naPagina(289) // → "/piloto/livro?v=ab12…#page=295"
//
// A soma da abertura é o miolo disto: `#page=` conta páginas do ARQUIVO e o
// catálogo grava a página impressa no rodapé. Ver `plataforma.Config.LivroAbertura`
// para a medição.
func (l enderecoDoLivro) naPagina(pagina int) string {
	if l.Base == "" || pagina <= 0 {
		return ""
	}
	return l.Base + "#page=" + strconv.Itoa(pagina+l.Abertura)
}

// livroServido é o que o servidor guarda: onde o arquivo está e como falar dele.
type livroServido struct {
	caminho  string
	digito   string
	endereco enderecoDoLivro
}

// rotaDoLivro é o caminho DENTRO do piloto; o público leva o `/piloto` na frente
// porque é isso que o navegador pede (o `buildMux` monta com `StripPrefix`).
const rotaDoLivro = "/livro"

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
			Base:     "/piloto" + rotaDoLivro + "?v=" + digito,
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

// avisaSeNaoLinearizado diz, no boot, se o navegador vai ter de baixar o livro
// INTEIRO antes de mostrar a página pedida.
//
// Medido no PDF da casa: 89.489.751 bytes e `Optimized: no`. Um PDF não
// linearizado não tem a tabela que permite ao visualizador pedir só as faixas
// da página, então `#page=295` custa o arquivo todo. `qpdf --linearize` leva
// 4,5s e devolve 78.622.788 bytes com `/Linearized 1`.
//
// Aviso e não conserto: linearizar aqui obrigaria o servidor a depender do
// `qpdf` instalado e a gravar um segundo arquivo de 78 MB no boot. O que o
// servidor pode fazer barato é NOMEAR o problema e o comando que o resolve.
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
	log.Printf("livro: %s não está linearizado — o navegador baixa o arquivo inteiro antes de mostrar a página. Conserto: qpdf --linearize %s %s", caminho, caminho, caminho+".linear")
}

// ehLinearizado procura a marca do PDF linearizado no começo do arquivo.
func ehLinearizado(cabeca []byte) bool {
	return strings.Contains(string(cabeca), "/Linearized")
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
