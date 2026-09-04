package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"t20engine/plataforma"
	"t20engine/web/ui"
	"time"

	"t20engine/db/sqlcgen"
)

// A CRÔNICA como dado (ALE-255): a página de uma campanha aberta no tomo.
//
// A diferença que mais pesa em relação à tela da SPA está na CONTAGEM DE IDAS
// À REDE. Lá a cena monta e dispara TRÊS consultas — campanha, sessões e
// membros —, cada uma com o próprio estado de carregando e o próprio esqueleto;
// a visão geral mostra números que só existem depois que as três voltam. Aqui é
// uma resposta só, e o número já vem escrito.
//
// E a ABA é o `?tab=`, que já era o estado na SPA — o comentário de lá conta
// que a versão em React precisava espelhar isso num `useState` com dois efeitos
// e um debounce de 250ms. No servidor não há o que espelhar: o parâmetro de
// consulta É o estado, e ele chega junto com o pedido.

type oneView struct {
	ID        int64
	Nome      string
	Descricao string
	// EhMestre decide o que a tela OFERECE. A trava é do servidor em cada rota
	// de escrita; isto é UX.
	EhMestre bool
	// DonoOutro é o nome do dono quando quem olha é admin e NÃO é o dono. Vazio
	// para o dono e para o jogador: marcar a mesa de um jogador trocaria o
	// "Jogando" dele por "Mesa de Fulano", que é defeito que o e2e já pegou.
	DonoOutro string

	Aba     string
	Abas    []oneTab
	Herois  []heroAtTable
	Sessoes []sessionRow

	// Os três sinetes da visão geral, contados aqui e não na tela: a tela que
	// conta é a tela que discorda de si mesma quando alguém muda o filtro de um
	// lado só.
	TotalHerois       int
	TotalSessoes      int
	SessoesEncerradas int
	// SessaoViva é a única ação de sessão que a cabeça da página oferece.
	SessaoVivaID       int64
	NumeroDaSessaoViva int64
	// CriadaEm é "desde quando esta mesa existe", e ela fica na linha de meta
	// porque é o que separa uma crônica de anos de uma aberta ontem.
	CriadaEm string

	// Lugares é o ACERVO da campanha (ALE-292), e ele só é carregado na aba dele.
	//
	// Sob demanda e não sempre: uma crônica longa tem dezenas de lugares, e a
	// visão geral não mostra nenhum — buscá-los a cada abertura da página seria
	// ler o acervo para desenhar três sinetes.
	Lugares []PlaceRow
	// Chaos são as aparências oferecidas ao lugar NOVO.
	Chaos []GroundOption

	// RegrasIgnoradas é o conjunto DESLIGADO, e não o ligado: o padrão do livro
	// é a regra valer, então guardar as exceções é guardar o que alguém
	// decidiu — e uma regra nova nasce em vigor sem migração de dados.
	RegrasIgnoradas []string
	// LinkDoConvite é o CAMINHO do convite desta mesa, ou "" quando ela não tem
	// um (ALE-287). Caminho e não URL: quem prefixa a origem é o navegador — ver
	// a razão medida em `ui.MintedInvite`.
	LinkDoConvite string
	// Erros e Aviso servem à aba de configuração, que é a única com formulário.
	Erros plataforma.FieldErrorMap
	Aviso string
}

type oneTab struct {
	ID     string
	Rotulo string
	Ativa  bool
}

type heroAtTable struct {
	Nome string
	// EhMestre é "este personagem é do dono da mesa?".
	//
	// Booleano e não a string do papel: a tela desenha uma COROA ou não desenha
	// nada, e um campo de texto convidaria a inventar um terceiro estado que a
	// autorização não tem. Ela conhece dois — dono e o resto.
	EhMestre  bool
	Iniciais  string
	Gradiente string
}

type sessionRow struct {
	ID     int64
	Numero int64
	Titulo string
	Data   string
	Estado string
	Viva   bool
}

