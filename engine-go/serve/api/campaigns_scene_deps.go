package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"t20engine/infra/db/dbvalue"

	"t20engine/domain/board"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
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
	boards   *board.BoardStore
	sessions *live.SessionStore
}

func (s *Server) campaignsHost() campaignsHost {
	return campaignsHost{sceneCore: s.sceneCore(), rules: s.campaignRules(), boards: s.boards, sessions: s.sessions}
}

// List traduz o `campaignList` para a forma que a CENA declarou.
//
// A cena NÃO consome o `campaignListDTO` direto: ele é a resposta de
// `GET /campaigns`, com tag `json:` em cada campo, e uma tela que o lesse
// passaria a depender do formato de um endpoint que ela não serve.
//
// O mapeamento é aqui e a CONSULTA continua uma só: duplicá-la do lado da cena
// seria trocar um acoplamento por uma cópia.
func (h campaignsHost) List(ctx context.Context, userID int64, admin bool) ([]campaigns.ListRow, error) {
	linhas, err := h.rules.campaignList(ctx, AuthUser{ID: userID, IsAdmin: admin})
	if err != nil {
		return nil, err
	}
	fora := make([]campaigns.ListRow, 0, len(linhas))
	for _, c := range linhas {
		linha := campaigns.ListRow{ID: c.ID, Name: c.Name, Role: c.Role}
		if c.Description != nil {
			linha.Description = *c.Description
		}
		if c.OwnerName != nil {
			linha.OwnerName = *c.OwnerName
		}
		if c.Character != nil {
			linha.Character = &campaigns.RowCharacter{
				ID: c.Character.ID, Name: c.Character.Name,
				Level: c.Character.Level, Classes: c.Character.Classes,
			}
		}
		fora = append(fora, linha)
	}
	return fora, nil
}

// RoleIn é o papel de quem pede numa campanha, e quantos membros ela tem.
func (h campaignsHost) RoleIn(ctx context.Context, userID int64, c sqlcgen.Campaign) (string, int, error) {
	return h.rules.roleIn(ctx, AuthUser{ID: userID}, c)
}

// OwnerNames traduz o dono de cada campanha em nome, para a lista do admin.
func (h campaignsHost) OwnerNames(ctx context.Context, cs []sqlcgen.Campaign, quemPede int64) map[int64]string {
	return h.rules.ownerNames(ctx, cs, quemPede)
}

// IgnoredRules são as regras que o mestre DESLIGOU nesta campanha.
func (h campaignsHost) IgnoredRules(ctx context.Context, campanhaID int64) []string {
	return h.rules.ignoredRulesOf(ctx, campanhaID)
}

// SaveIgnoredRules troca o conjunto INTEIRO, e é idempotente de propósito.
func (h campaignsHost) SaveIgnoredRules(ctx context.Context, campanhaID int64, regras []string) error {
	return h.rules.saveIgnoredRules(ctx, campanhaID, regras)
}

// SaveText grava o nome e a descrição da campanha.
//
// O SQL mora aqui e não na cena: cena que compõe SQL é cena com o banco dentro.
// O que atravessa a fronteira é a PERGUNTA — o hospedeiro é que sabe o nome da
// coluna, que vazio é NULL e que a linha tem um `updatedAt` a tocar.
func (h campaignsHost) SaveText(ctx context.Context, campanhaID int64, nome, descricao string) error {
	var set setBuilder
	set.Add("name = ?", nome)
	set.Add("description = ?", nullableArg(trimOrNull(&descricao)))
	return set.execTouched(ctx, h.rules.db, "UPDATE campaigns", campanhaID)
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

// OpenTable abre a mesa COM link de convite.
//
// Chamar o `CreateCampaign` direto faz a mesa nascer sem link — e sem link ela
// não aceita ninguém. Cunhar é do hospedeiro: é `crypto/rand` e é a política de
// quem entra, nenhuma das duas coisas é da tela.
func (h campaignsHost) OpenTable(
	ctx context.Context, donoID int64, nome, descricao string,
) (int64, error) {
	agora := dbvalue.NowISO()
	c, err := h.rules.createCampaign(ctx, sqlcgen.CreateCampaignParams{
		Ownerid: donoID, Name: nome, Description: descricaoOuNulo(descricao),
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		return 0, err
	}
	return c.ID, nil
}

// InviteLink devolve "" quando a mesa não tem link, e isso é estado normal:
// campanha antiga nasceu sem.
func (h campaignsHost) InviteLink(ctx context.Context, campanhaID int64) string {
	return h.rules.inviteOf(ctx, campanhaID)
}

func (h campaignsHost) RotateInvite(ctx context.Context, campanhaID int64) (string, error) {
	return h.rules.rotateInvite(ctx, campanhaID)
}

// descricaoOuNulo traduz o vazio da tela para o NULO do banco.
//
// Os dois querem dizer "sem descrição", e a diferença importa numa direção só:
// gravar string vazia faria a coluna distinguir "não escreveu" de "apagou o que
// tinha", e a tela não oferece essa diferença a ninguém.
func descricaoOuNulo(texto string) sql.NullString {
	if t := strings.TrimSpace(texto); t != "" {
		return sql.NullString{String: t, Valid: true}
	}
	return sql.NullString{}
}

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
