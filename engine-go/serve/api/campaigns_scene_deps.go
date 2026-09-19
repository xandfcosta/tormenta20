package api

import (
	"context"
	"net/http"
	"t20engine/app/boards"
	"t20engine/app/session"

	"t20engine/domain/board"
	"t20engine/serve/web/campaigns"
)

// A CENA DE CAMPANHAS e o adaptador que cumpre a porta dela (`campaigns.Deps`).
//
// Os métodos moram num arquivo próprio, e não espalhados pelos arquivos de
// domínio, porque juntos eles são UMA coisa: a tradução entre o vocabulário da
// cena e o do hospedeiro. Ler todos de uma vez é o que mostra se a fronteira
// está no lugar — e o sinal de que está é nenhum deles desenhar nada.
//
// Quem os cumpre é o NÚCLEO mais o acervo de lugares, e não o `*Server`. O
// `campaignRules` ainda está aqui por um método só — a faxina de memória de
// apagar a campanha —, e o que ele já respondeu ("de quem é esta mesa" e "quem
// pode entrar nela") virou `session.Access` e `campaign.Seating` na ALE-348.
type campaignsHost struct {
	sceneCore
	rules campaignRules
	// boards é o acervo de LUGARES da campanha, e é a única coisa do domínio ao
	// vivo que esta cena alcança — pelas três perguntas da porta, e não pelo
	// store inteiro. A crônica lista, cria e apaga um lugar; quem MONTA a cena é
	// a cena do tabuleiro.
	//
	// O `sessions` NÃO abre a mesma concessão: ele não atravessa a porta da
	// cena, e vive aqui só para a faxina de memória de apagar a campanha — que é
	// do hospedeiro, e que a cena pede como PERGUNTA (`CampaignDeleted`) e não
	// como store.
	boards   *boards.Store
	sessions *session.Store
}

func (s *Server) campaignsHost() campaignsHost {
	return campaignsHost{sceneCore: s.sceneCore(), rules: s.campaignRules(), boards: s.boards, sessions: s.sessions}
}

// RequesterIsAdmin diz se QUEM PEDE administra o servidor.
//
// O nome não é `IsAdmin` porque este `*Server` já tem um — `IsAdmin(email)`,
// que a administração pede e que olha a CONFIGURAÇÃO. São perguntas diferentes
// com a mesma cara, e reusar o nome faria uma responder pela outra.
func (h campaignsHost) RequesterIsAdmin(r *http.Request) bool { return currentUser(r).IsAdmin }

// CampaignDeleted é a faxina de memória das sessões da campanha.
func (h campaignsHost) CampaignDeleted(ctx context.Context, campanhaID int64) {
	campaignDeleted(ctx, h.rules.queries, h.boards, h.sessions, campanhaID)
}

// ── o ACERVO DE LUGARES da crônica ───────────────────────────────────────────

// Places lista o acervo, já dizendo qual lugar está numa MESA agora.
//
// O casamento é pelo NOME e não pelo id, como o acervo da Mesa já faz: o nome é
// a identidade do lugar dentro da campanha — é assim que o `Archive` decide se
// sobrescreve —, e uma cena aberta do zero com o nome de um lugar guardado É
// aquele lugar, porque é a conta que o arquivamento fará quando ela fechar.
func (h campaignsHost) Places(ctx context.Context, campanhaID int64) []campaigns.PlaceRow {
	naMesa := h.boards.PlacesOnATable(ctx, campanhaID)
	lugares := h.boards.Places(ctx, campanhaID)
	fora := make([]campaigns.PlaceRow, 0, len(lugares))
	for _, l := range lugares {
		fora = append(fora, campaigns.PlaceRow{
			ID: l.ID, Nome: l.Name, Pecas: l.Tokens,
			Quando: l.UpdatedAt, NaMesaID: naMesa[l.Name],
		})
	}
	return fora
}

func (h campaignsHost) NewPlace(ctx context.Context, campanhaID int64, nome, chao string) (int64, error) {
	lugar, err := h.boards.NewPlace(ctx, campanhaID, nome, chao)
	return lugar.ID, err
}

func (h campaignsHost) RemovePlace(ctx context.Context, campanhaID, lugarID int64) error {
	return h.boards.RemovePlace(ctx, campanhaID, lugarID)
}

// Grounds traduz as aparências do tabuleiro para a forma que a tela desenha.
func (h campaignsHost) Grounds() []campaigns.GroundOption {
	fora := make([]campaigns.GroundOption, 0, len(board.PlaceGrounds))
	for _, c := range board.PlaceGrounds {
		fora = append(fora, campaigns.GroundOption{ID: c.ID, Rotulo: c.Rotulo})
	}
	return fora
}
