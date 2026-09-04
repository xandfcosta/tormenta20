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

// A MESA, com adaptador próprio — a ÚLTIMA cena a largar o servidor
// (ALE-278, fatia 6).
//
// Trinta e uma assinaturas, a maior porta do projeto. O adaptador é o núcleo
// mais um `tableRules`, e o `tableRules` toca quase todos os campos do
// `*Server` — o que está certo, e a razão está escrita lá: a Mesa É a mesa ao
// vivo, e a mesa ao vivo é o que aqueles stores guardam. O que ela NÃO tem é o
// livro, os trincos por personagem, a espera do desligamento e os métodos das
// outras dez cenas.
type tableHost struct {
	sceneCore
	rules tableRules
}

func (s *Server) tableHost() tableHost {
	return tableHost{sceneCore: s.sceneCore(), rules: s.tableRules()}
}

// O adaptador cumprindo a porta da MESA (`table.Deps`, ALE-278).
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
func (h tableHost) Boards() *tabuleiro.BoardStore      { return h.rules.boards }
func (h tableHost) Sessions() *aovivo.SessionStore     { return h.rules.sessions }
func (h tableHost) Presence() *aovivo.PresenceRegistry { return h.rules.presence }
func (h tableHost) SSE() *aovivo.SSEHub                { return h.rules.sse }

// CharacterChanged avisa que uma ficha da mesa mexeu. A regra é da FICHA e a
// Mesa a pede emprestada, que é o que o campo `sheet` do `tableRules` diz.
func (h tableHost) CharacterChanged(characterID int64) {
	h.rules.sheet.characterChanged(characterID)
}
func (h tableHost) Bus() *events.Bus { return h.rules.bus }

// IsAdminRequester diz se quem pede administra.
//
// O nome NÃO é `IsAdmin`: aquele já existe com `(email string)`, e é outra
// pergunta — "este e-mail é de admin?" contra "quem está pedindo AGORA é?". É a
// colisão que a cena das campanhas registrou, e a regra que ela deixou: um
// contrato que já existe ganha quando é a MESMA pergunta; quando só a cara é a
// mesma, forçar um nome só junta duas coisas diferentes.
func (h tableHost) IsAdminRequester(ctx context.Context, userID int64) bool {
	u, err := h.rules.queries.GetUserByID(ctx, userID)
	if err != nil {
		return false
	}
	return h.rules.cfg.isAdmin(u.Email)
}

// SessionForCaller é a trava de acesso à mesa.
func (h tableHost) SessionForCaller(
	ctx context.Context, userID, campaignID, sessionID int64,
) (sqlcgen.Session, string, int, error) {
	return h.rules.campaign.sessionForCaller(ctx, AuthUser{ID: userID}, campaignID, sessionID)
}

// PlaceDraftCampaign é a trava do RASCUNHO DE LUGAR (ALE-292).
//
// O `loadOwnedCampaign` é a MESMA porta que renomear, apagar, convidar e abrir
// sessão já atravessam: só o dono passa, com o desvio do admin. Montar o acervo
// da campanha é da mesma família — não é um gesto de mesa, é um gesto de dono.
//
// Ela não pergunta mais nada: a outra trava do rascunho — o lugar que está
// aberto numa mesa — é do domínio, e o `EditPlace` a resolve contra todas as
// sessões da campanha.
func (h tableHost) PlaceDraftCampaign(
	ctx context.Context, userID, campaignID int64,
) (sqlcgen.Campaign, int, error) {
	return h.rules.campaign.loadOwnedCampaign(ctx, AuthUser{ID: userID}, campaignID)
}

// ── o estado AO VIVO ─────────────────────────────────────────────────────────

// StartSessionForTable e EndSessionForTable abrem e encerram a partida e
// devolvem o estado AO VIVO, que é o que a cena redesenha.
//
// A leitura da linha mora aqui e não na cena: ela existia lá só para ser
// passada de volta ao hospedeiro, que é a forma "duas perguntas em sequência
// viram uma" que a porta de entrar deixou escrita.
func (h tableHost) StartSessionForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error) {
	sess, err := h.rules.queries.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if _, err := h.rules.StartSession(ctx, sess); err != nil {
		return nil, err
	}
	return h.rules.sessions.GetState(sessionID), nil
}

func (h tableHost) EndSessionForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error) {
	sess, err := h.rules.queries.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if _, err := h.rules.EndSession(ctx, sess); err != nil {
		return nil, err
	}
	return h.rules.sessions.GetState(sessionID), nil
}

func (h tableHost) EndSceneForTable(userID, campaignID, sessionID int64) (*aovivo.SessionRuntimeState, error) {
	return h.rules.endSceneForTable(AuthUser{ID: userID}, campaignID, sessionID)
}

func (h tableHost) RestartCombatForTable(ctx context.Context, sessionID int64) (*aovivo.SessionRuntimeState, error) {
	if err := h.rules.RestartCombat(ctx, sessionID); err != nil {
		return nil, err
	}
	return h.rules.sessions.GetState(sessionID), nil
}

func (h tableHost) RestParty(
	userID, campaignID, sessionID int64, escopo, condicao string,
) (int, int, error) {
	return h.rules.restParty(AuthUser{ID: userID}, campaignID, sessionID, escopo, condicao)
}

