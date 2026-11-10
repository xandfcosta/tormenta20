package sheetui

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/starfederation/datastar-go/datastar"

	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
)

// O FLUXO AO VIVO DA FICHA SOLTA: o mesmo aviso que a mesa dá à "Minha ficha", para a ficha aberta em
// outra aba acompanhar o dano, a condição e a regra da mesa sem o jogador recarregar.
//
// Ele só avisa a VERSÃO nova (o `updatedAt` da linha); quem repede a ficha é o cliente, na aba em que
// está (`StandaloneRefetch`). Mandar a ficha inteira por aqui seria recomputar sete painéis a cada
// mudança, e o remendo da ficha já sabe pedir só o que muda.
func (s Scene) sheetLive(w http.ResponseWriter, r *http.Request) {
	id, ok := uRLCharacter(w, r)
	if !ok {
		return
	}
	// A ASSINATURA vem antes da leitura: uma mudança entre as duas se perderia.
	sub, stop := s.deps.Bus().Subscribe(events.OfCharacter(id))
	defer stop()
	// A recusa vem ANTES do `NewSSE`, que já escreve os cabeçalhos: quem não é dono merece um 403 legível.
	row, status, err := s.reachableCharacter(r.Context(), id, s.deps.CurrentUserID(r))
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	sse := datastar.NewSSE(w, r)
	// A VERSÃO DE AGORA, uma vez por conexão: o fluxo reconecta depois de um deploy ou de o celular
	// dormir, e é assim que a página descobre o que mudou enquanto estava fora. A guarda do cliente não
	// repede quando a versão é a mesma.
	if err := sse.PatchSignals([]byte(`{"sheet_version":` + strconv.Quote(row.Updatedat) + `}`)); err != nil {
		return
	}
	announced := row.Updatedat
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.deps.LiveStreamsClosing():
			// ABORTA em vez de `return`, pela razão do fluxo da mesa: fechar limpo é resposta completa
			// para o Datastar, que não reconectaria depois do deploy.
			panic(http.ErrAbortHandler)
		case <-sub.C:
			row, err := s.deps.Queries().GetCharacter(r.Context(), id)
			if err != nil || row.Updatedat == announced {
				continue
			}
			if err := sse.PatchSignals([]byte(`{"sheet_version":` + strconv.Quote(row.Updatedat) + `}`)); err != nil {
				return
			}
			announced = row.Updatedat
		}
	}
}

// standaloneRefetch é o ouvido da ficha SOLTA: o fluxo dela avisa a versão nova, e a ficha se repede onde
// está, com a aba e a visão (`commandRoute`). A da mesa tem o dela (`SheetRefetch`), com a aba num sinal.
//
// Ele ESPERA um pouco (`__debounce` no atributo): o gesto da própria aba grava, avisa e ainda monta a
// resposta, e o aviso chegava antes dela, com a tela na versão velha. Esperando, a resposta do gesto já
// pousou e a guarda vê a versão nova. Um gesto mais lento que a espera custa só uma busca a mais.
//
// @example standaloneRefetch(v) // "$sheet_version !== …dataset.versao && @get('/personagens/13?tab=combat')"
func standaloneRefetch(v View) string {
	return sheetVersionGuard + sheetGet(v)
}

// reachableCharacter é a linha da ficha, se quem pede a alcança: a ficha, os gestos e o fluxo dela perguntam
// o mesmo, e a resposta é uma só.
//
// Alcança o DONO, e o MESTRE da campanha da cópia, que abre e edita a ficha do jogador pelo "Ficha" da fila
// (decisão do dono). O rascunho continua só do jogador: ele não tem campanha, e não há mestre a perguntar.
func (s Scene) reachableCharacter(ctx context.Context, id, userID int64) (sqlcgen.Character, int, error) {
	row, err := s.deps.Queries().GetCharacter(ctx, id)
	if err != nil {
		return sqlcgen.Character{}, http.StatusNotFound, fmt.Errorf("personagem %d não existe", id)
	}
	if row.Ownerid == userID {
		return row, http.StatusOK, nil
	}
	if row.CampaignId.Valid {
		campaign, err := s.deps.Queries().GetCampaign(ctx, row.CampaignId.Int64)
		if err == nil && campaign.Ownerid == userID {
			return row, http.StatusOK, nil
		}
	}
	return sqlcgen.Character{}, http.StatusForbidden, fmt.Errorf("esta ficha não é sua")
}

// mastersTheCharacter diz se quem pede é o MESTRE da campanha da ficha. É ele quem encanta (p333), seja a
// ficha de um jogador, seja a dele mesmo na mesa que ele mestra. A ficha fora de campanha não tem mestre.
func (s Scene) mastersTheCharacter(ctx context.Context, row sqlcgen.Character, userID int64) bool {
	if !row.CampaignId.Valid {
		return false
	}
	campaign, err := s.deps.Queries().GetCampaign(ctx, row.CampaignId.Int64)
	return err == nil && campaign.Ownerid == userID
}
