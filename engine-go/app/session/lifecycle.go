package session

import (
	"context"
	"database/sql"
	"fmt"

	"t20engine/app"
	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// Lifecycle é o CICLO DE VIDA de uma sessão, do pedido à gravação.
//
// Cada método dele faz a mesma sequência, e é ela que define a camada: AUTORIZA,
// pergunta a DECISÃO à regra pura, GRAVA, e acerta o estado em memória. A cena
// recebe o resultado e desenha; ela não sabe em que ordem nada disso acontece.
// BoardsOfASession é o que o CICLO precisa do tabuleiro, e é UM método: esquecer
// os mapas de uma sessão que deixou de existir.
//
// Uma PORTA e não o `*boards.Store` inteiro, e a razão é de direção (ALE-376):
// com o store, este pacote importava `app/boards`, e isso proibia o caminho
// contrário — justo o que o gesto que atravessa tabuleiro e fila precisa. Pedir
// a PERGUNTA em vez do objeto desfez o impedimento sem custar nada: era uma
// chamada só.
type BoardsOfASession interface {
	SessionDeleted(sessionID int64)
}

type Lifecycle struct {
	db       *sql.DB
	queries  *sqlcgen.Queries
	sessions *Store
	boards   BoardsOfASession
	access   Access
}

func NewLifecycle(
	db *sql.DB, q *sqlcgen.Queries, sessions *Store, boards BoardsOfASession,
) Lifecycle {
	return Lifecycle{db: db, queries: q, sessions: sessions, boards: boards, access: NewAccess(q)}
}

// Access é a trava que este caso de uso usa, exposta para quem precisa só dela.
func (l Lifecycle) Access() Access { return l.access }

// SetStatus leva a sessão ao status pedido — `live.StatusActive` ou
// `live.StatusEnded`.
//
// UM método e não `Start`/`End` separados porque a DECISÃO é uma só e mora numa
// função só (`live.ChangeToStatus`): "iniciar" numa sessão encerrada REABRE, e
// duas portas para a mesma decisão são duas chances de uma delas esquecer esse
// caso.
func (l Lifecycle) SetStatus(
	ctx context.Context, who app.Caller, campaignID, sessionID int64, requested string,
) (*live.SessionRuntimeState, error) {
	sess, err := l.access.GM(ctx, who, campaignID, sessionID)
	if err != nil {
		return nil, err
	}
	change, err := live.ChangeToStatus(sess.Status, requested)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, app.ErrRefused)
	}
	now := dbvalue.NowISO()
	switch change {
	case live.StatusUnchanged:
		// Clicar duas vezes não escreve nada, e não é erro. O estado volta
		// mesmo assim: quem clicou merece ver a tela que já era a certa.
	case live.StatusStartsFresh:
		_, err = l.queries.StartSessionFresh(ctx, sqlcgen.StartSessionFreshParams{
			StartedAt: sql.NullString{String: now, Valid: true}, UpdatedAt: now, ID: sessionID,
		})
	case live.StatusReopens:
		_, err = l.queries.ReopenSession(ctx, sqlcgen.ReopenSessionParams{
			UpdatedAt: now, ID: sessionID,
		})
	case live.StatusEnds:
		_, err = l.queries.EndSession(ctx, sqlcgen.EndSessionParams{
			EndedAt: sql.NullString{String: now, Valid: true}, UpdatedAt: now, ID: sessionID,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("gravar a sessão %d como %q: %w", sessionID, requested, err)
	}
	return l.sessions.State(ctx, sessionID)
}

// Rename troca o título, e o VAZIO é legítimo: a sessão tem NÚMERO, que é a
// identidade dela, e o título é o apelido da noite.
//
// # Por que um `UPDATE` escrito à mão, e não uma consulta gerada
//
// A coluna `title` não tem query no sqlc, e esta camada É onde uma escrita sem
// consulta gerada pode morar — foi por não haver este lugar que ela vivia no
// `serve/api`, ao lado do middleware. O conserto definitivo é escrever a
// consulta no `query.sql` e regerar; até lá, o SQL está aqui, inteiro e visível,
// em vez de montado por um construtor genérico.
func (l Lifecycle) Rename(
	ctx context.Context, who app.Caller, campaignID, sessionID int64, title string,
) error {
	if _, err := l.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return err
	}
	var value any
	if title != "" {
		value = title
	}
	if _, err := l.db.ExecContext(ctx,
		"UPDATE sessions SET title = ?, updatedAt = ? WHERE id = ?",
		value, dbvalue.NowISO(), sessionID,
	); err != nil {
		return fmt.Errorf("gravar o título da sessão %d: %w", sessionID, err)
	}
	return nil
}

