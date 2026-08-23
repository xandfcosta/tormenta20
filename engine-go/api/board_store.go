package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// errNoBoard é a resposta a "mexa no tabuleiro" quando a sessão não tem um. É
// erro e não um tabuleiro vazio de cortesia: vazio desenharia uma grade de 0×0
// na tela, e o mestre acharia que abriu.
var errNoBoard = errors.New("esta sessão não tem tabuleiro aberto")

// boardStore guarda o tabuleiro vivo de cada sessão em memória, com lastro na
// tabela session_boards.
//
// Mutex PRÓPRIO, separado do `sessionStore`: lá o mutex é global a todas as
// sessões, e um tabuleiro movimentado numa mesa serializaria a edição de PV de
// outra mesa. Aqui a mesma trava vale para todos os tabuleiros — quando o custo
// aparecer, ela vira uma por sessão sem mudar quem chama (ALE-124).
type boardStore struct {
	mu     sync.Mutex
	boards map[int64]*BoardState
	// loaded marca a sessão já consultada no banco, para "sem tabuleiro" não
	// virar uma ida ao disco por mensagem.
	loaded map[int64]bool
	// dirty: a última gravação falhou. Espelha o do rastreador, e é o que
	// transforma "gravação falhando em silêncio" em aviso na tela da mesa.
	dirty map[int64]bool
	newID func() string
	q     *sqlcgen.Queries
}

func newBoardStore(q *sqlcgen.Queries, newID func() string) *boardStore {
	return &boardStore{
		boards: map[int64]*BoardState{},
		loaded: map[int64]bool{},
		dirty:  map[int64]bool{},
		newID:  newID,
		q:      q,
	}
}

// cloneBoard copia o tabuleiro para o broadcast. As peças são valores; a cópia
// da fatia é o que impede uma mensagem concorrente de mexer no que já está
// sendo serializado.
func cloneBoard(b *BoardState) *BoardState {
	if b == nil {
		return nil
	}
	out := *b
	out.Tokens = make([]BoardToken, len(b.Tokens))
	copy(out.Tokens, b.Tokens)
	// Os marcadores são valores, mas a FATIA é compartilhada: sem a cópia, uma
	// mensagem concorrente mexeria no que já está sendo serializado (ALE-132).
	out.Markers = make([]BoardMarker, len(b.Markers))
	copy(out.Markers, b.Markers)
	// O provisório é PONTEIRO, e uma cópia rasa deixaria o instantâneo do
	// broadcast apontando para o mesmo movimento que a mensagem seguinte
	// substitui. É a família de defeito da `cloneState` (ALE-132): o que sai
	// no fio tem de ser cópia de verdade.
	if b.Pending != nil {
		pending := *b.Pending
		pending.Path = append([]engine.Square(nil), b.Pending.Path...)
		out.Pending = &pending
	}
	return &out
}

// get devolve o tabuleiro da sessão (nil quando não há), hidratando do banco na
// primeira leitura.
func (bs *boardStore) get(ctx context.Context, sessionID int64) *BoardState {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	return cloneBoard(bs.boards[sessionID])
}

// hydrateLocked traz o tabuleiro do banco na primeira leitura da sessão.
//
// O `loaded` só é marcado no SUCESSO, e essa ordem é a correção da ALE-155.
// Marcando antes da query, um erro transiente de banco na primeira leitura
// ficava cacheado como "esta sessão não tem tabuleiro" **até o processo
// reiniciar** — a falha de LEITURA com o mesmo modo silencioso que a de
// gravação tinha antes da ALE-124.
//
// `sql.ErrNoRows` é a exceção deliberada: "sessão sem tabuleiro" é uma resposta
// legítima e definitiva, então ela MARCA e evita uma ida ao disco por mensagem.
// Qualquer outro erro deixa a sessão sem marca, e a mensagem seguinte tenta de
// novo.
func (bs *boardStore) hydrateLocked(ctx context.Context, sessionID int64) {
	if bs.loaded[sessionID] {
		return
	}
	blob, err := bs.q.GetSessionBoard(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		bs.loaded[sessionID] = true // ausência é resposta, não falha
		return
	}
	if err != nil {
		// Sem marcar: a próxima mensagem tenta de novo em vez de servir um
		// "sem tabuleiro" que só existe porque o banco piscou.
		log.Printf("session %d: board load failed (%v); tentará de novo", sessionID, err)
		return
	}
	bs.loaded[sessionID] = true
	var parsed BoardState
	if err := json.Unmarshal([]byte(blob), &parsed); err != nil {
		log.Printf("session %d: board blob malformed (%v); tratando como sem tabuleiro", sessionID, err)
		return
	}
	if parsed.Tokens == nil {
		parsed.Tokens = []BoardToken{}
	}
	bs.boards[sessionID] = &parsed
}

