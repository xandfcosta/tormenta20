package api

import (
	"context"
	"net/http"

	"t20engine/aovivo"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/events"
	"t20engine/tabuleiro"
	"t20engine/web/sheetui"
	"t20engine/web/table"
)

// O `*Server` cumprindo a porta da MESA (`table.Deps`, ALE-278).
//
// É a porta mais larga da série, e o arquivo é o lugar de dizer por quê: esta é
// a única cena que MOVIMENTA estado ao vivo. As outras leem o banco e desenham;
// esta abre e encerra cena, move peça, pinta terreno, vira turno e empurra tudo
// para quem está olhando.
//
// O sinal de que a fronteira está no lugar continua o mesmo: nenhum destes
// métodos desenha nada, e nenhum handler da cena toca banco fora do `Queries`.

// Os quatro STORES do estado ao vivo, inteiros.
//
// Eles são tipos de OUTROS pacotes, e é isso que os deixa atravessar: a cena
// recebe o vocabulário do domínio ao vivo, não o hospedeiro com outro nome.
// Embrulhá-los método a método daria oitenta entradas na porta e nenhuma
// fronteira a mais — é a mesma concessão do `Queries`, e ela tem o mesmo sinal
// de estar no lugar.
func (s *Server) Boards() *tabuleiro.BoardStore      { return s.boards }
func (s *Server) Sessions() *aovivo.SessionStore     { return s.sessions }
func (s *Server) Presence() *aovivo.PresenceRegistry { return s.presence }
func (s *Server) SSE() *aovivo.SSEHub                { return s.sse }
func (s *Server) Bus() *events.Bus                   { return s.bus }

// IsAdminRequester diz se quem pede administra.
//
// O nome NÃO é `IsAdmin`: aquele já existe com `(email string)`, e é outra
// pergunta — "este e-mail é de admin?" contra "quem está pedindo AGORA é?". É a
// colisão que a cena das campanhas registrou, e a regra que ela deixou: um
// contrato que já existe ganha quando é a MESMA pergunta; quando só a cara é a
// mesma, forçar um nome só junta duas coisas diferentes.
func (s *Server) IsAdminRequester(ctx context.Context, userID int64) bool {
	u, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return false
	}
	return s.cfg.IsAdmin(u.Email)
}

// SessionForCaller é a trava de acesso à mesa.
func (s *Server) SessionForCaller(
	ctx context.Context, userID, campaignID, sessionID int64,
) (sqlcgen.Session, string, int, error) {
	return s.sessionForCaller(ctx, AuthUser{ID: userID}, campaignID, sessionID)
}

// ── o estado AO VIVO ─────────────────────────────────────────────────────────

// StartSessionForTable e EndSessionForTable abrem e encerram a partida e
// devolvem o estado AO VIVO, que é o que a cena redesenha.
//
// A leitura da linha mora aqui e não na cena: ela existia lá só para ser
// passada de volta ao hospedeiro, que é a forma "duas perguntas em sequência
// viram uma" que a porta de entrar deixou escrita.
func (s *Server) StartSessionForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error) {
	sess, err := s.queries.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.StartSession(ctx, sess); err != nil {
		return nil, err
	}
	return s.sessions.GetState(sessionID), nil
}

func (s *Server) EndSessionForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error) {
	sess, err := s.queries.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.EndSession(ctx, sess); err != nil {
		return nil, err
	}
	return s.sessions.GetState(sessionID), nil
}

func (s *Server) EndSceneForTable(userID, campaignID, sessionID int64) (*aovivo.SessionRuntimeState, error) {
	return s.endSceneForTable(AuthUser{ID: userID}, campaignID, sessionID)
}

