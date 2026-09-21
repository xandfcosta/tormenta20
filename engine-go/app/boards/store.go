package boards

import "t20engine/domain/live"

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"t20engine/infra/db/dbvalue"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
)

// errNoBoard é a resposta a "mexa no tabuleiro" quando a sessão não tem um. É
// erro e não um tabuleiro vazio de cortesia: vazio desenharia uma grade de 0×0
// na tela, e o mestre acharia que abriu.
var errNoBoard = errors.New("esta sessão não tem tabuleiro aberto")

// openBoardsCeiling — quantos tabuleiros uma sessão pode ter abertos ao mesmo
// tempo.
//
// Existe pelo mesmo motivo do `MaxTokens`: sem teto o estado cresce sem
// limite e TODA hidratação e TODA gravação o carregam. Oito frentes é uma cena
// que nenhuma mesa joga.
const openBoardsCeiling = 8

// Store guarda os tabuleiros vivos de cada sessão em memória, com lastro na
// tabela open_boards.
//
// Mutex PRÓPRIO, separado do `session.Store`: lá o mutex é global a todas as
// sessões, e um tabuleiro movimentado numa mesa serializaria a edição de PV de
// outra mesa. Aqui a mesma trava vale para todos os tabuleiros — quando o custo
// aparecer, ela vira uma por sessão sem mudar quem chama.
type Store struct {
	Mu sync.Mutex
	// boards é uma LISTA por sessão, e a ordem dela é a de abertura: ela é a
	// ordem das abas na tela, e a primeira é a aba PADRÃO de quem ainda não
	// escolheu. Um mapa por id perderia a ordem e faria a barra de abas mudar de
	// forma a cada carga.
	boards map[int64][]*board.BoardState
	// loaded marca a sessão já consultada no banco, para "sem tabuleiro" não
	// virar uma ida ao disco por mensagem.
	loaded map[int64]bool
	// Dirty: a última gravação falhou. Espelha o do rastreador, e é o que
	// transforma "gravação falhando em silêncio" em aviso na tela da mesa.
	//
	// Continua por SESSÃO e não por tabuleiro, porque o aviso é da mesa: uma tarja
	// por aba faria o mestre conferir oito lugares para saber se o disco está
	// vivo.
	Dirty map[int64]bool
	// bus é por onde as mudanças deste tabuleiro viram notícia.
	//
	// Barramento e não um registro de ouvintes por store: dois stores com travas
	// próprias, cada um chamando o aviso do outro de dentro da trava, é como se
	// escreve um abraço mortal. O barramento é FOLHA — pega só a própria trava e
	// nunca chama de volta um store (ver `events.Bus.Publish`) —, e mesmo assim a
	// publicação sai de fora da trava.
	bus   *events.Bus
	newID func() string
	q     *sqlcgen.Queries
}

func NewStore(q *sqlcgen.Queries, newID func() string, bus *events.Bus) *Store {
	return &Store{
		bus:    bus,
		boards: map[int64][]*board.BoardState{},
		loaded: map[int64]bool{},
		Dirty:  map[int64]bool{},
		newID:  newID,
		q:      q,
	}
}

// cloneBoard copia o tabuleiro para o broadcast. As peças são valores; a cópia
// da fatia é o que impede uma mensagem concorrente de mexer no que já está
// sendo serializado.
func cloneBoard(b *board.BoardState) *board.BoardState {
	if b == nil {
		return nil
	}
	out := *b
	out.Tokens = make([]board.BoardToken, len(b.Tokens))
	copy(out.Tokens, b.Tokens)
	// Os marcadores são valores, mas a FATIA é compartilhada: sem a cópia, uma
	// mensagem concorrente mexeria no que já está sendo serializado.
	out.Markers = make([]board.BoardMarker, len(b.Markers))
	copy(out.Markers, b.Markers)
	// O provisório é PONTEIRO, e uma cópia rasa deixaria o instantâneo do
	// broadcast apontando para o mesmo movimento que a mensagem seguinte
	// substitui. O que sai no fio tem de ser cópia de verdade.
	if b.Pending != nil {
		pending := *b.Pending
		pending.Path = append([]engine.Square(nil), b.Pending.Path...)
		out.Pending = &pending
	}
	return &out
}