// oneTabs: a de configuração só existe para quem mestra.
//
// `?tab=config` na URL de um jogador CAI para a visão geral em vez de mostrar
// uma seção que o trilho dele não tem. A trava de verdade é do servidor em cada
// rota; isto evita a tela meio desenhada.
func oneTabs(ehMestre bool, pedida string) []oneTab {
	todas := []oneTab{
		{ID: "visao", Rotulo: "Visão geral"},
		{ID: "sessoes", Rotulo: "Sessões"},
		{ID: "membros", Rotulo: "Membros"},
	}
	if ehMestre {
		// LUGARES antes de CONFIG, e a ordem é a do uso: preparar a próxima cena
		// é trabalho de toda semana, e configurar a mesa acontece uma vez. Config
		// fecha o trilho porque é o que se procura quando já se sabe o que
		// procurar (ALE-292).
		todas = append(todas, oneTab{ID: "lugares", Rotulo: "Lugares"})
		todas = append(todas, oneTab{ID: "config", Rotulo: "Config"})
	}
	escolhida := "visao"
	for _, a := range todas {
		if a.ID == pedida {
			escolhida = pedida
		}
	}
	for i := range todas {
		todas[i].Ativa = todas[i].ID == escolhida
	}
	return todas
}

// RegraEmVigor: o conjunto guardado é o das DESLIGADAS.
func (v oneView) RegraEmVigor(id string) bool {
	return !slices.Contains(v.RegrasIgnoradas, id)
}

// optionalRule é o verbete que a tela mostra. O texto vive no servidor porque
// ele cita a PÁGINA do livro, e página citada é dado de regra, não de layout.
type optionalRule struct {
	ID        string
	Titulo    string
	Descricao string
}

var optionalRules = []optionalRule{
	{
		ID:     "carga",
		Titulo: "Limites de carga",
		Descricao: "Passar do limite sobrecarrega: −5 de penalidade de armadura e −3m de deslocamento (p141). " +
			"Os espaços continuam somados na mochila mesmo com a regra desligada.",
	},
}

