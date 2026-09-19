package campaign

import (
	"context"
	"database/sql"
	"fmt"

	"t20engine/app"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// Seen é uma campanha COMO QUEM PEDE a vê: a linha dela, mais o papel de quem
// olha e o herói que ele tem lá dentro.
//
// Ela não tem tag `json:` e não vai ter. Os dois chamadores querem formas
// diferentes — a rota JSON tem um DTO com tags, a cena monta o cartão dela —, e
// é justamente por isso que o caso de uso devolve uma forma que não é a de
// nenhum dos dois: com a de um, o outro passaria a depender do formato de um fio
// que ele não fala. (A cena declarava um tipo de linha só dela, que deixou de
// existir com esta fatia.)
type Seen struct {
	ID      int64
	OwnerID int64
	Name    string
	// Description é PONTEIRO porque o banco distingue "sem descrição" de
	// "descrição vazia", e a rota JSON leva essa diferença no fio. Colapsar aqui
	// faria a resposta trocar `null` por `""` para quem já a consome.
	Description *string
	CreatedAt   string
	UpdatedAt   string
	// Role é `app.RoleGM` ou `app.RolePlayer`.
	Role string
	// OwnerName vem preenchido SÓ numa campanha que quem pede não possui — hoje,
	// um admin vendo as de todo mundo.
	OwnerName string
	// Character é o herói de quem pede NESTA campanha, quando há um.
	Character *SeenCharacter
}

// SeenCharacter é o herói que quem pede tem numa campanha da lista.
type SeenCharacter struct {
	ID      int64
	Name    string
	Level   int64
	Classes []sheet.ClassDTO
}

// Directory é o ACERVO de campanhas: quais existem para quem pede, e com que
// papel.
type Directory struct {
	queries *sqlcgen.Queries
}

func NewDirectory(q *sqlcgen.Queries) Directory { return Directory{queries: q} }

// Visible são as campanhas que esta pessoa alcança: as dela, as em que joga —
// e, para quem ADMINISTRA o servidor, todas.
//
// A exceção do admin não é conveniência: sem ela, o dono do servidor só
// alcançaria a mesa de outra pessoa digitando a URL, e o papel que ele recebe é
// `gm` porque é com as ferramentas de mestre que ele vem consertar a mesa.
func (d Directory) Visible(ctx context.Context, quem app.Caller) ([]Seen, error) {
	linhas, err := d.visibleRows(ctx, quem)
	if err != nil {
		return nil, err
	}
	donos := d.OwnerNames(ctx, linhas, quem.ID)
	fora := make([]Seen, 0, len(linhas))
	for _, c := range linhas {
		vista := Seen{
			ID: c.ID, OwnerID: c.Ownerid, Name: c.Name,
			Description: dbvalue.NullToPtr(c.Description),
			CreatedAt:   c.Createdat, UpdatedAt: c.Updatedat,
			Role: app.RolePlayer,
		}
		switch {
		case c.Ownerid == quem.ID:
			vista.Role = app.RoleGM
		case quem.IsAdmin:
			// A condição é `IsAdmin` e NÃO "o mapa de donos tem um nome": um
			// jogador também não é dono aqui, e apoiar-se no mapa faria uma
			// edição futura no `OwnerNames` entregar "gm" a ele em silêncio.
			vista.Role, vista.OwnerName = app.RoleGM, donos[c.Ownerid]
		}
		vista.Character = d.characterOf(ctx, c.ID, quem.ID)
		fora = append(fora, vista)
	}
	return fora, nil
}

// visibleRows é o `where` da lista, e é ELE que decide quem vê o quê.
func (d Directory) visibleRows(ctx context.Context, quem app.Caller) ([]sqlcgen.Campaign, error) {
	if quem.IsAdmin {
		linhas, err := d.queries.ListAllCampaigns(ctx)
		if err != nil {
			return nil, fmt.Errorf("listar todas as campanhas para o admin %d: %w", quem.ID, err)
		}
		return linhas, nil
	}
	linhas, err := d.queries.ListCampaignsForUser(ctx, quem.ID)
	if err != nil {
		return nil, fmt.Errorf("listar as campanhas de %d: %w", quem.ID, err)
	}
	return linhas, nil
}

// characterOf é o herói que quem pede tem NESTA campanha, ou nulo.
//
// Sem herói não é erro: quem mestra não tem personagem na própria mesa, e é o
// caso mais comum da lista.
func (d Directory) characterOf(ctx context.Context, campanhaID, quemPede int64) *SeenCharacter {
	linha, err := d.queries.CallerCharacterInCampaign(ctx, sqlcgen.CallerCharacterInCampaignParams{
		Campaignid: campanhaID, Ownerid: quemPede,
	})
	if err != nil {
		return nil
	}
	classes, _ := d.queries.ListClassesByCharacter(ctx, linha.ID)
	heroi := &SeenCharacter{
		ID: linha.ID, Name: linha.Name, Level: linha.Level, Classes: []sheet.ClassDTO{},
	}
	for _, c := range classes {
		heroi.Classes = append(heroi.Classes, sheet.ClassDTO{ClassName: c.Classname, Level: c.Level})
	}
	return heroi
}

// OwnerNames rotula as mesas que quem pede NÃO possui, numa consulta só.
//
// A lista é curta hoje, e um N+1 aqui cresceria com o servidor. Falha vira mapa
// vazio: a lista sem o nome do dono ainda é a lista, e derrubá-la por causa de
// um rótulo seria trocar a tela inteira por um detalhe dela.
func (d Directory) OwnerNames(
	ctx context.Context, linhas []sqlcgen.Campaign, quemPede int64,
) map[int64]string {
	var ids []int64
	for _, c := range linhas {
		if c.Ownerid != quemPede {
			ids = append(ids, c.Ownerid)
		}
	}
	nomes := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return nomes
	}
	donos, err := d.queries.ListUsersByIDs(ctx, ids)
	if err != nil {
		return nomes
	}
	for _, u := range donos {
		nomes[u.ID] = displayName(u.Name, u.Email)
	}
	return nomes
}

// displayName prefere o nome escolhido e cai no e-mail, que é como o jogador é
// chamado em todo o resto do app.
//
// Mora com o único chamador que tem. O `web/hub` tem um `displayName` PRÓPRIO e
// diferente — ele recebe o espectador inteiro —, e juntar os dois num só faria
// uma função responder duas perguntas parecidas de lugares que não se conhecem.
func displayName(nome sql.NullString, email string) string {
	if nome.Valid && nome.String != "" {
		return nome.String
	}
	return email
}
