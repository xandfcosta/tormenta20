package api

import (
	"context"
	"net/http"
	"t20engine/app/boards"
	"t20engine/app/session"
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
	// boards e sessions vivem aqui para UMA coisa só: a faxina de memória de
	// apagar a campanha, que é do hospedeiro e que a cena pede como PERGUNTA
	// (`CampaignDeleted`) e não como store.
	//
	// O acervo de LUGARES não passa mais por aqui — a cena recebe o
	// `boards.Store` direto, como a Mesa (ALE-348).
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
