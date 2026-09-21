package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"t20engine/app"
	"t20engine/domain/board"
	"t20engine/infra/wire"
	"t20engine/serve/web/ui"
	"time"

	"t20engine/infra/db/sqlcgen"
)

// A CRÔNICA como dado: a página de uma campanha aberta no tomo.
//
// A página inteira sai de UMA resposta — campanha, sessões e membros —, com os
// números da visão geral já escritos, em vez de uma consulta por bloco com o
// próprio estado de carregando.
//
// E a ABA é o `?tab=`: o parâmetro de consulta É o estado, e chega junto com o
// pedido — não há o que espelhar no cliente.

type oneView struct {
	ID          int64
	Name        string
	Description string
	// IsGM decide o que a tela OFERECE. A trava é do servidor em cada rota
	// de escrita; isto é UX.
	IsGM bool
	// OtherOwner é o nome do dono quando quem olha é admin e NÃO é o dono. Vazio
	// para o dono e para o jogador: marcar a mesa de um jogador trocaria o
	// "Jogando" dele por "Mesa de Fulano".
	OtherOwner string

	Aba      string
	Tabs     []oneTab
	Heroes   []heroAtTable
	Sessions []sessionRow

	// Os três sinetes da visão geral, contados aqui e não na tela: a tela que
	// conta é a tela que discorda de si mesma quando alguém muda o filtro de um
	// lado só.
	TotalHeroes   int
	TotalSessions int
	EndedSessions int
	// SessaoViva é a única ação de sessão que a cabeça da página oferece.
	LiveSessionID     int64
	LiveSessionNumber int64
	// CreatedAt é "desde quando esta mesa existe", e ela fica na linha de meta
	// porque é o que separa uma crônica de anos de uma aberta ontem.
	CreatedAt string

	// Places é o ACERVO da campanha, e ele só é carregado na aba dele.
	//
	// Sob demanda e não sempre: uma crônica longa tem dezenas de lugares, e a
	// visão geral não mostra nenhum — buscá-los a cada abertura da página seria
	// ler o acervo para desenhar três sinetes.
	Places []PlaceRow
	// Chaos são as aparências oferecidas ao lugar NOVO.
	Chaos []GroundOption

	// IgnoredRules é o conjunto DESLIGADO, e não o ligado: o padrão do livro
	// é a regra valer, então guardar as exceções é guardar o que alguém
	// decidiu — e uma regra nova nasce em vigor sem migração de dados.
	IgnoredRules []string
	// InviteLink é o CAMINHO do convite desta mesa, ou "" quando ela não tem
	// um. Caminho e não URL: quem prefixa a origem é o navegador — ver a razão
	// em `ui.MintedInvite`.
	InviteLink string
	// Erros e Aviso servem à aba de configuração, que é a única com formulário.
	Erros  wire.FieldErrorMap
	Notice string
}

type oneTab struct {
	ID     string
	Label  string
	Active bool
}

type heroAtTable struct {
	Name string
	// IsGM é "este personagem é do dono da mesa?".
	//
	// Booleano e não a string do papel: a tela desenha uma COROA ou não desenha
	// nada, e um campo de texto convidaria a inventar um terceiro estado que a
	// autorização não tem. Ela conhece dois — dono e o resto.
	IsGM     bool
	Initials string
	Gradient string
}

type sessionRow struct {
	ID     int64
	Number int64
	Title  string
	Data   string
	State  string
	Alive  bool
}

// oneTabs: a de configuração só existe para quem mestra.
//
// `?tab=config` na URL de um jogador CAI para a visão geral em vez de mostrar
// uma seção que o trilho dele não tem. A trava de verdade é do servidor em cada
// rota; isto evita a tela meio desenhada.
func oneTabs(isGM bool, requested string) []oneTab {
	all := []oneTab{
		{ID: "visao", Label: "Visão geral"},
		{ID: "sessoes", Label: "Sessões"},
		{ID: "membros", Label: "Membros"},
	}
	if isGM {
		// LUGARES antes de CONFIG, e a ordem é a do uso: preparar a próxima cena
		// é trabalho de toda semana, e configurar a mesa acontece uma vez. Config
		// fecha o trilho porque é o que se procura quando já se sabe o que
		// procurar.
		all = append(all, oneTab{ID: "lugares", Label: "Lugares"})
		all = append(all, oneTab{ID: "config", Label: "Config"})
	}
	chosen := "visao"
	for _, a := range all {
		if a.ID == requested {
			chosen = requested
		}
	}
	for i := range all {
		all[i].Active = all[i].ID == chosen
	}
	return all
}