// SaveNotes grava as notas do mestre, e ela NÃO apara o texto.
//
// A diferença para o título é a que importa: aparar a cada 1,2s comeria a linha
// em branco que o mestre acabou de abrir para escrever o próximo parágrafo.
// Quem salva UMA vez, ao fechar, apara; este salva no meio da digitação. Vazio
// continua virando NULL.
//
// O `UPDATE` é escrito à mão pela mesma razão do título — a coluna não tem
// consulta no sqlc, e esta camada é onde uma escrita sem consulta gerada pode
// morar.
func (l Lifecycle) SaveNotes(
	ctx context.Context, who app.Caller, campaignID, sessionID int64, text string,
) error {
	if _, err := l.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return err
	}
	var value any
	if text != "" {
		value = text
	}
	if _, err := l.db.ExecContext(ctx,
		"UPDATE sessions SET notes = ?, updatedAt = ? WHERE id = ?",
		value, dbvalue.NowISO(), sessionID,
	); err != nil {
		return fmt.Errorf("gravar as notas da sessão %d: %w", sessionID, err)
	}
	return nil
}

// RestartCombat esvazia a fila e os turnos SEM tirar a partida do ar.
//
// ESQUECER O CACHE é metade do gesto, e a metade que não aparece: a fila mora em
// MEMÓRIA e o banco só é lido na primeira carga, então escrever a linha limpa
// sem derrubar o cache deixa a sessão servindo a fila velha — o reinício
// "funciona" e nada muda na tela. Um guarda que medisse o BANCO não pega isto:
// ele já está vazio antes do reinício, porque a fila nunca chegou lá.
//
// E a releitura depois do `Forget` também é obrigatória: sem ela o `State`
// devolve a cópia em memória sem passar pelo banco, e a próxima carga fria
// discordaria desta.
func (l Lifecycle) RestartCombat(
	ctx context.Context, who app.Caller, campaignID, sessionID int64,
) (*live.SessionRuntimeState, error) {
	if _, err := l.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return nil, err
	}
	if err := l.queries.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: live.EmptyTrackerJSON, UpdatedAt: dbvalue.NowISO(), ID: sessionID,
	}); err != nil {
		return nil, fmt.Errorf("limpar a fila da sessão %d: %w", sessionID, err)
	}
	l.sessions.Forget(sessionID)
	state, err := l.sessions.State(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("reler a fila da sessão %d: %w", sessionID, err)
	}
	return state, nil
}

// Delete apaga a sessão e TUDO que ela deixou em memória.
//
// A ORDEM importa numa direção só: avisar os stores ANTES deixaria uma janela em
// que a sessão ainda responde e o estado em memória já não existe — uma
// requisição nesse instante recriaria o que se acabou de apagar.
//
// São DOIS esquecimentos e não um. Sem o primeiro, a fila em cache continua
// respondendo por uma sessão que não existe mais; sem o segundo, o tabuleiro
// fica no mapa do `BoardStore` batendo na chave estrangeira a cada gravação, e
// a marca de gravação falhando não sai mais — só um `Persist` bem sucedido a
// apaga, e nenhum vai suceder.
//
// O banco limpa o resto sozinho: `open_boards` sai por CASCATA com a sessão
// (migração 00010), e a fila mora na própria linha dela.
func (l Lifecycle) Delete(ctx context.Context, who app.Caller, campaignID, sessionID int64) error {
	if _, err := l.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return err
	}
	if err := l.queries.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("apagar a sessão %d: %w", sessionID, err)
	}
	l.boards.SessionDeleted(sessionID)
	l.sessions.SessionDeleted(sessionID)
	return nil
}
