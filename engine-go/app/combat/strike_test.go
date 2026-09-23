package combat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// AS DECISÕES do gesto de atacar. A conta do d20 contra a Defesa é do
// `engine.ResolveAttack` e tem os casos do livro lá; o que se prende aqui é o
// que ESTA camada decide: quem pode, qual d20 vale, e — a que mais importa —
// contra QUEM a Defesa é medida.

// queueDouble é a mesa dos casos: um herói e um ogro.
type queueDouble struct {
	state  *live.SessionRuntimeState
	stored *live.PendingAttack
}

func (f *queueDouble) State(context.Context, int64) (*live.SessionRuntimeState, error) {
	return f.state, nil
}

func (f *queueDouble) ProposeAttack(_ context.Context, _ int64, a live.PendingAttack) (*live.SessionRuntimeState, error) {
	f.stored = &a
	return f.state, live.ProposeAttack(f.state, a)
}

// CharacterActionFits deixa agir: a economia de ação é prendida pelo handler,
// na mesa de verdade (`serve/api/table_attack_test.go`).
func (f *queueDouble) CharacterActionFits(context.Context, int64, engine.ActionCost) error {
	return nil
}

// sheetDouble é a porta traduzida à mão: cada linha da queueDouble vira um combatente com
// os números que o caso quer.
type sheetDouble map[string]Combatant

func (f sheetDouble) Of(_ context.Context, _ int64, e live.InitiativeEntry) (Combatant, error) {
	c, ok := f[e.ID]
	if !ok {
		return Combatant{}, errors.New("combatente fora do caso")
	}
	return c, nil
}

func aTable(t *testing.T) *queueDouble {
	t.Helper()
	st := live.EmptyRuntimeState()
	st.Initiative = []live.InitiativeEntry{
		{ID: "heroi", Label: "Arwen", Type: "character"},
		{ID: "ogro", Label: "Ogro", Type: "npc"},
	}
	return &queueDouble{state: st}
}

func aLongsword() engine.WeaponCard {
	return engine.WeaponCard{Name: "Espada longa", Damage: "1d8", DamageBonus: 3, Attack: 5, CritRange: 19, CritMult: 2}
}

func fixedDie(value int) func(int) (int, error) {
	return func(int) (int, error) { return value, nil }
}

// A DEFESA MEDIDA É A DO ALVO, e este é o caso que separa "funciona" de
// "funciona por acaso": quem ataca tem Defesa 30 e o alvo tem 10. Um código que
// lesse a Defesa do atacante erraria o ataque — e um que lesse qualquer uma das
// duas passaria se as duas fossem iguais.
func TestTheDefenseThatMattersIsTheTargetOne(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}, Defense: 30},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, fixedDie(4))

	d20 := 10
	out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
	})
	if err != nil {
		t.Fatalf("propor: %v", err)
	}
	if !out.Hit {
		t.Errorf("15 contra a Defesa 10 do ALVO acerta; contra a 30 de quem ataca, erraria")
	}
	if out.TargetEntryID != "ogro" || out.AttackerEntryID != "heroi" {
		t.Errorf("o provisório saiu com atacante %q e alvo %q", out.AttackerEntryID, out.TargetEntryID)
	}
	if table.stored == nil || table.stored.ByUserID != 7 {
		t.Errorf("o provisório tem de ser GUARDADO na mesa, com quem rolou junto")
	}
}

// QUEM NÃO É DONO NÃO ROLA pelo personagem alheio, e o mestre rola por
// qualquer um. A posse chega RESOLVIDA do gateway, contra o banco.
func TestOnlyTheOwnerOrTheGameMasterRollsTheAttack(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}},
		"ogro":  {EntryID: "ogro", Defense: 10},
	}, table, fixedDie(15))
	req := Request{SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro"}

	_, err := strike.Propose(context.Background(), app.Caller{ID: 9}, "player", req)
	if !errors.Is(err, app.ErrRefused) {
		t.Errorf("quem não é dono não rola pelo personagem, e veio %v", err)
	}
	if !strings.Contains(err.Error(), "Arwen") {
		t.Errorf("a recusa diz de quem se fala, e veio %q", err)
	}
	if _, err := strike.Propose(context.Background(), app.Caller{ID: 1}, "gm", req); err != nil {
		t.Errorf("o mestre rola por qualquer um, e veio %v", err)
	}
	owner := req
	owner.OwnsAttacker = true
	if _, err := strike.Propose(context.Background(), app.Caller{ID: 9}, "player", owner); err != nil {
		t.Errorf("o dono rola pelo próprio personagem, e veio %v", err)
	}
}

// O D20 RECEBIDO É CONFERIDO. É a mesma linha do `SelfEntry` da iniciativa: um
// número fora da faixa não é um dado, é um pedido montado à mão.
func TestAD20OutsideTheRangeIsRefused(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Weapons: []engine.WeaponCard{aLongsword()}},
		"ogro":  {EntryID: "ogro", Defense: 10},
	}, table, fixedDie(4))

	for _, value := range []int{0, 21, -3, 40} {
		v := value
		_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
			SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &v, OwnsAttacker: true,
		})
		if !errors.Is(err, app.ErrRefused) {
			t.Errorf("d20 %d tinha de ser recusado, e veio %v", value, err)
		}
	}
	if table.stored != nil {
		t.Error("um pedido recusado não guarda provisório nenhum")
	}
}

// SEM D20 O SERVIDOR ROLA, e é o caminho do gesto de menu, que não tem onde
// digitar.
func TestWithoutAD20TheServerRollsIt(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Weapons: []engine.WeaponCard{aLongsword()}},
		"ogro":  {EntryID: "ogro", Defense: 10},
	}, table, fixedDie(19))

	out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", OwnsAttacker: true,
	})
	if err != nil {
		t.Fatalf("propor: %v", err)
	}
	if out.Roll != 19 {
		t.Errorf("o d20 = %d, e o dado desta mesa está cravado em 19", out.Roll)
	}
	if !out.Critical {
		t.Errorf("19 com margem 19 é crítico, e o provisório tem de dizer isso à mesa")
	}
}

// NINGUÉM ATACA A SI MESMO, e recusar na porta é mais barato que descobrir
// depois que o alvo e o atacante são a mesma linha da queueDouble.
func TestNobodyAttacksThemselves(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Weapons: []engine.WeaponCard{aLongsword()}},
	}, table, fixedDie(10))

	_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "heroi", OwnsAttacker: true,
	})
	if !errors.Is(err, app.ErrRefused) {
		t.Errorf("atacar a si mesmo tem de ser recusado, e veio %v", err)
	}
}

// QUEM NÃO EMPUNHA ARMA não ataca, e a frase diz o nome de quem — "não tem arma
// empunhada" sem sujeito manda procurar em nove sheetDouble.
func TestWithoutAWieldedWeaponThereIsNoAttack(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen"},
		"ogro":  {EntryID: "ogro", Defense: 10},
	}, table, fixedDie(10))

	_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", OwnsAttacker: true,
	})
	if !errors.Is(err, app.ErrRefused) {
		t.Fatalf("sem arma empunhada o ataque é recusado, e veio %v", err)
	}
	if !strings.Contains(err.Error(), "Arwen") {
		t.Errorf("a recusa tem de dizer de quem se fala, e veio %q", err)
	}
}