// open abre (ou substitui) o tabuleiro da sessão.
func (bs *boardStore) open(ctx context.Context, sessionID int64, place, terrain string) *BoardState {
	b := newBoard(place, terrain)
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	// A versão CONTINUA de onde estava: um cliente que ainda tem o tabuleiro
	// velho precisa reconhecer o novo como mais recente, e não como um retorno
	// ao passado.
	if old := bs.boards[sessionID]; old != nil && old.Version >= b.Version {
		b.Version = old.Version + 1
	}
	bs.boards[sessionID] = b
	return cloneBoard(b)
}

// close encerra o tabuleiro: some da memória e do banco. As posições não são
// arquivadas AINDA — os Lugares da crônica são fatia própria (ALE-124).
//
// Devolve as transições de saúde como o `persist`, e pelo mesmo motivo
// (ALE-155): se o DELETE falha, a memória diz "fechado" e o banco mantém a
// linha — no próximo boot o tabuleiro FANTASMA volta, com as peças de uma cena
// que a mesa já encerrou. Antes isso morria numa linha de log.
func (bs *boardStore) close(ctx context.Context, sessionID int64) (dirty, changed bool) {
	bs.mu.Lock()
	delete(bs.boards, sessionID)
	bs.loaded[sessionID] = true
	bs.mu.Unlock()

	err := bs.q.DeleteSessionBoard(ctx, sessionID)

	bs.mu.Lock()
	defer bs.mu.Unlock()
	prev := bs.dirty[sessionID]
	dirty = err != nil
	changed = prev != dirty
	if dirty {
		bs.dirty[sessionID] = true
		log.Printf("session %d: board delete failed (%v)", sessionID, err)
		return dirty, changed
	}
	delete(bs.dirty, sessionID)
	return dirty, changed
}

// apply roda uma mutação pura sob a trava e devolve o instantâneo para o
// broadcast. Recusa quando não há tabuleiro: mexer no que não existe é erro de
// quem chamou, não um tabuleiro criado por acidente.
func (bs *boardStore) apply(ctx context.Context, sessionID int64, fn func(*BoardState) error) (*BoardState, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	b := bs.boards[sessionID]
	if b == nil {
		return nil, errNoBoard
	}
	if err := fn(b); err != nil {
		return nil, err
	}
	return cloneBoard(b), nil
}

// addToken põe a peça no tabuleiro. Sem posição declarada, ela nasce no
// primeiro quadrado livre da fileira de entrada — senão duas peças criadas
// seguidas ficariam uma em cima da outra (ALE-166).
func (bs *boardStore) addToken(ctx context.Context, sessionID int64, t BoardToken, temPosicao bool) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error {
		if !temPosicao {
			spot := nextFreeSpot(b)
			t.X, t.Y = spot.x, spot.y
		}
		return addToken(b, t, bs.newID)
	})
}

func (bs *boardStore) removeToken(ctx context.Context, sessionID int64, tokenID string) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { removeToken(b, tokenID); return nil })
}

// duplicateToken põe outra igual ao lado, numerada pelo SERVIDOR: dois clientes
// duplicando ao mesmo tempo não podem inventar o mesmo "Zumbi 3" (ALE-192).
func (bs *boardStore) duplicateToken(ctx context.Context, sessionID int64, tokenID string) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error {
		return duplicateToken(b, tokenID, bs.newID)
	})
}

func (bs *boardStore) updateToken(ctx context.Context, sessionID int64, tokenID string, patch tokenPatch) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { return updateToken(b, tokenID, patch) })
}

