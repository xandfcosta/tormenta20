package table

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar-go/datastar"

	"t20engine/engine"
	"t20engine/tabuleiro"
	"t20engine/web/ui"
)

// OS GESTOS DO RASCUNHO (ALE-292).
//
// Cada um é a MESMA mutação pura que a mesa aplica, num `EditPlace` em vez de
// num tabuleiro vivo. Eles são gêmeos dos da Mesa e não os mesmos, e a escolha
// é deliberada: o gêmeo é fiação, sem uma linha de regra — quem confere a
// coordenada, o teto de peças e o tamanho da criatura continua sendo o
// `tabuleiro`, num lugar só.
//
// O que os separa não é acidente de encanamento, é o que o gesto SIGNIFICA. Na
// mesa, arrastar propõe um movimento com custo e vez, e alguém confirma; aqui a
// peça vai para a casa e acabou. Fundir os dois caminhos num handler com um `if`
// faria a Mesa — a superfície mais exercitada do app — depender de uma
// ramificação que só o rascunho percorre.

// DraftRoutes registra as quatorze rotas do rascunho.
//
// O endereço é da CAMPANHA e não da mesa: o lugar é do acervo e sobrevive a
// qualquer sessão, que é a diferença entre o rascunho e a cortina.
func (s Scene) DraftRoutes(r chi.Router) {
	r.Get("/campanhas/{campaignId}/lugares/{placeId}", s.handleDraftPage)
	base := "/campanhas/{campaignId}/lugares/{placeId}/tabuleiro"
	// O TRAÇO, como na mesa: as rotas de terreno recebem de ONDE ATÉ ONDE o dedo
	// andou desde o aviso anterior. Ver `tabuleiro.StrokeSquares`.
	r.Post(base+"/terreno/{especie}/{x}/{y}/ate/{x2}/{y2}", s.draftCommand(draftPaintsTerrain))
	r.Post(base+"/terreno/limpar/{x}/{y}/ate/{x2}/{y2}", s.draftCommand(draftClearsTerrain))
	r.Post(base+"/terreno/{especie}/retangulo/{x}/{y}/{x2}/{y2}", s.draftCommand(draftFillsRect))
	r.Post(base+"/terreno/limpar/retangulo/{x}/{y}/{x2}/{y2}", s.draftCommand(draftClearsRect))
	r.Post(base+"/pecas/nova/{x}/{y}", s.draftCommand(draftNewLoosePiece))
	// MOVER é o gesto que NÃO tem gêmeo na mesa, e é a diferença do rascunho:
	// lá o arrasto manda uma PARADA e o servidor devolve uma proposta com custo,
	// aqui ele põe a peça na casa. Ver `draftMoveDrop`.
	r.Post(base+"/pecas/{id}/mover/{x}/{y}", s.draftCommand(draftMovesToken))
	r.Post(base+"/pecas/{id}/editar", s.draftCommand(draftEditsToken))
	r.Post(base+"/pecas/{id}/duplicar", s.draftCommand(draftDuplicatesToken))
	r.Post(base+"/pecas/{id}/remover", s.draftCommand(draftRemovesToken))
	r.Post(base+"/pecas/{id}/visibilidade", s.draftCommand(draftTogglesVisibility))
	r.Post(base+"/marcadores/novo/{x}/{y}", s.draftCommand(draftMarksTheSpot))
	r.Post(base+"/marcadores/{id}/revelar", s.draftCommand(draftRevealsMarker))
	r.Post(base+"/marcadores/{id}/cor/{cor}", s.draftCommand(draftPaintsMarker))
	r.Post(base+"/marcadores/{id}/remover", s.draftCommand(draftErasesMarker))
	// MEDIR não é comandar, e por isso as duas de baixo NÃO passam pelo
	// `draftCommand`: elas não mutam nada e respondem só com sinais. Um
	// `EditPlace` aqui gravaria o acervo a cada movimento do dedo sobre a régua.
	r.Post(base+"/regua", s.handleDraftRuler)
	r.Post(base+"/gabarito/{tipo}/{tamanho}/{x}/{y}/{mx}/{my}", s.handleDraftTemplate)
}

