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

// comCacheDeEstatico embrulha o servidor de arquivos com a política de cache.
//
// COM a versão certa na consulta: um ano e `immutable`, porque a URL identifica
// o conteúdo — se o conteúdo mudar, o dígito muda e a URL é outra.
//
// SEM ela: `no-cache`, e é deliberado ser o pior caso. Um endereço sem versão
// pode ter sido guardado por alguém antes de um deploy, e servi-lo como eterno
// é prender a pessoa numa folha velha sem nenhum gesto que a resgate.
func comCacheDeEstatico(interno http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("v") == versaoDosEstaticos {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		// O `ETag` sai nos DOIS casos: ele é o que dá 304 a quem chega sem
		// versão, e é a única coisa que o `embed` não podia oferecer sozinho
		// (modtime zero). Forte e entre aspas, como manda o RFC.
		w.Header().Set("ETag", `"`+versaoDosEstaticos+`"`)
		if strings.Contains(r.Header.Get("If-None-Match"), versaoDosEstaticos) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		interno.ServeHTTP(w, r)
	})
}
