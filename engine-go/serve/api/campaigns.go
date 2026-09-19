package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/httpio"

	"t20engine/app"
	"t20engine/app/campaign"
	"t20engine/app/session"
	// O APELIDO é o preço medido de `app/campaign` morar ao lado de
	// `domain/campaign`, e ele é cobrado em DOIS arquivos — este e o
	// `serve/web/campaigns/routes.go`. O plural, que é o padrão da casa para
	// desviar da colisão com o domínio, custaria SETE: `app/campaigns` colide
	// com a CENA (ver o `doc.go` do pacote).
	regra "t20engine/domain/campaign"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// CampaignDTO is the base campaign row (create/update responses).
type CampaignDTO struct {
	ID          int64   `json:"id"`
	OwnerID     int64   `json:"ownerId"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type campaignCharacterDTO struct {
	ID      int64            `json:"id"`
	Name    string           `json:"name"`
	Level   int64            `json:"level"`
	Classes []sheet.ClassDTO `json:"classes"`
}

// campaignListDTO adds the caller's role + own member character (GET /campaigns).
type campaignListDTO struct {
	CampaignDTO
	Role      string                `json:"role"`
	Character *campaignCharacterDTO `json:"character"`
	// OwnerName só existe numa mesa que o chamador NÃO possui, o que hoje quer
	// dizer um admin vendo as de todo mundo. Ausente é o caso normal, então a
	// tela marca a exceção em vez de marcar toda linha.
	OwnerName *string `json:"ownerName,omitempty"`
}

type campaignDetailDTO struct {
	CampaignDTO
	Role string `json:"role"`
	// IgnoredRules acompanha o detalhe porque é nele que a campanha se configura
	// — pedir uma segunda rota para desenhar os interruptores faria a tela piscar
	// entre "tudo ligado" e o estado real.
	IgnoredRules []string `json:"ignoredRules"`
	// Mesma regra da lista: só numa mesa que o chamador não possui. Aqui ele
	// importa MAIS — esta é a tela onde se renomeia e se apaga.
	OwnerName *string `json:"ownerName,omitempty"`
}

func campaignScalars(c sqlcgen.Campaign) CampaignDTO {
	return CampaignDTO{
		ID: c.ID, OwnerID: c.Ownerid, Name: c.Name, Description: dbvalue.NullToPtr(c.Description),
		CreatedAt: c.Createdat, UpdatedAt: c.Updatedat,
	}
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	vistas, err := s.campaignDirectory().Visible(r.Context(), callerOf(currentUser(r)))
	if err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not list campaigns")
		return
	}
	httpio.WriteJSON(w, http.StatusOK, campaignListJSON(vistas))
}

// campaignListJSON veste o resultado do caso de uso na forma do FIO.
//
// A tradução mora aqui e não no caso de uso porque a tag `json:` é a forma de um
// protocolo: o `Seen` que o `app/campaign` devolve não tem tag nenhuma, e é isso
// que deixa a cena em templ ler a MESMA regra sem depender do formato de um
// endpoint que ela não serve.
func campaignListJSON(vistas []campaign.Seen) []campaignListDTO {
	fora := make([]campaignListDTO, 0, len(vistas))
	for _, v := range vistas {
		linha := campaignListDTO{
			CampaignDTO: CampaignDTO{
				ID: v.ID, OwnerID: v.OwnerID, Name: v.Name, Description: v.Description,
				CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
			},
			Role: v.Role,
		}
		if v.OwnerName != "" {
			nome := v.OwnerName
			linha.OwnerName = &nome
		}
		if v.Character != nil {
			linha.Character = &campaignCharacterDTO{
				ID: v.Character.ID, Name: v.Character.Name,
				Level: v.Character.Level, Classes: v.Character.Classes,
			}
		}
		fora = append(fora, linha)
	}
	return fora
}

// A LISTA DE CAMPANHAS mora no `app/campaign` (`Directory.Visible`), inteira.
//
// Eram QUATRO funções aqui, e todas deixaram de existir: a que montava a lista,
// a do `where` que decide quem vê o quê, a que rotulava o dono e o rótulo de
// nome que só ela usava. As quatro decidiam QUEM VÊ O QUÊ — e um `where` que
// ramifica pelo admin é a regra, não encanamento. O que sobrou
// deste lado é a tradução para o FIO, logo acima (ALE-348).

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if !httpio.DecodeJSON(w, r, &body) {
		return
	}
	// As DUAS recusas de uma vez, e em pt-BR: a mesma regra respondendo duas
	// frases diferentes conforme o transporte é o que faz uma delas envelhecer.
	name, descricaoTexto, erros := regra.ValidateText(body.Name, body.Description)
	if len(erros) > 0 {
		httpio.WriteValidationError(w, erros)
		return
	}
	descricao := trimOrNull(&descricaoTexto)
	now := dbvalue.NowISO()
	c, err := s.campaignRules().createCampaign(r.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: currentUser(r).ID, Name: name, Description: descricao,
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not create campaign")
		return
	}
	httpio.WriteJSON(w, http.StatusCreated, campaignScalars(c))
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.campaignRules().ownedCampaign(w, r, id); !ok {
		return
	}
	// ANTES de apagar: a campanha leva as sessões por cascata, e depois disso não
	// há mais como perguntar quais eram — o estado em memória delas ficaria
	// batendo na chave estrangeira até o processo reiniciar.
	s.CampaignDeleted(r.Context(), id)
	if err := s.queries.DeleteCampaign(r.Context(), id); err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not delete campaign")
		return
	}
	httpio.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// A REGRA DE ACESSO A UMA CAMPANHA mora no `app/session` inteira (`Access`), e
// não sobrou tradução nenhuma dela aqui.
//
// Havia duas traduções, e as duas eram duas linhas: chamar o caso de uso e
// trocar a recusa TIPADA por um número de HTTP. As duas deixaram de existir. A cena das
// campanhas chamava a segunda pela porta, com uma assinatura que declarava
// `(papel string, membros int, err error)` e entregava o STATUS no lugar do
// número de membros (ALE-348). Hoje ela chama o `Access` direto.
//
// Quem ainda precisa do número usa o `statusForAccess`, que é o que ele sempre
// foi: tradução de transporte, no lugar onde o transporte mora.

func (rules campaignRules) access() session.Access { return session.NewAccess(rules.queries) }

// callerOf traduz o usuário do hospedeiro na forma que o caso de uso recebe: um
// id e se administra, e nada mais.
func callerOf(user AuthUser) app.Caller {
	return app.Caller{ID: user.ID, IsAdmin: user.IsAdmin}
}

// statusForAccess é a tradução da recusa TIPADA no número do HTTP.
//
// Ela mora aqui e não no `app/`: um caso de uso que devolvesse 403 não poderia
// ser chamado de outro transporte, que é a única coisa que aquela camada compra.
func statusForAccess(err error) int {
	switch {
	case errors.Is(err, app.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, app.ErrForbidden):
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

// loadOwnedCampaign é a regra de campanha SÓ DO DONO, independente de
// transporte: passa o mestre (dono), e o resto recebe Forbidden. Esta função é o
// gargalo de meia dúzia de sítios (renomear/apagar, convite, membros, sessões),
// e é por isso que a exceção do admin custa uma condição só.
// O corpo mora no `app/session` (ALE-344); aqui sobra o número do HTTP.
func (rules campaignRules) loadOwnedCampaign(ctx context.Context, user AuthUser, id int64) (sqlcgen.Campaign, int, error) {
	c, err := rules.access().OwnedCampaign(ctx, callerOf(user), id)
	if err != nil {
		return c, statusForAccess(err), err
	}
	return c, http.StatusOK, nil
}

func (rules campaignRules) ownedCampaign(w http.ResponseWriter, r *http.Request, id int64) (sqlcgen.Campaign, bool) {
	c, status, err := rules.loadOwnedCampaign(r.Context(), currentUser(r), id)
	if err != nil {
		httpio.WriteError(w, status, err.Error())
		return c, false
	}
	return c, true
}

func generateInviteToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
