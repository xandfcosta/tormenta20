package table

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar-go/datastar"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/serve/web/ui"
)

// OS GESTOS DO RASCUNHO.
//
// Cada um é a MESMA mutação pura que a mesa aplica, num `EditPlace` em vez de
// num tabuleiro vivo. São GÊMEOS dos da Mesa e não os mesmos: o gêmeo é fiação,
// sem uma linha de regra — quem confere a coordenada, o teto de peças e o
// tamanho da criatura continua sendo o `tabuleiro`, num lugar só.
//
// O que os separa é o que o gesto SIGNIFICA. Na mesa, arrastar propõe um
// movimento com custo e vez, e alguém confirma; aqui a peça vai para a casa e
// acabou. Fundir os dois num handler com um `if` faria a Mesa — a superfície
// mais exercitada do app — depender de uma ramificação que só o rascunho
// percorre.

// DraftRoutes registra as quatorze rotas do rascunho.
//
// O endereço é da CAMPANHA e não da mesa: o lugar é do acervo e sobrevive a
// qualquer sessão, que é a diferença entre o rascunho e a cortina.
func (s Scene) DraftRoutes(r chi.Router) {
	r.Get("/campanhas/{campaignId}/lugares/{placeId}", s.handleDraftPage)
	base := "/campanhas/{campaignId}/lugares/{placeId}/tabuleiro"
	// O TRAÇO, como na mesa: as rotas de terreno recebem de ONDE ATÉ ONDE o dedo
	// andou desde o aviso anterior. Ver `board.StrokeSquares`.
	r.Post(base+"/terreno", s.draftCommand(draftPaintsTerrain))
	r.Post(base+"/terreno/limpar", s.draftCommand(draftClearsTerrain))
	r.Post(base+"/terreno/retangulo", s.draftCommand(draftFillsRect))
	r.Post(base+"/terreno/limpar/retangulo", s.draftCommand(draftClearsRect))
	r.Post(base+"/pecas/nova", s.draftCommand(draftNewLoosePiece))
	// MOVER é o gesto que NÃO tem gêmeo na mesa, e é a diferença do draft:
	// lá o arrasto manda uma PARADA e o servidor devolve uma proposta com custo,
	// aqui ele põe a peça na casa. Ver `draftMoveDrop`.
	r.Post(base+"/pecas/{id}/mover", s.draftCommand(draftMovesToken))
	r.Post(base+"/pecas/{id}/editar", s.draftCommand(draftEditsToken))
	r.Post(base+"/pecas/{id}/duplicar", s.draftCommand(draftDuplicatesToken))
	r.Post(base+"/pecas/{id}/remover", s.draftCommand(draftRemovesToken))
	r.Post(base+"/pecas/{id}/visibilidade", s.draftCommand(draftTogglesVisibility))
	r.Post(base+"/marcadores/novo", s.draftCommand(draftMarksTheSpot))
	r.Post(base+"/marcadores/{id}/revelar", s.draftCommand(draftRevealsMarker))
	r.Post(base+"/marcadores/{id}/cor/{cor}", s.draftCommand(draftPaintsMarker))
	r.Post(base+"/marcadores/{id}/remover", s.draftCommand(draftErasesMarker))
	// MEDIR não é comandar, e por isso as duas de baixo NÃO passam pelo
	// `draftCommand`: elas não mutam nada e respondem só com sinais. Um
	// `EditPlace` aqui gravaria o acervo a cada movimento do dedo sobre a régua.
	r.Post(base+"/regua", s.handleDraftRuler)
	r.Post(base+"/gabarito", s.handleDraftTemplate)
}

// ── MEDIR o rascunho ─────────────────────────────────────────────────────────
//
// A régua e o gabarito são desenhados no rascunho porque não são `GMOnly` e o
// trilho inteiro veio junto com o `boardTable`. As rotas PRECISAM vir junto: sem
// elas o trilho oferece um gesto que o servidor não atende — 404, tela que não
// muda, e nada explicando por quê. Quem varre isso é o
// `TestEveryDraftToolHasARoute`.
//
// E elas pertencem aqui e não só ao combate: *"cabe a bola de fogo nesta
// sala?"* é pergunta de PREPARAÇÃO — é montando a cripta que se decide o
// tamanho dela.

