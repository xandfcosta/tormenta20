package combat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"t20engine/app"
	"t20engine/domain/board"
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

// O `Skill` entra porque a MANOBRA o lê: ela é ataque corpo a corpo (p234), e
// uma carta sem perícia é recusada — a regra falha no que não reconhece, que é a
// direção segura. O dublê sem ele mentia por omissão: toda carta de verdade sai
// do `ComputeWeaponCards` com "Luta" ou "Pontaria".
func aLongsword() engine.WeaponCard {
	return engine.WeaponCard{Name: "Espada longa", Skill: "Luta", Damage: "1d8", DamageBonus: 3, Attack: 5, CritRange: 19, CritMult: 2}
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
	}, table, semTabuleiro{}, fixedDie(4))

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
	}, table, semTabuleiro{}, fixedDie(15))
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
	}, table, semTabuleiro{}, fixedDie(4))

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
	}, table, semTabuleiro{}, fixedDie(19))

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
	}, table, semTabuleiro{}, fixedDie(10))

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
	}, table, semTabuleiro{}, fixedDie(10))

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

// O DEFENSOR DA MANOBRA ROLA LUTA, e não a Defesa (T20 p234).
//
// É o caso que separa "funciona" de "funciona por acaso", como o da Defesa
// acima: o alvo tem Defesa 30 e Luta 2. Um código que lesse a Defesa daria 30 ao
// outro lado do teste oposto, e a manobra nunca venceria — com o número saindo
// plausível na faixa, porque 30 é um total possível.
//
// A página diz isso por extenso: *"mesmo que ela esteja usando uma arma de
// ataque à distância, deve fazer o teste usando seu valor de Luta"*.
func TestTheManeuverDefenderRollsMeleeAndNotDefense(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}, Defense: 30},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 30, Melee: 2},
	}, table, semTabuleiro{}, fixedDie(11))

	d20 := 10
	out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro",
		D20: &d20, OwnsAttacker: true, Maneuver: "derrubar",
	})
	if err != nil {
		t.Fatalf("propor a manobra: %v", err)
	}
	if out.Maneuver == nil {
		t.Fatal("o provisório saiu sem a conta da manobra")
	}
	// O d20 do defensor é o do servidor (11), e o Luta dele é 2: total 13.
	// Lendo a Defesa daria 41, e a manobra perderia sempre.
	if out.Maneuver.Opposed != 13 {
		t.Errorf("o lado do defensor deu %d e a conta é 11 + Luta 2 = 13.\n"+
			"41 quer dizer que o código leu a DEFESA (30) do alvo, e a p234 manda "+
			"ele rolar LUTA", out.Maneuver.Opposed)
	}
	// 10 + 5 = 15 contra 13: a diferença é +2, e é ela que o provisório leva. Quem
	// olha os dois números e decide é o mestre.
	if out.Maneuver.Margin != 2 {
		t.Errorf("15 contra 13 dá diferença +2, e deu %+v", *out.Maneuver)
	}
	if out.Damage != 0 {
		t.Errorf("a manobra causou %d de dano, e a p234 diz que ela faz algo DIFERENTE "+
			"de causar dano", out.Damage)
	}
}

