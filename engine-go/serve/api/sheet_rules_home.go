package api

import (
	"t20engine/app/session"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
)

// O QUE AINDA SOBRA DAS REGRAS DE ESCRITA DA FICHA (ALE-278, fatia 6).
//
// Três coisas: gravar proficiência, gravar perícia nova, e avisar a Mesa. Todo
// o resto virou `app/character.Plays` na ALE-347.
//
// **O `*sql.DB` saiu**, e essa é a medida da fatia: nenhuma escrita da ficha
// compõe SQL deste lado da fronteira. As três que sobram falam pelo `queries`,
// que é consulta gerada, e as quatro que compunham `SET` à mão sobre
// `character_items` e `characters` foram para o caso de uso, onde o SQL está
// escrito inteiro e visível.
type sheetRules struct {
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
		queries: s.queries, catalogs: s.catalogs,
		bus: s.bus, sessions: s.sessions, sse: s.sse,
	}
}