// Get devolve UM tabuleiro da sessão (nil quando não há), hidratando do banco na
// primeira leitura.
//
// `boardID` vazio significa "a aba padrão", e não "erro": quem entra na
// sessão pela primeira vez ainda não escolheu aba nenhuma, e o que ele vê é o
// primeiro tabuleiro aberto. Um id que não existe devolve NIL em vez de cair no
// padrão — a aba que o mestre fechou tem de sumir da tela de quem estava nela,
// e não virar outra cena em silêncio.
func (bs *Store) Get(ctx context.Context, sessionID int64, boardID string) *board.BoardState {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	return cloneBoard(bs.findLocked(sessionID, boardID))
}

// OpenBoards devolve os tabuleiros da sessão na ordem de abertura — é o que a
// barra de abas desenha.
//
// Cópias, como o `Get`: quem recebe a lista a redige por papel e a serializa,
// e devolver os ponteiros vivos deixaria o `board.BoardForRole` do chamador
// escrevendo no estado da mesa.
func (bs *Store) OpenBoards(ctx context.Context, sessionID int64) []*board.BoardState {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	open := make([]*board.BoardState, 0, len(bs.boards[sessionID]))
	for _, b := range bs.boards[sessionID] {
		open = append(open, cloneBoard(b))
	}
	return open
}

// DefaultBoardID é o id da aba de quem ainda não escolheu: a mais antiga.
//
// A MAIS ANTIGA e não a última aberta: o mestre abre a taverna sob cortina
// enquanto a mesa olha a cripta, e por "a última" a mesa inteira seria puxada
// para uma cortina sem ninguém pedir. Quem move a mesa de propósito é o FORÇAR,
// que é gesto.
func (bs *Store) DefaultBoardID(ctx context.Context, sessionID int64) string {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	if open := bs.boards[sessionID]; len(open) > 0 {
		return open[0].ID
	}
	return ""
}

// nextSeqLocked é o número da PRÓXIMA aba desta sessão.
//
// `max + 1` e não `len + 1`: fechar a aba do meio deixaria dois tabuleiros com
// o mesmo número, e dois números iguais é o empate que esta coluna existe para
// não ter. Buraco na sequência não custa nada — ela só serve para ordenar.
func (bs *Store) nextSeqLocked(sessionID int64) int64 {
	var max int64
	for _, b := range bs.boards[sessionID] {
		if b.Seq > max {
			max = b.Seq
		}
	}
	return max + 1
}

// findLocked resolve o id na lista da sessão, com a trava já na mão.
func (bs *Store) findLocked(sessionID int64, boardID string) *board.BoardState {
	open := bs.boards[sessionID]
	if len(open) == 0 {
		return nil
	}
	if boardID == "" {
		return open[0]
	}
	for _, b := range open {
		if b.ID == boardID {
			return b
		}
	}
	return nil
}

// hydrateLocked traz os tabuleiros do banco na primeira leitura da sessão.
//
// O `loaded` só é marcado no SUCESSO. Marcando antes da query, um erro
// transiente de banco na primeira leitura fica cacheado como "esta sessão não
// tem tabuleiro" **até o processo reiniciar**.
//
// A lista VAZIA é a exceção deliberada: "sessão sem tabuleiro" é uma resposta
// legítima e definitiva, então ela MARCA e evita uma ida ao disco por mensagem.
// Qualquer outro erro deixa a sessão sem marca, e a mensagem seguinte tenta de
// novo.
func (bs *Store) hydrateLocked(ctx context.Context, sessionID int64) {
	if bs.loaded[sessionID] {
		return
	}
	rows, err := bs.q.ListOpenBoards(ctx, sessionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// Sem marcar: a próxima mensagem tenta de novo em vez de servir um
		// "sem tabuleiro" que só existe porque o banco piscou.
		log.Printf("session %d: board Load failed (%v); tentará de novo", sessionID, err)
		return
	}
	bs.loaded[sessionID] = true
	open := make([]*board.BoardState, 0, len(rows))
	for _, row := range rows {
		var parsed board.BoardState
		if err := json.Unmarshal([]byte(row.State), &parsed); err != nil {
			log.Printf("session %d: board %s blob malformed (%v); tratando como sem tabuleiro",
				sessionID, row.Boardid, err)
			continue
		}
		if parsed.Tokens == nil {
			parsed.Tokens = []board.BoardToken{}
		}
		// O ID e a SEQUÊNCIA vêm da COLUNA e não do JSON, pela mesma razão do nome
		// do lugar no `Reopen`: duas verdades sobre quem é este tabuleiro é como
		// elas divergem, e a de fora é a que o upsert usa.
		parsed.ID = row.Boardid
		parsed.Seq = row.Openseq
		open = append(open, &parsed)
	}
	if len(open) > 0 {
		bs.boards[sessionID] = open
	}
}

