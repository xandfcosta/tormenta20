package api

import (
	"database/sql"
	"t20engine/aovivo"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/events"
)

// AS REGRAS DE ESCRITA DA FICHA, com casa própria (ALE-278, fatia 6).
//
// Dez métodos que mudam a ficha de alguém: subir de nível, conjurar, consumir
// um item, aplicar o efeito de uma magia, gravar proficiência e perícia nova, e
// a reserva de PV temporários sob o vale-o-maior da p256.
//
// Elas leem o MOTOR além do banco, e é isso que as separa das regras de
// campanha: subir de nível recalcula PV e PM (`syncLevelVitals`), conjurar
// pergunta o custo ao círculo, e nenhuma dessas contas é uma consulta. O
// `*sql.DB` está aqui pela mesma razão de sempre — consumir uma dose e gastar o
// item são a mesma transação.
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
	sessions *aovivo.SessionStore
	sse      *aovivo.SSEHub
}

func (s *Server) sheetRules() sheetRules {
	return sheetRules{
		db: s.db, queries: s.queries, catalogs: s.catalogs,
		bus: s.bus, sessions: s.sessions, sse: s.sse,
	}
}
