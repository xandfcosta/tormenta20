package api

import (
	"context"
	"errors"
	"net/http"

	"t20engine/db/sqlcgen"
	"t20engine/web/campaigns"
)

// A CENA DE CAMPANHAS e o adaptador que cumpre a porta dela (`campaigns.Deps`,
// ALE-278).
//
// Os onze métodos moram num arquivo próprio, e não espalhados pelos arquivos de
// domínio, porque juntos eles são UMA coisa: a tradução entre o vocabulário da
// cena e o do hospedeiro. Ler os onze de uma vez é o que mostra se a fronteira
// está no lugar — e o sinal de que está é nenhum deles desenhar nada.
//
// # Quem os cumpre deixou de ser o `*Server` (fatia 6)
//
// Das doze assinaturas que a porta pede, quatro são do núcleo e OITO são regra
// de campanha — listar as mesas de quem olha, dizer o papel de cada um, entrar
// numa mesa, guardar as regras ignoradas. Por isso o adaptador é o núcleo mais
// um `campaignRules`: o que esta cena precisa da casa é exatamente "as regras
// de quem é dono do quê", e não um servidor.
type campaignsHost struct {
	sceneCore
	rules campaignRules
}

func (s *Server) campaignsHost() campaignsHost {
	return campaignsHost{sceneCore: s.sceneCore(), rules: s.campaignRules()}
}

// List traduz o `campaignList` para a forma que a CENA declarou.
//
// A cena consumia o `campaignListDTO` direto, e ele é a resposta de
// `GET /campaigns`: tag `json:` em cada campo, nome do fio. Uma tela que o
// lesse passaria a depender do formato de um endpoint que ela não serve — é o
// que a administração recusou com o `backupDTO`.
//
// O mapeamento é aqui e a CONSULTA continua uma só: duplicá-la do lado da cena
// seria trocar um acoplamento por uma cópia, e cópia de consulta é a família de
// defeito que esta épica mais encontrou.
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
// Ela existe porque a cena montava `setBuilder` + `execTouched` +
// `"UPDATE campaigns"` à mão. Cena que compõe SQL é cena com o banco dentro, e
// o que atravessa a fronteira agora é a PERGUNTA — o hospedeiro é que sabe o
// nome da coluna, que vazio é NULL e que a linha tem um `updatedAt` a tocar.
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
// Quem CLASSIFICA é o hospedeiro, quem escolhe a FRASE é a cena: a decisão que a
// porta de entrar deixou escrita (ALE-278).
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
// com a mesma cara, e o compilador recusaria as duas com o mesmo nome. É o
// contrário do caso do `MintAccountInvite`: lá um contrato existente ganhou
// porque era a MESMA pergunta; aqui ele não ganha porque não é.
func (h campaignsHost) RequesterIsAdmin(r *http.Request) bool { return currentUser(r).IsAdmin }
