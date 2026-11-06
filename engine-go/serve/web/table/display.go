package table

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app"
	"t20engine/domain/board"
	"t20engine/domain/live"
	"t20engine/infra/events"
	"t20engine/serve/web/routes"
	"t20engine/serve/web/ui"
)

// A TELA DA MESA (ALE-608): o tabuleiro que o mestre põe no display grande que
// a mesa inteira vê. Ela não tem ação nenhuma e mostra a cena COMO A MESA a vê
// — pelo mesmo `BoardForRole` da lente, e não por uma segunda regra —, na aba
// e no enquadramento do MESTRE. Ele a CONGELA para mexer escondido: a tela
// fica com o retrato daquele instante até ele soltar.
//
// É PÁGINA PRÓPRIA, e não uma `sidePage` da mesa, porque não precisa da mesa:
// a view dela é o tabuleiro e o recorte, e o fluxo dela acorda também pelo
// enquadramento, que não é evento da sessão.

// displayPath é o pedaço do endereço depois da sessão.
const displayPath = "tela"

func (s Scene) DisplayRoutes(r chi.Router) {
	base := sessionPattern + "/" + displayPath
	r.Get(base, s.handleDisplayPage)
	r.Get(base+"/fluxo", s.handleDisplayStream)
	r.Post(base+"/enquadramento", s.handleDisplayFraming)
	// CONGELAR e SOLTAR são dois verbos, e não um que alterna como a lente: o
	// clique duplo no Congelar deixaria a tela solta, mostrando à mesa o que o
	// mestre congelou para esconder.
	r.Post(base+"/congelar", s.gmBoardCommand(freezeDisplay))
	r.Post(base+"/soltar", s.gmBoardCommand(thawDisplay))
}

// displayAddress é o endereço da tela desta sessão.
func displayAddress(v View) string { return sidePageAddress(v, displayPath) }

// displayGm resolve a sessão e exige o mestre. A tela não é dos jogadores: ela
// é o tabuleiro que o MESTRE escolheu mostrar, no recorte dele.
func (s Scene) displayGm(w http.ResponseWriter, r *http.Request) (campaignID, sessionID, userID, number int64, ok bool) {
	campaignID, sessionID, ok = tableParams(w, r)
	if !ok {
		return 0, 0, 0, 0, false
	}
	userID = s.deps.CurrentUserID(r)
	sess, role, err := s.access.Session(r.Context(), app.Caller{ID: userID}, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), statusOf(err))
		return 0, 0, 0, 0, false
	}
	if role != "gm" {
		http.Error(w, "a tela da mesa é do mestre", http.StatusForbidden)
		return 0, 0, 0, 0, false
	}
	return campaignID, sessionID, userID, sess.Sessionnumber, true
}

// displayNow é o que a tela desenha agora: o retrato, se congelada, ou o
// tabuleiro de agora no recorte de agora.
func (s Scene) displayNow(ctx context.Context, userID, campaignID, sessionID int64) (displayView, error) {
	if frozen, ok := s.displays.Frozen(sessionID); ok {
		return frozen, nil
	}
	b, err := s.displayBoard(ctx, userID, campaignID, sessionID)
	if err != nil {
		return displayView{}, err
	}
	return displayView{Board: b, Framing: s.displays.Framing(sessionID)}, nil
}

// displayBoard é o tabuleiro da aba do MESTRE, redigido para a MESA.
//
// O estado da fila também passa pelo papel de jogador: é dele que sai a barra
// de PV de cada peça, e o combatente cujo PV o mestre escondeu não pode ter a
// barra na tela da mesa.
func (s Scene) displayBoard(ctx context.Context, gmID, campaignID, sessionID int64) (BoardView, error) {
	if _, err := s.deps.Sessions().State(ctx, sessionID); err != nil {
		return BoardView{}, err
	}
	fresh, err := s.deps.Sessions().RefreshCharacterVitals(ctx, sessionID)
	if err != nil {
		return BoardView{}, err
	}
	st := live.StateForRole("player", fresh)
	aba, _, _, err := s.pullTab(ctx, sessionID, gmID)
	if err != nil {
		return BoardView{}, err
	}
	onBoard, err := s.deps.Boards().Get(ctx, sessionID, aba)
	if err != nil {
		return BoardView{}, err
	}
	scene := board.BoardForRole("player", onBoard)
	return boardViewOf(scene, st, saudeDaFila(st), turnCombatant(st), board.Mover{Role: "player"}, nil, campaignID, sessionID), nil
}