// RegraEmVigor: o conjunto guardado é o das DESLIGADAS.
func (v oneView) RegraEmVigor(id string) bool {
	return !slices.Contains(v.IgnoredRules, id)
}

// optionalRule é o verbete que a tela mostra. O texto vive no servidor porque
// ele cita a PÁGINA do livro, e página citada é dado de regra, não de layout.
type optionalRule struct {
	ID          string
	Title       string
	Description string
}

var optionalRules = []optionalRule{
	{
		ID:    "carga",
		Title: "Limites de carga",
		Description: "Passar do limite sobrecarrega: −5 de penalidade de armadura e −3m de deslocamento (p141). " +
			"Os espaços continuam somados na mochila mesmo com a regra desligada.",
	},
}

func (v oneView) AbaAtiva() string {
	for _, a := range v.Tabs {
		if a.Active {
			return a.ID
		}
	}
	return "visao"
}

func (s Scene) LoadOne(ctx context.Context, euID int64, admin bool, id int64, aba string) (oneView, error) {
	c, err := s.deps.Queries().GetCampaign(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return oneView{}, errNoSuchCampaign
	}
	if err != nil {
		return oneView{}, err
	}
	// A MESMA regra de acesso que o ciclo da sessão usa: dono é "gm", quem tem
	// personagem na mesa é "player", e o resto não entra. Chamada DIRETO, e não
	// por uma entrada da porta que só a repassava.
	role, err := s.access.RoleIn(ctx, app.Caller{ID: euID, IsAdmin: admin}, c)
	if err != nil {
		return oneView{}, err
	}

	v := oneView{
		ID: c.ID, Name: c.Name, Description: c.Description.String,
		IsGM:         role == "gm",
		CreatedAt:    shortDate(c.Createdat),
		IgnoredRules: s.life.IgnoredRules(ctx, c.ID),
		Erros:        wire.FieldErrorMap{},
	}
	// O nome do DONO só aparece numa campanha que não é de quem está olhando, o
	// que hoje quer dizer um admin. A pergunta "sou admin?" chega por parâmetro
	// e não pelo usuário inteiro, pela mesma razão de sempre: o tipo do usuário
	// é do hospedeiro.
	if admin && c.Ownerid != euID {
		v.OtherOwner = s.collection.OwnerNames(ctx, []sqlcgen.Campaign{c}, euID)[c.Ownerid]
	}
	v.Tabs = oneTabs(v.IsGM, aba)
	// O LINK só é LIDO para quem mestra, e essa é a fronteira desta tela: a aba
	// de configuração não existe para o jogador, mas "não desenhar" é UX — não
	// carregar é a regra. Um jogador que forjasse `?tab=config` receberia a
	// visão geral (ver `oneTabs`), e mesmo assim o link não teria sido lido.
	if v.IsGM {
		if token := s.life.InviteOf(ctx, c.ID); token != "" {
			v.InviteLink = "/campanhas/entrar?token=" + url.QueryEscape(token)
		}
	}
	// O ACERVO é lido só na ABA dele, e pela mesma regra do link acima: não
	// desenhar é UX, não carregar é a decisão. Uma crônica de dois anos tem
	// dezenas de lugares, e nenhuma outra aba mostra um.
	if v.IsGM && v.AbaAtiva() == "lugares" {
		v.Places = s.placesOf(ctx, c.ID)
		v.Chaos = groundOptions()
	}

	members, err := s.deps.Queries().ListMembers(ctx, id)
	if err != nil {
		return oneView{}, err
	}
	// O MESTRE PRIMEIRO, e o resto na ordem que veio: numa mesa de seis, quem
	// mestra ser o primeiro da lista é o que faz o elenco se ler como grupo em
	// vez de como fila.
	//
	// Quem mestra é o DONO da campanha, e não quem tem `Role` de mestre — a
	// coluna vale `'player'` em toda linha, e comparar por ela devolve zero para
	// todo par, deixando a lista na ordem em que veio. O dono é a MESMA verdade
	// que o `RoleIn` usa para autorizar, e é o que faz a tela e a autorização não
	// poderem divergir.
	isGMs := func(m sqlcgen.ListMembersRow) bool { return m.Charownerid == c.Ownerid }
	slices.SortStableFunc(members, func(a, b sqlcgen.ListMembersRow) int {
		switch {
		case isGMs(a) == isGMs(b):
			return 0
		case isGMs(a):
			return -1
		default:
			return 1
		}
	})
	for _, m := range members {
		if !isGMs(m) {
			v.TotalHeroes++
		}
		name := memberName(m.Charname, m.Characterid)
		v.Heroes = append(v.Heroes, heroAtTable{
			Name: name, IsGM: isGMs(m),
			Initials: ui.Monogram(name), Gradient: ui.NameGradient(name),
		})
	}

	sessions, err := s.deps.Queries().ListSessions(ctx, id)
	if err != nil {
		return oneView{}, err
	}
	v.TotalSessions = len(sessions)
	// DA MAIS NOVA PARA A MAIS VELHA, e por isso o laço é de trás para a frente:
	// o `ListSessions` ordena por número CRESCENTE, então "as recentes" são as
	// ÚLTIMAS. Tomar as primeiras mostraria as mais antigas — o que não aparece
	// numa mesa com três sessões, e a tela não tem como avisar que está
	// mentindo.
	for i := len(sessions) - 1; i >= 0; i-- {
		sess := sessions[i]
		if sess.Status == "ended" {
			v.EndedSessions++
		}
		alive := sess.Status == "active"
		if alive {
			v.LiveSessionID, v.LiveSessionNumber = sess.ID, sess.Sessionnumber
		}
		v.Sessions = append(v.Sessions, sessionRow{
			ID: sess.ID, Number: sess.Sessionnumber,
			Title: sess.Title.String,
			Data:  shortDate(sess.Createdat),
			State: readableState(sess.Status), Alive: alive,
		})
	}
	return v, nil
}