// Marcadores (ALE-195): o lugar apontado no mapa que não é peça.
func (bs *boardStore) addMarker(ctx context.Context, sessionID int64, m BoardMarker) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { return addMarker(b, m, bs.newID) })
}

func (bs *boardStore) updateMarker(ctx context.Context, sessionID int64, markerID string, patch markerPatch) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { return updateMarker(b, markerID, patch) })
}

func (bs *boardStore) removeMarker(ctx context.Context, sessionID int64, markerID string) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { removeMarker(b, markerID); return nil })
}

func (bs *boardStore) paintTerrain(ctx context.Context, sessionID int64, square engine.Square, difficult bool) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error {
		paintTerrain(b, square, difficult)
		return nil
	})
}

func (bs *boardStore) populate(
	ctx context.Context, sessionID int64, st *SessionRuntimeState, chosen entrySelection,
) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error {
		populateBoard(b, st, bs.newID, chosen)
		return nil
	})
}

// setSpeeds grava o orçamento de várias peças de uma vez. Uma mutação só, e um
// broadcast só: um `updateToken` por peça faria a mesa receber seis tabuleiros
// seguidos ao trazer o grupo.
func (bs *boardStore) setSpeeds(ctx context.Context, sessionID int64, speeds map[string]int) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error {
		for i := range b.Tokens {
			if squares, ok := speeds[b.Tokens[i].ID]; ok && squares > 0 {
				b.Tokens[i].SpeedSquares = squares
			}
		}
		b.Version++
		return nil
	})
}

// persist grava o tabuleiro e devolve as transições de saúde, como o do
// rastreador: a mesa não para porque o disco piscou, mas ela precisa SABER
// quando parou de gravar.
//
// Esta devolução nasceu de um defeito real: a tabela `session_boards` sumiu do
// banco de desenvolvimento (a migração constava aplicada e a tabela não
// existia), e o tabuleiro passou um dia vivendo só em memória — cada gravação
// falhava numa linha de log que ninguém lê, e na tela estava tudo perfeito até
// o processo reiniciar. Falha permanente de gravação não é "o disco piscou".
func (bs *boardStore) persist(ctx context.Context, sessionID int64) (dirty, changed bool) {
	bs.mu.Lock()
	b := cloneBoard(bs.boards[sessionID])
	bs.mu.Unlock()
	if b == nil {
		return false, false
	}
	blob, err := json.Marshal(b)
	if err != nil {
		log.Printf("session %d: board marshal failed (%v)", sessionID, err)
		return false, false
	}
	err = bs.q.SaveSessionBoard(ctx, sqlcgen.SaveSessionBoardParams{
		Sessionid: sessionID, State: string(blob), Updatedat: nowISO(),
	})

	bs.mu.Lock()
	defer bs.mu.Unlock()
	prev := bs.dirty[sessionID] // ausente ⇒ false (saudável)
	dirty = err != nil
	changed = prev != dirty
	if dirty {
		bs.dirty[sessionID] = true
		log.Printf("session %d: board persist failed (%v)", sessionID, err)
		return dirty, changed
	}
	delete(bs.dirty, sessionID)
	return dirty, changed
}

// proposeMove, commitMove e cancelMove são as três portas do movimento (ALE-124).
// A posse e o orçamento chegam RESOLVIDOS do gateway: quem consulta o banco é
// ele, e a trava daqui não pode esperar por I/O.

func (bs *boardStore) proposeMove(ctx context.Context, sessionID int64, st *SessionRuntimeState, tokenID string, path []engine.Square, by mover, speedSquares int) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error {
		// O orçamento fresco do motor entra na peça ANTES da medição: sem isso,
		// a armadura vestida no meio da sessão só valeria no movimento seguinte.
		if token := findToken(b, tokenID); token != nil && speedSquares > 0 {
			token.SpeedSquares = speedSquares
		}
		return proposeMove(b, st, tokenID, path, by)
	})
}

func (bs *boardStore) commitMove(ctx context.Context, sessionID int64, st *SessionRuntimeState, version int64, by mover) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { return commitMove(b, st, version, by) })
}

func (bs *boardStore) cancelMove(ctx context.Context, sessionID int64, by mover) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { return cancelMove(b, by) })
}
