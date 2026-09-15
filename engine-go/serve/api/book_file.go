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

// livroServido é o que o servidor guarda: onde o arquivo está e como falar dele.
type livroServido struct {
	caminho  string
	digito   string
	endereco bookui.BookAddress
}

// O endereço do leitor mora em `web/routes` porque o `bookui` o cita para montar
// o selo de página. Ele não é versionado: é uma página HTML servida com
// `no-store`, e quem carrega versão é o PDF que ela pede.

// A divisão com `web/reader` é por DEPENDÊNCIA e não por tamanho: o que ficou
// aqui lê `config.Config`, chama `os.Stat` e devolve um `http.Handler` sobre
// um arquivo do disco do dono da mesa. Uma cena que recebesse a `Config` para
// saber onde o PDF está teria o hospedeiro dentro dela.

// abreOLivro lê a configuração UMA vez, no boot.
//
// Ausência de arquivo é degradação normal e não queda: a mesa inteira funciona
// sem o livro, e derrubar o servidor por causa de um botão seria trocar um
// problema pequeno por um grande. O aviso vai para o log com o caminho que
// falhou, porque configurar e não ver o botão é o sintoma sem explicação.
func abreOLivro(cfg config.Config) livroServido {
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
		endereco: bookui.BookAddress{
			Base:     routes.Book + "?v=" + digito,
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
	if s.livro.caminho == "" {
		return http.NotFoundHandler()
	}
	return comCacheVersionado(s.livro.digito, "private", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.livro.caminho)
	}))
}

// BookAddress cumpre a porta da Mesa do Mestre (`master.Deps`).
//
// Invólucro fino de um campo, e é assim que o `api` cumpre toda porta de cena:
// quem escolhe o que atravessa a fronteira é o CONSUMIDOR, e o hospedeiro se
// dobra ao que ele pediu. A cena não recebe a `Config` nem o `livroServido` —
// ela recebe o endereço pronto, que é a única coisa que os componentes do livro
// precisam saber.
