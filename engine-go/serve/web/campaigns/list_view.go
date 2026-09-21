package campaigns

import (
	"context"
	"strconv"
	"strings"
	"t20engine/app"
	"t20engine/app/campaign"
	"t20engine/domain/search"
	"t20engine/serve/web/ui"
)

// A cena de CAMPANHAS como dado — uma cena de SELEÇÃO: um cursor que anda, um
// palco que muda com ele, um trilho e uma busca.
//
// A decisão que governa o formato: o servidor entrega TODOS os palcos já
// desenhados e o cursor é um sinal do cliente. Com uma dúzia de campanhas numa
// mesa doméstica isso custa alguns kilobytes e faz ←/→ serem instantâneos, sem
// requisição. O contrário — uma ida ao servidor por passo do cursor —
// transformaria navegar por teclado numa conversa com a rede, e teclado é onde
// a latência mais aparece.
//
// A BUSCA, essa sim, vai ao servidor. Filtrar no cliente obrigaria o cursor, o
// índice e o primeiro/último a serem do cliente também, e aí a cena viraria
// ilha por acidente.

type listView struct {
	Search string
	// Role é "todas" | "gm" | "player". Fica na URL junto com a busca para a
	// tela filtrada ser um endereço que se guarda e se recarrega.
	Role        string
	Campaigns   []campaignCard
	CursorID    int64
	HasAny      bool
	FilteredAll bool
	// Neighbors espelha `Campanhas` na ordem do trilho — ver `ui.NeighborAt`.
	// Montado uma vez aqui em vez de dois por palco desenhado, e no tipo
	// compartilhado porque o vizinho é a MESMA peça da cena do elenco.
	Neighbors []ui.Neighbor
}

type campaignCard struct {
	ID       int64
	Name     string
	Synopsis string
	Role     string
	Initials string
	// Gradient é a capa derivada do nome — ver `ui.NameGradient`.
	Gradient  string
	Live      bool
	SessionID int64
	Mine      *myCharacter
}

type myCharacter struct {
	Name     string
	Classes  string
	Initials string
	Gradient string
}

// LoadList monta a cena.
func (s Scene) LoadList(ctx context.Context, euID int64, admin bool, query, role string) (listView, error) {
	list, err := s.collection.Visible(ctx, app.Caller{ID: euID, IsAdmin: admin})
	if err != nil {
		return listView{}, err
	}
	alive, err := s.liveSessions(ctx, euID)
	if err != nil {
		return listView{}, err
	}

	v := listView{Search: query, Role: knownRole(role), HasAny: len(list) > 0}
	for _, c := range list {
		if !passesRole(c.Role, v.Role) {
			continue
		}
		// Os campos indexados: nome e sinopse.
		if !search.Matches([]string{c.Name, textOrEmpty(c.Description)}, query) {
			continue
		}
		v.Campaigns = append(v.Campaigns, cardOf(c, alive))
	}
	v.FilteredAll = v.HasAny && len(v.Campaigns) == 0
	if len(v.Campaigns) > 0 {
		// O cursor nasce na primeira, e é sempre uma que EXISTE na lista
		// filtrada: um cursor apontando para campanha filtrada fora deixaria o
		// palco vazio com o trilho cheio.
		v.CursorID = v.Campaigns[0].ID
	}
	for i, c := range v.Campaigns {
		v.Neighbors = append(v.Neighbors, ui.Neighbor{
			ID: c.ID, Name: c.Name, Monogram: c.Initials, Gradient: c.Gradient, Index: i,
		})
	}
	return v, nil
}

func cardOf(c campaign.Seen, alive map[int64]int64) campaignCard {
	card := campaignCard{
		ID:       c.ID,
		Name:     c.Name,
		Synopsis: textOrEmpty(c.Description),
		Role:     roleLabel(c.Role, c.OwnerName),
		Initials: ui.Monogram(c.Name),
		Gradient: ui.NameGradient(c.Name),
	}
	if sid, ok := alive[c.ID]; ok {
		card.Live, card.SessionID = true, sid
	}
	if c.Character != nil {
		card.Mine = &myCharacter{
			Name:     c.Character.Name,
			Classes:  classesInLine(c.Character),
			Initials: ui.Monogram(c.Character.Name),
			Gradient: ui.NameGradient(c.Character.Name),
		}
	}
	return card
}

// liveSessions responde, numa consulta só, quais campanhas têm partida rolando.
//
// UMA consulta e não N+1, uma por campanha: a fan-out do cliente é o remendo
// que aparece quando a resposta não existe no servidor, e ela já apareceu em
// duas telas.
func (s Scene) liveSessions(ctx context.Context, userID int64) (map[int64]int64, error) {
	rows, err := s.deps.Queries().LiveSessionsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	alive := make(map[int64]int64, len(rows))
	for _, l := range rows {
		// O sqlc tipa o `MIN(s.id)` de um GROUP BY como `interface{}`, porque
		// agregação pode devolver NULL. Aqui nunca devolve — o grupo só existe
		// se houver linha —, mas o tipo é o que é.
		if id, ok := asInt64(l.Sessionid); ok {
			alive[l.Campaignid] = id
		}
	}
	return alive, nil
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	}
	return 0, false
}

// classesInLine: "Arcanista 5 / Guerreiro 2". Sem classe nenhuma cai no nível,
// que é o que sobra para dizer.
func classesInLine(c *campaign.SeenCharacter) string {
	parts := make([]string, 0, len(c.Classes))
	for _, cl := range c.Classes {
		parts = append(parts, cl.ClassName+" "+strconv.FormatInt(cl.Level, 10))
	}
	if len(parts) == 0 {
		return "Nv " + strconv.FormatInt(c.Level, 10)
	}
	return strings.Join(parts, " / ")
}

// knownRole fecha o filtro nos três valores que existem: qualquer outra coisa
// na URL vira "todas" em vez de esconder a lista inteira.
func knownRole(role string) string {
	if role == "gm" || role == "player" {
		return role
	}
	return "todas"
}

func passesRole(campaignRole, filter string) bool {
	if filter == "todas" {
		return true
	}
	if campaignRole == "" {
		return filter == "player"
	}
	return campaignRole == filter
}

func valueOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// textOrEmpty achata o ponteiro da descrição.
//
// O caso de uso a carrega como PONTEIRO porque o banco — e a rota JSON — fazem
// diferença entre "sem descrição" e "descrição vazia". A TELA não faz: as duas
// desenham um cartão sem sinopse.
func textOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
