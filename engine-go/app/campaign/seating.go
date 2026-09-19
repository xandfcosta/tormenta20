package campaign

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"t20engine/app"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS RECUSAS DE SENTAR À MESA, e elas são EXPORTADAS.
//
// Era o contrário: os sentinelas moravam no `serve/api` e a cena não podia
// lê-los — ler um erro do hospedeiro é alcançar o hospedeiro —, então o
// adaptador os traduzia num enum que a cena declarava só para isso. Com as
// recusas aqui, no `app/`, quem está ACIMA pode lê-las: a tradução do meio
// deixou de existir (ALE-348).
//
// Cada uma embrulha a recusa GENÉRICA do grupo, e as duas camadas servem a
// leitores diferentes: o transporte pergunta "que número é este?" pela
// genérica, e a tela pergunta "que frase eu mostro?" pela específica.
var (
	// ErrNoSuchCampaign é a mesa que não existe.
	ErrNoSuchCampaign = fmt.Errorf("campanha não existe: %w", app.ErrNotFound)
	// ErrNeedsInvite é a mesa fechada: sem o token EXATO ninguém que não seja o
	// dono senta.
	ErrNeedsInvite = fmt.Errorf("convite válido é obrigatório: %w", app.ErrForbidden)
	// ErrNotYourHero junta "o personagem não existe" e "o personagem é de outra
	// pessoa", e a junção é DELIBERADA: distinguir diria a um estranho se um id
	// de personagem existe, e ele não deveria conseguir sondar isso.
	ErrNotYourHero = fmt.Errorf("o herói não é seu: %w", app.ErrForbidden)
	// ErrAlreadyHasHero é quem já tem um herói nesta mesa.
	ErrAlreadyHasHero = fmt.Errorf("já tem personagem nesta campanha: %w", app.ErrRefused)
	// ErrHeroAlreadyThere é o herói que já foi copiado para esta mesa — inclusive
	// quando quem descobre isso é a releitura DENTRO da transação, que perdeu a
	// corrida para um pedido simultâneo.
	ErrHeroAlreadyThere = fmt.Errorf("personagem já está na campanha: %w", app.ErrRefused)
)

// Seating é sentar alguém à mesa: as travas, a cópia e o membro.
type Seating struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

func NewSeating(db *sql.DB, q *sqlcgen.Queries) Seating {
	return Seating{db: db, queries: q}
}

// Seat aplica as travas e faz o instantâneo.
//
// A ORDEM importa: campanha, depois convite, depois personagem. Checar o
// personagem antes do convite diria a um estranho se um id de personagem existe
// — informação que ele não deveria conseguir sondar sem estar convidado.
//
// NÃO recebe `app.Caller`: quem senta senta com o PRÓPRIO herói, e é isso que a
// trava do dono do personagem confere. Um administrador não senta pela cara —
// ele precisaria de um herói dele na mesa como qualquer um.
func (s Seating) Seat(
	ctx context.Context, quemPede, campanhaID, heroiID int64, convite string,
) error {
	c, err := s.queries.GetCampaign(ctx, campanhaID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoSuchCampaign
	}
	if err != nil {
		return fmt.Errorf("carregar a campanha %d: %w", campanhaID, err)
	}
	// O dono entra sem convite; qualquer outra pessoa precisa do token EXATO.
	if c.Ownerid != quemPede {
		if !c.Invitetoken.Valid || convite == "" || convite != c.Invitetoken.String {
			return ErrNeedsInvite
		}
	}

	dono, err := s.queries.GetCharacterOwner(ctx, heroiID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && dono != quemPede) {
		return ErrNotYourHero
	}
	if err != nil {
		return fmt.Errorf("ler o dono do personagem %d: %w", heroiID, err)
	}

	// AS DUAS TRAVAS ABAIXO JÁ FALHARAM ABERTAS uma vez: o erro era descartado
	// com `_`, e erro de banco virava `false`, que significa "pode entrar".
	// Checagem de autorização ou de unicidade NUNCA descarta erro: na dúvida,
	// NEGA.
	temHeroi, err := s.queries.HasPlayerPc(ctx, sqlcgen.HasPlayerPcParams{
		Campaignid: campanhaID, Ownerid: quemPede,
	})
	if err != nil {
		return fmt.Errorf("conferir se %d já tem herói na campanha %d: %w", quemPede, campanhaID, err)
	}
	if temHeroi {
		return ErrAlreadyHasHero
	}

	// Modelo de INSTANTÂNEO: o personagem do elenco é um molde, e a mesa guarda
	// uma CÓPIA dele. A deduplicação é por "este molde já foi copiado aqui" e
	// não por participação do molde — o molde nunca é membro.
	temCopia, err := s.hasCopyOf(ctx, s.db, heroiID, campanhaID)
	if err != nil {
		return fmt.Errorf("conferir a cópia de %d na campanha %d: %w", heroiID, campanhaID, err)
	}
	if temCopia {
		return ErrHeroAlreadyThere
	}
	return s.seat(ctx, heroiID, campanhaID)
}

// seat clona o personagem para a mesa e cria o membro NA MESMA transação.
//
// Em duas escritas separadas, um `CreateMember` que falhasse deixaria a cópia
// ÓRFÃ no banco — e cópia órfã é pior que nada, porque a deduplicação passa a
// responder "já está na mesa" e o herói fica impedido de entrar PARA SEMPRE,
// sem membro nenhum que se possa remover para desfazer (ALE-156).
//
// # A CHECAGEM É REFEITA aqui dentro, e é isto que fecha a corrida
//
// A de fora existe para a mensagem amigável e para o caminho rápido; ela roda
// SEM transação, então dois pedidos simultâneos passam os dois por ela. Com o
// `_txlock=immediate`, a transação já nasce com a trava de escrita, então o
// segundo pedido ESPERA o primeiro terminar — e esta releitura enxerga o que
// ele gravou. Repetição de propósito: a de fora é pela MENSAGEM, esta é pela
// VERDADE.
//
// É a mesma forma do commit de movimento no tabuleiro: entre decidir e
// escrever, a mesa pode ter mudado.
func (s Seating) seat(ctx context.Context, heroiID, campanhaID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir a transação de sentar à mesa: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	temCopia, err := s.hasCopyOf(ctx, tx, heroiID, campanhaID)
	if err != nil {
		return fmt.Errorf("reconferir a cópia de %d na campanha %d: %w", heroiID, campanhaID, err)
	}
	if temCopia {
		return ErrHeroAlreadyThere
	}

	copiaID, err := cloneCharacterTx(ctx, tx, heroiID, campanhaID)
	if err != nil {
		return err
	}
	if _, err := s.queries.WithTx(tx).CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: campanhaID, Characterid: copiaID, Addedat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("gravar o membro da campanha %d: %w", campanhaID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fechar a transação de sentar à mesa: %w", err)
	}
	return nil
}

// consultor é o que o `hasCopyOf` precisa, e ele existe para a MESMA pergunta
// ser feita por fora e por dentro da transação.
//
// Duas funções com o mesmo `SELECT` divergiriam na primeira vez que a
// deduplicação mudasse de forma — e as duas respondem quem entra na mesa.
type consultor interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// hasCopyOf diz se este molde já foi copiado para esta mesa.
func (s Seating) hasCopyOf(ctx context.Context, q consultor, heroiID, campanhaID int64) (bool, error) {
	var existe bool
	err := q.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM characters WHERE sourceCharacterId = ? AND campaignId = ?)`,
		heroiID, campanhaID).Scan(&existe)
	return existe, err
}