func (v oneView) AbaAtiva() string {
	for _, a := range v.Abas {
		if a.Ativa {
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
	// A MESMA regra de acesso da rota JSON e do gateway do socket (ALE-120):
	// dono é "gm", quem tem personagem na mesa é "player", e o resto não entra.
	papel, _, err := s.deps.RoleIn(ctx, euID, c)
	if err != nil {
		return oneView{}, err
	}

	v := oneView{
		ID: c.ID, Nome: c.Name, Descricao: c.Description.String,
		EhMestre:        papel == "gm",
		CriadaEm:        shortDate(c.Createdat),
		RegrasIgnoradas: s.deps.IgnoredRules(ctx, c.ID),
		Erros:           plataforma.FieldErrorMap{},
	}
	// O nome do DONO só aparece numa campanha que não é de quem está olhando, o
	// que hoje quer dizer um admin. A pergunta "sou admin?" chega por parâmetro
	// e não pelo usuário inteiro, pela mesma razão de sempre: o tipo do usuário
	// é do hospedeiro.
	if admin && c.Ownerid != euID {
		v.DonoOutro = s.deps.OwnerNames(ctx, []sqlcgen.Campaign{c}, euID)[c.Ownerid]
	}
	v.Abas = oneTabs(v.EhMestre, aba)
	// O LINK só é LIDO para quem mestra, e essa é a fronteira desta tela: a aba
	// de configuração não existe para o jogador, mas "não desenhar" é UX — não
	// carregar é a regra. Um jogador que forjasse `?tab=config` receberia a
	// visão geral (ver `oneTabs`), e mesmo assim o link não teria sido lido.
	if v.EhMestre {
		if token := s.deps.InviteLink(ctx, c.ID); token != "" {
			v.LinkDoConvite = "/campanhas/entrar?token=" + url.QueryEscape(token)
		}
	}
	// O ACERVO é lido só na ABA dele, e pela mesma regra do link acima: não
	// desenhar é UX, não carregar é a decisão. Uma crônica de dois anos tem
	// dezenas de lugares, e nenhuma outra aba mostra um.
	if v.EhMestre && v.AbaAtiva() == "lugares" {
		v.Lugares = s.deps.Places(ctx, c.ID)
		v.Chaos = s.deps.Grounds()
	}

	membros, err := s.deps.Queries().ListMembers(ctx, id)
	if err != nil {
		return oneView{}, err
	}
	// O MESTRE PRIMEIRO, e o resto na ordem que veio. É a regra do `sortRoster`
	// da SPA, portada: numa mesa de seis, quem mestra ser o primeiro da lista é
	// o que faz o elenco se ler como grupo em vez de como fila.
	//
	// ELA NUNCA ACONTECEU até a ALE-287, e não por engano de ordenação: a
	// comparação era sobre `m.Role`, uma coluna que valia `'player'` em toda
	// linha. `a.Role == b.Role` dava sempre verdadeiro, a função devolvia zero
	// para todo par, e a lista saía na ordem em que veio. A coroa ao lado do
	// nome (ver `heroRow`) nunca foi desenhada pela mesma razão.
	//
	// Quem mestra é o DONO da campanha, e essa é a MESMA verdade que o `roleIn`
	// usa para autorizar. Perguntar ao dono do personagem em vez de a uma coluna
	// é o que faz a tela e a autorização não poderem divergir.
	ehDoMestre := func(m sqlcgen.ListMembersRow) bool { return m.Charownerid == c.Ownerid }
	slices.SortStableFunc(membros, func(a, b sqlcgen.ListMembersRow) int {
		switch {
		case ehDoMestre(a) == ehDoMestre(b):
			return 0
		case ehDoMestre(a):
			return -1
		default:
			return 1
		}
	})
	for _, m := range membros {
		if !ehDoMestre(m) {
			v.TotalHerois++
		}
		nome := memberName(m.Charname, m.Characterid)
		v.Herois = append(v.Herois, heroAtTable{
			Nome: nome, EhMestre: ehDoMestre(m),
			Iniciais: ui.Monogram(nome), Gradiente: ui.NameGradient(nome),
		})
	}

	sessoes, err := s.deps.Queries().ListSessions(ctx, id)
	if err != nil {
		return oneView{}, err
	}
	v.TotalSessoes = len(sessoes)
	// DA MAIS NOVA PARA A MAIS VELHA. O `ListSessions` ordena por número
	// CRESCENTE, e a primeira versão desta cena pegava as três primeiras para
	// "sessões recentes" — mostrando as três mais ANTIGAS. O defeito não
	// aparece numa mesa com três sessões, só numa que já jogou bastante, e a
	// tela não tem como avisar que está mentindo. Peguei comparando com a da
	// SPA, que mostra 5, 4, 3 onde a minha mostrava 1, 2, 3.
	for i := len(sessoes) - 1; i >= 0; i-- {
		sess := sessoes[i]
		if sess.Status == "ended" {
			v.SessoesEncerradas++
		}
		viva := sess.Status == "active"
		if viva {
			v.SessaoVivaID, v.NumeroDaSessaoViva = sess.ID, sess.Sessionnumber
		}
		v.Sessoes = append(v.Sessoes, sessionRow{
			ID: sess.ID, Numero: sess.Sessionnumber,
			Titulo: sess.Title.String,
			Data:   shortDate(sess.Createdat),
			Estado: readableState(sess.Status), Viva: viva,
		})
	}
	return v, nil
}

// memberName: personagem sem nome vira "Personagem N" e não linha em branco.
// Um membro invisível na lista é pior que um nome feio — ele some da contagem
// que o olho faz.
func memberName(nome string, id int64) string {
	if nome != "" {
		return nome
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