func (s Scene) handleDisplayPage(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, userID, number, ok := s.displayGm(w, r)
	if !ok {
		return
	}
	v, err := s.displayNow(r.Context(), userID, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Title:   fmt.Sprintf("Tela da mesa · Sessão %d", number),
		Signals: tableSignalsExpr(s.deps.BuildRevision()),
		Init:    fmt.Sprintf("@get('%s/%s/fluxo')", routes.Session(campaignID, sessionID), displayPath),
	}, displayPage(v))
}

// handleDisplayStream é o fluxo da tela. Ele CONTA a tela enquanto ela está
// aberta — é o que acende o interruptor no tabuleiro do mestre — e acorda por
// três coisas: o barramento da sessão (uma peça andou), o enquadramento e o
// congelamento (`displays.Changed`), e o batimento, que cobre o que muda fora
// dos stores, como o PV que vem da ficha.
func (s Scene) handleDisplayStream(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, userID, _, ok := s.displayGm(w, r)
	if !ok {
		return
	}
	s.displays.Join(sessionID)
	defer s.displays.Leave(sessionID)
	sub, stop := s.deps.Bus().Subscribe(events.OfSession(sessionID))
	defer stop()

	sse := datastar.NewSSE(w, r)
	var last [32]byte
	write := func() {
		v, err := s.displayNow(r.Context(), userID, campaignID, sessionID)
		if err != nil {
			// Erro passageiro: a tela fica no último quadro bom, e o próximo
			// ciclo tenta de novo.
			return
		}
		fragment, err := ui.RenderFragment(r.Context(), displayRegion(v))
		if err != nil {
			return
		}
		// Congelada, o retrato é o mesmo a cada ciclo, e a digital cala.
		if digest := sha256.Sum256([]byte(fragment)); digest != last && sse.PatchElements(fragment) == nil {
			last = digest
		}
	}
	// O canal é pego ANTES de cada desenho: o `notifyLocked` fecha e apaga o
	// canal que existe, e um recorte que chegasse durante o desenho, com o
	// canal ainda por pegar, não acordaria ninguém até o batimento.
	changed := s.displays.Changed(sessionID)
	write()
	beat := time.NewTicker(tableHeartbeat)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.deps.LiveStreamsClosing():
			// ABORTA como o fluxo da mesa: fechar limpo é resposta completa, e a
			// tela não reconectaria depois do deploy.
			panic(http.ErrAbortHandler)
		case <-sub.C:
		case <-changed:
		case <-beat.C:
		}
		changed = s.displays.Changed(sessionID)
		write()
	}
}