// handleDraftRuler é o gêmeo mais fino do arquivo: ele não olha o tabuleiro.
//
// A régua lê as paradas dos SINAIS e devolve a leitura de cada perna mais o
// total, e nada disso depende de o mapa ser a mesa ou o acervo. O que muda em
// relação ao da Mesa é só a trava — e ela muda de natureza, que é a razão de o
// gêmeo existir em vez de a rota ser registrada duas vezes.
func (s Scene) handleDraftRuler(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.draftGm(w, r); !ok {
		return
	}
	stops, err := rulerStops(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeSignals(w, r, polylineReading(stops))
}

// handleDraftTemplate desenha a área e diz quem ela pega (T20 p225).
//
// A cena vem do `PlaceScene` e NÃO passa pelo `BoardForRole`, ao contrário da
// Mesa. Lá a redação por papel existe porque quem pergunta "quem o cone pega?"
// não pode descobrir por aí a peça que a cortina esconde DELE; aqui não há
// outro papel — o rascunho é privativo por construção, e quem não mestra a
// campanha não chega até esta linha (`draftGm`).
//
// A peça ESCONDIDA entra na conta de propósito, e é a consequência que vale
// dizer: montando a emboscada, o mestre pergunta se a bola de fogo pega o
// assassino que a mesa ainda não vê. Redigir aqui esconderia dele a resposta que
// ele veio buscar.
func (s Scene) handleDraftTemplate(w http.ResponseWriter, r *http.Request) {
	c, ok := s.draftGm(w, r)
	if !ok {
		return
	}
	requested, origin, mira, err := pointsFromBody(r)
	if err != nil {
		http.Error(w, "a origem e a mira do gabarito precisam ser dois pares de números", http.StatusBadRequest)
		return
	}
	kind, err := urlTemplate(requested.Shape)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// A MIRA ainda não foi dada quando ela é a própria origem: o cone e a linha
	// precisam apontar para algum lado, e escolher um pelo servidor seria
	// inventar a decisão que falta.
	if pointsTemplate(kind) && mira == origin {
		writeSignals(w, r, map[string]any{
			"template_path": "", "template_text": "Clique de novo para apontar.",
		})
		return
	}
	squares := engine.AreaSquares(origin, engine.Area{
		Kind: kind, Size: templateSize(requested.Size),
		Direction: templateDirection(origin, mira),
	})
	// A cena pode ter sumido entre desenhar a tela e medir — outro navegador do
	// mestre pode ter apagado o lugar. O desenho sai de qualquer jeito; o que
	// falta é a lista de quem ele pega, e "ninguém" é a resposta honesta para um
	// mapa que não existe mais.
	scene, err := s.deps.Boards().PlaceScene(r.Context(), c.CampaignID, c.PlaceID)
	if err != nil {
		scene = nil
	}
	writeSignals(w, r, map[string]any{
		"template_path": squaresPath(squares),
		"template_text": takesTemplateWho(scene, squares),
	})
}

// handleDraftPage desenha a cena guardada, pronta para montar.
func (s Scene) handleDraftPage(w http.ResponseWriter, r *http.Request) {
	c, ok := s.draftGm(w, r)
	if !ok {
		return
	}
	view, err := s.draftPageOf(r.Context(), c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	// SEM `Init`, ao contrário da Mesa: não há stream. O rascunho tem um leitor
	// só e nada acontece nele que não tenha sido esta pessoa fazendo — uma
	// conexão SSE aberta aqui ficaria esperando um evento que ninguém publica.
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: "Rascunho · " + view.Place,
		Sinais: tableSignalsExpr(),
	}, draftBody(view))
}

