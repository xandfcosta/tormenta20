package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/assets"
	"t20engine/serve/web/bookui"
	"t20engine/serve/web/characters"
	"t20engine/serve/web/finder"
	"t20engine/serve/web/ui"

	"github.com/a-h/templ"
)

// O NÚCLEO QUE TODA CENA PEDE: as poucas assinaturas que MAIS DE UMA cena quer.
// O que só uma pede vira campo do adaptador dela — uma união que ninguém
// precisa como união é um objeto-deus com nome de servidor.
//
// Ele é VALOR e não ponteiro: não há estado para mutar aqui, e copiar três
// ponteiros por cena é mais barato que a pergunta "quem mais está segurando
// isto".
type sceneCore struct {
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
	book     bookui.BookAddress
}

// Queries é a concessão mais larga da casa, e o que a mantém honesta é o que
// ela NÃO entrega: `*sql.DB` não atravessa, então nenhuma cena monta SQL nem
// abre transação (ver os `boundary_test.go`).
func (c sceneCore) Queries() *sqlcgen.Queries { return c.queries }

// Catalogs é o motor primado — o mesmo que o oráculo usa.
func (c sceneCore) Catalogs() *engine.Catalogs { return c.catalogs }

// BookAddress é onde o PDF do livro atende, quando `LIVRO_PDF` aponta para um.
func (c sceneCore) BookAddress() bookui.BookAddress { return c.book }

// Asset monta o endereço versionado de um estático.
func (c sceneCore) Asset(file string) string { return assets.URL(file) }

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
		dto, err := sheet.Load(ctx, c.queries, c.catalogs, row)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

// CampaignCopies são as cópias de campanha de quem pede, com a mesa, o nível e o PV de cada uma.
func (c sceneCore) CampaignCopies(ctx context.Context, ownerID int64) ([]characters.CampaignCopy, error) {
	rows, err := c.queries.ListCampaignCopiesOfOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]characters.CampaignCopy, 0, len(rows))
	for _, row := range rows {
		if !row.CampaignId.Valid || !row.SourceCharacterId.Valid {
			continue
		}
		campaign, err := c.queries.GetCampaign(ctx, row.CampaignId.Int64)
		if err != nil {
			return nil, fmt.Errorf("a mesa %d da cópia %d: %w", row.CampaignId.Int64, row.ID, err)
		}
		live, err := c.queries.HasLiveSessionForCharacter(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		dto, err := sheet.Load(ctx, c.queries, c.catalogs, row)
		if err != nil {
			return nil, err
		}
		out = append(out, characters.CampaignCopy{
			ID: row.ID, DraftID: row.SourceCharacterId.Int64, Campaign: campaign.Name, Level: dto.Level,
			PV: characters.HeroCardOf(c.catalogs, dto).PV, Live: live,
		})
	}
	slices.SortStableFunc(out, func(a, b characters.CampaignCopy) int { return strings.Compare(a.Campaign, b.Campaign) })
	return out, nil
}

// WritePage renderiza uma tela inteira direto na resposta.
//
// O corpo entra como `templ.Component` e não como HTML já renderizado: quem
// compõe é o compilador, e não sobra passada intermediária onde escapar
// errado.
func (c sceneCore) WritePage(
	w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component,
) {
	var buf bytes.Buffer
	// A CASCA RECEBE o que ela não pode conhecer: o endereço dos estáticos, as
	// três sobreposições e o modo de cor que o cookie traz. Este é o único
	// lugar que monta uma página, e pôr os campos em cada `ui.Page{…}` seria
	// repetir em toda cena o que não varia.
	p.Asset = assets.URL
	p.Overlays = []templ.Component{finder.Dialog(), bookui.BookDialog(), bookui.EntryDialog()}
	p.ColorMode = ui.ColorModeOf(r)
	if err := ui.Layout(p, body).Render(r.Context(), &buf); err != nil {
		// Em buffer e não direto no `w`: um erro no meio da renderização já
		// teria mandado 200 e meia página, e o jogador veria uma tela cortada
		// sem nenhum sinal de que faltou coisa.
		http.Error(w, fmt.Sprintf("render da página: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// O status vem por parâmetro porque um formulário RECUSADO devolve a mesma
	// tela, e responder 200 a uma recusa mente para tudo o que não é navegador.
	// Ele é escrito DEPOIS dos cabeçalhos: o `WriteHeader` os congela, e um
	// `Set` depois dele não faz nada e some sem erro.
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