// Open abre MAIS UM tabuleiro na sessão e devolve o que nasceu.
//
// Ele ACRESCENTA em vez de substituir: o grupo se separou e a cripta não pode
// custar a taverna. Quem tira uma cena da mesa é o `Close`, que arquiva; abrir
// nunca destrói.
//
// A versão nasce em 1 e não continua a de ninguém: são dois tabuleiros
// diferentes, com dois contadores, e continuar um no outro faria o número
// mentir sobre quantas vezes ESTA cena mudou.
func (bs *Store) Open(ctx context.Context, sessionID int64, place, terrain string) (*board.BoardState, error) {
	b, err := bs.openLocked(ctx, sessionID, place, terrain)
	if err != nil {
		return nil, err
	}
	bs.bus.Publish(events.BoardOpened{SessionID: sessionID})
	return b, nil
}

func (bs *Store) openLocked(ctx context.Context, sessionID int64, place, terrain string) (*board.BoardState, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	if len(bs.boards[sessionID]) >= openBoardsCeiling {
		return nil, fmt.Errorf(
			"esta sessão já tem %d tabuleiros abertos (teto %d): feche um antes de abrir outro",
			len(bs.boards[sessionID]), openBoardsCeiling)
	}
	b := board.NewBoard(bs.newID(), place, terrain)
	b.Seq = bs.nextSeqLocked(sessionID)
	bs.boards[sessionID] = append(bs.boards[sessionID], b)
	return cloneBoard(b), nil
}

// Close encerra UM tabuleiro: some da memória e do banco. As posições não são
// arquivadas AQUI — quem arquiva é o gateway, antes de chamar.
//
// Devolve as transições de saúde como o `Persist`: se o DELETE falha, a memória
// diz "fechado" e o banco mantém a linha — no próximo boot o tabuleiro FANTASMA
// volta, com as peças de uma cena que a mesa já encerrou.
func (bs *Store) Close(ctx context.Context, sessionID int64, boardID string) (Dirty, changed bool) {
	bs.Mu.Lock()
	target := bs.findLocked(sessionID, boardID)
	if target == nil {
		bs.Mu.Unlock()
		return bs.Dirty[sessionID], false
	}
	closed := target.ID
	remaining := make([]*board.BoardState, 0, len(bs.boards[sessionID]))
	for _, b := range bs.boards[sessionID] {
		if b.ID != closed {
			remaining = append(remaining, b)
		}
	}
	if len(remaining) == 0 {
		delete(bs.boards, sessionID)
	} else {
		bs.boards[sessionID] = remaining
	}
	bs.loaded[sessionID] = true
	bs.Mu.Unlock()
	bs.bus.Publish(events.BoardClosed{SessionID: sessionID})

	// SEM CANCELAMENTO: o `ctx` que chega aqui é o da REQUISIÇÃO, e ele morre
	// quando quem clicou vai embora — a aba fechada, o telefone bloqueado, a rede
	// caindo entre o clique e a resposta. A linha ficaria no banco, o `Dirty`
	// acenderia, e não há quem tente de novo.
	//
	// **Limpeza que depende de o cliente esperar não é limpeza.** O tabuleiro já
	// saiu da memória três linhas acima; deixar a linha no banco faria a próxima
	// hidratação trazer de volta uma cena que o mestre encerrou.
	//
	// `WithoutCancel` e não `context.Background()`: os valores do contexto (prazo
	// do servidor, rastros) continuam valendo — o que se descarta é o cancelamento,
	// que é a única coisa que pertence ao cliente.
	err := bs.q.DeleteOpenBoard(context.WithoutCancel(ctx), sqlcgen.DeleteOpenBoardParams{
		Sessionid: sessionID, Boardid: closed,
	})

	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	prev := bs.Dirty[sessionID]
	Dirty = err != nil
	changed = prev != Dirty
	if Dirty {
		bs.Dirty[sessionID] = true
		log.Printf("session %d: board delete failed (%v)", sessionID, err)
		return Dirty, changed
	}
	delete(bs.Dirty, sessionID)
	return Dirty, changed
}

