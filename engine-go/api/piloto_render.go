package api

import (
	"bytes"
	"embed"
	"fmt"
	"net/http"
	"t20engine/web/bookui"
	"t20engine/web/finder"

	"github.com/a-h/templ"
	"t20engine/web/ui"
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

// WritePage renderiza uma tela inteira do piloto direto na resposta.
//
// O corpo entra como `templ.Component` e não como HTML já renderizado. É a
// diferença que a ALE-227 comprou: antes eram DUAS passadas — `ui.RenderFragment`
// devolvia uma string, ela virava `template.HTML` (a anotação "confie em mim")
// e só então entrava no layout — porque o `html/template` não sabe invocar um
// sub-template por nome dinâmico. Agora quem compõe é o compilador, e não há
// passada intermediária onde escapar errado.
func (s *Server) WritePage(
	w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component,
) {
	var buf bytes.Buffer
	// A CASCA RECEBE o que ela não pode conhecer (ALE-278, fatia 4): o endereço
	// dos estáticos, que são embutidos aqui, e as três sobreposições, que leem
	// catálogo. Este é o único lugar do projeto que monta uma página, então é
	// aqui que a injeção cabe — pôr os campos em cada `ui.Page{…}` seria repetir
	// dezoito vezes o que não varia.
	p.Asset = EstaticoDoPiloto
	p.Overlays = []templ.Component{finder.Dialog(), bookui.BookDialog(), bookui.EntryDialog()}
	if err := ui.Layout(p, corpo).Render(r.Context(), &buf); err != nil {
		// Em buffer e não direto no `w`: um erro no meio da renderização já
		// teria mandado 200 e meia página, e o jogador veria uma tela cortada
		// sem nenhum sinal de que faltou coisa.
		http.Error(w, fmt.Sprintf("render da página do piloto: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// O status vem por parâmetro porque um formulário RECUSADO devolve a mesma
	// tela (ALE-229), e responder 200 a uma recusa mente para tudo o que não é
	// um navegador — teste, log, monitoração. E ele é escrito DEPOIS dos
	// cabeçalhos: `WriteHeader` os congela, então um `Set` depois dele não faz
	// nada e some sem erro.
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