// memberName: personagem sem nome vira "Personagem N" e não linha em branco.
// Um membro invisível na lista é pior que um nome feio — ele some da contagem
// que o olho faz.
func memberName(name string, id int64) string {
	if name != "" {
		return name
	}
	return "Personagem " + strconv.FormatInt(id, 10)
}

// shortDate corta o carimbo ISO em dd/mm/aaaa. A data importa na crônica
// porque a lista de sessões é uma LINHA DO TEMPO — sem ela, "Sessão 4" não diz
// se foi ontem ou em março.
func shortDate(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	return t.Format("02/01/2006")
}

func readableState(status string) string {
	switch status {
	case "active":
		return "Ao vivo"
	case "ended":
		return "Encerrada"
	default:
		return "Planejada"
	}
}

// placesOf lista o acervo de cenas guardadas, já dizendo qual lugar está numa
// MESA agora.
//
// O casamento é pelo NOME e não pelo id, como o acervo da Mesa já faz: o nome é
// a identidade do lugar dentro da campanha — é assim que o `Archive` decide se
// sobrescreve —, e uma cena aberta do zero com o nome de um lugar guardado É
// aquele lugar, porque é a conta que o arquivamento fará quando ela fechar.
func (s Scene) placesOf(ctx context.Context, campaignID int64) []PlaceRow {
	onTable := s.places.PlacesOnATable(ctx, campaignID)
	saved := s.places.Places(ctx, campaignID)
	outside := make([]PlaceRow, 0, len(saved))
	for _, l := range saved {
		outside = append(outside, PlaceRow{
			ID: l.ID, Name: l.Name, Tokens: l.Tokens,
			When: l.UpdatedAt, AtTableID: onTable[l.Name],
		})
	}
	return outside
}

// groundOptions são as aparências que um lugar pode ter, na forma do
// formulário.
//
// LIDAS do catálogo e nunca copiadas: uma lista escrita aqui ofereceria um chão
// que o servidor não conhece no dia em que a sexta nascer. Era esse o argumento
// para elas atravessarem a porta, e ele continua de pé — o que mudou é que a
// cena lê o catálogo direto, que é `domain/` e está abaixo dela.
func groundOptions() []GroundOption {
	outside := make([]GroundOption, 0, len(board.PlaceGrounds))
	for _, c := range board.PlaceGrounds {
		outside = append(outside, GroundOption{ID: c.ID, Label: c.Label})
	}
	return outside
}