// O BÔNUS DE MANOBRA DE CADA LADO entra no lado dele, e o do defensor NUNCA no
// ataque — é a decisão que a ALE-406 tomou pondo o escopo na chave.
func TestEachSideGetsItsOwnManeuverBonus(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()},
			ManeuverOffense: map[string]int{"agarrar": 2}},
		"ogro": {EntryID: "ogro", Label: "Ogro", Melee: 0,
			ManeuverDefense: map[string]int{"agarrar": 5}},
	}, table, semTabuleiro{}, fixedDie(10))

	d20 := 10
	out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro",
		D20: &d20, OwnsAttacker: true, Maneuver: "agarrar",
	})
	if err != nil {
		t.Fatalf("propor a manobra: %v", err)
	}
	// Quem ataca: 10 + 5 do Luta da espada + 2 do bônus de ofensa = 17.
	if out.Total != 17 {
		t.Errorf("o lado de quem agarra deu %d e a conta é 10 + 5 + 2 = 17", out.Total)
	}
	// Quem defende: 10 + 0 de Luta + 5 do Desejo de Liberdade = 15.
	if out.Maneuver.Opposed != 15 {
		t.Errorf("o lado de quem se solta deu %d e a conta é 10 + 0 + 5 = 15.\n"+
			"Somar o +5 de defesa no ATAQUE daria 22 e a manobra venceria sempre — é o "+
			"defeito que a ALE-406 consertou pondo o escopo na chave", out.Maneuver.Opposed)
	}
}

// O ARCO NÃO DERRUBA NINGUÉM, e a recusa vem desta camada lendo a PERÍCIA da
// carta (T20 p234: *"não é possível fazer manobras de combate com ataques à
// distância"*).
//
// A regra sabe recusar — o `ResolveManeuver` tem o caso —, mas quem lhe diz que a
// arma é de disparo é o `Strike`. Sem este caso, um `Ranged: false` fixo passaria
// verde: o arco derrubaria o ogro e a recusa nunca seria exercida pelo gesto.
func TestABowCannotManeuver(t *testing.T) {
	arco := engine.WeaponCard{Name: "Arco longo", Skill: "Pontaria", Damage: "1d8", Attack: 5}
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{arco}},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Melee: 0},
	}, table, semTabuleiro{}, fixedDie(10))

	d20 := 20
	_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro",
		D20: &d20, OwnsAttacker: true, Maneuver: "derrubar",
	})
	if err == nil {
		t.Fatal("o arco derrubou o ogro: a p234 diz que manobra é ataque corpo a corpo")
	}
	if !strings.Contains(err.Error(), "corpo a corpo") {
		t.Errorf("a recusa tinha de dizer o motivo do livro, e veio %q", err)
	}
	if table.stored != nil {
		t.Error("a manobra recusada virou provisório na mesa")
	}
	// E O GOLPE COM ARCO CONTINUA VALENDO: a proibição é da MANOBRA, não do
	// ataque. Sem esta metade, um `Ranged` lido errado proibiria atirar.
	if _, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro",
		D20: &d20, OwnsAttacker: true,
	}); err != nil {
		t.Errorf("atirar de arco foi recusado (%v), e a p234 só proíbe a MANOBRA", err)
	}
}

// semTabuleiro é a mesa que não abriu mapa, que é a maior parte de uma sessão.
//
// Dublê NOMEADO e não `nil`: o `nil` também passa — o `specialSituations` o
// trata —, mas ele diz "esqueci de montar" e este diz "esta mesa não tem mapa",
// que é um caso do livro.
type semTabuleiro struct{}

func (semTabuleiro) Between(context.Context, int64, string, board.AttackTargetOnTheBoard) ([]engine.SpecialSituation, error) {
	return nil, nil
}

// comTabuleiro devolve as situações que lhe deram, sem olhar quem pergunta.
type comTabuleiro struct{ situations []engine.SpecialSituation }

func (c comTabuleiro) Between(context.Context, int64, string, board.AttackTargetOnTheBoard) ([]engine.SpecialSituation, error) {
	return c.situations, nil
}