// draftCommand é o gateway dos gestos: trava, muta, redesenha.
//
// Irmão do `boardCommand` e separado dele porque as três pontas são outras. A
// trava é da CAMPANHA e não da sessão; não há `PublishBoardState`, porque não há
// mesa esperando; e a resposta redesenha UMA região, porque a página do rascunho
// tem uma só — a Mesa manda nove.
func (s Scene) draftCommand(
	mutate func(Scene, draftCtx, *board.BoardState) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := s.draftGm(w, r)
		if !ok {
			return
		}
		// A leitura dos sinais vem ANTES do `NewSSE`, e a ordem é obrigatória: o
		// `NewSSE` assume a resposta e fecha o corpo do pedido. A ordem inversa passa
		// VERDE em teste de handler e falha no servidor de verdade.
		_, err := s.deps.Boards().EditPlace(r.Context(), c.CampaignID, c.PlaceID,
			func(b *board.BoardState) error { return mutate(s, c, b) })

		sse := datastar.NewSSE(w, r)
		sentence := ""
		if err != nil {
			sentence = err.Error()
		}
		// REDESENHA NOS DOIS CAMINHOS, inclusive na recusa: é o que mostra que a
		// cena continua como estava, ao lado da frase que diz por quê. Sem isso
		// uma recusa deixa a tela mostrando o gesto que o servidor não aceitou.
		if view, readErr := s.draftPageOf(r.Context(), c); readErr == nil {
			if fragment, drawErr := ui.RenderFragment(r.Context(), draftBoardRegion(view)); drawErr == nil {
				_ = sse.PatchElements(fragment)
			}
		}
		_ = sse.MarshalAndPatchSignals(map[string]any{"command_error": sentence})
	}
}

// ── o TERRENO (T20 p238) ─────────────────────────────────────────────────────

func draftPaintsTerrain(st Scene, c draftCtx, b *board.BoardState) error {
	requested, trait, err := strokeFromBody(c.R)
	if err != nil {
		return err
	}
	species := board.KnownTerrainKind(requested.Kind)
	// O `erase` continua sendo MODO da ferramenta e não uma rota própria, como
	// na mesa: ele vale para o arraste inteiro, e não para um quadrado.
	on := !requested.Erase
	for _, square := range trait {
		board.PaintTerrain(b, square, species, on)
	}
	return nil
}

func draftClearsTerrain(st Scene, c draftCtx, b *board.BoardState) error {
	_, trait, err := strokeFromBody(c.R)
	if err != nil {
		return err
	}
	for _, square := range trait {
		board.ClearSquare(b, square)
	}
	return nil
}

func draftFillsRect(st Scene, c draftCtx, b *board.BoardState) error {
	requested, squares, err := rectFromBody(c.R)
	if err != nil {
		return err
	}
	species := board.KnownTerrainKind(requested.Kind)
	for _, square := range squares {
		board.PaintTerrain(b, square, species, true)
	}
	return nil
}

func draftClearsRect(st Scene, c draftCtx, b *board.BoardState) error {
	_, squares, err := rectFromBody(c.R)
	if err != nil {
		return err
	}
	for _, square := range squares {
		board.ClearSquare(b, square)
	}
	return nil
}

// ── as PEÇAS ─────────────────────────────────────────────────────────────────

// draftNewLoosePiece é a peça avulsa: a porta, o baú, o barril.
//
// Ela lê a MESMA tira que a mesa lê (`loosePieceSignals`), com as mesmas
// recusas — nome obrigatório, tamanho do livro (p107), aparência conhecida.
func draftNewLoosePiece(st Scene, c draftCtx, b *board.BoardState) error {
	drawing, square, err := loosePieceSignals(c.R)
	if err != nil {
		return err
	}
	return board.AddToken(b, board.BoardToken{
		Label: drawing.Name, Kind: drawing.Appearance, Footprint: drawing.Size,
		X: square.X, Y: square.Y,
	}, st.deps.Boards().NewID)
}

// draftMovesToken põe a peça na casa, sem proposta e sem custo.
func draftMovesToken(st Scene, c draftCtx, b *board.BoardState) error {
	square, err := squareOnly(c.R)
	if err != nil {
		return err
	}
	return board.UpdateToken(b, chi.URLParam(c.R, "id"),
		board.ParseTokenPatch(map[string]any{"x": square.X, "y": square.Y}))
}

