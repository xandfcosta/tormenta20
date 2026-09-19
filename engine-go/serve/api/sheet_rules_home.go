package api

import (
	"database/sql"
	"t20engine/app/session"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
)

// O QUE AINDA SOBRA DAS REGRAS DE ESCRITA DA FICHA (ALE-278, fatia 6).
//
// Elas estão MUDANDO DE CASA fatia a fatia: conjurar, beber uma dose e subir de
// nível já viraram `app/character.Plays`, e este tipo é o que ainda não foi —
// aplicar o efeito de uma magia, gravar proficiência e perícia nova, a reserva
// de PV temporários, e o aviso à Mesa.
//
// O `*sql.DB` continua aqui pelas escritas de item do adaptador da cena
// (`setBuilder` sobre `character_items`), que são a próxima fatia.
type sheetRules struct {
	db       *sql.DB
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
	// O AVISO é dependência e não efeito colateral: quem grava a ficha tem de
	// contar à Mesa, senão o mestre vê o PV velho. Ele entra por três
	// colaboradores em vez de um `*Server` porque é exatamente isso que ele usa
	// — e porque um GANCHO opcional já nasceu desligado uma vez neste
	// repositório (ver `characterChanged`).
	bus      *events.Bus
	sessions *session.Store
	sse      *live.SSEHub
}

func (s *Server) sheetRules() sheetRules {
	return sheetRules{
		db: s.db, queries: s.queries, catalogs: s.catalogs,
		bus: s.bus, sessions: s.sessions, sse: s.sse,
	}
}
