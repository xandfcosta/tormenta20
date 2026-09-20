package api

import (
	"net/http"
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
}

func (s *Server) campaignsHost() campaignsHost {
	return campaignsHost{sceneCore: s.sceneCore(), rules: s.campaignRules()}
}

// RequesterIsAdmin diz se QUEM PEDE administra o servidor.
//
// O nome não é `IsAdmin` porque este `*Server` já tem um — `IsAdmin(email)`,
// que a administração pede e que olha a CONFIGURAÇÃO. São perguntas diferentes
// com a mesma cara, e reusar o nome faria uma responder pela outra.
func (h campaignsHost) RequesterIsAdmin(r *http.Request) bool { return currentUser(r).IsAdmin }
