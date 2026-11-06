package table

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/serve/web/master"
	"t20engine/serve/web/ui"
)

// AS PÁGINAS À PARTE DO MESTRE (ALE-608): o que era a coluna das Consultas,
// cada um numa aba do navegador. A mesa fica com a fila, o tabuleiro e a linha
// do turno; o que se abre com uma pergunta na cabeça mora ao lado dela.
//
// São a MESMA cena e as mesmas regiões, penduradas noutro documento — o Grupo
// é o `gmGroup`, o Elenco é o `gmCastList` —, e por isso o que muda entre a
// mesa e uma página é só a MARCA (`?pagina=`), que viaja no fluxo e nos
// comandos. Sem ela, a resposta de um − no Grupo remendaria as dez regiões da
// mesa numa página que não tem nenhuma delas.

// sidePage é qual página à parte uma view desenha. O vazio é a mesa.
type sidePage string

const (
	pageTable    sidePage = ""
	pageGroup    sidePage = "grupo"
	pageCast     sidePage = "elenco"
	pageBestiary sidePage = "bestiario"
)

// pageParam é o nome da marca na query.
const pageParam = "pagina"

// sidePageLink é um botão do cabeçalho: o rótulo, o caminho depois da sessão e
// a linha de baixo que o menu das telas estreitas escreve.
type sidePageLink struct {
	Label, Path, Hint string
}

// sidePageLinks são as páginas na ordem do cabeçalho. As NOTAS não são uma
// `sidePage`: a cena delas já existia (`notesWindowPage`), com uma view mínima,
// e não precisa da mesa inteira para desenhar uma caixa de texto.
var sidePageLinks = []sidePageLink{
	{"Notas", notesPath, "o texto da sessão"},
	{"Grupo", string(pageGroup), "quem está na mesa, PV e PM"},
	{"Elenco", string(pageCast), "os NPCs da campanha"},
	{"Bestiário", string(pageBestiary), "as criaturas do livro"},
}

// SidePageFor traduz o nome da marca na página, e devolve a mesa para o que
// não é página. É exportada porque a BANCADA do hospedeiro pergunta ao mesmo
// `TableRegions` da produção o que o fluxo de cada página manda.
func SidePageFor(mark string) sidePage {
	switch p := sidePage(mark); p {
	case pageGroup, pageCast, pageBestiary:
		return p
	}
	return pageTable
}

// dressPage lê a marca do pedido. Só o mestre tem páginas à parte: para o
// jogador a marca não existe, e ele recebe a mesa dele.
func (v *View) dressPage(r *http.Request) {
	if v.GM == nil {
		return
	}
	v.Page = SidePageFor(r.URL.Query().Get(pageParam))
}

// pageMark é a marca que o fluxo e os comandos desta view levam, ou "" na mesa.
func (v View) pageMark() string {
	if v.Page == pageTable {
		return ""
	}
	return "?" + pageParam + "=" + string(v.Page)
}

// sidePageAddress é o endereço de uma página desta sessão.
func sidePageAddress(v View, path string) string { return v.SessionBase() + "/" + path }

// sidePageTarget é o NOME da aba de uma página: clicar de novo traz a que já
// está aberta em vez de abrir outra, e a da sessão 4 não é a da 7.
func sidePageTarget(v View, path string) string {
	return fmt.Sprintf("t20-%s-%d", path, v.SessionID)
}

// opensSidePage é o mesmo elo escrito como gesto, para quem não é `<a>`.
func opensSidePage(v View, page sidePage) string {
	return fmt.Sprintf("window.open('%s', '%s')", sidePageAddress(v, string(page)), sidePageTarget(v, string(page)))
}

// sidePageTitle é o "Grupo · Sessão 4" do topo e da aba do navegador.
func sidePageTitle(v View, label string) string {
	return fmt.Sprintf("%s · Sessão %d", label, v.SessionNum)
}

func (s Scene) SidePageRoutes(r chi.Router) {
	r.Get(sessionPattern+"/"+string(pageGroup), s.handleSidePage(pageGroup))
	r.Get(sessionPattern+"/"+string(pageCast), s.handleSidePage(pageCast))
	// O Bestiário divide o endereço com o remendo do painel: quem decide é o
	// cabeçalho do Datastar, no `handleBestiaryTable`.
}

