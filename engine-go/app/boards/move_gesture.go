package boards

import (
	"context"

	"t20engine/app/session"
	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// OS GESTOS QUE ATRAVESSAM O TABULEIRO E A FILA (ALE-376).
//
// Cada store é tudo-ou-nada sozinho desde a ALE-373 e a ALE-375. O que faltava
// é o gesto que escreve nos DOIS: ele gravava em duas transações, e a segunda
// podia falhar com a primeira já feita.
//
// # Por que eles moram aqui, e não na cena
//
// Porque a TRANSAÇÃO é o contorno do gesto, e desenhar esse contorno é decidir
// o que o gesto É — trabalho de caso de uso, não de quem sabe o que é um
// `http.ResponseWriter`. O `TestNoPresentationLayerOpensATransaction` cobra
// isso, e a cena passou a chamar estes métodos em vez de encadear os dois
// stores.
//
// # Por que em `app/boards` e não num pacote novo
//
// Porque o destino de uma função é a DEPENDÊNCIA dela, e a parte cara destes
// gestos é a mutação do tabuleiro — o store, a trava e o retrato estão todos
// aqui. A fila entra por `app/session`, que este pacote passa a importar; o
// caminho contrário não existe, então não há ciclo.

// Gestures são os gestos de tabuleiro que também escrevem na fila.
type Gestures struct {
	boards   *Store
	sessions *session.Store
	units    session.Units
	// access é a TRAVA da sessão. Ela entra porque o `RestartCombat` é gesto do
	// MESTRE e a conferência é do caso de uso — a cena decide o que desenhar, e
	// quem decide se o gesto pode é quem o executa.
	access session.Access
}

func NewGestures(
	boards *Store, sessions *session.Store, units session.Units, access session.Access,
) Gestures {
	return Gestures{boards: boards, sessions: sessions, units: units, access: access}
}

// ConfirmMove faz a peça pousar E cobra a ação do turno, na MESMA transação.
//
// # O que estava errado
//
// As duas gravações saíam em transações diferentes, nesta ordem: a peça pousava
// e só então a ação era cobrada. Uma falha na cobrança deixava **o movimento
// feito e o turno intacto** — o combatente andou de graça, e ninguém na mesa
// tem como notar, porque a tela mostra exatamente o que se espera de um
// movimento que deu certo.
//
// # A ORDEM interna não mudou, e ela é regra do livro
//
// A CONFERÊNCIA vem antes do pouso e a COBRANÇA depois. Cobrar antes basta para
// o número ficar certo e não basta para a mesa: entre a conferência e a cobrança
// o `CommitMove` ainda pode recusar — e aí o turno não é cobrado, que é o lado
// seguro dos dois. O que a transação acrescenta é a outra ponta: agora a
// cobrança que falha DESFAZ o pouso.
//
// QUEM ANDA É QUEM ESTÁ NA VEZ, e só dele se cobra: o mestre move peça fora de
// turno o tempo todo — arrumando a cena, empurrando um NPC —, e cobrar dele a
// ação de outro combatente tiraria do turno de quem não se mexeu (p233).
func (g Gestures) ConfirmMove(
	ctx context.Context, sessionID int64, boardID string, who board.Mover,
) (*board.BoardState, error) {
	var landed *board.BoardState
	err := g.units.Do(ctx, func(u session.Unit) error {
		inUnit := g.inUnit(ctx, u)
		state, err := g.sessions.State(inUnit, sessionID)
		if err != nil {
			return err
		}
		moved, err := g.boards.Get(inUnit, sessionID, boardID)
		if err != nil {
			return err
		}
		onTurn := board.MovedTokenIsOnTurn(state, moved)
		if onTurn {
			if err := g.sessions.ActionFits(inUnit, sessionID, engine.ActionMovement); err != nil {
				return err
			}
		}
		// Versão ZERO: o `CommitMove` só compara quando ela é positiva, e quem
		// confirma acabou de ver a cena que o servidor desenhou. A trava contra
		// a mesa ter mudado é a REVALIDAÇÃO da vez, que o `CommitMove` refaz.
		landed, err = g.boards.CommitMove(inUnit, sessionID, boardID, state, 0, who)
		if err != nil || !onTurn {
			return err
		}
		_, err = g.sessions.SpendAction(inUnit, sessionID, engine.ActionMovement)
		return err
	})
	if err != nil {
		return nil, err
	}
	return landed, nil
}

// inUnit é o contexto que carrega a transação para os DOIS donos de dado.
//
// São duas marcas porque são dois mecanismos, e cada um já existia: o
// `session.WithUnit` faz um `Do` aninhado REUSAR esta transação em vez de pedir
// a segunda conexão (ALE-374), e o `WithSnapshots` faz o store do tabuleiro
// gravar por ela em vez de pelo retrato que ele guardou no construtor.
//
// Sem a segunda, a peça andaria por fora da transação — e o gesto pareceria
// atômico sendo exatamente o que ele era antes.
func (g Gestures) inUnit(ctx context.Context, u session.Unit) context.Context {
	return WithSnapshots(session.WithUnit(ctx, u), NewSnapshots(u.InTx))
}