func (s *Server) RestartCombatForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error) {
	if err := s.RestartCombat(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.sessions.GetState(sessionID), nil
}

func (s *Server) RestParty(
	userID, campaignID, sessionID int64, escopo, condicao string,
) (int, int, error) {
	return s.restParty(AuthUser{ID: userID}, campaignID, sessionID, escopo, condicao)
}

func (s *Server) SelfInitiativeEntry(
	userID, campaignID, characterID, d20 int64,
) (aovivo.InitiativeEntry, error) {
	return s.selfInitiativeEntry(userID, campaignID, characterID, d20)
}

func (s *Server) MaterializeEntry(
	ctx context.Context, userID, campaignID int64, pedido map[string]any,
) (aovivo.InitiativeEntry, error) {
	return s.materializeEntry(ctx, userID, campaignID, pedido)
}

// PlayerCombatants traduz o `combatant` do hospedeiro na forma que a CENA
// declarou — os campos daqui são minúsculos, e tipo não exportado não atravessa
// fronteira nenhuma.
func (s *Server) PlayerCombatants(ctx context.Context, campaignID int64) ([]table.Combatant, error) {
	linhas, err := s.listPlayerCombatants(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	fora := make([]table.Combatant, 0, len(linhas))
	for _, c := range linhas {
		fora = append(fora, table.Combatant{
			CharacterID: c.characterID, Name: c.name,
			HpCurrent: c.hpCurrent, HpMax: c.hpMax,
			MpCurrent: c.mpCurrent, MpMax: c.mpMax,
		})
	}
	return fora, nil
}

func (s *Server) PopulateParty(sessionID int64, quem []table.Combatant) (*aovivo.SessionRuntimeState, error) {
	linhas := make([]combatant, 0, len(quem))
	for _, c := range quem {
		linhas = append(linhas, combatant{
			characterID: c.CharacterID, name: c.Name,
			hpCurrent: c.HpCurrent, hpMax: c.HpMax,
			mpCurrent: c.MpCurrent, mpMax: c.MpMax,
		})
	}
	return s.populateParty(sessionID, linhas)
}

func (s *Server) InitiativeBonus(ctx context.Context, characterID int64) (int64, error) {
	return s.initiativeBonus(ctx, characterID)
}

func (s *Server) ComputedSheet(ctx context.Context, row sqlcgen.Character) (engine.ComputedSheetV2, error) {
	return s.ComputeSheet(ctx, row)
}

func (s *Server) SpeedsForBoard(board *tabuleiro.BoardState) map[string]int {
	return s.speedsForBoard(board)
}

// ── PUBLICAR, que é do hospedeiro ────────────────────────────────────────────

func (s *Server) PublishSessionState(sessionID int64, estado *aovivo.SessionRuntimeState) {
	s.publishSessionState(sessionID, estado)
}

func (s *Server) PublishBoardState(sessionID int64, board *tabuleiro.BoardState) {
	s.publishBoardState(sessionID, board)
}

func (s *Server) PublishWhatIsLeft(ctx context.Context, sessionID int64) {
	s.publishWhatIsLeft(ctx, sessionID)
}

// ── as DUAS escritas que a cena montava em SQL ───────────────────────────────
//
// A tabela `sessions` não tem query própria no sqlc para estas duas colunas —
// quem escreve é um SET montado —, e por isso a cena montava `setBuilder` +
// `"UPDATE sessions"` aqui dentro. Cena que compõe SQL é cena com o banco
// dentro, e a resposta é a PERGUNTA: quem sabe o nome da coluna, que vazio é
// NULL e que a linha tem um `updatedAt` a carimbar é o hospedeiro.

func (s *Server) SaveSessionTitle(ctx context.Context, sessionID int64, titulo string) error {
	var set setBuilder
	set.Add("title = ?", nullableArg(trimOrNull(&titulo)))
	return set.execTouched(ctx, s.db, "UPDATE sessions", sessionID)
}

// SaveNotes grava as notas do mestre, e ela NÃO apara o texto.
//
// A diferença com o título é a que importa: aparar a cada 1,2s comeria a linha
// em branco que o mestre acabou de abrir para escrever o próximo parágrafo. O
// handler JSON apara porque salva UMA vez, ao fechar; este salva no meio da
// digitação. Vazio continua virando NULL.
func (s *Server) SaveNotes(ctx context.Context, sessionID int64, texto string) error {
	var set setBuilder
	if texto == "" {
		set.Add("notes = ?", nil)
	} else {
		set.Add("notes = ?", texto)
	}
	return set.execTouched(ctx, s.db, "UPDATE sessions", sessionID)
}

// ── a casca e a ficha embutida ───────────────────────────────────────────────
//
// O `BookAddress` NÃO está aqui: ele já existia no `piloto_livro.go`, com a
// forma exata que a porta pede. É a regra que a administração deixou — um
// contrato que já existe ganha, e declarar um segundo daria ao `*Server` dois
// nomes para a mesma coisa.

// PlayerSheet é a ficha EMBUTIDA de quem senta à mesa (ALE-275).
//
// A Mesa pede o painel PRONTO em vez de montar a cena da ficha: montá-la lá
// obrigaria a Mesa a cumprir a `sheetui.Deps` inteira — dezoito métodos que ela
// não usa — só para desenhar um painel. Nulo é caminho normal, e a falha é
// silenciosa de propósito: estar numa mesa é mais importante que ver a própria
// ficha dentro dela.
func (s *Server) PlayerSheet(r *http.Request, characterID int64) *sheetui.View {
	ficha, _, err := sheetui.New(s).Load(
		r.Context(), currentUser(r).ID, characterID, sheetui.AskedTab(""), "", sheetui.Signals{})
	if err != nil {
		return nil
	}
	ficha.Embutida = true
	return &ficha
}