// SessionDeleted é o FIM DA VIDA dos tabuleiros de uma sessão.
//
// Ela não é o `Close`, e a diferença importa. O `Close` é o mestre ENCERRANDO
// uma cena, e uma falha de disco ali é notícia — a mesa precisa saber que parou
// de gravar. Aqui a SESSÃO deixou de existir, e com ela qualquer motivo para
// gravar: sem esta porta, o mapa em memória sobreviveria à sessão, o `Persist`
// seguinte bateria na chave estrangeira de `open_boards`, e o `Dirty` acenderia
// para NUNCA mais sair — um alarme que toca sozinho é como se aprende a ignorar
// o alarme.
//
// A MARCA sai junto, e é o oposto do que o `SessionStore.Forget` faz: lá o
// `Dirty` fica porque a sessão continua existindo e o próximo `Persist` ainda
// tem de avisar que ela voltou ao normal. A premissa daquele argumento é a
// sessão continuar viva, e é exatamente ela que esta porta desmente.
//
// Não toca no BANCO: a linha de `open_boards` some por CASCATA quando a sessão
// é apagada (migração 00010). Apagá-la aqui seria a segunda verdade sobre quem
// limpa, e a que roda depois falharia por não achar nada.
func (bs *Store) SessionDeleted(sessionID int64) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	delete(bs.boards, sessionID)
	delete(bs.loaded, sessionID)
	delete(bs.Dirty, sessionID)
}

// apply roda uma mutação pura sobre UM tabuleiro, sob a trava, e devolve o
// instantâneo para o broadcast. Recusa quando aquele tabuleiro não existe: mexer
// no que não existe é erro de quem chamou, não um tabuleiro criado por acidente.
func (bs *Store) apply(
	ctx context.Context, sessionID int64, boardID string, fn func(*board.BoardState) error,
) (*board.BoardState, error) {
	b, err := bs.applyLocked(ctx, sessionID, boardID, fn)
	if err != nil {
		return nil, err
	}
	bs.bus.Publish(events.BoardChanged{SessionID: sessionID})
	return b, nil
}

func (bs *Store) applyLocked(
	ctx context.Context, sessionID int64, boardID string, fn func(*board.BoardState) error,
) (*board.BoardState, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	b := bs.findLocked(sessionID, boardID)
	if b == nil {
		return nil, errNoBoard
	}
	if err := fn(b); err != nil {
		return nil, err
	}
	return cloneBoard(b), nil
}

// board.AddToken põe a peça no tabuleiro, NA CASA que ela traz. A posição é sempre
// declarada: o gesto de criar peça POSICIONA — o modo liga, o clique numa casa
// diz onde, e a coordenada viaja no caminho.
func (bs *Store) AddToken(ctx context.Context, sessionID int64, boardID string, t board.BoardToken) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.AddToken(b, t, bs.newID)
	})
}

func (bs *Store) RemoveToken(ctx context.Context, sessionID int64, boardID, tokenID string) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error { board.RemoveToken(b, tokenID); return nil })
}

// board.DuplicateToken põe outra igual ao lado, numerada pelo SERVIDOR: dois clientes
// duplicando ao mesmo tempo não podem inventar o mesmo "Zumbi 3".
//
// O `loop` é o que a cópia vai ser — ver o `board.DuplicateToken` do estado, onde a
// decisão está escrita. Ele chega PRONTO porque a linha nova mora no
// `session.Store`, e este store não o conhece.
func (bs *Store) DuplicateToken(ctx context.Context, sessionID int64, boardID, tokenID string, loop *live.InitiativeEntry) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.DuplicateToken(b, tokenID, loop, bs.newID)
	})
}

// board.PasteToken põe a cópia NESTE tabuleiro, e a original pode ser de outro.
func (bs *Store) PasteToken(ctx context.Context, sessionID int64, boardID string, template board.BoardToken, loop *live.InitiativeEntry, x, y int) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.PasteToken(b, template, loop, x, y, bs.newID)
	})
}

func (bs *Store) UpdateToken(ctx context.Context, sessionID int64, boardID, tokenID string, patch board.TokenPatch) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error { return board.UpdateToken(b, tokenID, patch) })
}

// board.ClearSquare é a BORRACHA: tira todo terreno de um quadrado.
func (bs *Store) ClearSquare(ctx context.Context, sessionID int64, boardID string, square engine.Square) (*board.BoardState, error) {
	return bs.ClearStroke(ctx, sessionID, boardID, []engine.Square{square})
}

