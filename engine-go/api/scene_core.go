package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/sheet"
	"t20engine/web/bookui"
	"t20engine/web/finder"
	"t20engine/web/ui"

	"github.com/a-h/templ"
)

// O NÚCLEO QUE TODA CENA PEDE, e o começo do fim do `*Server` como porta
// (ALE-278, fatia 6).
//
// Onze cenas declaram porta própria e, até aqui, quem as cumpria era um tipo
// só: o `*Server`, com 89 métodos exportados existindo para satisfazer a UNIÃO
// de todas elas. A medição que motivou a divisão: das 76 assinaturas pedidas,
// **67 têm exatamente uma cena pedindo**. Uma união que ninguém precisa como
// união é um objeto-deus com nome de servidor — e o preço dela não é teórico, é
// o que a ALE-277 mediu do outro lado: método sem chamador não quebra
// compilação, então 104 manipuladores mortos atravessaram onze fatias sendo
// lidos como código vivo.
//
// As NOVE assinaturas compartilhadas ficam aqui, e a repartição segue quantas
// cenas pedem cada uma: `WritePage` (11), `Queries` (7), `CurrentUserID` (6),
// `Catalogs` (4), `BookAddress` (3) e mais quatro pares. O resto vira campo do
// adaptador da cena que o pede.
//
// Ele é VALOR e não ponteiro de propósito: não há estado para mutar aqui, e
// copiar três ponteiros por cena é mais barato que a pergunta "quem mais está
// segurando isto".
type sceneCore struct {
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
	livro    bookui.BookAddress
}

// Queries é o acesso ao banco pelas consultas geradas.
//
// É a concessão mais larga da casa e ela é consciente: sete cenas leem e
// escrevem a linha delas por aqui. O que a mantém honesta é o que ela NÃO
// entrega — `*sql.DB` não atravessa, então nenhuma cena monta SQL nem abre
// transação (ver os `fronteira_test.go`).
func (c sceneCore) Queries() *sqlcgen.Queries { return c.queries }

// Catalogs é o motor primado — o mesmo que o oráculo usa.
func (c sceneCore) Catalogs() *engine.Catalogs { return c.catalogs }

// BookAddress é onde o PDF do livro atende, quando `LIVRO_PDF` aponta para um.
func (c sceneCore) BookAddress() bookui.BookAddress { return c.livro }

// Asset monta o endereço versionado de um estático.
func (c sceneCore) Asset(arquivo string) string { return EstaticoDoPiloto(arquivo) }

// CurrentUserID lê quem está pedindo do contexto que o `requirePage` escreveu.
//
// Ela é método e não função de pacote porque a CHAVE do contexto é deste
// pacote: uma segunda chave com o mesmo nome, declarada noutro pacote, não lê o
// mesmo valor.
func (c sceneCore) CurrentUserID(r *http.Request) int64 { return currentUser(r).ID }

// CharacterList carrega o elenco de quem pede, agregado por agregado.
func (c sceneCore) CharacterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error) {
	rows, err := c.queries.ListCharactersByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]sheet.CharacterDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := sheet.Load(ctx, c.queries, row)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

// WritePage renderiza uma tela inteira direto na resposta.
//
// O corpo entra como `templ.Component` e não como HTML já renderizado. É a
// diferença que a ALE-227 comprou: antes eram DUAS passadas — `ui.RenderFragment`
// devolvia uma string, ela virava `template.HTML` (a anotação "confie em mim")
// e só então entrava no layout — porque o `html/template` não sabe invocar um
// sub-template por nome dinâmico. Agora quem compõe é o compilador, e não há
// passada intermediária onde escapar errado.
func (c sceneCore) WritePage(
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
