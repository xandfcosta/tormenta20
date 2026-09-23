package boards

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"t20engine/app/session"
	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
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
	// bus é por onde as mudanças deste tabuleiro viram notícia.
	//
	// Barramento e não um registro de ouvintes por store: dois stores com travas
	// próprias, cada um chamando o aviso do outro de dentro da trava, é como se
	// escreve um abraço mortal. O barramento é FOLHA — pega só a própria trava e
	// nunca chama de volta um store (ver `events.Bus.Publish`) —, e mesmo assim a
	// publicação sai de fora da trava.
	bus   *events.Bus
	newID func() string
	// snapshots é o RETRATO do tabuleiro no banco, e é ele a fonte da verdade:
	// toda mutação passa por ele antes de existir para alguém (ALE-375).
	snapshots BoardSnapshots
	// q é o caderno de consultas do ACERVO DE LUGARES — o `campaign_places` e o
	// rascunho, que moram em `places.go` e `place_draft.go`.
	//
	// Ele NÃO grava tabuleiro, e é por isso que convive com o retrato acima: o
	// acervo é outra tabela e outro assunto (arquivar uma cena, listar os
	// lugares de uma campanha), e passá-lo pela porta do retrato faria a porta
	// crescer para caber o que ela não descreve.
	q *sqlcgen.Queries
}

func NewStore(snapshots BoardSnapshots, q *sqlcgen.Queries, newID func() string, bus *events.Bus) *Store {
	return &Store{
		bus:       bus,
		boards:    map[int64][]*board.BoardState{},
		loaded:    map[int64]bool{},
		newID:     newID,
		snapshots: snapshots,
		q:         q,
	}
}

// installWhenItCounts põe o tabuleiro mudado na memória — AGORA, ou depois do
// commit quando há transação aberta.
//
// Dentro de uma unidade, gravar não é o fim: a transação ainda pode ser
// desfeita por um passo seguinte, e a memória não participa disso. Instalando
// na hora, o `rollback` desfaz o disco e deixa o mapa que a mesa VÊ adiantado —
// medido na ALE-376, a peça ficava na casa nova depois de o gesto inteiro ser
// recusado.
func (bs *Store) installWhenItCounts(ctx context.Context, alvo, draft *board.BoardState) {
	if open, ok := session.UnitFrom(ctx); ok && open.AfterCommit != nil {
		pronto := *draft
		open.AfterCommit(func() { *alvo = pronto })
		return
	}
	*alvo = *draft
}