// handleSidePage serve uma página à parte.
//
// A view é a da MESA (`LoadView`), e não uma mínima como a das notas: o Grupo
// e o Elenco são as mesmas regiões que o fluxo remenda, e o fluxo as monta pelo
// mesmo `LoadView`. Duas montagens para a mesma região seriam duas verdades
// sobre o PV de alguém.
//
// A TRAVA é a da mesa: o Grupo diz o PV de todo mundo, e o Elenco e o
// Bestiário, os números de quem a mesa vai enfrentar.
func (s Scene) handleSidePage(page sidePage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, sessionID, ok := tableParams(w, r)
		if !ok {
			return
		}
		view, status, err := s.LoadView(r.Context(), s.deps.CurrentUserID(r), campaignID, sessionID)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		if view.GM == nil {
			http.Error(w, "as páginas à parte da mesa são do mestre", http.StatusForbidden)
			return
		}
		view.Page = page
		p := ui.Page{Signals: tableSignalsExpr(s.deps.BuildRevision())}
		switch page {
		case pageGroup:
			p.Title = sidePageTitle(view, "Grupo")
			p.Init = fmt.Sprintf("@get('%s')", streamOfPage(view))
			s.deps.WritePage(w, r, http.StatusOK, p, gmGroupPage(view))
		case pageCast:
			p.Title = sidePageTitle(view, "Elenco")
			p.Init = fmt.Sprintf("@get('%s')", streamOfPage(view))
			s.deps.WritePage(w, r, http.StatusOK, p, gmCastPage(view))
		case pageBestiary:
			// SEM FLUXO: o livro não muda no meio da sessão, e o que o Pôr na
			// fila tem a dizer volta na resposta dele (`entry_sent`).
			p.Title = sidePageTitle(view, "Bestiário")
			s.deps.WritePage(w, r, http.StatusOK, p, gmBestiaryPage(view, s.forTableBestiary(r, campaignID, sessionID)))
		}
	}
}

// bestiaryPageDraft semeia o AJUSTE da criatura que a página já abre pelo
// endereço (`?criatura=ogro&abrir=1`): o remendo do painel só semeia quando o
// clique abre OUTRA criatura, e a carga fria nasceria com PV 0 no ajuste.
func bestiaryPageDraft(b master.BestiaryView) string {
	if b.Chosen == nil {
		return "{}"
	}
	draft, err := json.Marshal(entryDraft(*b.Chosen))
	if err != nil {
		return "{}"
	}
	return string(draft)
}

// groupAtTable é a contagem do cabeçalho do Grupo: "2 de 3 na mesa".
func groupAtTable(group []Member) string {
	here := 0
	for _, m := range group {
		if m.Presence != nil && m.Presence.AtTable {
			here++
		}
	}
	return fmt.Sprintf("%d de %d na mesa", here, len(group))
}

// memberMeta é a linha de baixo do nome: o nível, as classes e a Defesa —
// partida quando o personagem está caído (p394), porque é aqui que o mestre a
// confere para decidir se o ataque acerta.
func memberMeta(m Member) string {
	meta := fmt.Sprintf("Nível %d", m.Level)
	if m.Classes != "" {
		meta += " · " + m.Classes
	}
	meta += " · Defesa " + m.Defense
	if m.Presence != nil && !m.Presence.AtTable {
		meta += " · fora da mesa"
	}
	return meta
}

// poolOrNothing não desenha o pool que o personagem não tem: um "PM 0/0" de
// quem não usa mana é uma barra vazia que parece um herói esgotado.
func poolOrNothing(b tableBar) *tableBar {
	if b.Max == 0 {
		return nil
	}
	return &b
}

// npcMeta é a linha do livro, e a origem quando o mestre o renomeou.
func npcMeta(npc castNpc) string {
	if npc.FromBook != "" && npc.FromBook != npc.Name {
		return npc.Summary + " · cópia de " + npc.FromBook
	}
	return npc.Summary
}

// nameMatches é o filtro de uma linha pela busca da página: vazio mostra
// todos, e o nome casa por pedaço, sem caixa. O nome vai como literal JSON, que
// é texto JavaScript válido com aspas e acentos escapados.
func nameMatches(signal, name string) string {
	literal, err := json.Marshal(name)
	if err != nil {
		literal = []byte(`""`)
	}
	return fmt.Sprintf("$%s === '' || %s.toLowerCase().includes($%s.toLowerCase())", signal, literal, signal)
}
