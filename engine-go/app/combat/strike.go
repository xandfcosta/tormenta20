// Package combat é o gesto de ATACAR: quem pode, o que a regra decide, e o que
// fica guardado para a mesa ver.
//
// Ele existe porque o ataque atravessa três contextos que não se conhecem — a
// FICHA diz com o que se ataca e o quanto se defende, o LIVRO resolve o d20
// contra a Defesa, e o REGIME guarda o provisório na fila. Nenhum dos três pode
// importar os outros dois, e juntá-los é exatamente o trabalho desta camada.
package combat

import (
	"context"
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// Combatant é o que o gesto precisa saber de quem está na mesa, e só isso.
//
// Ele é achatado de propósito: por trás de uma linha da fila pode haver uma
// ficha de personagem, um verbete do bestiário ou um bloco que o mestre
// escreveu, e as três respondem as mesmas quatro perguntas. Quem sabe de qual
// se trata é o adaptador da porta, não o gesto.
type Combatant struct {
	EntryID string
	Label   string
	// Weapons são as armas empunhadas, pelo motor. VAZIO para quem não tem
	// ficha — e é por isso que esta fatia só deixa PERSONAGEM atacar: o verbete
	// do bestiário traz o ataque como texto ("crítico 19"), e texto não resolve.
	Weapons         []engine.WeaponCard
	Defense         int
	DamageReduction int
	CritImmune      bool
}

// Combatants é a porta que traduz uma linha da fila em combatente.
type Combatants interface {
	Of(ctx context.Context, entry live.InitiativeEntry) (Combatant, error)
}

// Tables é a porta do estado da mesa.
type Tables interface {
	GetState(sessionID int64) *live.SessionRuntimeState
	ProposeAttack(sessionID int64, ataque live.PendingAttack) (*live.SessionRuntimeState, error)
}

// Strike resolve e propõe ataques.
type Strike struct {
	combatentes Combatants
	mesas       Tables
	rolar       func(faces int) (int, error)
}

func NewStrike(c Combatants, t Tables, rolar func(faces int) (int, error)) Strike {
	return Strike{combatentes: c, mesas: t, rolar: rolar}
}

// Request é o pedido de ataque.
type Request struct {
	SessionID       int64
	AttackerEntryID string
	TargetEntryID   string
	// Weapon é qual das armas empunhadas, por índice. Zero é a primeira, que é
	// o caso de quase toda ficha.
	Weapon int
	// D20 é a rolagem QUE JÁ ACONTECEU na mesa, quando aconteceu.
	//
	// Nulo é o servidor rolar. Os dois caminhos existem porque as duas mesas
	// existem, e este repositório já tem os dois precedentes escritos: a
	// INICIATIVA recebe o d20 do jogador (ele rolou um dado de verdade e
	// digitou), e a bolsa inicial rola no servidor. Aceitar nulo é o que
	// permite o gesto de menu — que não tem onde digitar — sem fechar a porta
	// para a mesa que rola dado na mão.
	D20 *int
}

// Propose rola o ataque e guarda o provisório. Ninguém perde PV aqui: quem
// confirma é o mestre, pela mesma divisa do movimento no tabuleiro.
func (s Strike) Propose(ctx context.Context, quem app.Caller, papel string, pedido Request) (live.PendingAttack, error) {
	estado := s.mesas.GetState(pedido.SessionID)
	if estado == nil {
		return live.PendingAttack{}, fmt.Errorf("a sessão %d não tem mesa aberta: %w", pedido.SessionID, app.ErrNotFound)
	}
	atacante, err := entryOf(estado, pedido.AttackerEntryID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	alvo, err := entryOf(estado, pedido.TargetEntryID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if atacante.ID == alvo.ID {
		return live.PendingAttack{}, fmt.Errorf("ninguém ataca a si mesmo: %w", app.ErrRefused)
	}

	quemAtaca, err := s.combatentes.Of(ctx, atacante)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if len(quemAtaca.Weapons) == 0 {
		return live.PendingAttack{}, fmt.Errorf(
			"%s não tem arma empunhada com que atacar: %w", quemAtaca.Label, app.ErrRefused)
	}
	if pedido.Weapon < 0 || pedido.Weapon >= len(quemAtaca.Weapons) {
		return live.PendingAttack{}, fmt.Errorf(
			"%s empunha %d arma(s) e o pedido veio na %d: %w",
			quemAtaca.Label, len(quemAtaca.Weapons), pedido.Weapon, app.ErrRefused)
	}
	quemApanha, err := s.combatentes.Of(ctx, alvo)
	if err != nil {
		return live.PendingAttack{}, err
	}

	d20, err := s.oD20(pedido.D20)
	if err != nil {
		return live.PendingAttack{}, err
	}
	arma := quemAtaca.Weapons[pedido.Weapon]
	fora, err := engine.ResolveAttack(arma, engine.AttackTarget{
		Defense:         quemApanha.Defense,
		DamageReduction: quemApanha.DamageReduction,
		CritImmune:      quemApanha.CritImmune,
	}, d20, s.rolar)
	if err != nil {
		return live.PendingAttack{}, err
	}

	provisorio := live.PendingAttack{
		AttackerEntryID: atacante.ID, TargetEntryID: alvo.ID, Weapon: arma.Name,
		Roll: fora.Roll, Total: fora.Total, Hit: fora.Hit, Critical: fora.Critical,
		Dice: fora.Dice, RawDamage: fora.RawDamage, Absorbed: fora.Absorbed,
		Damage: fora.Damage, ByUserID: quem.ID,
	}
	if _, err := s.mesas.ProposeAttack(pedido.SessionID, provisorio); err != nil {
		return live.PendingAttack{}, err
	}
	return provisorio, nil
}

// oD20 devolve a rolagem, do cliente ou do servidor.
//
// O RECEBIDO É CONFERIDO, e é a mesma linha do `SelfEntry` da iniciativa: um
// número fora de 1..20 não é um dado, é um pedido montado à mão — e o servidor
// que o aceita dá crítico a quem digitou 40.
func (s Strike) oD20(recebido *int) (int, error) {
	if recebido == nil {
		rolagem, err := s.rolar(20)
		if err != nil {
			return 0, fmt.Errorf("rolar o d20: %w", err)
		}
		return rolagem, nil
	}
	if *recebido < 1 || *recebido > 20 {
		return 0, fmt.Errorf("o d20 rolado foi %d, e um d20 vai de 1 a 20: %w", *recebido, app.ErrRefused)
	}
	return *recebido, nil
}

func entryOf(st *live.SessionRuntimeState, entryID string) (live.InitiativeEntry, error) {
	if i := live.FindEntryIndex(st, entryID); i >= 0 {
		return st.Initiative[i], nil
	}
	return live.InitiativeEntry{}, fmt.Errorf("%q não está na fila: %w", entryID, app.ErrNotFound)
}