// ── MEDIR o rascunho (ALE-293) ───────────────────────────────────────────────
//
// A régua e o gabarito são desenhadas no rascunho desde a ALE-292, porque elas
// não são `SoMestre` e o trilho inteiro veio junto com o `boardTable`. As rotas
// não vieram, e o resultado era o pior defeito desta casa: **o gesto oferecido
// que o servidor não atende** — 404, tela que não muda, e nada explicando por
// quê. Quem varre isso agora é o `TestEveryDraftToolHasARoute`.
//
// E elas PERTENCEM aqui, o que eu tinha julgado errado. O argumento de deixá-las
// de fora era que medem "dá para acertar daqui?", pergunta de combate. Decisão
// do dono: *"cabe a bola de fogo nesta sala?"* é pergunta de PREPARAÇÃO — é
// montando a cripta que se decide o tamanho dela.

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
	paradas, err := rulerStops(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeSignals(w, r, polylineReading(paradas))
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
	tipo, err := urlTemplate(chi.URLParam(r, "tipo"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	origem, erroOrigem := quadradoDoCaminho(r, "x", "y")
	mira, erroMira := quadradoDoCaminho(r, "mx", "my")
	if erroOrigem != nil || erroMira != nil {
		http.Error(w, "a origem e a mira do gabarito precisam ser dois pares de números", http.StatusBadRequest)
		return
	}
	// A MIRA ainda não foi dada quando ela é a própria origem: o cone e a linha
	// precisam apontar para algum lado, e escolher um pelo servidor seria
	// inventar a decisão que falta.
	if pointsTemplate(tipo) && mira == origem {
		writeSignals(w, r, map[string]any{
			"gabaritopath": "", "gabaritotexto": "Clique de novo para apontar.",
		})
		return
	}
	casas := engine.AreaSquares(origem, engine.Area{
		Kind: tipo, Size: templateSize(chi.URLParam(r, "tamanho")),
		Direction: templateDirection(origem, mira),
	})
	// A cena pode ter sumido entre desenhar a tela e medir — outro navegador do
	// mestre pode ter apagado o lugar. O desenho sai de qualquer jeito; o que
	// falta é a lista de quem ele pega, e "ninguém" é a resposta honesta para um
	// mapa que não existe mais.
	cena, err := s.deps.Boards().PlaceScene(r.Context(), c.CampaignID, c.PlaceID)
	if err != nil {
		cena = nil
	}
	writeSignals(w, r, map[string]any{
		"gabaritopath":  squaresPath(casas),
		"gabaritotexto": takesTemplateWho(cena, casas),
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
		Titulo: "Rascunho · " + view.Lugar,
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
	mutar func(Scene, draftCtx, *tabuleiro.BoardState) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := s.draftGm(w, r)
		if !ok {
			return
		}
		// A leitura dos sinais vem ANTES do `NewSSE`, e a ordem é obrigatória: o
		// `NewSSE` assume a resposta e fecha o corpo do pedido. A ordem inversa
		// passou VERDE em teste de handler e falhou no servidor de verdade, e o
		// comentário do `handleTableInitiative` conta a história inteira.
		_, err := s.deps.Boards().EditPlace(r.Context(), c.CampaignID, c.PlaceID,
			func(b *tabuleiro.BoardState) error { return mutar(s, c, b) })

		sse := datastar.NewSSE(w, r)
		frase := ""
		if err != nil {
			frase = err.Error()
		}
		// REDESENHA NOS DOIS CAMINHOS, inclusive na recusa: é o que mostra que a
		// cena continua como estava, ao lado da frase que diz por quê. Sem isso
		// uma recusa deixa a tela mostrando o gesto que o servidor não aceitou.
		if view, erroDeLeitura := s.draftPageOf(r.Context(), c); erroDeLeitura == nil {
			if fragmento, erroDeDesenho := ui.RenderFragment(r.Context(), draftBoardRegion(view)); erroDeDesenho == nil {
				_ = sse.PatchElements(fragmento)
			}
		}
		_ = sse.MarshalAndPatchSignals(map[string]any{"erroDoComando": frase})
	}
}

// ── o TERRENO (T20 p238) ─────────────────────────────────────────────────────

func draftPaintsTerrain(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	traco, err := tracoDaURL(c.R)
	if err != nil {
		return err
	}
	especie := tabuleiro.KnownTerrainKind(chi.URLParam(c.R, "especie"))
	// O `apagar` continua sendo MODO da ferramenta e não caminho, como na mesa:
	// ele vale para o arraste inteiro, e não para um quadrado.
	ligado := c.R.URL.Query().Get("apagar") == ""
	for _, casa := range traco {
		tabuleiro.PaintTerrain(b, casa, especie, ligado)
	}
	return nil
}

func draftClearsTerrain(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	traco, err := tracoDaURL(c.R)
	if err != nil {
		return err
	}
	for _, casa := range traco {
		tabuleiro.ClearSquare(b, casa)
	}
	return nil
}

func draftFillsRect(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	casas, err := urlRect(c.R)
	if err != nil {
		return err
	}
	especie := tabuleiro.KnownTerrainKind(chi.URLParam(c.R, "especie"))
	for _, casa := range casas {
		tabuleiro.PaintTerrain(b, casa, especie, true)
	}
	return nil
}

func draftClearsRect(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	casas, err := urlRect(c.R)
	if err != nil {
		return err
	}
	for _, casa := range casas {
		tabuleiro.ClearSquare(b, casa)
	}
	return nil
}

// ── as PEÇAS ─────────────────────────────────────────────────────────────────

// draftNewLoosePiece é a peça avulsa (ALE-291): a porta, o baú, o barril.
//
// Ela lê a MESMA tira que a mesa lê (`loosePieceSignals`), com as mesmas
// recusas — nome obrigatório, tamanho do livro (p107), aparência conhecida.
func draftNewLoosePiece(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	casa, err := quadradoDaURL(c.R)
	if err != nil {
		return err
	}
	desenho, err := loosePieceSignals(c.R)
	if err != nil {
		return err
	}
	return tabuleiro.AddToken(b, tabuleiro.BoardToken{
		Label: desenho.Nome, Kind: desenho.Aparencia, Footprint: desenho.Tamanho,
		X: casa.X, Y: casa.Y,
	}, st.deps.Boards().NewID)
}

// draftMovesToken põe a peça na casa, sem proposta e sem custo.
func draftMovesToken(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	casa, err := quadradoDaURL(c.R)
	if err != nil {
		return err
	}
	return tabuleiro.UpdateToken(b, chi.URLParam(c.R, "id"),
		tabuleiro.ParseTokenPatch(map[string]any{"x": casa.X, "y": casa.Y}))
}

func draftEditsToken(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	var sinais tokenSignals
	if err := datastar.ReadSignals(c.R, &sinais); err != nil {
		return fmt.Errorf("não entendi o formulário da peça: %v", err)
	}
	nome := strings.TrimSpace(sinais.Nome)
	if nome == "" {
		return errors.New("a peça precisa de um nome")
	}
	if !tokenSize(sinais.Tamanho) {
		return fmt.Errorf("uma peça ocupa 1, 2, 3 ou 6 quadrados de lado (p107); veio %d", sinais.Tamanho)
	}
	return tabuleiro.UpdateToken(b, chi.URLParam(c.R, "id"),
		tabuleiro.ParseTokenPatch(map[string]any{"label": nome, "footprint": sinais.Tamanho}))
}

func draftDuplicatesToken(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	return tabuleiro.DuplicateToken(b, chi.URLParam(c.R, "id"), st.deps.Boards().NewID)
}

func draftRemovesToken(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	id := chi.URLParam(c.R, "id")
	if tabuleiro.FindToken(b, id) == nil {
		return fmt.Errorf("peça %q não está no rascunho", id)
	}
	tabuleiro.RemoveToken(b, id)
	return nil
}

// draftTogglesVisibility esconde a peça que a mesa NÃO deve ver quando a cena
// chegar à mesa — a emboscada montada na quinta-feira.
//
// ALTERNA lendo o estado atual, como na mesa: escrever o valor desejado faria a
// tela ser a fonte da verdade de um estado que é do servidor.
func draftTogglesVisibility(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	id := chi.URLParam(c.R, "id")
	peca := tabuleiro.FindToken(b, id)
	if peca == nil {
		return fmt.Errorf("peça %q não está no rascunho", id)
	}
	escondida := !peca.Hidden
	return tabuleiro.UpdateToken(b, id, tabuleiro.ParseTokenPatch(map[string]any{"hidden": escondida}))
}

// ── os MARCADORES (ALE-195) ──────────────────────────────────────────────────

func draftMarksTheSpot(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	casa, err := quadradoDaURL(c.R)
	if err != nil {
		return err
	}
	return tabuleiro.AddMarker(b, tabuleiro.BoardMarker{
		X: casa.X, Y: casa.Y,
		Text:  tabuleiro.NextMarkerLetter(b.Markers),
		Color: tabuleiro.DefaultMarkerColor(),
		// ESCONDIDO ao nascer, e é a razão de o marcador existir: marcar a
		// armadilha na frente da mesa entrega a armadilha.
		Hidden: true,
	}, st.deps.Boards().NewID)
}

func draftRevealsMarker(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	marcador, err := draftMarker(c, b)
	if err != nil {
		return err
	}
	return tabuleiro.UpdateMarker(b, marcador.ID, tabuleiro.MarkerReveal(!marcador.Hidden))
}

func draftPaintsMarker(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	marcador, err := draftMarker(c, b)
	if err != nil {
		return err
	}
	cor := chi.URLParam(c.R, "cor")
	if !tabuleiro.KnownMarkerColor(cor) {
		return fmt.Errorf("a cor %q não existe; as do mapa são %s", cor, coresEmPortugues())
	}
	return tabuleiro.UpdateMarker(b, marcador.ID, tabuleiro.NewMarkerColor(cor))
}

func draftErasesMarker(st Scene, c draftCtx, b *tabuleiro.BoardState) error {
	marcador, err := draftMarker(c, b)
	if err != nil {
		return err
	}
	tabuleiro.RemoveMarker(b, marcador.ID)
	return nil
}

// draftMarker acha o marcador que o gesto aponta, na cena já lida.
//
// Devolve o MARCADOR e não o id pela mesma razão do `urlMarker` da mesa: revelar
// ALTERNA, e alternar sem ler é escrever `true` por cima de `true`.
func draftMarker(c draftCtx, b *tabuleiro.BoardState) (tabuleiro.BoardMarker, error) {
	id := chi.URLParam(c.R, "id")
	for _, m := range b.Markers {
		if m.ID == id {
			return m, nil
		}
	}
	return tabuleiro.BoardMarker{}, fmt.Errorf("marcador %q não está no rascunho", id)
}
