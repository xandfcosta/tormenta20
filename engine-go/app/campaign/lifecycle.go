package campaign

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"t20engine/app"
	"t20engine/app/session"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/secret"
)

// Lifecycle é o CICLO DE VIDA de uma campanha: abrir, renomear, cunhar o link
// de convite, escolher quais regras opcionais valem e APAGAR.
//
// Cada método faz a mesma sequência, e é ela que define a camada: AUTORIZA,
// decide, grava. A trava é a MESMA do ciclo da sessão (`session.Access`), e não
// uma segunda — ver o `doc.go` do pacote.
//
// Irmão do `session.Lifecycle` de propósito, e com os mesmos verbos: uma
// campanha e uma sessão são coisas diferentes com o mesmo formato de gesto, e
// dois nomes para isso fariam quem lê procurar a diferença onde não há.
// SessionsForgotten é o que apagar uma CAMPANHA precisa do ciclo da sessão, e é
// UM método: tirar da memória o que uma sessão que deixou de existir deixou.
//
// Aqui moravam os dois stores — o da fila e o do tabuleiro —, e eles existiam
// para as DUAS linhas que este pacote escrevia à mão. Pedir o gesto em vez dos
// objetos fez a sequência morar num lugar só e tirou a dependência deste pacote
// em `app/boards`, que era de uma chamada (ALE-377).
type SessionsForgotten interface {
	ForgetSession(sessionID int64)
}

type Lifecycle struct {
	db       *sql.DB
	queries  *sqlcgen.Queries
	access   session.Access
	sessions SessionsForgotten
}

func NewLifecycle(
	db *sql.DB, q *sqlcgen.Queries, lock session.Access, sessions SessionsForgotten,
) Lifecycle {
	return Lifecycle{db: db, queries: q, access: lock, sessions: sessions}
}

// Open abre uma mesa, e ela nasce COM link de convite.
//
// O `CreateCampaign` gerado pelo sqlc NÃO escreve o `inviteToken`, e uma mesa
// com a coluna nula não aceita ninguém: o `Seat` recusa já no
// `!c.Invitetoken.Valid`, antes de olhar o que a pessoa digitou.
//
// Cunhar AQUI e não no `INSERT` é o que garante que os dois caminhos passem por
// isto — a cena de campanhas e a rota JSON que o e2e usa como fixture. É um
// `UPDATE` logo depois do `INSERT` e não uma coluna com `DEFAULT` porque o
// token é aleatório de verdade (`crypto/rand`), e o SQLite não tem de onde
// tirar isso.
//
// NÃO é transação: o pior caso é uma mesa sem link, e ele tem conserto pela
// tela (`RotateInvite`). Abrir mesa não AUTORIZA nada — quem está autenticado
// abre a própria.
func (l Lifecycle) Open(
	ctx context.Context, owner int64, name, description string,
) (int64, error) {
	now := dbvalue.NowISO()
	c, err := l.queries.CreateCampaign(ctx, sqlcgen.CreateCampaignParams{
		Ownerid: owner, Name: name, Description: textOrNull(description),
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		return 0, fmt.Errorf("abrir a campanha %q de %d: %w", name, owner, err)
	}
	if _, err := l.mintInvite(ctx, c.ID); err != nil {
		return 0, err
	}
	return c.ID, nil
}

// Rename grava o nome e a descrição.
//
// # Por que um `UPDATE` escrito à mão, e não uma consulta gerada
//
// Estas duas colunas não têm query no sqlc, e esta camada É onde uma escrita
// sem consulta gerada pode morar — foi por não haver este lugar que ela vivia
// no `serve/api`, montada por um construtor genérico de `SET`. Com esta fatia
// aquele construtor perde o ÚLTIMO chamador. O conserto definitivo é escrever a
// consulta no `query.sql` e regerar; até lá o SQL está aqui, inteiro e visível
// — a mesma decisão que o `session.Rename` e o `character.SaveCustomItem`
// tomaram.
func (l Lifecycle) Rename(
	ctx context.Context, who app.Caller, campaignID int64, name, description string,
) error {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return err
	}
	if _, err := l.db.ExecContext(ctx,
		"UPDATE campaigns SET name = ?, description = ?, updatedAt = ? WHERE id = ?",
		name, nullOrText(description), dbvalue.NowISO(), campaignID,
	); err != nil {
		return fmt.Errorf("gravar o texto da campanha %d: %w", campaignID, err)
	}
	return nil
}

// RotateInvite cunha um link novo e INVALIDA o anterior.
//
// É o mesmo gesto para três coisas, e é por isso que ele tem um nome só: a mesa
// que nasce, a mesa antiga que nunca teve link, e o mestre que quer cortar quem
// já tem o link na mão.
//
// Cunhar link é DAR ACESSO à mesa, então a trava é a do dono e não a de membro.
func (l Lifecycle) RotateInvite(
	ctx context.Context, who app.Caller, campaignID int64,
) (string, error) {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return "", err
	}
	return l.mintInvite(ctx, campaignID)
}

