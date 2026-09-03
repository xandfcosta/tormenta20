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
	"t20engine/web/bookui"
	"t20engine/web/routes"

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

// O ENDEREÇO DO LIVRO mora em `web/bookui` desde a ALE-278.
//
// Ele é a assinatura de todos os componentes de lá — `PageSeal`, `Chunk`,
// `CrossRef` recebem um `bookui.BookAddress` —, e é um tipo de VALOR com um
// método que monta URL. Ficar aqui obrigaria o pacote de apresentação do livro
// a importar a cena que serve o PDF.

// livroServido é o que o servidor guarda: onde o arquivo está e como falar dele.
type livroServido struct {
	caminho  string
	digito   string
	endereco bookui.BookAddress
}

// O endereço desta cena mora em `web/routes` desde a ALE-278, porque a Mesa o
// cita. O comentário que estava aqui explicava o `StripPrefix` do `buildMux`, e
// ele já era falso: o prefixo saiu na ALE-280.

// O endereço do leitor mora em `web/routes` desde a ALE-278: o `bookui` o cita
// para montar o selo de página, e endereço citado de outra cena é o critério de
// entrada de lá. Ele não é versionado porque é uma página HTML servida com
// `no-store`; quem carrega versão é o PDF que ela pede.

// A CENA do leitor saiu para `web/reader` na ALE-278, e este arquivo ficou com
// a outra metade: ler a configuração no boot, cunhar o dígito de cache, avisar
// sobre linearização e SERVIR o arquivo com faixas.
//
// A divisão não é por tamanho, é a de sempre — dependência. O que ficou lê
// `plataforma.Config`, chama `os.Stat` e devolve um `http.Handler` sobre um
// arquivo do disco do dono da mesa; nada disso é cena, e uma cena que
// recebesse a `Config` para saber onde o PDF está teria o hospedeiro dentro
// dela. O que saiu desenha uma página e não toca o disco.

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
	// `TestWithoutConfigurationTheBookRouteGives404`, que nasceu vermelho por isso.
	if s.livro.caminho == "" {
		return http.NotFoundHandler()
	}
	return comCacheVersionado(s.livro.digito, "private", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.livro.caminho)
	}))
}

// BookAddress cumpre a porta da Mesa do Mestre (`master.Deps`, ALE-278).
//
// Invólucro fino de um campo, e é assim que o `api` cumpre toda porta de cena
// desde a forja: quem escolhe o que atravessa a fronteira é o CONSUMIDOR, e o
// hospedeiro se dobra ao que ele pediu. A cena não recebe a `Config` nem o
// `livroServido` — ela recebe o endereço pronto, que é a única coisa que os
// componentes do livro precisam saber.
func (s *Server) BookAddress() bookui.BookAddress { return s.livro.endereco }
