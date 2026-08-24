package api

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"t20engine/plataforma"
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

type cronicaView struct {
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
	Abas    []abaDaCronica
	Herois  []heroiNaMesa
	Sessoes []sessaoNaCronica

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

	// RegrasIgnoradas é o conjunto DESLIGADO, e não o ligado: o padrão do livro
	// é a regra valer, então guardar as exceções é guardar o que alguém
	// decidiu — e uma regra nova nasce em vigor sem migração de dados.
	RegrasIgnoradas []string
	// Erros e Aviso servem à aba de configuração, que é a única com formulário.
	Erros plataforma.FieldErrorMap
	Aviso string
}

type abaDaCronica struct {
	ID     string
	Rotulo string
	Ativa  bool
}

type heroiNaMesa struct {
	Nome      string
	Papel     string
	Iniciais  string
	Gradiente string
}

type sessaoNaCronica struct {
	ID     int64
	Numero int64
	Titulo string
	Data   string
	Estado string
	Viva   bool
}

// abasDaCronica: a de configuração só existe para quem mestra.
//
// `?tab=config` na URL de um jogador CAI para a visão geral em vez de mostrar
// uma seção que o trilho dele não tem. A trava de verdade é do servidor em cada
// rota; isto evita a tela meio desenhada.
func abasDaCronica(ehMestre bool, pedida string) []abaDaCronica {
	todas := []abaDaCronica{
		{ID: "visao", Rotulo: "Visão geral"},
		{ID: "sessoes", Rotulo: "Sessões"},
		{ID: "membros", Rotulo: "Membros"},
	}
	if ehMestre {
		todas = append(todas, abaDaCronica{ID: "config", Rotulo: "Config"})
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
func (v cronicaView) RegraEmVigor(id string) bool {
	return !slices.Contains(v.RegrasIgnoradas, id)
}

// regraOpcional é o verbete que a tela mostra. O texto vive no servidor porque
// ele cita a PÁGINA do livro, e página citada é dado de regra, não de layout.
type regraOpcional struct {
	ID        string
	Titulo    string
	Descricao string
}

var regrasOpcionais = []regraOpcional{
	{
		ID:     "carga",
		Titulo: "Limites de carga",
		Descricao: "Passar do limite sobrecarrega: −5 de penalidade de armadura e −3m de deslocamento (p141). " +
			"Os espaços continuam somados na mochila mesmo com a regra desligada.",
	},
}

func (v cronicaView) AbaAtiva() string {
	for _, a := range v.Abas {
		if a.Ativa {
			return a.ID
		}
	}
	return "visao"
}

func (s *Server) carregaCronica(ctx context.Context, eu AuthUser, id int64, aba string) (cronicaView, error) {
	c, err := s.queries.GetCampaign(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return cronicaView{}, errCampanhaInexistente
	}
	if err != nil {
		return cronicaView{}, err
	}
	// A MESMA regra de acesso da rota JSON e do gateway do socket (ALE-120):
	// dono é "gm", quem tem personagem na mesa é "player", e o resto não entra.
	papel, _, err := s.roleIn(ctx, eu, c)
	if err != nil {
		return cronicaView{}, err
	}

	v := cronicaView{
		ID: c.ID, Nome: c.Name, Descricao: c.Description.String,
		EhMestre:        papel == "gm",
		CriadaEm:        dataCurta(c.Createdat),
		RegrasIgnoradas: s.ignoredRulesOf(ctx, c.ID),
		Erros:           plataforma.FieldErrorMap{},
	}
	if eu.IsAdmin && c.Ownerid != eu.ID {
		v.DonoOutro = s.ownerNames(ctx, []sqlcgen.Campaign{c}, eu.ID)[c.Ownerid]
	}
	v.Abas = abasDaCronica(v.EhMestre, aba)

	membros, err := s.queries.ListMembers(ctx, id)
	if err != nil {
		return cronicaView{}, err
	}
	// O MESTRE PRIMEIRO, e o resto na ordem que veio. É a regra do `sortRoster`
	// da SPA, portada: numa mesa de seis, quem mestra ser o primeiro da lista é
	// o que faz o elenco se ler como grupo em vez de como fila.
	slices.SortStableFunc(membros, func(a, b sqlcgen.ListMembersRow) int {
		if a.Role == b.Role {
			return 0
		}
		if a.Role == "gm" {
			return -1
		}
		return 1
	})
	for _, m := range membros {
		if m.Role == "player" {
			v.TotalHerois++
		}
		nome := nomeDoMembro(m.Charname, m.Characterid)
		v.Herois = append(v.Herois, heroiNaMesa{
			Nome: nome, Papel: m.Role,
			Iniciais: iniciais(nome), Gradiente: gradienteDaCampanha(nome),
		})
	}

	sessoes, err := s.queries.ListSessions(ctx, id)
	if err != nil {
		return cronicaView{}, err
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
		v.Sessoes = append(v.Sessoes, sessaoNaCronica{
			ID: sess.ID, Numero: sess.Sessionnumber,
			Titulo: sess.Title.String,
			Data:   dataCurta(sess.Createdat),
			Estado: estadoLegivel(sess.Status), Viva: viva,
		})
	}
	return v, nil
}

// nomeDoMembro: personagem sem nome vira "Personagem N" e não linha em branco.
// Um membro invisível na lista é pior que um nome feio — ele some da contagem
// que o olho faz.
func nomeDoMembro(nome string, id int64) string {
	if nome != "" {
		return nome
	}
	return "Personagem " + strconv.FormatInt(id, 10)
}

// dataCurta corta o carimbo ISO em dd/mm/aaaa. A data importa na crônica
// porque a lista de sessões é uma LINHA DO TEMPO — sem ela, "Sessão 4" não diz
// se foi ontem ou em março.
func dataCurta(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	return t.Format("02/01/2006")
}

func estadoLegivel(status string) string {
	switch status {
	case "active":
		return "Ao vivo"
	case "ended":
		return "Encerrada"
	default:
		return "Planejada"
	}
}
