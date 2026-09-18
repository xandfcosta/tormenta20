package api

import (
	"context"
	"fmt"
	"net/http"

	"t20engine/domain/board"
	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
	"t20engine/serve/web/sheetui"
	"t20engine/serve/web/table"
)

// A MESA, com adaptador próprio, e a maior porta do projeto.
//
// O adaptador é o núcleo mais um `tableRules`, e o `tableRules` toca quase todos
// os campos do `*Server` — o que está certo, e a razão está escrita lá: a Mesa É
// a mesa ao vivo, e a mesa ao vivo é o que aqueles stores guardam. O que ela NÃO
// tem é o livro, os trincos por personagem, a espera do desligamento e os
// métodos das outras cenas.
type tableHost struct {
	sceneCore
	rules tableRules
}

func (s *Server) tableHost() tableHost {
	return tableHost{sceneCore: s.sceneCore(), rules: s.tableRules()}
}

// O adaptador cumprindo a porta da MESA (`table.Deps`).
//
// É a porta mais larga da série, e o arquivo é o lugar de dizer por quê: esta é
// a única cena que MOVIMENTA estado ao vivo. As outras leem o banco e desenham;
// esta abre e encerra cena, move peça, pinta terreno, vira turno e empurra tudo
// para quem está olhando.
//
// O sinal de que a fronteira está no lugar: nenhum destes métodos desenha nada,
// e nenhum handler da cena toca banco fora do `Queries`.

// Os quatro STORES do estado ao vivo, inteiros.
//
// Eles são tipos de OUTROS pacotes, e é isso que os deixa atravessar: a cena
// recebe o vocabulário do domínio ao vivo, não o hospedeiro com outro nome.
// Embrulhá-los método a método daria oitenta entradas na porta e nenhuma
// fronteira a mais — é a mesma concessão do `Queries`, e ela tem o mesmo sinal
// de estar no lugar.
func (h tableHost) Boards() *board.BoardStore        { return h.rules.boards }
func (h tableHost) Sessions() *live.SessionStore     { return h.rules.sessions }
func (h tableHost) Presence() *live.PresenceRegistry { return h.rules.presence }
func (h tableHost) SSE() *live.SSEHub                { return h.rules.sse }

// CharacterChanged avisa que uma ficha da mesa mexeu. A regra é da FICHA e a
// Mesa a pede emprestada, que é o que o campo `sheet` do `tableRules` diz.
func (h tableHost) CharacterChanged(characterID int64) {
	h.rules.sheet.characterChanged(characterID)
}
func (h tableHost) Bus() *events.Bus { return h.rules.bus }

// SessionForCaller é a trava de acesso à mesa.
func (h tableHost) SessionForCaller(
	ctx context.Context, userID, campaignID, sessionID int64,
) (sqlcgen.Session, string, int, error) {
	return h.rules.campaign.sessionForCaller(ctx, AuthUser{ID: userID}, campaignID, sessionID)
}

// PlaceDraftCampaign é a trava do RASCUNHO DE LUGAR.
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

func (h tableHost) EndSceneForTable(userID, campaignID, sessionID int64) (*live.SessionRuntimeState, error) {
	return h.rules.endSceneForTable(AuthUser{ID: userID}, campaignID, sessionID)
}

func (h tableHost) RestParty(
	userID, campaignID, sessionID int64, escopo, condicao string,
) (int, int, error) {
	return h.rules.restParty(AuthUser{ID: userID}, campaignID, sessionID, escopo, condicao)
}

func (h tableHost) SelfInitiativeEntry(
	userID, campaignID, characterID, d20 int64,
) (live.InitiativeEntry, error) {
	return h.rules.selfInitiativeEntry(userID, campaignID, characterID, d20)
}

// CloneCreatureBlock copia o bloco e devolve o id da cópia.
//
// Uma leitura e uma escrita, sem transação: o bloco é uma linha só, e não há
// segundo passo que possa falhar deixando a cópia órfã — que é o que obriga o
// `cloneCharacterTx` a ter dono de transação.
//
// A CAMPANHA vem de fora e não do bloco lido, e isso é deliberado: é o servidor
// que sabe em qual mesa o gesto aconteceu, e copiar o `campaignId` da origem
// deixaria um bloco de outra campanha entrar nesta pelo id na URL.
func (h tableHost) CloneCreatureBlock(ctx context.Context, creatureID, campaignID int64, nome string) (int64, error) {
	origem, err := h.rules.queries.GetCampaignCreature(ctx, creatureID)
	if err != nil {
		return 0, fmt.Errorf("o bloco %d não foi encontrado: %w", creatureID, err)
	}
	if origem.Campaignid != campaignID {
		return 0, fmt.Errorf("o bloco %d é de outra campanha", creatureID)
	}
	agora := dbvalue.NowISO()
	copia, err := h.rules.queries.CreateCampaignCreature(ctx, sqlcgen.CreateCampaignCreatureParams{
		Campaignid: campaignID, Name: nome, Block: origem.Block,
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		return 0, fmt.Errorf("copiar o bloco %d: %w", creatureID, err)
	}
	return copia.ID, nil
}

func (h tableHost) MaterializeEntry(
	ctx context.Context, userID, campaignID int64, pedido map[string]any,
) (live.InitiativeEntry, error) {
	return h.rules.materializeEntry(ctx, userID, campaignID, pedido)
}

func (h tableHost) PopulateParty(sessionID int64, quem []table.Combatant) (*live.SessionRuntimeState, error) {
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

func (h tableHost) SpeedsForBoard(board *board.BoardState) map[string]int {
	return h.rules.speedsForBoard(board)
}

// ── PUBLICAR, que é do hospedeiro ────────────────────────────────────────────

// Os DOIS passos ficam escritos aqui, separados: o disco primeiro, o fio depois.
// Com a gravação escondida dentro do publicador, apagar o publicador — que é a
// leitura natural de "isto emite para ninguém", já que o `SSEHub` não tem
// ouvinte em produção — levaria a gravação junto, e a mesa passaria a viver só
// em memória.
func (h tableHost) PublishSessionState(sessionID int64, estado *live.SessionRuntimeState) {
	h.rules.saveSession(sessionID)
	h.rules.publishSessionState(sessionID, estado)
}

func (h tableHost) PublishBoardState(sessionID int64, board *board.BoardState) {
	h.rules.saveBoard(sessionID, board)
	h.rules.publishBoardState(sessionID, board)
}

func (h tableHost) PublishWhatIsLeft(ctx context.Context, sessionID int64) {
	h.rules.publishWhatIsLeft(ctx, sessionID)
}

// ── a escrita montada em SQL ─────────────────────────────────────────────────
//
// A coluna `notes` não tem query própria no sqlc — quem escreve é um SET
// montado. Ela mora no hospedeiro e não na cena porque cena que compõe SQL é
// cena com o banco dentro. O TÍTULO era o irmão dela e saiu na ALE-344: ele
// mora no `app/session`, que é onde uma escrita sem consulta gerada pode morar.

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
// O `BookAddress` NÃO está aqui: ele vem embutido do `sceneCore`
// (`scene_core.go`), junto com as outras cinco assinaturas que MAIS DE UMA cena
// pede. Declarar um segundo daria ao `*Server` dois nomes para a mesma coisa.

// PlayerSheet é a ficha EMBUTIDA de quem senta à mesa.
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
