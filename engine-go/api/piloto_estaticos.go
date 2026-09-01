package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"sort"
	"strings"
)

// O CACHE dos estáticos do piloto, e o defeito que ele conserta é de EXPERIÊNCIA
// e não de desempenho.
//
// Medido: `curl -D -` na folha devolvia `200 OK` e `Content-Length` e MAIS NADA
// — sem `Cache-Control`, sem `ETag`, sem `Last-Modified`. Sem validador não há
// 304, então o navegador rebaixava 113KB de CSS BLOQUEANTE DE RENDERIZAÇÃO a
// cada troca de página. Como o documento novo não pinta antes da folha chegar, o
// navegador desiste de segurar os pixels da página anterior e mostra o branco —
// o "flick" que o dono viu em toda navegação do Datastar. A ficha não piscava
// porque ela é rota de SPA: não há navegação de documento.
//
// A CAUSA é uma armadilha do `embed`: arquivo embutido tem modtime ZERO, e o
// `http.ServeContent` não emite `Last-Modified` de um tempo nulo nem inventa
// `ETag`. O `http.FileServer` estava certo; o sistema de arquivos por baixo é
// que não tinha o que datar.
//
// A escolha é URL VERSIONADA e não `max-age` curto, e a diferença é o objetivo:
// `max-age` curto ainda paga uma revalidação, e revalidação de folha bloqueante
// ainda atrasa a primeira pintura. Com a versão no caminho o navegador não
// pergunta nada — e não existe janela de folha velha depois de um deploy,
// porque binário novo muda o dígito e o dígito muda a URL.

// versaoDosEstaticos é o dígito do CONJUNTO, e não um por arquivo.
//
// Um só porque eles mudam juntos: são embutidos no mesmo binário, e um deploy
// que mexe na folha e no script publica os dois. Dígito por arquivo daria três
// invalidações independentes para ganhar nada — e a folha e o `cena.js` saem do
// mesmo build.
var versaoDosEstaticos = digitoDosEstaticos()

func digitoDosEstaticos() string {
	var caminhos []string
	_ = fs.WalkDir(pilotoFS, "piloto/static", func(caminho string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			caminhos = append(caminhos, caminho)
		}
		return nil
	})
	// ORDENADO, senão o dígito muda entre dois boots do MESMO binário e o cache
	// do navegador é jogado fora por nada. `WalkDir` já anda em ordem lexical,
	// mas depender disso é depender de detalhe de implementação para um valor
	// que decide invalidação de cache.
	sort.Strings(caminhos)

	soma := sha256.New()
	for _, caminho := range caminhos {
		conteudo, err := pilotoFS.ReadFile(caminho)
		if err != nil {
			continue
		}
		soma.Write([]byte(caminho))
		soma.Write(conteudo)
	}
	return hex.EncodeToString(soma.Sum(nil))[:12]
}

// EstaticoDoPiloto monta o endereço versionado de um arquivo estático.
//
// Usar isto e nunca escrever o caminho à mão: caminho cru continua funcionando,
// e é servido SEM cache de propósito — então a página que o escrever à mão volta
// a piscar, em silêncio, e ninguém liga uma coisa à outra.
func EstaticoDoPiloto(arquivo string) string {
	return "/piloto/static/" + arquivo + "?v=" + versaoDosEstaticos
}

// comCacheVersionado embrulha o servidor de arquivos com a política de cache.
//
// COM a versão certa na consulta: um ano e `immutable`, porque a URL identifica
// o conteúdo — se o conteúdo mudar, o dígito muda e a URL é outra.
//
// SEM ela: `no-cache`, e é deliberado ser o pior caso. Um endereço sem versão
// pode ter sido guardado por alguém antes de um deploy, e servi-lo como eterno
// é prender a pessoa numa folha velha sem nenhum gesto que a resgate.
// O `alcance` é `public` ou `private`, e não é detalhe: `public` autoriza um
// cache COMPARTILHADO a guardar a resposta, o que está certo para a folha e as
// fontes (elas saem sem sessão) e errado para o que só sai depois do
// `requirePagina` — ver `piloto_livro.go`.
func comCacheVersionado(versao, alcance string, interno http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("v") == versao {
			w.Header().Set("Cache-Control", alcance+", max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		// O `ETag` sai nos DOIS casos: ele é o que dá 304 a quem chega sem
		// versão, e é a única coisa que o `embed` não podia oferecer sozinho
		// (modtime zero). Forte e entre aspas, como manda o RFC.
		w.Header().Set("ETag", `"`+versao+`"`)
		// `versao != ""` porque `strings.Contains(x, "")` é VERDADEIRO: uma
		// versão vazia responderia 304 a qualquer pedido, e o corpo nunca sairia.
		if versao != "" && strings.Contains(r.Header.Get("If-None-Match"), versao) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		interno.ServeHTTP(w, r)
	})
}

// FontesDoPiloto serve as fontes que a FOLHA pede em `/fonts/…`.
//
// O defeito que a criou, relatado pelo dono com o cabeçalho na mão: `GET
// /fonts/cinzel-latin.woff2` devolvia 404 no binário sem SPA, e a Cinzel caía
// para uma serifada do sistema em TODA tela — que é justamente o binário em que
// a cena era revisada. O `@font-face` do `index.css` pede o caminho ABSOLUTO, e
// quem o resolvia era o `dist` da SPA.
//
// As fontes eram CÓPIA das da SPA, com o
// `TestAsFontesEmbutidasSaoAsMesmasDaSPA` prendendo as duas byte a byte. Com a
// SPA apagada (ALE-272, fatia 10c) elas deixaram de ser cópia: são as fontes, e
// o guarda saiu junto com o outro lado que ele comparava.
func (s *Server) FontesDoPiloto() http.Handler {
	sub, err := fs.Sub(pilotoFS, "piloto/static/fonts")
	if err != nil {
		panic("piloto: fontes embutidas ausentes: " + err.Error())
	}
	// `StripPrefix` porque o `FileServer` recebe `/fonts/x.woff2` e procuraria
	// `fonts/x.woff2` DENTRO do sub-FS, que já tem a pasta como raiz. Sem ele o
	// 404 continua e parece que o embed falhou — medido.
	return http.StripPrefix("/fonts", comCacheVersionado(versaoDosEstaticos, "public", http.FileServer(http.FS(sub))))
}

// FaviconDoPiloto serve o `/favicon.svg` que o layout pede.
//
// Ele era do `public/` da SPA e o `dist` o servia; com a SPA apagada (ALE-272,
// fatia 10c) o arquivo veio para os estáticos do piloto. Sem esta rota o ícone
// da aba fica com o padrão do navegador — não quebra nada e ninguém liga à
// causa, que é a marca desta família de perda numa migração.
func (s *Server) FaviconDoPiloto() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bruto, err := pilotoFS.ReadFile("piloto/static/favicon.svg")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(bruto)
	})
}