// A SITUAÇÃO DO TABULEIRO CHEGA AO d20, e ela chega pela PORTA.
//
// O caso do `serve/api` prende o trajeto inteiro — o mestre pinta, a peça está
// na casa, a conta muda. Este prende a junta: que o `Propose` PERGUNTA ao
// tabuleiro e passa a resposta adiante. Sem ele, trocar o `ResolveAttackUnder`
// de volta pelo `ResolveAttack` deixaria a porta ligada e inerte.
func TestTheBoardSituationReachesTheRoll(t *testing.T) {
	table := aTable(t)
	alvo := sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}, Defense: 30},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}
	d20 := 10
	pedido := Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
	}

	nua, err := NewStrike(alvo, table, semTabuleiro{}, fixedDie(4)).
		Propose(context.Background(), app.Caller{ID: 7}, "player", pedido)
	if err != nil {
		t.Fatalf("propor sem tabuleiro: %v", err)
	}

	table = aTable(t) // o provisório anterior fica em pé, e um segundo é recusado
	coberto, err := NewStrike(alvo, table,
		comTabuleiro{situations: []engine.SpecialSituation{engine.TargetUnderLightCover}},
		fixedDie(4)).
		Propose(context.Background(), app.Caller{ID: 7}, "player", pedido)
	if err != nil {
		t.Fatalf("propor com cobertura: %v", err)
	}
	if coberto.Defense != nua.Defense+5 {
		t.Errorf("com a cobertura o ataque enfrentou Defesa %d e sem ela %d — a p239 dá +5",
			coberto.Defense, nua.Defense)
	}
}

// E O ERRO DO TABULEIRO SOBE, em vez de virar uma cobertura que some.
//
// Um mapa que não se consegue ler faria o ataque acontecer com +5 a menos e
// ninguém saberia que faltou. É a mesma linha da leitura da fila (ALE-373):
// número errado em silêncio é pior que gesto recusado.
func TestABoardThatCannotBeReadRefusesTheAttack(t *testing.T) {
	_, err := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}, Defense: 30},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, aTable(t), tabuleiroQuebrado{}, fixedDie(4)).
		Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
			SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", OwnsAttacker: true,
		})
	if err == nil {
		t.Fatal("o ataque aconteceu sobre um tabuleiro que não pôde ser lido")
	}
}

type tabuleiroQuebrado struct{}

func (tabuleiroQuebrado) Between(context.Context, int64, string, board.AttackTargetOnTheBoard) ([]engine.SpecialSituation, error) {
	return nil, errors.New("o disco sumiu")
}

// A MANOBRA IGNORA A TROCA DO TIPO DE DANO, e este caso é de DECISÃO e não de
// encanamento: ele prende o RAMO, afirmando que o mesmo pedido com a flag ligada
// e desligada dá o mesmo total.
//
// A primeira versão dele morava no `serve/api` e era VÁCUA: ela procurava o
// rótulo "trocando o tipo de dano" no `Situations` do provisório, e a manobra
// NUNCA preenche esse campo — o laço percorria uma lista vazia e passaria mesmo
// se a manobra pagasse os −5. Um teste de ausência sem prova de que o canal
// existiria é a armadilha que o `CLAUDE.md` nomeia.
//
// Uma manobra não causa dano, então trocar o TIPO dele não significa nada ali —
// e uma penalidade sobre um teste sem dano é um número que ninguém consegue
// explicar na mesa.
func TestTheManeuverIgnoresTheDamageTypeChoice(t *testing.T) {
	roll := func(switches bool) int {
		t.Helper()
		strike := NewStrike(sheetDouble{
			"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}, Melee: 6},
			"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10, Melee: 2},
		}, aTable(t), semTabuleiro{}, fixedDie(4))
		d20, opposed := 10, 5
		out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
			SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro",
			Maneuver: "derrubar", D20: &d20, OpposedD20: &opposed, OwnsAttacker: true,
			SwitchesDamageType: switches,
		})
		if err != nil {
			t.Fatalf("propor a manobra (trocando=%v): %v", switches, err)
		}
		return out.Total
	}
	nua, trocada := roll(false), roll(true)
	if trocada != nua {
		t.Errorf("a manobra somou %d com a troca ligada e %d sem ela — a p236 cobra os −5 "+
			"de quem troca o tipo do DANO, e manobra não causa dano", trocada, nua)
	}
}
