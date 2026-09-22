package api

import (
	"database/sql"

	"t20engine/app/boards"
	"t20engine/app/character"
	"t20engine/app/session"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
)

// AS REGRAS DA MESA AO VIVO, com casa própria: a iniciativa, o descanso, quem se
// move e quanto anda, os vitais espelhados no acompanhamento, e a PUBLICAÇÃO do
// quadro para as duas salas por papel.
//
// # Por que ele carrega quase tudo, e por que isso está certo
//
// Esta porta toca quase todos os campos do `*Server`: `boards`, `sessions`,
// `presence`, `sse`, `bus`, `queries`, `catalogs` e `db`. Um adaptador que
// carrega quase tudo parece a divisão ter falhado, e é o contrário: a Mesa É a
// mesa ao vivo, e a mesa ao vivo é o que esses stores guardam.
//
// A diferença entre isto e receber o `*Server` não é o tamanho da lista, é o
// que ela **não** tem — o `livro`, o `charMu`, a cena da
// Mesa dentro dela mesma, e os métodos das outras dez cenas. Aqui está escrito
// de que a mesa depende; no `*Server` estava escrito "de tudo".
//
// Ele guarda também as regras de campanha e as da ficha, porque a Mesa pergunta
// às duas: quem pode entrar nesta sessão é regra de campanha, e o painel da
// ficha embutida é regra de ficha.
type tableRules struct {
	db       *sql.DB
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
	boards   *boards.Store
	sessions *session.Store
	presence *live.PresenceRegistry
	sse      *live.SSEHub
	bus      *events.Bus
	campaign campaignRules
	sheet    sheetRules
	// sheetScene é o adaptador da FICHA, e a Mesa o carrega por uma razão só: o
	// painel da ficha embutida. Ela pede a cena PRONTA em vez de montá-la — ver
	// o `PlayerSheet`.
	sheetScene sheetHost
	// sheetPlays acompanha o adaptador acima: montar a cena da ficha pede os
	// gestos dela, e o painel embutido só LÊ. Passá-lo zerado seria guardar um
	// ponteiro nulo esperando o primeiro gesto que alguém chamasse daqui.
	sheetPlays character.Plays
}

func (s *Server) tableRules() tableRules {
	return tableRules{
		db:      s.db,
		queries: s.queries, catalogs: s.catalogs,
		boards: s.boards, sessions: s.sessions, presence: s.presence,
		sse: s.sse, bus: s.bus,
		campaign: s.campaignRules(), sheet: s.sheetRules(),
		sheetScene: s.sheetHost(), sheetPlays: s.characterPlays(),
	}
}