func draftEditsToken(st Scene, c draftCtx, b *board.BoardState) error {
	var signals tokenSignals
	if err := datastar.ReadSignals(c.R, &signals); err != nil {
		return fmt.Errorf("não entendi o formulário da peça: %v", err)
	}
	name := strings.TrimSpace(signals.Name)
	if name == "" {
		return errors.New("a peça precisa de um nome")
	}
	if !tokenSize(signals.Size) {
		return fmt.Errorf("uma peça ocupa 1, 2, 3 ou 6 quadrados de lado (p107); veio %d", signals.Size)
	}
	return board.UpdateToken(b, chi.URLParam(c.R, "id"),
		board.ParseTokenPatch(map[string]any{"label": name, "footprint": signals.Size}))
}

func draftDuplicatesToken(st Scene, c draftCtx, b *board.BoardState) error {
	// O rascunho não tem fila: laço nulo, sempre peão mudo.
	return board.DuplicateToken(b, chi.URLParam(c.R, "id"), nil, st.deps.Boards().NewID)
}

func draftRemovesToken(st Scene, c draftCtx, b *board.BoardState) error {
	id := chi.URLParam(c.R, "id")
	if board.FindToken(b, id) == nil {
		return fmt.Errorf("peça %q não está no rascunho", id)
	}
	board.RemoveToken(b, id)
	return nil
}

// draftTogglesVisibility esconde a peça que a mesa NÃO deve ver quando a cena
// chegar à mesa — a emboscada montada na quinta-feira.
//
// ALTERNA lendo o estado atual, como na mesa: escrever o valor desejado faria a
// tela ser a fonte da verdade de um estado que é do servidor.
func draftTogglesVisibility(st Scene, c draftCtx, b *board.BoardState) error {
	id := chi.URLParam(c.R, "id")
	token := board.FindToken(b, id)
	if token == nil {
		return fmt.Errorf("peça %q não está no rascunho", id)
	}
	hidden := !token.Hidden
	return board.UpdateToken(b, id, board.ParseTokenPatch(map[string]any{"hidden": hidden}))
}

// ── os MARCADORES ────────────────────────────────────────────────────────────

func draftMarksTheSpot(st Scene, c draftCtx, b *board.BoardState) error {
	square, err := squareOnly(c.R)
	if err != nil {
		return err
	}
	return board.AddMarker(b, board.BoardMarker{
		X: square.X, Y: square.Y,
		Text:  board.NextMarkerLetter(b.Markers),
		Color: board.DefaultMarkerColor(),
		// ESCONDIDO ao nascer, e é a razão de o marcador existir: marcar a
		// armadilha na frente da mesa entrega a armadilha.
		Hidden: true,
	}, st.deps.Boards().NewID)
}

func draftRevealsMarker(st Scene, c draftCtx, b *board.BoardState) error {
	marker, err := draftMarker(c, b)
	if err != nil {
		return err
	}
	return board.UpdateMarker(b, marker.ID, board.MarkerReveal(!marker.Hidden))
}

func draftPaintsMarker(st Scene, c draftCtx, b *board.BoardState) error {
	marker, err := draftMarker(c, b)
	if err != nil {
		return err
	}
	color := chi.URLParam(c.R, "cor")
	if !board.KnownMarkerColor(color) {
		return fmt.Errorf("a cor %q não existe; as do mapa são %s", color, coresEmPortugues())
	}
	return board.UpdateMarker(b, marker.ID, board.NewMarkerColor(color))
}

func draftErasesMarker(st Scene, c draftCtx, b *board.BoardState) error {
	marker, err := draftMarker(c, b)
	if err != nil {
		return err
	}
	board.RemoveMarker(b, marker.ID)
	return nil
}

// draftMarker acha o marcador que o gesto aponta, na cena já lida.
//
// Devolve o MARCADOR e não o id pela mesma razão do `urlMarker` da mesa: revelar
// ALTERNA, e alternar sem ler é escrever `true` por cima de `true`.
func draftMarker(c draftCtx, b *board.BoardState) (board.BoardMarker, error) {
	id := chi.URLParam(c.R, "id")
	for _, m := range b.Markers {
		if m.ID == id {
			return m, nil
		}
	}
	return board.BoardMarker{}, fmt.Errorf("marcador %q não está no rascunho", id)
}
