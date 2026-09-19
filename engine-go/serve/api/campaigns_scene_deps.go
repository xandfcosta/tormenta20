package api

import (
	"context"
	"errors"
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
// Quem os cumpre é o núcleo mais um `campaignRules`, e não o `*Server`: o que
// esta cena precisa da casa é exatamente "as regras de quem é dono do quê".
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

// Join senta alguém à mesa e devolve o MOTIVO da recusa, não o erro.
//
// Aqui é o único lugar do repositório que conhece as duas listas: os sete
// sentinelas do `joinTable` e os seis motivos que a cena declara. A cena colapsa
// "personagem não existe" e "personagem é de outra pessoa" num motivo só, porque
// as duas viram a mesma frase — e distinguir diria a um estranho se um id
// existe.
//
// Quem CLASSIFICA é o hospedeiro, quem escolhe a FRASE é a cena.
func (h campaignsHost) Join(ctx context.Context, campanhaID, heroiID, quemPede int64, convite string) campaigns.JoinRefusal {
	_, err := h.rules.joinTable(ctx, joinRequest{
		CampanhaID: campanhaID, PersonagemID: heroiID,
		Convite: convite, Papel: "player", QuemPede: quemPede,
	})
	switch {
	case err == nil:
		return campaigns.JoinOK
	case errors.Is(err, errCampanhaInexistente):
		return campaigns.JoinNoSuchCampaign
	case errors.Is(err, errConviteExigido):
		return campaigns.JoinNeedsInvite
	case errors.Is(err, errPersonagemInexistente), errors.Is(err, errPersonagemDeOutro):
		return campaigns.JoinNotYourHero
	case errors.Is(err, errJaTemPersonagem):
		return campaigns.JoinAlreadyHasHero
	case errors.Is(err, errAlreadyInCampaign):
		return campaigns.JoinHeroAlreadyThere
	default:
		return campaigns.JoinFailed
	}
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
