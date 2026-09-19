package api

import (
	"net/http"

	"t20engine/infra/config"
	"t20engine/serve/web/hub"
	"t20engine/serve/web/routes"
)

// O HUB, com adaptador próprio (ALE-278, fatia 6).
//
// Das seis assinaturas que ele pede, duas são do núcleo e quatro estão aqui. O
// que o adaptador carrega além do núcleo é a CONFIGURAÇÃO, e só ela: cunhar
// convite já é função de pacote sobre as consultas, e as outras duas não leem
// estado nenhum.
type hubHost struct {
	sceneCore
	cfg config.Config
}

func (s *Server) hubHost() hubHost { return hubHost{sceneCore: s.sceneCore(), cfg: s.cfg} }

// CurrentViewer traduz quem está pedindo para a língua do HUB (ALE-278).
//
// A tradução é o preço da fronteira, e ela é barata: quatro campos. O que ela
// compra é o hub não conhecer o `AuthUser` — e portanto não importar este
// pacote, que o importa de volta para montar rota.
func (h hubHost) CurrentViewer(r *http.Request) hub.Viewer {
	eu := currentUser(r)
	return hub.Viewer{ID: eu.ID, Email: eu.Email, Name: eu.Name, IsAdmin: eu.IsAdmin}
}

// ExpiredSessionCookie é o que o hub pede da CASA: o formato do biscoito
// depende da configuração e da política, e nenhuma das duas é da tela. Cunhar
// convite era o outro, e virou caso de uso (ALE-349).
func (h hubHost) ExpiredSessionCookie() *http.Cookie { return sessionCookie(h.cfg, "", -1) }

// TableRoute é o endereço de uma sessão ao vivo. Quem sabe onde cada cena está
// montada é quem monta.
func (h hubHost) TableRoute(campaignID, sessionID int64) string {
	return routes.Session(campaignID, sessionID)
}
