package api

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

// Os ESTÁTICOS do piloto Datastar (ALE-219). Embutidos, como os catálogos: o
// binário continua sendo UM arquivo, que é a premissa de produção deste
// projeto.
//
// Só os estáticos: desde a ALE-227 os templates são código Go gerado pelo
// `templ`, então não há mais `.html` para embutir — e some junto a classe de
// erro que o `template.Must` existia para pegar cedo. Template com sintaxe
// quebrada agora não COMPILA, que é mais cedo que o boot.
//
//go:embed piloto/static/*
var pilotoFS embed.FS

// escrevePagina renderiza uma tela inteira do piloto direto na resposta.
//
// O corpo entra como `templ.Component` e não como HTML já renderizado. É a
// diferença que a ALE-227 comprou: antes eram DUAS passadas — `renderFragmento`
// devolvia uma string, ela virava `template.HTML` (a anotação "confie em mim")
// e só então entrava no layout — porque o `html/template` não sabe invocar um
// sub-template por nome dinâmico. Agora quem compõe é o compilador, e não há
// passada intermediária onde escapar errado.
func (s *Server) escrevePagina(w http.ResponseWriter, r *http.Request, p paginaPiloto, corpo templ.Component) {
	var buf bytes.Buffer
	if err := layout(p, corpo).Render(r.Context(), &buf); err != nil {
		// Em buffer e não direto no `w`: um erro no meio da renderização já
		// teria mandado 200 e meia página, e o jogador veria uma tela cortada
		// sem nenhum sinal de que faltou coisa.
		http.Error(w, fmt.Sprintf("render da página do piloto: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(buf.Bytes())
}

// renderFragmento devolve o HTML de um componente, para viajar num
// `datastar-patch-elements`.
func renderFragmento(ctx context.Context, c templ.Component) (string, error) {
	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		return "", fmt.Errorf("render de fragmento do piloto: %w", err)
	}
	return buf.String(), nil
}
