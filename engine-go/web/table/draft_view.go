package table

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"t20engine/tabuleiro"
)

// O RASCUNHO DE LUGAR desenhado (ALE-292): a superfície do tabuleiro apontada
// para o ACERVO em vez de para a mesa.
//
// A cena é a MESMA — mesmo trilho, mesmos pincéis, mesma peça avulsa, mesmos
// marcadores — porque montar a cripta na quinta-feira é o mesmo trabalho que
// montá-la ao vivo. O que muda é o destino dos gestos (`BoardView.Base`) e o que
// não existe fora da sessão (`BoardView.Rascunho`).
//
// Ela mora no pacote da Mesa e não num pacote próprio porque é o TABULEIRO que
// ela reusa, e o tabuleiro é daqui: um pacote separado precisaria exportar o
// `boardTable`, o `boardViewOf` e a dúzia de decisões que os dois carregam — e
// duas cópias do `BoardView` é como as duas superfícies divergem em silêncio.

// draftView é a página do rascunho.
type draftView struct {
	CampaignID int64
	// CampanhaNome é para onde o "voltar" leva, escrito: fora da sessão o mestre
	// veio da crônica, e um "voltar" sem nome não diz para onde.
	CampanhaNome string
	Lugar        string
	// GravadoEm é o instante da última gravação, já legível.
	//
	// Ele existe porque o rascunho grava a CADA GESTO, sem botão de salvar: sem
	// nada na tela dizendo isso, a única leitura possível é "será que perdi o
	// que fiz?", e a resposta certa é invisível.
	GravadoEm string
	Tabuleiro BoardView
}

// draftBoardOf monta o tabuleiro do rascunho a partir da cena guardada.
//
// Ela passa pelo MESMO `boardViewOf` da mesa, com o estado ao vivo NULO — e os
// três ajudantes que ele chama já tratam o nulo como caminho normal
// (`MapCandidates`, `moveBoard`, `reachAndTarget` devolvem vazio). Não há
// segunda tradução de peça, de marcador nem de terreno: uma segunda seria a que
// desenha a peça Grande em 1×1 no dia em que a primeira mudar.
//
// O papel é `gm` porque o rascunho é privativo por construção — quem não mestra
// a campanha não chega até aqui (ver `draftGm`). Não é a redação do
// `BoardForRole` sendo pulada: é que não há mesa para redigir nada PARA.
func draftBoardOf(cena *tabuleiro.BoardState, campaignID, placeID int64) BoardView {
	v := boardViewOf(cena, nil, nil, "", tabuleiro.Mover{Role: "gm"}, nil, campaignID, 0)
	// O `boardViewOf` escreve o `Base` da MESA, que é o destino de 99% das
	// chamadas dele. Aqui ele é reescrito, e o par é o que faz os gestos
	// desta tela postarem no acervo em vez de numa sessão que não existe.
	v.Rascunho = true
	v.Base = placeDraftBase(campaignID, placeID)
	return v
}

// draftParams lê a campanha e o lugar da URL.
//
// Irmã do `tableParams`, e separada dele porque o segundo id é OUTRA coisa: lá
// é a sessão, aqui é o lugar do acervo. Uma função só com um nome genérico para
// "os dois ids" esconderia justamente a diferença que esta issue existe para
// desenhar.
func draftParams(w http.ResponseWriter, r *http.Request) (campaignID, placeID int64, ok bool) {
	campanha, erroCampanha := strconv.ParseInt(chi.URLParam(r, "campaignId"), 10, 64)
	lugar, erroLugar := strconv.ParseInt(chi.URLParam(r, "placeId"), 10, 64)
	if erroCampanha != nil || erroLugar != nil {
		http.Error(w, fmt.Sprintf("endereço de rascunho inválido: campanha %q, lugar %q",
			chi.URLParam(r, "campaignId"), chi.URLParam(r, "placeId")), http.StatusBadRequest)
		return 0, 0, false
	}
	return campanha, lugar, true
}

// draftGm é a trava, e ela é do SERVIDOR: só quem MESTRA a campanha monta o
// acervo dela.
//
// Ela é resolvida UMA vez, no gateway, pela mesma razão que a aba do comando é:
// a mesma pergunta escrita em quinze mutações é quinze lugares para a décima
// sexta esquecer. A OUTRA trava do rascunho — o lugar que está aberto numa mesa
// — não está aqui: ela é do domínio, e o `EditPlace` a aplica a todo gesto.
func (s Scene) draftGm(w http.ResponseWriter, r *http.Request) (draftCtx, bool) {
	campaignID, placeID, ok := draftParams(w, r)
	if !ok {
		return draftCtx{}, false
	}
	userID := s.deps.CurrentUserID(r)
	campanha, status, err := s.deps.PlaceDraftCampaign(r.Context(), userID, campaignID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return draftCtx{}, false
	}
	return draftCtx{
		R: r, CampaignID: campaignID, PlaceID: placeID, CampanhaNome: campanha.Name,
	}, true
}

// draftCtx é o que todo gesto do rascunho recebe, já resolvido.
type draftCtx struct {
	R          *http.Request
	CampaignID int64
	PlaceID    int64
	// CampanhaNome é o nome da crônica, para o cabeçalho e o "voltar".
	CampanhaNome string
}

// draftPageOf monta a página inteira: a cena guardada, o tabuleiro e a moldura.
func (s Scene) draftPageOf(ctx context.Context, c draftCtx) (draftView, error) {
	cena, err := s.deps.Boards().PlaceScene(ctx, c.CampaignID, c.PlaceID)
	if err != nil {
		return draftView{}, err
	}
	return draftView{
		CampaignID:   c.CampaignID,
		CampanhaNome: c.CampanhaNome,
		Lugar:        cena.Place,
		GravadoEm:    savedAt(time.Now()),
		Tabuleiro:    draftBoardOf(cena, c.CampaignID, c.PlaceID),
	}, nil
}

// savedAt escreve a HORA e não a data: o rascunho grava a cada gesto, então o
// que a pessoa quer confirmar é "isto acabou de acontecer" — uma data completa
// responderia a outra pergunta e ocuparia a linha inteira.
func savedAt(quando time.Time) string {
	return fmt.Sprintf("gravado às %s", quando.Format("15:04"))
}
