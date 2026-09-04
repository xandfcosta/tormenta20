package tabuleiro

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"t20engine/db/sqlcgen"
)

// O RASCUNHO DE LUGAR (ALE-292): montar a próxima cena FORA da sessão.
//
// A cortina resolve "montar a cripta enquanto a mesa olha a taverna" DURANTE a
// sessão (ALE-202). O rascunho resolve o outro tempo, que é o da ALE-191:
// preparar a sessão de sábado na quinta-feira, sem ninguém conectado. Decisão do
// dono — os dois convivem, e o GLOSSARIO registra a linha entre eles.
//
// # Por que não há um estado do rascunho em memória
//
// Porque não há o que guardar entre dois gestos. O tabuleiro vivo mora em
// memória por três razões que o rascunho não tem: a mesa inteira lê o mesmo
// estado a cada quadro do SSE, gravar é assíncrono, e a versão precisa subir
// para o cliente reconhecer o que é mais novo. Aqui há um leitor só, o gesto é
// humano, e a próxima leitura vem do disco de qualquer jeito.
//
// Então cada gesto LÊ a cena guardada, aplica a MESMA função pura que a mesa
// aplica (`AddToken`, `PaintTerrain`, `RemoveMarker`…) e grava de volta. Sem
// cache, sem despejo, sem uma segunda verdade sobre onde as peças estão.

// EditPlace roda uma mutação sobre a cena guardada de um lugar.
//
// `liveSessionID` é a sessão AO VIVO da campanha, ou zero quando não há uma —
// ela chega resolvida de quem chamou, como o orçamento chega ao `ProposeMove`,
// porque quem sabe qual sessão está viva é o gateway e a trava daqui não pode
// esperar por I/O que ela mesma dispare.
//
// A gravação passa pelo `SavePlaceScene`, então a conferência dele vale para
// todo gesto: a mutação é pura e não sabe do teto de peças nem da coordenada
// sã. Uma recusa deixa o acervo INTACTO, porque quem escreve é a gravação e ela
// não chega a acontecer.
func (bs *BoardStore) EditPlace(
	ctx context.Context, campaignID, liveSessionID, placeID int64, fn func(*BoardState) error,
) (*BoardState, error) {
	cena, err := bs.PlaceScene(ctx, campaignID, placeID)
	if err != nil {
		return nil, err
	}
	if err := bs.refusesIfLive(ctx, liveSessionID, cena.Place); err != nil {
		return nil, err
	}
	if err := fn(cena); err != nil {
		return nil, err
	}
	if err := bs.SavePlaceScene(ctx, campaignID, placeID, cena); err != nil {
		return nil, err
	}
	return cena, nil
}

// refusesIfLive recusa o rascunho do lugar que está ABERTO numa mesa ao vivo.
//
// Seriam duas verdades sobre onde as peças estão, e a de fora perderia em
// SILÊNCIO: encerrar a aba chama o `Archive`, que sobrescreve o lugar de mesmo
// nome — a noite de trabalho no rascunho sumiria sem uma linha na tela.
//
// Pelo NOME e não pelo id, porque é o nome que identifica o lugar dentro da
// campanha — é assim que o `Archive` decide se sobrescreve, e é a mesma conta
// que o acervo já faz para escrever "nesta mesa agora".
func (bs *BoardStore) refusesIfLive(ctx context.Context, liveSessionID int64, nome string) error {
	if liveSessionID == 0 {
		return nil
	}
	for _, aberto := range bs.OpenBoards(ctx, liveSessionID) {
		if aberto.Place == nome {
			return fmt.Errorf(
				"%q está aberto numa mesa agora: encerre a aba antes de montar o rascunho, senão o que você montar aqui some quando ela for encerrada",
				nome)
		}
	}
	return nil
}

// NewPlace devolve o lugar em que o mestre vai montar a cena, criando-o vazio
// quando ele ainda não existe.
//
// DEVOLVE O EXISTENTE em vez de recusar o nome repetido, e não é conveniência: o
// nome É a identidade do lugar dentro da campanha (ver o `Archive`, que
// sobrescreve por nome). Recusar diria que há dois conceitos onde há um, e criar
// um segundo deixaria duas tavernas quase iguais no acervo — que é exatamente o
// que o `Archive` existe para não fazer.
//
// O `terrain` só vale para o lugar que NASCE: pedi-lo de novo sobre uma cena
// montada repintaria o chão dela por baixo do pano, e quem quer trocar o chão
// tem o gesto do próprio rascunho para isso.
func (bs *BoardStore) NewPlace(ctx context.Context, campaignID int64, name, terrain string) (Place, error) {
	if existente, err := bs.q.FindCampaignPlaceByName(ctx, sqlcgen.FindCampaignPlaceByNameParams{
		Campaignid: campaignID,
		Name:       name,
	}); err == nil {
		return Place{
			ID: existente.ID, Name: existente.Name,
			Tokens: countTokens(existente.State), UpdatedAt: existente.Updatedat,
		}, nil
	}
	// A cena nasce com a versão em 1 e as peças em fatia VAZIA, como a do
	// `Open`: `null` no JSON derruba quem indexa `tokens.length`, e é o mesmo
	// cuidado que o `storedScene` toma na volta.
	blob, err := json.Marshal(&BoardState{
		Version: 1, Place: name, Terrain: terrain, Tokens: []BoardToken{},
	})
	if err != nil {
		return Place{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	row, err := bs.q.SaveCampaignPlace(ctx, sqlcgen.SaveCampaignPlaceParams{
		Campaignid: campaignID, Name: name, State: string(blob),
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		return Place{}, err
	}
	return Place{ID: row.ID, Name: row.Name, Tokens: 0, UpdatedAt: row.Updatedat}, nil
}