// handleDisplayFraming guarda o recorte que o tabuleiro do mestre manda.
//
// Responde 204 e nada mais: o mestre não tem o que redesenhar, e a tela acorda
// pelo `displays.Changed`.
func (s Scene) handleDisplayFraming(w http.ResponseWriter, r *http.Request) {
	_, sessionID, _, _, ok := s.displayGm(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var f displayFraming
	if err := datastar.ReadSignals(r, &f); err != nil {
		http.Error(w, fmt.Sprintf("o enquadramento não é JSON: %v", err), http.StatusBadRequest)
		return
	}
	if f.Width <= 0 || f.Height <= 0 {
		http.Error(w, fmt.Sprintf("o enquadramento precisa de largura e altura positivas, em quadrados; chegou %gx%g", f.Width, f.Height), http.StatusBadRequest)
		return
	}
	s.displays.Frame(sessionID, f)
	w.WriteHeader(http.StatusNoContent)
}

// freezeDisplay tira o retrato do que a mesa está vendo AGORA. Não muda a
// cena — como a lente, devolve nil e não acorda a mesa —, e o que redesenha o
// interruptor do mestre é a resposta do próprio comando.
func freezeDisplay(st Scene, c commandCtx) (*board.BoardState, error) {
	if _, frozen := st.displays.Frozen(c.SessionID); frozen {
		return nil, nil
	}
	b, err := st.displayBoard(c.R.Context(), c.User, c.CampaignID, c.SessionID)
	if err != nil {
		return nil, err
	}
	st.displays.Freeze(c.SessionID, displayView{Board: b, Framing: st.displays.Framing(c.SessionID)}, time.Now())
	return nil, nil
}

func thawDisplay(st Scene, c commandCtx) (*board.BoardState, error) {
	st.displays.Thaw(c.SessionID)
	return nil, nil
}

// displayFramingSignals é o recorte como os sinais da tela. O nome leva o
// prefixo da família (`display_*`) porque `x` e `width` sozinhos não diriam de
// quem são.
func displayFramingSignals(f displayFraming) string {
	signals, err := json.Marshal(map[string]float64{
		"display_x": f.X, "display_y": f.Y, "display_width": f.Width, "display_height": f.Height,
	})
	if err != nil {
		return "{}"
	}
	return string(signals)
}

// displayFits encaixa o recorte do mestre na tela: o quadrado é o maior que
// faz o recorte inteiro caber, e a janela põe o meio dele no meio da tela. É a
// conta do `planPoint` ao contrário — o plano anda `translate(-janela)` —, a
// mesma do `centerViewport`.
// Sem recorte ainda (o mestre não mexeu no mapa desde que a tela abriu), a
// janela fica onde nasceu.
//
// Ele também roda no `data-on:resize__window`, e por isso a última instrução
// não é um `if` — ver o `datastar.md`.
const displayFits = "const r = document.getElementById('" + sceneId + "')?.getBoundingClientRect(); " +
	"const sq = r && r.width > 0 && $display_width > 0 && $display_height > 0 ? Math.min(r.width / $display_width, r.height / $display_height) : 0; " +
	"sq > 0 && ($square = sq, $" + sinalDaVistaX + " = $display_x * sq - r.width / 2, $" + sinalDaVistaY + " = $display_y * sq - r.height / 2)"

// followsFraming é o lado do MESTRE: o meio e o tamanho do que ele está vendo,
// em quadrados, mandados quando a janela dele anda ou o zoom muda. Com o
// tabuleiro escondido (o celular na Fila), a cena mede zero e nada sai.
//
// É um `data-effect` que LÊ os três sinais, e não um `data-on-signal-patch`:
// medido na ALE-608, o ouvinte de remendo disparava no zoom e calava no
// arrasto da vista, com a janela andando 260px e nenhum pedido saindo. O
// efeito é a mesma reatividade que move o `style` do mapa, e ela não falha. O
// adiamento de 300ms é à mão (`framingTimer` no próprio nó): um pedido por
// quadro de arrasto seria um por pixel.
func followsFraming(v View) string {
	x, y := planPoint("r.width / 2", "r.height / 2")
	return fmt.Sprintf("$square; $%s; $%s; clearTimeout(el.framingTimer); "+
		"el.framingTimer = setTimeout(() => { const r = document.getElementById('%s')?.getBoundingClientRect(); "+
		"r && r.width > 0 && $square > 0 && @post('%s/enquadramento', {payload: {x: %s, y: %s, width: r.width / $square, height: r.height / $square}}) }, 300)",
		sinalDaVistaX, sinalDaVistaY, sceneId, displayAddress(v), x, y)
}

func freezeCommand(v View) string {
	return fmt.Sprintf("@post('%s/congelar')", displayAddress(v))
}

func thawCommand(v View) string {
	return fmt.Sprintf("@post('%s/soltar')", displayAddress(v))
}
