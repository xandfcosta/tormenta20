package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"t20engine/db/sqlcgen"
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
	newID  func() string
	q      *sqlcgen.Queries
}

func newBoardStore(q *sqlcgen.Queries, newID func() string) *boardStore {
	return &boardStore{boards: map[int64]*BoardState{}, loaded: map[int64]bool{}, newID: newID, q: q}
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

func (bs *boardStore) hydrateLocked(ctx context.Context, sessionID int64) {
	if bs.loaded[sessionID] {
		return
	}
	bs.loaded[sessionID] = true
	blob, err := bs.q.GetSessionBoard(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return // sessão sem tabuleiro: ausência é resposta, não falha
	}
	if err != nil {
		log.Printf("session %d: board load failed (%v)", sessionID, err)
		return
	}
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
func (bs *boardStore) close(ctx context.Context, sessionID int64) {
	bs.mu.Lock()
	delete(bs.boards, sessionID)
	bs.loaded[sessionID] = true
	bs.mu.Unlock()
	if err := bs.q.DeleteSessionBoard(ctx, sessionID); err != nil {
		log.Printf("session %d: board delete failed (%v)", sessionID, err)
	}
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

func (bs *boardStore) addToken(ctx context.Context, sessionID int64, t BoardToken) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { return addToken(b, t, bs.newID) })
}

func (bs *boardStore) removeToken(ctx context.Context, sessionID int64, tokenID string) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { removeToken(b, tokenID); return nil })
}

func (bs *boardStore) updateToken(ctx context.Context, sessionID int64, tokenID string, patch tokenPatch) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { return updateToken(b, tokenID, patch) })
}

func (bs *boardStore) populate(ctx context.Context, sessionID int64, st *SessionRuntimeState) (*BoardState, error) {
	return bs.apply(ctx, sessionID, func(b *BoardState) error { populateBoard(b, st, bs.newID); return nil })
}

// persist grava o tabuleiro. Best-effort como o do rastreador: a mesa não pode
// parar porque o disco piscou, e a memória continua sendo a verdade da sessão.
func (bs *boardStore) persist(ctx context.Context, sessionID int64) {
	bs.mu.Lock()
	b := cloneBoard(bs.boards[sessionID])
	bs.mu.Unlock()
	if b == nil {
		return
	}
	blob, err := json.Marshal(b)
	if err != nil {
		log.Printf("session %d: board marshal failed (%v)", sessionID, err)
		return
	}
	if err := bs.q.SaveSessionBoard(ctx, sqlcgen.SaveSessionBoardParams{
		Sessionid: sessionID, State: string(blob), Updatedat: nowISO(),
	}); err != nil {
		log.Printf("session %d: board persist failed (%v)", sessionID, err)
	}
}