// ClearStroke apaga o segmento inteiro numa gravação só.
//
// UMA transação para o traço e não uma por casa, e a diferença não é desempenho:
// `apply` sobe a versão do tabuleiro e publica para a mesa. Uma gravação por casa
// faria a mesa receber dez quadros para um gesto só, e cada um deles com metade
// do traço desenhada.
func (bs *Store) ClearStroke(ctx context.Context, sessionID int64, boardID string, trait []engine.Square) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		for _, square := range trait {
			board.ClearSquare(b, square)
		}
		return nil
	})
}

// board.ReturnToken desfaz o último pouso.
func (bs *Store) ReturnToken(ctx context.Context, sessionID int64, boardID, tokenID string) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error { return board.ReturnToken(b, tokenID) })
}

// Marcadores: o lugar apontado no mapa que não é peça.
func (bs *Store) AddMarker(ctx context.Context, sessionID int64, boardID string, m board.BoardMarker) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error { return board.AddMarker(b, m, bs.newID) })
}

func (bs *Store) UpdateMarker(ctx context.Context, sessionID int64, boardID, markerID string, patch board.MarkerPatch) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error { return board.UpdateMarker(b, markerID, patch) })
}

func (bs *Store) RemoveMarker(ctx context.Context, sessionID int64, boardID, markerID string) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error { board.RemoveMarker(b, markerID); return nil })
}

func (bs *Store) PaintTerrain(
	ctx context.Context, sessionID int64, boardID string, square engine.Square, species board.TerrainKind, on bool,
) (*board.BoardState, error) {
	return bs.PaintStroke(ctx, sessionID, boardID, []engine.Square{square}, species, on)
}

// PaintStroke pinta o segmento inteiro numa gravação só — ver `ClearStroke`.
func (bs *Store) PaintStroke(
	ctx context.Context, sessionID int64, boardID string, trait []engine.Square, species board.TerrainKind, on bool,
) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		for _, square := range trait {
			board.PaintTerrain(b, square, species, on)
		}
		return nil
	})
}

func (bs *Store) Populate(
	ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, chosen board.EntrySelection,
) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		board.PopulateBoard(b, st, bs.newID, chosen)
		return nil
	})
}

// SetSpeeds grava o orçamento de várias peças de uma vez. Uma mutação só, e um
// broadcast só: um `board.UpdateToken` por peça faria a mesa receber seis tabuleiros
// seguidos ao trazer o grupo.
func (bs *Store) SetSpeeds(ctx context.Context, sessionID int64, boardID string, speeds map[string]int) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		for i := range b.Tokens {
			if squares, ok := speeds[b.Tokens[i].ID]; ok && squares > 0 {
				b.Tokens[i].SpeedSquares = squares
			}
		}
		b.Version++
		return nil
	})
}

// Persist grava UM tabuleiro e devolve as transições de saúde, como o do
// rastreador: a mesa não para porque o disco piscou, mas ela precisa SABER
// quando parou de gravar. Falha permanente de gravação não é "o disco piscou".
//
// UM e não todos os abertos da sessão, e a razão é o tamanho: com oito abas,
// gravar todas a cada peça que anda seria oito serializações e oito upserts por
// gesto. Quem chama sabe qual aba mudou porque acabou de mutá-la.
//
// A saúde continua sendo da SESSÃO: ver o campo `Dirty`.

// SaveFailed diz se a última gravação do tabuleiro desta sessão falhou.
//
// ESTADO e não notícia, e a diferença é o que faz o aviso servir: ele vale
// enquanto durar, então quem abre a aba dez minutos depois da primeira falha
// merece vê-lo. Um evento perdido é um evento que não existiu.
//
// Sob a trava porque o `Dirty` é escrito pelo `Persist`, que roda em goroutine.
func (bs *Store) SaveFailed(sessionID int64) bool {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	return bs.Dirty[sessionID]
}