func (h tableHost) SelfInitiativeEntry(
	userID, campaignID, characterID, d20 int64,
) (aovivo.InitiativeEntry, error) {
	return h.rules.selfInitiativeEntry(userID, campaignID, characterID, d20)
}

func (h tableHost) MaterializeEntry(
	ctx context.Context, userID, campaignID int64, pedido map[string]any,
) (aovivo.InitiativeEntry, error) {
	return h.rules.materializeEntry(ctx, userID, campaignID, pedido)
}

// PlayerCombatants traduz o `combatant` do hospedeiro na forma que a CENA
// declarou — os campos daqui são minúsculos, e tipo não exportado não atravessa
// fronteira nenhuma.
func (h tableHost) PlayerCombatants(ctx context.Context, campaignID int64) ([]table.Combatant, error) {
	linhas, err := h.rules.listPlayerCombatants(ctx, campaignID)
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

func (h tableHost) PopulateParty(sessionID int64, quem []table.Combatant) (*aovivo.SessionRuntimeState, error) {
	linhas := make([]combatant, 0, len(quem))
	for _, c := range quem {
		linhas = append(linhas, combatant{
			characterID: c.CharacterID, name: c.Name,
			hpCurrent: c.HpCurrent, hpMax: c.HpMax,
			mpCurrent: c.MpCurrent, mpMax: c.MpMax,
		})
	}
	return h.rules.populateParty(sessionID, linhas)
}

func (h tableHost) InitiativeBonus(ctx context.Context, characterID int64) (int64, error) {
	return h.rules.initiativeBonus(ctx, characterID)
}

func (h tableHost) ComputedSheet(ctx context.Context, row sqlcgen.Character) (engine.ComputedSheetV2, error) {
	return h.rules.sheet.ComputeSheet(ctx, row)
}

func (h tableHost) SpeedsForBoard(board *tabuleiro.BoardState) map[string]int {
	return h.rules.speedsForBoard(board)
}

// SaveFailed junta os DOIS stores numa pergunta só.
//
// Para quem está mestrando não existe "o tabuleiro não salvou" e "a fila não
// salvou": existe "a mesa não está sendo salva". Separar daria à tela uma
// decisão que ela não tem o que fazer com — os dois têm a mesma causa (o disco)
// e o mesmo remédio (parar e chamar alguém).
func (h tableHost) SaveFailed(sessionID int64) bool {
	return h.rules.boards.SaveFailed(sessionID) || h.rules.sessions.SaveFailed(sessionID)
}

// ── PUBLICAR, que é do hospedeiro ────────────────────────────────────────────

// Os DOIS passos ficam escritos aqui, e essa é a separação que a ALE-288 fez: o
// disco primeiro, o fio depois. Antes a gravação era uma linha dentro do
// publicador, e como o `SSEHub` não tem ouvinte em produção, apagar o publicador
// — que é a leitura natural de "isto emite para ninguém" — levaria a gravação
// junto e a mesa passaria a viver só em memória (ALE-154).
func (h tableHost) PublishSessionState(sessionID int64, estado *aovivo.SessionRuntimeState) {
	h.rules.saveSession(sessionID)
	h.rules.publishSessionState(sessionID, estado)
}

func (h tableHost) PublishBoardState(sessionID int64, board *tabuleiro.BoardState) {
	h.rules.saveBoard(sessionID, board)
	h.rules.publishBoardState(sessionID, board)
}

func (h tableHost) PublishWhatIsLeft(ctx context.Context, sessionID int64) {
	h.rules.publishWhatIsLeft(ctx, sessionID)
}

// ── as DUAS escritas que a cena montava em SQL ───────────────────────────────
//
// A tabela `sessions` não tem query própria no sqlc para estas duas colunas —
// quem escreve é um SET montado —, e por isso a cena montava `setBuilder` +
// `"UPDATE sessions"` aqui dentro. Cena que compõe SQL é cena com o banco
// dentro, e a resposta é a PERGUNTA: quem sabe o nome da coluna, que vazio é
// NULL e que a linha tem um `updatedAt` a carimbar é o hospedeiro.

func (h tableHost) SaveSessionTitle(ctx context.Context, sessionID int64, titulo string) error {
	var set setBuilder
	set.Add("title = ?", nullableArg(trimOrNull(&titulo)))
	return set.execTouched(ctx, h.rules.db, "UPDATE sessions", sessionID)
}

// SaveNotes grava as notas do mestre, e ela NÃO apara o texto.
//
// A diferença com o título é a que importa: aparar a cada 1,2s comeria a linha
// em branco que o mestre acabou de abrir para escrever o próximo parágrafo. O
// handler JSON apara porque salva UMA vez, ao fechar; este salva no meio da
// digitação. Vazio continua virando NULL.
func (h tableHost) SaveNotes(ctx context.Context, sessionID int64, texto string) error {
	var set setBuilder
	if texto == "" {
		set.Add("notes = ?", nil)
	} else {
		set.Add("notes = ?", texto)
	}
	return set.execTouched(ctx, h.rules.db, "UPDATE sessions", sessionID)
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
func (h tableHost) PlayerSheet(r *http.Request, characterID int64) *sheetui.View {
	ficha, _, err := sheetui.New(h.rules.sheetScene).Load(
		r.Context(), currentUser(r).ID, characterID, sheetui.AskedTab(""), "", sheetui.Signals{})
	if err != nil {
		return nil
	}
	ficha.Embutida = true
	return &ficha
}