// snapshotsFor devolve o retrato que ESTE gesto tem de usar: o da transação
// aberta, quando o contexto carrega um, e o do store no resto das vezes.
//
// O do contexto GANHA por construção, e é o que faz um gesto que atravessa
// tabuleiro e fila gravar os dois na mesma transação (ALE-376).
func (bs *Store) snapshotsFor(ctx context.Context) BoardSnapshots {
	if inTx, ok := ctx.Value(snapshotsKey{}).(BoardSnapshots); ok && inTx != nil {
		return inTx
	}
	return bs.snapshots
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
// ELA DEVOLVE ERRO desde a ALE-375, e o nil deixou de significar duas coisas:
// antes, "o banco não respondeu" e "esta sessão não tem tabuleiro" saíam pelo
// mesmo nil — e o segundo é uma resposta plausível, então a mesa abria vazia
// sobre uma falha de leitura. É a mesma mentira que a leitura da FILA contava
// até a ALE-373. O nil que sobra quer dizer uma coisa só: não há aquele
// tabuleiro.
func (bs *Store) Get(ctx context.Context, sessionID int64, boardID string) (*board.BoardState, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	if err := bs.hydrateLocked(ctx, sessionID); err != nil {
		return nil, err
	}
	return cloneBoard(bs.findLocked(sessionID, boardID)), nil
}

// OpenBoards devolve os tabuleiros da sessão na ordem de abertura — é o que a
// barra de abas desenha.
//
// Cópias, como o `Get`: quem recebe a lista a redige por papel e a serializa,
// e devolver os ponteiros vivos deixaria o `board.BoardForRole` do chamador
// escrevendo no estado da mesa.
func (bs *Store) OpenBoards(ctx context.Context, sessionID int64) ([]*board.BoardState, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	if err := bs.hydrateLocked(ctx, sessionID); err != nil {
		return nil, err
	}
	open := make([]*board.BoardState, 0, len(bs.boards[sessionID]))
	for _, b := range bs.boards[sessionID] {
		open = append(open, cloneBoard(b))
	}
	return open, nil
}

// DefaultBoardID é o id da aba de quem ainda não escolheu: a mais antiga.
//
// A MAIS ANTIGA e não a última aberta: o mestre abre a taverna sob cortina
// enquanto a mesa olha a cripta, e por "a última" a mesa inteira seria puxada
// para uma cortina sem ninguém pedir. Quem move a mesa de propósito é o FORÇAR,
// que é gesto.
func (bs *Store) DefaultBoardID(ctx context.Context, sessionID int64) (string, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	if err := bs.hydrateLocked(ctx, sessionID); err != nil {
		return "", err
	}
	if open := bs.boards[sessionID]; len(open) > 0 {
		return open[0].ID, nil
	}
	return "", nil
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
// O `loaded` só é marcado no SUCESSO, e agora isso é consequência e não
// cuidado: o erro SOBE, então não há caminho em que a sessão seja marcada sobre
// uma leitura que falhou. Marcando antes da query, um erro transiente ficaria
// cacheado como "esta sessão não tem tabuleiro" até o processo reiniciar.
//
// A lista VAZIA marca: "sessão sem tabuleiro" é resposta legítima e definitiva,
// e ela evita uma ida ao disco por mensagem.
func (bs *Store) hydrateLocked(ctx context.Context, sessionID int64) error {
	if bs.loaded[sessionID] {
		return nil
	}
	open, err := bs.snapshotsFor(ctx).Read(ctx, sessionID)
	if err != nil {
		// Sem marcar: a próxima mensagem tenta de novo em vez de servir um
		// "sem tabuleiro" que só existe porque o banco piscou.
		return err
	}
	bs.loaded[sessionID] = true
	if len(open) > 0 {
		bs.boards[sessionID] = open
	}
	return nil
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
	if err := bs.hydrateLocked(ctx, sessionID); err != nil {
		return nil, err
	}
	if len(bs.boards[sessionID]) >= openBoardsCeiling {
		return nil, fmt.Errorf(
			"esta sessão já tem %d tabuleiros abertos (teto %d): feche um antes de abrir outro",
			len(bs.boards[sessionID]), openBoardsCeiling)
	}
	b := board.NewBoard(bs.newID(), place, terrain)
	b.Seq = bs.nextSeqLocked(sessionID)
	// A GRAVAÇÃO VEM ANTES DE A ABA EXISTIR PARA ALGUÉM. Invertida, o mestre vê
	// a aba nova na barra, monta a cena nela, e no próximo boot ela não está lá
	// — porque o INSERT que nunca aconteceu era o que a fazia existir.
	if err := bs.snapshotsFor(ctx).Save(ctx, sessionID, b); err != nil {
		return nil, err
	}
	bs.boards[sessionID] = append(bs.boards[sessionID], b)
	return cloneBoard(b), nil
}

// Close encerra UM tabuleiro: some da memória e do banco. As posições não são
// arquivadas AQUI — quem arquiva é o gateway, antes de chamar.
//
// Devolve as transições de saúde como o `Persist`: se o DELETE falha, a memória
// diz "fechado" e o banco mantém a linha — no próximo boot o tabuleiro FANTASMA
// volta, com as peças de uma cena que a mesa já encerrou.
// O APAGAR VEM ANTES DE A ABA SUMIR DA MEMÓRIA (ALE-375), e a ordem inverteu.
// Ela saía da memória primeiro e o DELETE ia depois, num caminho que podia
// falhar: a aba sumia da tela, a linha ficava no banco, e no boot seguinte o
// tabuleiro FANTASMA voltava com as peças de uma cena que a mesa já encerrou.
// Agora o DELETE que falha RECUSA o gesto, e a aba continua onde estava.
//
// E COM ISSO O `WithoutCancel` SAIU. Ele existia porque a limpeza acontecia
// depois da resposta, e um cliente que fosse embora a levaria junto — "limpeza
// que depende de o cliente esperar não é limpeza". Dentro do gesto não há mais
// o que proteger do cancelamento: o gesto cancelado não fecha a aba, o banco
// não muda, e a mesa continua íntegra.
func (bs *Store) Close(ctx context.Context, sessionID int64, boardID string) error {
	bs.Mu.Lock()
	target := bs.findLocked(sessionID, boardID)
	if target == nil {
		bs.Mu.Unlock()
		return nil
	}
	closed := target.ID
	if err := bs.snapshotsFor(ctx).Delete(ctx, sessionID, closed); err != nil {
		bs.Mu.Unlock()
		return err
	}
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
	return nil
}

// SessionDeleted é o FIM DA VIDA dos tabuleiros de uma sessão: sem esta porta,
// o mapa em memória sobreviveria à sessão que o continha.
//
// Ela não é o `Close`, e a diferença encolheu com a ALE-375 sem sumir. O
// `Close` é o mestre ENCERRANDO uma cena, e ele GRAVA — o DELETE mora dentro do
// gesto, e a falha dele recusa. Aqui a sessão deixou de existir, e com ela
// qualquer motivo para tocar no disco.
//
// Não toca no BANCO: a linha de `open_boards` some por CASCATA quando a sessão
// é apagada (migração 00010). Apagá-la aqui seria a segunda verdade sobre quem
// limpa, e a que roda depois falharia por não achar nada.
//
// > Aqui morava metade de um argumento sobre o `Dirty`, que explicava por que a
// > marca de sujeira saía junto: ela acenderia para sempre quando o `Persist`
// > seguinte batesse na chave estrangeira de uma sessão apagada. A marca não
// > existe mais, e nem o `Persist` — a razão inteira saiu com eles.
func (bs *Store) SessionDeleted(sessionID int64) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	delete(bs.boards, sessionID)
	delete(bs.loaded, sessionID)
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

// A MUDANÇA RODA SOBRE UMA CÓPIA, e é isso que faz a gravação recusada não
// deixar rastro: o tabuleiro vivo só é substituído DEPOIS de o disco aceitar.
// Mutando o vivo direto, uma gravação que falhasse deixaria a mesa vendo uma
// peça que o banco não tem — que é exatamente o estado que o `Dirty` existia
// para avisar, e que esta fatia apaga em vez de avisar.
//
// # Por que aqui NÃO se relê do banco, e na fila sim
//
// O `Mutate` da fila lê o retrato antes de mudá-lo, e o comentário dele diz por
// quê: a função de mudança dela ESCREVE NA FICHA por outras portas, então
// partir do gravado é o que garante que ela parta do mundo real. Aqui a mutação
// é PURA — `domain/board` recebe estado e devolve estado, sem tocar em nada —,
// e a memória é escrita através a cada gesto: ela não tem como divergir do
// disco, porque nenhuma mutação sobrevive a uma gravação recusada. Reler seria
// uma ida ao disco por peça que anda, comprando uma garantia que já existe.
func (bs *Store) applyLocked(
	ctx context.Context, sessionID int64, boardID string, fn func(*board.BoardState) error,
) (*board.BoardState, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	if err := bs.hydrateLocked(ctx, sessionID); err != nil {
		return nil, err
	}
	live := bs.findLocked(sessionID, boardID)
	if live == nil {
		return nil, errNoBoard
	}
	draft := cloneBoard(live)
	if err := fn(draft); err != nil {
		return nil, err
	}
	// A GRAVAÇÃO É A MUTAÇÃO (ALE-375): o que não gravou não aconteceu, e o erro
	// sobe para quem clicou em vez de acender uma tarja que ninguém lê.
	if err := bs.snapshotsFor(ctx).Save(ctx, sessionID, draft); err != nil {
		return nil, err
	}
	bs.installWhenItCounts(ctx, live, draft)
	return cloneBoard(draft), nil
}

// board.AddToken põe a peça no tabuleiro, NA CASA que ela traz. A posição é sempre
// declarada: o gesto de criar peça POSICIONA — o modo liga, o clique numa casa
// diz onde, e a coordenada viaja no caminho.
func (bs *Store) AddToken(ctx context.Context, sessionID int64, boardID string, t board.BoardToken) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.AddToken(b, t, bs.newID)
	})
}

// UnbindTokens desamarra as peças cuja linha da fila não existe mais na fila
// dada. A peça fica; só o vínculo sai (ALE-377).
func (bs *Store) UnbindTokens(
	ctx context.Context, sessionID int64, boardID string, queue *live.SessionRuntimeState,
) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		board.UnbindOrphanTokens(b, queue)
		return nil
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

// Aqui moravam o `Persist`, o `SaveFailed` e a marca `Dirty`, e os três eram a
// mesma coisa dita em três lugares: a gravação saía DEPOIS, numa goroutine, e
// por isso precisava de um jeito de contar à mesa que tinha falhado.
//
// Com a gravação dentro da mutação não há o que avisar — o gesto que o disco
// recusa volta recusado, com a frase, e a mesa continua exatamente como estava
// (ALE-375). A tarja "a mesa não está sendo salva" saiu junto: um aviso que só
// podia aparecer depois do estrago é pior que uma recusa na hora.

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
