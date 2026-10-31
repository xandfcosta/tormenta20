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

// A cena de CAMPANHAS como dado: uma lista de cartões com a busca e o filtro de
// papel (ALE-526). Ela foi um carrossel — uma capa por vez, com cursor e trilho
// —, e virou lista porque ver todas de uma vez é o que a pessoa quer quando
// escolhe a mesa (decisão do dono).
//
// A BUSCA vai ao servidor: a regra de casar nome e sinopse é a do
// `search.Matches`, e uma segunda cópia no cliente divergiria dela.

type listView struct {
	Search string
	// Role é "todas" | "gm" | "player". Fica na URL junto com a busca para a
	// tela filtrada ser um endereço que se guarda e se recarrega.
	Role        string
	Campaigns   []campaignCard
	HasAny      bool
	FilteredAll bool
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
	// SessionNumber é o número da sessão AO VIVO, que o cartão diz no lugar da
	// contagem (ALE-526).
	SessionNumber int64
	// Heroes e Sessions são a contagem que o cartão parado diz.
	Heroes   int64
	Sessions int64
	Mine     *myCharacter
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
	tallies, err := s.tallies(ctx)
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
		card := cardOf(c, alive)
		count := tallies[c.ID]
		card.Heroes, card.Sessions = count.Heroes, count.Sessions
		v.Campaigns = append(v.Campaigns, card)
	}
	v.FilteredAll = v.HasAny && len(v.Campaigns) == 0
	return v, nil
}

func cardOf(c campaign.Seen, alive map[int64]liveSession) campaignCard {
	card := campaignCard{
		ID:       c.ID,
		Name:     c.Name,
		Synopsis: textOrEmpty(c.Description),
		Role:     roleLabel(c.Role, c.OwnerName),
		Initials: ui.Monogram(c.Name),
		Gradient: ui.NameGradient(c.Name),
	}
	if l, ok := alive[c.ID]; ok {
		card.Live, card.SessionID, card.SessionNumber = true, l.ID, l.Number
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
// liveSession é a sessão ao vivo de uma campanha: o id para o caminho e o
// número para o cartão.
type liveSession struct {
	ID, Number int64
}

func (s Scene) liveSessions(ctx context.Context, userID int64) (map[int64]liveSession, error) {
	rows, err := s.deps.Queries().LiveSessionsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	alive := make(map[int64]liveSession, len(rows))
	for _, l := range rows {
		if id, ok := asInt64(l.Sessionid); ok {
			alive[l.Campaignid] = liveSession{ID: id, Number: l.Sessionnumber}
		}
	}
	return alive, nil
}

// tallies é a contagem de heróis e sessões de cada campanha, numa consulta só.
func (s Scene) tallies(ctx context.Context) (map[int64]campaignTally, error) {
	rows, err := s.deps.Queries().CampaignTallies(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]campaignTally, len(rows))
	for _, r := range rows {
		out[r.Campaignid] = campaignTally{Heroes: r.Heroes, Sessions: r.Sessions}
	}
	return out, nil
}

type campaignTally struct {
	Heroes, Sessions int64
}

// cardMeta é a linha de baixo do nome: a sessão ao vivo, ou o papel e a
// contagem no singular certo.
func cardMeta(c campaignCard) string {
	if c.Live {
		return "Sessão " + strconv.FormatInt(c.SessionNumber, 10) + " ao vivo agora"
	}
	return c.Role + " · " + countOf(c.Heroes, "herói", "heróis") + " · " + countOf(c.Sessions, "sessão", "sessões")
}

func countOf(n int64, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.FormatInt(n, 10) + " " + many
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

func passesRole(campaignRole, filterText string) bool {
	if filterText == "todas" {
		return true
	}
	if campaignRole == "" {
		return filterText == "player"
	}
	return campaignRole == filterText
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
