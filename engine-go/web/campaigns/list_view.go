package campaigns

import (
	"context"
	"strconv"
	"strings"
	"t20engine/search"
	"t20engine/web/ui"
)

// A cena de CAMPANHAS como dado (ALE-234) — a primeira cena de SELEÇÃO da
// migração: um cursor que anda, um palco que muda com ele, um trilho e uma
// busca.
//
// A decisão que governa o formato: o servidor entrega TODOS os livros já
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
	Busca string
	// Papel é "todas" | "gm" | "player". Fica na URL junto com a busca para a
	// tela filtrada ser um endereço que se guarda e se recarrega.
	Papel       string
	Campanhas   []campaignCard
	CursorID    int64
	TemAlguma   bool
	FiltrouTudo bool
}

type campaignCard struct {
	ID       int64
	Nome     string
	Sinopse  string
	Papel    string
	Iniciais string
	// Gradiente é a capa derivada do nome — ver `ui.NameGradient`.
	Gradiente string
	AoVivo    bool
	SessaoID  int64
	Meu       *myCharacter
}

type myCharacter struct {
	Nome     string
	Classes  string
	Iniciais string
}

// LoadList monta a cena.
func (s Scene) LoadList(ctx context.Context, euID int64, admin bool, busca, papel string) (listView, error) {
	lista, err := s.deps.List(ctx, euID, admin)
	if err != nil {
		return listView{}, err
	}
	vivas, err := s.liveSessions(ctx, euID)
	if err != nil {
		return listView{}, err
	}

	v := listView{Busca: busca, Papel: knownRole(papel), TemAlguma: len(lista) > 0}
	for _, c := range lista {
		if !passesRole(c.Role, v.Papel) {
			continue
		}
		// Os MESMOS campos que a SPA indexa: nome e sinopse.
		if !search.Matches([]string{c.Name, c.Description}, busca) {
			continue
		}
		v.Campanhas = append(v.Campanhas, cardOf(c, vivas))
	}
	v.FiltrouTudo = v.TemAlguma && len(v.Campanhas) == 0
	if len(v.Campanhas) > 0 {
		// O cursor nasce na primeira, e é sempre uma que EXISTE na lista
		// filtrada: um cursor apontando para campanha filtrada fora deixaria o
		// palco vazio com o trilho cheio.
		v.CursorID = v.Campanhas[0].ID
	}
	return v, nil
}

func cardOf(c ListRow, vivas map[int64]int64) campaignCard {
	cartao := campaignCard{
		ID:        c.ID,
		Nome:      c.Name,
		Sinopse:   c.Description,
		Papel:     roleLabel(c.Role, c.OwnerName),
		Iniciais:  ui.Monogram(c.Name),
		Gradiente: ui.NameGradient(c.Name),
	}
	if sid, ok := vivas[c.ID]; ok {
		cartao.AoVivo, cartao.SessaoID = true, sid
	}
	if c.Character != nil {
		cartao.Meu = &myCharacter{
			Nome:     c.Character.Name,
			Classes:  classesInLine(c.Character),
			Iniciais: ui.Monogram(c.Character.Name),
		}
	}
	return cartao
}

// liveSessions responde, numa consulta só, quais campanhas têm partida rolando.
//
// Era o `createActiveSessionByCampaign`: N+1 requisições do cliente, uma por
// campanha. É a SEGUNDA fan-out idêntica que a migração encontra — a primeira
// era a do Hub (ALE-231) —, e duas telas com o mesmo remendo são o sinal de que
// o buraco estava na API.
func (s Scene) liveSessions(ctx context.Context, userID int64) (map[int64]int64, error) {
	linhas, err := s.deps.Queries().LiveSessionsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	vivas := make(map[int64]int64, len(linhas))
	for _, l := range linhas {
		// O sqlc tipa o `MIN(s.id)` de um GROUP BY como `interface{}`, porque
		// agregação pode devolver NULL. Aqui nunca devolve — o grupo só existe
		// se houver linha —, mas o tipo é o que é.
		if id, ok := asInt64(l.Sessionid); ok {
			vivas[l.Campaignid] = id
		}
	}
	return vivas, nil
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

// classesInLine: "Arcanista 5 / Guerreiro 2", como a SPA escreve. Sem classe
// nenhuma cai no nível, que é o que sobra para dizer.
func classesInLine(c *RowCharacter) string {
	partes := make([]string, 0, len(c.Classes))
	for _, cl := range c.Classes {
		partes = append(partes, cl.ClassName+" "+strconv.FormatInt(cl.Level, 10))
	}
	if len(partes) == 0 {
		return "Nv " + strconv.FormatInt(c.Level, 10)
	}
	return strings.Join(partes, " / ")
}

// knownRole fecha o filtro nos três valores que existem: qualquer outra coisa
// na URL vira "todas" em vez de esconder a lista inteira.
func knownRole(papel string) string {
	if papel == "gm" || papel == "player" {
		return papel
	}
	return "todas"
}

func passesRole(papelDaCampanha, filtro string) bool {
	if filtro == "todas" {
		return true
	}
	if papelDaCampanha == "" {
		return filtro == "player"
	}
	return papelDaCampanha == filtro
}

func valueOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
