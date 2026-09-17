// Package assets é o dono dos ESTÁTICOS do app: a folha, as ilhas de JS, as
// fontes e o favicon.
//
// Ele mora em `serve/web` e não no `serve/api` por uma razão mecânica: o
// `//go:embed` só alcança arquivos ABAIXO do diretório do pacote, então quem
// embute é quem hospeda. Enquanto a diretiva estava no `api`, todo o front que
// não é `.templ` — doze `.ts` e três `.css` — vivia dentro da raiz de
// composição, que é a última pasta onde alguém procura CSS.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"sort"

	"t20engine/infra/httpio"
)

//go:embed static/*
var files embed.FS

// version é o dígito do CONJUNTO, e não um por arquivo: eles são embutidos no
// mesmo binário e um deploy publica todos juntos. Dígito por arquivo daria três
// invalidações independentes para ganhar nada.
var version = digest()

func digest() string {
	var paths []string
	_ = fs.WalkDir(files, "static", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	// ORDENADO, senão o dígito muda entre dois boots do MESMO binário e o cache
	// do navegador é jogado fora por nada. O `WalkDir` já anda em ordem lexical,
	// mas depender disso é depender de detalhe de implementação para um valor
	// que decide invalidação de cache.
	sort.Strings(paths)

	sum := sha256.New()
	for _, path := range paths {
		content, err := files.ReadFile(path)
		if err != nil {
			continue
		}
		sum.Write([]byte(path))
		sum.Write(content)
	}
	return hex.EncodeToString(sum.Sum(nil))[:12]
}

// URL monta o endereço versionado de um estático.
//
// Usar isto e nunca escrever o caminho à mão: caminho cru continua funcionando e
// é servido SEM cache de propósito, então a página que o escrever à mão volta a
// piscar, em silêncio, e ninguém liga uma coisa à outra.
//
// @example assets.URL("scene.js") // "/static/scene.js?v=a1b2c3d4e5f6"
func URL(file string) string { return "/static/" + file + "?v=" + version }

// Handler serve `/static/*`.
//
// O cache é a razão de ele não ser um `http.FileServer` pelado: arquivo de
// `go:embed` tem modtime ZERO, e o `http.ServeContent` não emite
// `Last-Modified` de um tempo nulo nem inventa `ETag`. Sem validador não há 304,
// e o navegador rebaixa a folha BLOQUEANTE a cada navegação — o que aparece como
// um flash branco, não como lentidão.
func Handler() http.Handler {
	sub, err := fs.Sub(files, "static")
	if err != nil {
		panic("assets: o estático embutido está ausente: " + err.Error())
	}
	return httpio.WithVersionedCache(version, "public", http.FileServer(http.FS(sub)))
}

// FontsHandler serve as fontes que a FOLHA pede em `/fonts/…`.
//
// Sem esta rota, `GET /fonts/cinzel-latin.woff2` dá 404 e a Cinzel cai para uma
// serifada do sistema em TODA tela: o `@font-face` pede o caminho ABSOLUTO, e
// não há mais nada que o resolva.
func FontsHandler() http.Handler {
	sub, err := fs.Sub(files, "static/fonts")
	if err != nil {
		panic("assets: as fontes embutidas estão ausentes: " + err.Error())
	}
	// `StripPrefix` porque o `FileServer` recebe `/fonts/x.woff2` e procuraria
	// `fonts/x.woff2` DENTRO do sub-FS, que já tem a pasta como raiz.
	return http.StripPrefix("/fonts", httpio.WithVersionedCache(version, "public", http.FileServer(http.FS(sub))))
}

// FaviconHandler serve o `/favicon.svg` que o layout pede.
func FaviconHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := files.ReadFile("static/favicon.svg")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(raw)
	})
}