// mintInvite é a cunhagem sem trava, para o nascimento — que não tem dono a
// conferir porque acabou de escolher um.
func (l Lifecycle) mintInvite(ctx context.Context, campaignID int64) (string, error) {
	token, err := secret.Token()
	if err != nil {
		return "", err
	}
	if _, err := l.queries.SetInviteToken(ctx, sqlcgen.SetInviteTokenParams{
		InviteToken: sql.NullString{String: token, Valid: true},
		UpdatedAt:   dbvalue.NowISO(), ID: campaignID,
	}); err != nil {
		return "", fmt.Errorf("cunhar o convite da campanha %d: %w", campaignID, err)
	}
	return token, nil
}

// InviteOf é o link de uma mesa, ou "" quando ela não tem um.
//
// Vazio é estado NORMAL e não erro — campanhas antigas nasceram sem link, e o
// que a tela faz com isso é oferecer o botão de gerar.
func (l Lifecycle) InviteOf(ctx context.Context, campaignID int64) string {
	c, err := l.queries.GetCampaign(ctx, campaignID)
	if err != nil || !c.Invitetoken.Valid {
		return ""
	}
	return c.Invitetoken.String
}

// IgnoredRules são as regras opcionais que o mestre DESLIGOU nesta campanha.
//
// Devolve fatia vazia e nunca nula: `null` e `[]` chegam diferentes no JSON e
// quem lê teria de tratar os dois.
func (l Lifecycle) IgnoredRules(ctx context.Context, campaignID int64) []string {
	ignored, err := l.queries.ListIgnoredRulesForCampaign(ctx, campaignID)
	if err != nil || ignored == nil {
		return []string{}
	}
	return ignored
}

// SaveIgnoredRules troca o conjunto INTEIRO: limpa e reinsere.
//
// Substituição e não delta de propósito — o conjunto é pequeno e fechado, e
// mandar o estado final faz a operação ser idempotente. Um delta reenviado
// alternaria a regra duas vezes, que é exatamente o que um clique repetido numa
// conexão ruim produz.
func (l Lifecycle) SaveIgnoredRules(
	ctx context.Context, who app.Caller, campaignID int64, rules []string,
) error {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return err
	}
	if err := l.queries.ClearIgnoredRulesForCampaign(ctx, campaignID); err != nil {
		return fmt.Errorf("limpar as regras ignoradas da campanha %d: %w", campaignID, err)
	}
	now := dbvalue.NowISO()
	for _, rule := range rules {
		if err := l.queries.IgnoreRuleInCampaign(ctx, sqlcgen.IgnoreRuleInCampaignParams{
			Campaignid: campaignID, Rule: rule, Updatedat: now,
		}); err != nil {
			return fmt.Errorf("desligar a regra %q na campanha %d: %w", rule, campaignID, err)
		}
	}
	return nil
}

// textOrNull traduz o vazio da tela para o NULO do banco.
//
// Os dois querem dizer "sem descrição", e a diferença importa numa direção só:
// gravar string vazia faria a coluna distinguir "não escreveu" de "apagou o que
// tinha", e a tela não oferece essa diferença a ninguém.
func textOrNull(text string) sql.NullString {
	if t := strings.TrimSpace(text); t != "" {
		return sql.NullString{String: t, Valid: true}
	}
	return sql.NullString{}
}

// nullOrText é o mesmo para um `ExecContext`, que quer `nil` e não um
// `sql.NullString` inválido.
func nullOrText(text string) any {
	if ns := textOrNull(text); ns.Valid {
		return ns.String
	}
	return nil
}

// Delete apaga a campanha, e ESQUECE o estado em memória das sessões antes.
//
// # A ordem É o gesto
//
// Apagar a campanha leva as sessões por CASCATA, e depois disso não há mais como
// perguntar quais eram. Invertida, a lista volta vazia, ninguém é esquecido, e o
// tabuleiro de cada sessão fica no mapa em memória batendo numa chave
// estrangeira que não existe: a mesa se declara suja para sempre.
//
// Ela morava num handler, entre dois comentários que a explicavam — e uma
// invariante de sequência que depende de quem chama lembrar dela é uma
// invariante que se perde na segunda vez. Aqui ela não tem como ser esquecida
// (ALE-359), e o `TestDeletingACampaignForgetsItsSessionsFirst` a prende.
//
// # Esquecer é MELHOR ESFORÇO, apagar não é
//
// Se a listagem das sessões falhar, o esquecimento não acontece e a linha é
// apagada assim mesmo: o usuário pediu para apagar, e recusar por causa de um
// mapa em memória seria trocar o gesto dele por um detalhe de processo. O preço
// é estado obsoleto até o reinício, e ele está dito no log.
func (l Lifecycle) Delete(ctx context.Context, who app.Caller, campaignID int64) error {
	if _, err := l.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return err
	}
	l.forgetSessionsInMemory(ctx, campaignID)
	if err := l.queries.DeleteCampaign(ctx, campaignID); err != nil {
		return fmt.Errorf("apagar a campanha %d: %w", campaignID, err)
	}
	return nil
}

// forgetSessionsInMemory tira do mapa o estado vivo de toda sessão da campanha.
func (l Lifecycle) forgetSessionsInMemory(ctx context.Context, campaignID int64) {
	sessions, err := l.queries.ListSessions(ctx, campaignID)
	if err != nil {
		log.Printf("campanha %d: não deu para listar as sessões antes de apagar (%v); "+
			"o estado em memória delas fica até o reinício", campaignID, err)
		return
	}
	for _, sess := range sessions {
		l.sessions.ForgetSession(sess.ID)
	}
}
