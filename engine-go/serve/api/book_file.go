package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"t20engine/serve/web/bookui"
	"t20engine/serve/web/routes"

	"t20engine/infra/config"

	"t20engine/infra/httpio"
)

// O LIVRO servido pela mesa: o Tormenta 20 em PDF, entregue pelo próprio
// servidor, para o botão "abrir no livro" cair na página certa em QUALQUER
// navegador da mesa — e não só na máquina de quem tem o arquivo.
//
// A forma é por CONFIGURAÇÃO (`LIVRO_PDF`): sem ela nada é servido e nenhum
// botão aparece. O PDF está fora do módulo Go e é ignorado pelo git, então
// `go:embed` não o alcança — e embutir 89 MB no binário seria o oposto do que a
// premissa de "um binário" quer dizer.
//
// DIVIDIR o livro em seções quebraria o `#page=N`, que é justamente o que faz o
// botão apontar para a página certa, e cobraria um mapa página→arquivo para
// sempre.

// O ENDEREÇO DO LIVRO mora em `web/bookui`: ele é a assinatura de todos os
// componentes de lá, e ficar aqui obrigaria o pacote de apresentação do livro a
// importar a cena que serve o PDF.

// servedBook é o que o servidor guarda: onde o arquivo está e como falar dele.
type servedBook struct {
	path    string
	digest  string
	address bookui.BookAddress
}

// O endereço do leitor mora em `web/routes` porque o `bookui` o cita para montar
// o selo de página. Ele não é versionado: é uma página HTML servida com
// `no-store`, e quem carrega versão é o PDF que ela pede.

// A divisão com `web/reader` é por DEPENDÊNCIA e não por tamanho: o que ficou
// aqui lê `config.Config`, chama `os.Stat` e devolve um `http.Handler` sobre
// um arquivo do disco do dono da mesa. Uma cena que recebesse a `Config` para
// saber onde o PDF está teria o hospedeiro dentro dela.

// openServedBook lê a configuração UMA vez, no boot.
//
// Ausência de arquivo é degradação normal e não queda: a mesa inteira funciona
// sem o livro, e derrubar o servidor por causa de um botão seria trocar um
// problema pequeno por um grande. O aviso vai para o log com o caminho que
// falhou, porque configurar e não ver o botão é o sintoma sem explicação.
func openServedBook(cfg config.Config) servedBook {
	if cfg.BookPDF == "" {
		return servedBook{}
	}
	info, err := os.Stat(cfg.BookPDF)
	if err != nil || info.IsDir() {
		log.Printf("livro: %s não serve como PDF (%v) — o botão de abrir no livro não vai aparecer", cfg.BookPDF, err)
		return servedBook{}
	}
	warnIfNotLinearized(cfg.BookPDF)
	digest := bookFileDigest(info)
	return servedBook{
		path:   cfg.BookPDF,
		digest: digest,
		address: bookui.BookAddress{
			Base:     routes.Book + "?v=" + digest,
			Abertura: cfg.BookPageOffset,
		},
	}
}

// bookFileDigest versiona o endereço a partir do TAMANHO e da data do arquivo,
// e não do conteúdo.
//
// A diferença é medida: somar os 89 MB custa uma leitura do arquivo inteiro em
// todo boot, para invalidar um cache que só muda quando alguém TROCA o arquivo
// — e trocar um arquivo muda o tamanho ou a data. É o mesmo par que qualquer
// servidor de arquivos usa para cunhar `ETag`.
func bookFileDigest(info os.FileInfo) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%d-%d", info.Size(), info.ModTime().UnixNano()))
	return hex.EncodeToString(sum[:])[:12]
}

// warnIfNotLinearized diz, no boot, que o arquivo configurado não passou pelo
// `qpdf --linearize`.
//
// MEDIDO, e o número desmente a suposição óbvia: a linearização NÃO faz o
// visualizador pedir só as FAIXAS da página — pelo menos não no Chrome, abrindo
// o PDF como navegação de topo. Contando os bytes do loopback, abrir `#page=295`
// transferiu o ARQUIVO INTEIRO nos dois casos, cru e linearizado, e os dois
// números batem com o tamanho do arquivo.
//
// O ganho medido do `qpdf` é de TAMANHO — 12% —, e não de faixas. A renderização
// progressiva que a linearização promete não foi medida daqui: no localhost a
// transferência termina antes de haver o que ver. Quem for afirmá-la, meça antes.
//
// Aviso e não conserto: linearizar aqui obrigaria o servidor a depender do
// `qpdf` instalado e a gravar um segundo arquivo de 78 MB no boot. O que o
// servidor pode fazer barato é NOMEAR o comando.
func warnIfNotLinearized(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	// O dicionário de linearização é o PRIMEIRO objeto do arquivo, por
	// definição: se ele não está no começo, ele não existe.
	head := make([]byte, 2048)
	n, _ := io.ReadFull(f, head)
	if isLinearized(head[:n]) {
		return
	}
	log.Printf("livro: %s não está linearizado — medido, o navegador transfere o arquivo inteiro para abrir uma página, e o qpdf ainda o encolhe 12%%. Conserto: qpdf --linearize %s %s", path, path, path+".linear")
}

// isLinearized procura a marca do PDF linearizado no começo do arquivo.
func isLinearized(head []byte) bool {
	return strings.Contains(string(head), "/Linearized")
}

// BookFileHandler serve o PDF configurado, com faixas: o `http.ServeFile`
// responde `Range` sozinho.
//
// O alcance do cache é `private` porque esta rota sai DEPOIS do `requirePage`:
// `public` autorizaria um cache compartilhado a guardar e reentregar o livro de
// alguém que entrou para quem não entrou.
func (s *Server) BookFileHandler() http.Handler {
	// Sem livro a rota é 404 e PRONTO — o 404 não passa pela política de cache,
	// e essa ordem é conserto de um vermelho: um dígito VAZIO fazia
	// `strings.Contains(ifNoneMatch, "")` responder verdadeiro, e a rota
	// devolvia 304 para todo mundo em vez de 404. Visto no
	// `TestWithoutConfigurationTheBookRouteGives404`, que nasceu vermelho por isso.
	if s.book.path == "" {
		return http.NotFoundHandler()
	}
	return httpio.WithVersionedCache(s.book.digest, "private", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.book.path)
	}))
}

// BookAddress cumpre a porta da Mesa do Mestre (`master.Deps`).
//
// Invólucro fino de um campo, e é assim que o `api` cumpre toda porta de cena:
// quem escolhe o que atravessa a fronteira é o CONSUMIDOR, e o hospedeiro se
// dobra ao que ele pediu. A cena não recebe a `Config` nem o `servedBook` —
// ela recebe o endereço pronto, que é a única coisa que os componentes do livro
// precisam saber.
