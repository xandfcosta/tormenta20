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
	// State devolve a mesa gravada, e DEVOLVE ERRO: sem conseguir ler não dá
	// para saber de quem é a vez, e rolar um ataque assim é inventar (ALE-373).
	State(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error)
	ProposeAttack(ctx context.Context, sessionID int64, attack live.PendingAttack) (*live.SessionRuntimeState, error)
	// CharacterActionFits diz se este personagem pode gastar o custo AGORA: se
	// ele pode agir (o instante, pelo PV e pelas condições da ficha) e se sobrou
	// ação no turno. Não cobra nada — quem cobra é a confirmação.
	CharacterActionFits(ctx context.Context, characterID int64, cost engine.ActionCost) error
}

// Strike resolve e propõe ataques.
type Strike struct {
	combatants Combatants
	tables     Tables
	rollDie    func(faces int) (int, error)
}

func NewStrike(c Combatants, t Tables, rollDie func(faces int) (int, error)) Strike {
	return Strike{combatants: c, tables: t, rollDie: rollDie}
}

// Request é o pedido de ataque.
type Request struct {
	SessionID       int64
	AttackerEntryID string
	TargetEntryID   string
	// Weapon é qual das armas empunhadas, por índice. Zero é a primeira, que é
	// o caso de quase toda ficha.
	Weapon int
	// OwnsAttacker: quem pede é dono do personagem que ataca. Resolvido no
	// gateway, CONTRA O BANCO — o cliente não é fonte de posse. É o mesmo campo
	// e a mesma razão do `board.Mover.OwnsCharacter`.
	OwnsAttacker bool
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
func (s Strike) Propose(ctx context.Context, who app.Caller, role string, req Request) (live.PendingAttack, error) {
	state, err := s.tables.State(ctx, req.SessionID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if state == nil {
		return live.PendingAttack{}, fmt.Errorf("a sessão %d não tem mesa aberta: %w", req.SessionID, app.ErrNotFound)
	}
	attackerEntry, err := entryOf(state, req.AttackerEntryID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	target, err := entryOf(state, req.TargetEntryID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	// QUEM ROLA É O DONO, ou o mestre. Sem isto qualquer um na mesa rolaria o
	// ataque do personagem alheio que estiver na vez — e o provisório sairia com
	// o nome dele, que é pior do que não deixar atacar.
	if role != "gm" && !req.OwnsAttacker {
		return live.PendingAttack{}, fmt.Errorf(
			"%s não é seu personagem: %w", attackerEntry.Label, app.ErrRefused)
	}
	if attackerEntry.ID == target.ID {
		return live.PendingAttack{}, fmt.Errorf("ninguém ataca a si mesmo: %w", app.ErrRefused)
	}
	// AGREDIR É AÇÃO PADRÃO (p233), e a pergunta vem ANTES de rolar: um d20
	// rolado por quem está atordoado, ou já gastou a padrão, é um provisório que
	// a mesa vê e que nunca poderia ter acontecido.
	if attackerEntry.CharacterID != nil {
		if err := s.tables.CharacterActionFits(ctx, *attackerEntry.CharacterID, engine.ActionStandard); err != nil {
			return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
		}
	}

	striker, err := s.combatants.Of(ctx, attackerEntry)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if len(striker.Weapons) == 0 {
		return live.PendingAttack{}, fmt.Errorf(
			"%s não tem arma empunhada com que atacar: %w", striker.Label, app.ErrRefused)
	}
	if req.Weapon < 0 || req.Weapon >= len(striker.Weapons) {
		return live.PendingAttack{}, fmt.Errorf(
			"%s empunha %d arma(s) e o pedido veio na %d: %w",
			striker.Label, len(striker.Weapons), req.Weapon, app.ErrRefused)
	}
	victim, err := s.combatants.Of(ctx, target)
	if err != nil {
		return live.PendingAttack{}, err
	}

	d20, err := s.d20Of(req.D20)
	if err != nil {
		return live.PendingAttack{}, err
	}
	weapon := striker.Weapons[req.Weapon]
	out, err := engine.ResolveAttack(weapon, engine.AttackTarget{
		Defense:         victim.Defense,
		DamageReduction: victim.DamageReduction,
		CritImmune:      victim.CritImmune,
	}, d20, s.rollDie)
	if err != nil {
		return live.PendingAttack{}, err
	}

	pending := live.PendingAttack{
		AttackerEntryID: attackerEntry.ID, TargetEntryID: target.ID, Weapon: weapon.Name,
		Roll: out.Roll, Total: out.Total, Defense: victim.Defense,
		Hit: out.Hit, Critical: out.Critical,
		Dice: out.Dice, Faces: out.Faces, RawDamage: out.RawDamage, Absorbed: out.Absorbed,
		Damage: out.Damage, ByUserID: who.ID,
	}
	if _, err := s.tables.ProposeAttack(ctx, req.SessionID, pending); err != nil {
		return live.PendingAttack{}, err
	}
	return pending, nil
}

// d20Of devolve a rolagem, do cliente ou do servidor.
//
// O RECEBIDO É CONFERIDO, e é a mesma linha do `SelfEntry` da iniciativa: um
// número fora de 1..20 não é um dado, é um pedido montado à mão — e o servidor
// que o aceita dá crítico a quem digitou 40.
func (s Strike) d20Of(given *int) (int, error) {
	if given == nil {
		roll, err := s.rollDie(20)
		if err != nil {
			return 0, fmt.Errorf("rolar o d20: %w", err)
		}
		return roll, nil
	}
	if *given < 1 || *given > 20 {
		return 0, fmt.Errorf("o d20 rolado foi %d, e um d20 vai de 1 a 20: %w", *given, app.ErrRefused)
	}
	return *given, nil
}

func entryOf(st *live.SessionRuntimeState, entryID string) (live.InitiativeEntry, error) {
	if i := live.FindEntryIndex(st, entryID); i >= 0 {
		return st.Initiative[i], nil
	}
	return live.InitiativeEntry{}, fmt.Errorf("%q não está na fila: %w", entryID, app.ErrNotFound)
}