func (bs *Store) Persist(ctx context.Context, sessionID int64, boardID string) (Dirty, changed bool) {
	bs.Mu.Lock()
	b := cloneBoard(bs.findLocked(sessionID, boardID))
	bs.Mu.Unlock()
	if b == nil {
		return false, false
	}
	blob, err := json.Marshal(b)
	if err != nil {
		log.Printf("session %d: board marshal failed (%v)", sessionID, err)
		return false, false
	}
	// `openSeq` só entra no INSERT — o upsert não o toca (ver a query). Gravar o
	// tabuleiro é dizer que ele mudou, nunca que ele nasceu de novo, e a ordem
	// das abas na tela sai daquela coluna.
	err = bs.q.SaveOpenBoard(ctx, sqlcgen.SaveOpenBoardParams{
		Sessionid: sessionID, Boardid: b.ID, State: string(blob),
		Openseq: b.Seq, Updatedat: dbvalue.NowISO(),
	})

	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	prev := bs.Dirty[sessionID] // ausente ⇒ false (saudável)
	Dirty = err != nil
	changed = prev != Dirty
	if Dirty {
		bs.Dirty[sessionID] = true
		log.Printf("session %d: board %s Persist failed (%v)", sessionID, b.ID, err)
		return Dirty, changed
	}
	delete(bs.Dirty, sessionID)
	return Dirty, changed
}

// board.ProposeMove, board.CommitMove e board.CancelMove são as três portas do movimento.
// A posse e o orçamento chegam RESOLVIDOS do gateway: quem consulta o banco é
// ele, e a trava daqui não pode esperar por I/O.

func (bs *Store) ProposeMove(ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, tokenID string, path []engine.Square, by board.Mover, speedSquares int) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		// O orçamento fresco do motor entra na peça ANTES da medição: sem isso,
		// a armadura vestida no meio da sessão só valeria no movimento seguinte.
		if token := board.FindToken(b, tokenID); token != nil && speedSquares > 0 {
			token.SpeedSquares = speedSquares
		}
		return board.ProposeMove(b, st, tokenID, path, by)
	})
}

// board.ProposeMoveWithStops é a porta de quem monta o movimento CLICANDO, e ela
// existe para a lista de paradas ser guardada junto.
//
// Mesma trava, mesmo orçamento fresco, mesma medição: o que muda é a memória de
// ONDE a pessoa parou, que o caminho sozinho não deixa reconstruir. Sem ela,
// "desfazer a última perna" seria um palpite sobre o movimento que a mesa está
// vendo.
func (bs *Store) ProposeMoveWithStops(ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, tokenID string, stops []engine.Square, by board.Mover, speedSquares int) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		if token := board.FindToken(b, tokenID); token != nil && speedSquares > 0 {
			token.SpeedSquares = speedSquares
		}
		return board.ProposeMoveWithStops(b, st, tokenID, stops, by)
	})
}

func (bs *Store) CommitMove(ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, version int64, by board.Mover) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error { return board.CommitMove(b, st, version, by) })
}

func (bs *Store) CancelMove(ctx context.Context, sessionID int64, boardID string, by board.Mover) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error { return board.CancelMove(b, by) })
}

// SetCurtain fecha ou abre a CORTINA. Devolve `changed` falso quando o estado
// já era o pedido: fechar cortina fechada não é erro — dois cliques no telefone
// do mestre, ou duas abas abertas — mas também não é mutação, e publicar quadro
// por não-mudança acorda a mesa inteira à toa.
//
// A cortina é POR TABULEIRO porque `Curtained` é campo do `board.BoardState`: o
// mestre monta a taverna com a cortina fechada enquanto a mesa olha a cripta.
func (bs *Store) SetCurtain(ctx context.Context, sessionID int64, boardID string, closed bool) (*board.BoardState, bool, error) {
	var changed bool
	b, err := bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		changed = b.Curtained != closed
		if !changed {
			return nil
		}
		b.Curtained = closed
		// `Version` significa "o tabuleiro mudou", e fechar a cortina é uma
		// mudança: toda mutação aceita neste arquivo sobe o contador, e uma que não
		// subisse faria o número mentir para quem o lê — o descarte de quadro
		// atrasado do hub e o `commitMove`, que recusa confirmar sobre um tabuleiro
		// que mudou desde a proposta.
		//
		// Não é o bump que faz a cortina CHEGAR: o `EmitOrdered` descarta com `Seq <
		// lastSeq`, estritamente menor, então versão repetida PASSA. O contador é o
		// que mantém o número honesto.
		b.Version++
		return nil
	})
	return b, changed, err
}

// board.MoveGroup desloca as peças marcadas pelo mesmo delta.
//
// Uma transação para o grupo inteiro, pelo mesmo motivo do `PaintStroke`: o
// gesto é UM, e uma gravação por peça faria a mesa ver a horda chegar pela
// metade.
func (bs *Store) MoveGroup(
	ctx context.Context, sessionID int64, boardID string, ids []string, dx, dy int,
) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.MoveGroup(b, ids, dx, dy)
	})
}
