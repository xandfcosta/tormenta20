package boards

import (
	"context"
	"errors"
	"fmt"
	"sync"

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
// limite e TODA leitura e TODA gravação o carregam. Oito frentes é uma cena
// que nenhuma mesa joga.
const openBoardsCeiling = 8

// Store é o tabuleiro vivo de cada sessão: ele mora na tabela open_boards, e se lê e se grava SEMPRE
// no banco (decisão do dono). Aqui morava uma cópia em memória, um mapa por sessão, e ela saiu pela
// razão da cópia da mesa: o gancho de depois do commit a escrevia FORA da trava quando um gesto abria
// a unidade direto, e a cena reaberta do acervo vivia só nela até o primeiro gesto gravá-la.
//
// Mutex PRÓPRIO, separado do `session.Store`: lá o mutex é global a todas as
// sessões, e um tabuleiro movimentado numa mesa serializaria a edição de PV de
// outra mesa. Aqui a mesma trava vale para todos os tabuleiros — quando o custo
// aparecer, ela vira uma por sessão sem mudar quem chama.
type Store struct {
	Mu sync.Mutex
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
		newID:     newID,
		snapshots: snapshots,
		q:         q,
	}
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

// Get devolve UM tabuleiro da sessão (nil quando não há), lido do banco.
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
	open, err := bs.snapshotsFor(ctx).Read(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return FindBoard(open, boardID), nil
}

// OpenBoards devolve os tabuleiros da sessão na ordem de abertura — é o que a
// barra de abas desenha.
//
// Cópias, como o `Get`: quem recebe a lista a redige por papel e a serializa,
// e devolver os ponteiros vivos deixaria o `board.BoardForRole` do chamador
// escrevendo no estado da mesa.
func (bs *Store) OpenBoards(ctx context.Context, sessionID int64) ([]*board.BoardState, error) {
	return bs.snapshotsFor(ctx).Read(ctx, sessionID)
}

// DefaultBoardID é o id da aba de quem ainda não escolheu: a mais antiga.
//
// A MAIS ANTIGA e não a última aberta: o mestre abre a taverna sob cortina
// enquanto a mesa olha a cripta, e por "a última" a mesa inteira seria puxada
// para uma cortina sem ninguém pedir. Quem move a mesa de propósito é o FORÇAR,
// que é gesto.
func (bs *Store) DefaultBoardID(ctx context.Context, sessionID int64) (string, error) {
	ids, err := bs.OpenBoardIDs(ctx, sessionID)
	if err != nil || len(ids) == 0 {
		return "", err
	}
	return ids[0], nil
}

// OpenBoardIDs são os ids dos tabuleiros abertos, na ordem das abas, sem ler os tabuleiros: é o que a
// escolha da aba e a aba padrão perguntam, e a pergunta vem a cada gesto e a cada desenho.
func (bs *Store) OpenBoardIDs(ctx context.Context, sessionID int64) ([]string, error) {
	ids, err := bs.q.ListOpenBoardIDs(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ler as abas da sessão %d: %w", sessionID, err)
	}
	return ids, nil
}

// nextSeqLocked é o número da PRÓXIMA aba desta sessão.
//
// `max + 1` e não `len + 1`: fechar a aba do meio deixaria dois tabuleiros com
// o mesmo número, e dois números iguais é o empate que esta coluna existe para
// não ter. Buraco na sequência não custa nada — ela só serve para ordenar.
func nextSeq(open []*board.BoardState) int64 {
	var max int64
	for _, b := range open {
		if b.Seq > max {
			max = b.Seq
		}
	}
	return max + 1
}

// FindBoard acha o tabuleiro na lista dos abertos: o do id, ou a aba PADRÃO (a primeira) quando o id
// vem vazio; nil quando não há. Quem já leu a lista escolhe nela, sem ler o banco de novo.
func FindBoard(open []*board.BoardState, boardID string) *board.BoardState {
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
	open, err := bs.snapshotsFor(ctx).Read(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(open) >= openBoardsCeiling {
		return nil, fmt.Errorf(
			"esta sessão já tem %d tabuleiros abertos (teto %d): feche um antes de abrir outro",
			len(open), openBoardsCeiling)
	}
	b := board.NewBoard(bs.newID(), place, terrain)
	b.Seq = nextSeq(open)
	if err := bs.snapshotsFor(ctx).Save(ctx, sessionID, b); err != nil {
		return nil, err
	}
	return b, nil
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

// applyLocked lê o tabuleiro do banco, deixa a mutação pura mudar a cópia lida e grava: o que o disco
// recusa não chega a existir para ninguém (ALE-375). A trava guarda a ORDEM entre dois gestos no
// mesmo tabuleiro — sem ela, os dois liam a mesma versão e o segundo apagava o primeiro.
func (bs *Store) applyLocked(
	ctx context.Context, sessionID int64, boardID string, fn func(*board.BoardState) error,
) (*board.BoardState, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	open, err := bs.snapshotsFor(ctx).Read(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	draft := FindBoard(open, boardID)
	if draft == nil {
		return nil, errNoBoard
	}
	if err := fn(draft); err != nil {
		return nil, err
	}
	// A GRAVAÇÃO É A MUTAÇÃO (ALE-375): o que não gravou não aconteceu, e o erro
	// sobe para quem clicou em vez de acender uma tarja que ninguém lê.
	if err := bs.snapshotsFor(ctx).Save(ctx, sessionID, draft); err != nil {
		return nil, err
	}
	return draft, nil
}

// board.AddToken põe a peça no tabuleiro, NA CASA que ela traz. A posição é sempre
// declarada: o gesto de criar peça POSICIONA — o modo liga, o clique numa casa
// diz onde, e a coordenada viaja no caminho.
func (bs *Store) AddToken(ctx context.Context, sessionID int64, boardID string, t board.BoardToken) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.AddToken(b, t, bs.newID)
	})
}

// DamageObject tira PV de uma peça de cenário (p239).
//
// Ela existe porque o dano de um ataque a OBJETO não tem para onde ir no regime:
// o `DeltaVitals` escreve na LINHA da fila, e um objeto não tem linha. Quem
// escolhe entre as duas é a CENA, acima dos dois stores — cada um tem trava
// própria, e um chamando o outro de dentro da sua é como se escreve um abraço
// mortal (ver o comentário do `bus` acima).
func (bs *Store) DamageObject(
	ctx context.Context, sessionID int64, boardID, tokenID string, amount int,
) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		return board.DamageObject(b, tokenID, amount)
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
func (bs *Store) SetSpeeds(ctx context.Context, sessionID int64, boardID string, speeds map[string]board.TokenMovement) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
		for i := range b.Tokens {
			if movement, ok := speeds[b.Tokens[i].ID]; ok && movement.Squares > 0 {
				b.Tokens[i].SpeedSquares = movement.Squares
				b.Tokens[i].IgnoresDifficult, b.Tokens[i].OtherModes = movement.IgnoresDifficult, movement.Modes
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

// board.ProposeMoveWithStops, board.CommitMove e board.CancelMove são as três portas do movimento.
// A posse chega RESOLVIDA do gateway: quem consulta o banco é ele, e a trava daqui não pode esperar
// por I/O.
//
// O ORÇAMENTO JÁ ESTÁ NA PEÇA: ela mede a ficha ao entrar no mapa, e o aviso de que a ficha mudou a
// mede de novo (`RemeasureCharacter`). Houve aqui um parâmetro de "orçamento fresco" que regravava a
// peça antes de medir, e o único chamador passava zero — um segundo lugar escrevendo a mesma cópia,
// morto, e com um comentário dizendo que a armadura vestida valia por ele.

// ProposeMoveWithStops é a porta de quem monta o movimento CLICANDO, e ela existe para a lista de
// paradas ser guardada junto: sem a memória de ONDE a pessoa parou, que o caminho sozinho não deixa
// reconstruir, "desfazer a última perna" seria um palpite sobre o movimento que a mesa está vendo.
func (bs *Store) ProposeMoveWithStops(ctx context.Context, sessionID int64, boardID string, st *live.SessionRuntimeState, tokenID string, stops []engine.Square, by board.Mover) (*board.BoardState, error) {
	return bs.apply(ctx, sessionID, boardID, func(b *board.BoardState) error {
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
