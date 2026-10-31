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
	"t20engine/domain/board"
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
	// DamageReductionByType é a RD CONTRA UM TIPO (p229), por tipo de dano. Ela
	// viaja separada da geral porque alcança outra coisa: a redução de fogo come
	// a parcela de fogo e nada mais.
	DamageReductionByType map[string]int
	// CritExtraIgnoreChance é a chance, em porcento, de este combatente ignorar
	// o dano EXTRA de um crítico (p166, p338). Ela é INFORMADA à mesa, nunca
	// rolada pelo app.
	CritExtraIgnoreChance int
	CritImmune            bool
	// Melee é o valor de LUTA, e ele existe separado da Defesa porque a MANOBRA
	// o pede: o teste é oposto, e quem se defende rola Luta mesmo empunhando
	// arma de disparo (p234). Zero para quem não tem ficha — um bloco escrito à
	// mão não traz perícia.
	Melee int
	// ManeuverOffense e ManeuverDefense são os bônus que os modificadores dão a
	// cada manobra, por LADO. Dois mapas e não um porque o catálogo os separa: o
	// `Desejo de Liberdade` ajuda quem está sendo agarrado, não quem agarra.
	ManeuverOffense map[string]int
	ManeuverDefense map[string]int
	// Boost são as cartas com o impulso de PM pedido, e o PM que ele custa
	// (`engine.BoostedWeaponCards`). Nulo para quem não tem ficha — o bloco de
	// criatura não paga PM no golpe.
	Boost func(engine.BoostChoice) ([]engine.WeaponCard, int, error)
	// ChargesRecklessly diz que a ficha tem a Investida Imprudente (p42).
	ChargesRecklessly bool
	// ManaAvailable é o PM que a ficha pode gastar AGORA, poças temporárias
	// incluídas: o impulso que custa mais que isso é recusado na proposta.
	ManaAvailable int
}

// Combatants é a porta que traduz uma linha da fila em combatente.
type Combatants interface {
	Of(ctx context.Context, campaignID int64, entry live.InitiativeEntry) (Combatant, error)
}

// Tables é a porta do estado da mesa.
type Tables interface {
	// State devolve a mesa gravada, e DEVOLVE ERRO: sem conseguir ler não dá
	// para saber de quem é a vez, e rolar um ataque assim é inventar (ALE-373).
	State(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error)
	ProposeAttack(ctx context.Context, sessionID int64, attack live.PendingAttack) (*live.SessionRuntimeState, error)
	RuleAttack(ctx context.Context, sessionID int64, who live.Attacker, ruling string, toRoll *live.DamageToRoll) (*live.SessionRuntimeState, error)
	RecordAttackDamage(ctx context.Context, sessionID int64, who live.Attacker, rolled live.PendingAttack) (*live.SessionRuntimeState, error)
	// CharacterActionFits diz se este personagem pode gastar o custo AGORA: se
	// ele pode agir (o instante, pelo PV e pelas condições da ficha) e se sobrou
	// ação no turno. Não cobra nada — quem cobra é a confirmação.
	CharacterActionFits(ctx context.Context, characterID int64, cost engine.ActionCost) error
}

// Situations é a porta do TABULEIRO, e ela devolve regra em vez de mapa.
//
// A pergunta é "que linhas da Tabela 5-3 valem entre estas duas linhas da
// fila?", e não "onde estão as peças?" — uma porta que devolvesse coordenadas
// obrigaria este pacote a saber de corpo, de pegada e de qual espécie de casa
// faz o quê, que é tudo o que o `domain/board` já sabe.
//
// VAZIO É LEGÍTIMO e é o caso comum: uma mesa sem tabuleiro ataca sem situação
// nenhuma, e isso sai como lista vazia sem erro.
//
// O ERRO, porém, SOBE. Um tabuleiro que não se consegue ler viraria uma
// cobertura que some: o ataque aconteceria com +5 a menos e ninguém saberia
// que faltou. É a mesma linha da leitura da fila (ALE-373) — número errado em
// silêncio é pior que gesto recusado.
type Situations interface {
	Between(ctx context.Context, sessionID int64, attackerEntry string,
		target board.AttackTargetOnTheBoard) ([]engine.SpecialSituation, error)
}

// Strike resolve e propõe ataques.
type Strike struct {
	combatants Combatants
	tables     Tables
	situations Situations
	rollDie    func(faces int) (int, error)
}

func NewStrike(c Combatants, t Tables, s Situations, rollDie func(faces int) (int, error)) Strike {
	return Strike{combatants: c, tables: t, situations: s, rollDie: rollDie}
}

// specialSituations pergunta ao tabuleiro, e cala quando não há um.
//
// A porta nula é tratada aqui e não em cada chamador: ela é opcional por
// desenho — o combate existe sem mapa —, e um `if` por sítio seria a mesma
// decisão escrita três vezes.
func (s Strike) specialSituations(ctx context.Context, req Request) ([]engine.SpecialSituation, error) {
	doTabuleiro, err := s.situationsOnTheBoard(ctx, req)
	if err != nil {
		return nil, err
	}
	// A ESCOLHA DO TIPO DE DANO não vem do mapa: ela é do gesto, e é por isso
	// que ela se junta à lista AQUI e não no adaptador do tabuleiro. As duas
	// viajam no mesmo vetor porque o mecanismo é o mesmo — deslocar o teste de
	// ataque e dizer à mesa por quê.
	if charging, is := engine.ChargeSituation(req.Charge); is {
		doTabuleiro = append(doTabuleiro, charging)
	}
	if req.SwitchesDamageType {
		return append(doTabuleiro, engine.AttackerSwitchesTheDamageType), nil
	}
	return doTabuleiro, nil
}

// situationsOnTheBoard são só as que o MAPA produz, e cala quando não há mapa.
func (s Strike) situationsOnTheBoard(ctx context.Context, req Request) ([]engine.SpecialSituation, error) {
	if s.situations == nil {
		return nil, nil
	}
	return s.situations.Between(ctx, req.SessionID, req.AttackerEntryID,
		board.AttackTargetOnTheBoard{EntryID: req.TargetEntryID, TokenID: req.TargetObjectTokenID()})
}

// Request é o pedido de ataque.
type Request struct {
	// CampaignID é a dona do bloco de criatura, e por isso ela entra: sem ela
	// o combate lia o bloco de outra mesa (ALE-377).
	CampaignID      int64
	SessionID       int64
	AttackerEntryID string
	TargetEntryID   string
	// TargetObject é a PEÇA DE OBJETO atacada (p239), e é a alternativa ao
	// `TargetEntryID`: *"para objetos soltos, faça um ataque contra a Defesa do
	// objeto"*, e um objeto não tem linha na fila porque não tem turno.
	//
	// Nula é o caso comum — ataque a criatura. Quem a monta é a cena, lendo a
	// peça do tabuleiro; a REGRA (a Defesa pelo tamanho, a RD pelo material)
	// fica aqui, que é onde o motor é chamado.
	TargetObject *ObjectTarget
	// Weapon é qual das armas empunhadas, por índice. Zero é a primeira, que é
	// o caso de quase toda ficha.
	Weapon int
	// OwnsAttacker: quem pede é dono do personagem que ataca. Resolvido no
	// gateway, CONTRA O BANCO — o cliente não é fonte de posse. É o mesmo campo
	// e a mesma razão do `board.Mover.OwnsCharacter`.
	OwnsAttacker bool
	// Maneuver é a manobra do livro quando o gesto é uma MANOBRA e não um golpe
	// (p234): agarrar, derrubar, desarmar, empurrar ou quebrar. Vazio é golpe.
	Maneuver string
	// SwitchesDamageType é a ESCOLHA da p236: usar a arma contra a natureza dela
	// — o fio para derrubar, ou o punho para matar. Custa −5 nos dois sentidos.
	//
	// É do GOLPE e não da manobra: uma manobra não causa dano, então trocar o
	// tipo dele não significa nada ali.
	SwitchesDamageType bool
	// Boost é o impulso de PM que o jogador pediu para ESTE golpe — o Golpe
	// Divino, o Ataque Especial (ALE-487). Vazio é golpe sem impulso.
	Boost engine.BoostChoice
	// Charge é a investida que leva a este golpe (`ChargePlain`,
	// `ChargeReckless`); vazio é golpe parado. Ver `charge.go`.
	Charge string
	// OpposedD20 é o d20 de quem se DEFENDE da manobra, e ele existe pelo mesmo
	// motivo do `D20`: a mesa pode rolar na mão. Nulo é o servidor rolar.
	OpposedD20 *int
	// D20 é a rolagem QUE JÁ ACONTECEU na mesa, quando aconteceu.
	//
	// Nulo é o servidor rolar. Os dois caminhos existem porque as duas mesas
	// existem, e este repositório já tem os dois precedentes escritos: a
	// INICIATIVA recebe o d20 do jogador (ele rolou um dado de verdade e
	// digitou), e a bolsa inicial rola no servidor. Aceitar nulo é o que
	// permite o gesto de menu — que não tem onde digitar — sem fechar a porta
	// para a mesa que rola dado na mão.
	D20 *int
	// CountsAsNaturalTwenty diz que o `D20` "será considerado um 20 natural"
	// (Sorte de Nimb, p48): só o rolar de novo o acende (ALE-508).
	CountsAsNaturalTwenty bool
}

// strikerFor confere QUEM ATACA e monta o combatente dele.
//
// É tudo o que o golpe contra criatura e o golpe contra objeto dividem — a mesa
// aberta, a linha da vez, a posse resolvida contra o banco, a ação padrão da
// p233 e a arma empunhada. O que os separa vem depois, e é só o ALVO.
//
// Extraído quando o segundo chamador apareceu: as cinco conferências escritas
// duas vezes seriam a próxima correção acertando uma e esquecendo a outra.
func (s Strike) strikerFor(
	ctx context.Context, role string, req Request,
) (live.InitiativeEntry, Combatant, error) {
	state, err := s.tables.State(ctx, req.SessionID)
	if err != nil {
		return live.InitiativeEntry{}, Combatant{}, err
	}
	if state == nil {
		return live.InitiativeEntry{}, Combatant{}, fmt.Errorf(
			"a sessão %d não tem mesa aberta: %w", req.SessionID, app.ErrNotFound)
	}
	attackerEntry, err := entryOf(state, req.AttackerEntryID)
	if err != nil {
		return live.InitiativeEntry{}, Combatant{}, err
	}
	// QUEM ROLA É O DONO, ou o mestre. Sem isto qualquer um na mesa rolaria o
	// ataque do personagem alheio que estiver na vez — e o provisório sairia com
	// o nome dele, que é pior do que não deixar atacar.
	if role != "gm" && !req.OwnsAttacker {
		return live.InitiativeEntry{}, Combatant{}, fmt.Errorf(
			"%s não é seu personagem: %w", attackerEntry.Label, app.ErrRefused)
	}
	// AGREDIR É AÇÃO PADRÃO (p233), e a INVESTIDA é completa (p235). A
	// pergunta vem ANTES de rolar: um d20
	// rolado por quem está atordoado, ou já gastou a padrão, é um provisório que
	// a mesa vê e que nunca poderia ter acontecido.
	if attackerEntry.CharacterID != nil {
		if err := s.tables.CharacterActionFits(ctx, *attackerEntry.CharacterID, engine.StrikeAction(req.Charge)); err != nil {
			return live.InitiativeEntry{}, Combatant{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
		}
	}
	striker, err := s.combatants.Of(ctx, req.CampaignID, attackerEntry)
	if err != nil {
		return live.InitiativeEntry{}, Combatant{}, err
	}
	if len(striker.Weapons) == 0 {
		return live.InitiativeEntry{}, Combatant{}, fmt.Errorf(
			"%s não tem arma empunhada com que atacar: %w", striker.Label, app.ErrRefused)
	}
	return attackerEntry, striker, nil
}

// ObjectTarget é a peça de cenário que está sendo atacada, como a cena a lê.
//
// Ela traz tamanho e material CRUS e não Defesa e RD prontas: as duas são as
// escadas da p239, e deixar a cena calculá-las poria a regra do livro numa
// camada que não é dona dela.
type ObjectTarget struct {
	TokenID  string
	Label    string
	Size     string
	Material string
}

// TargetObjectTokenID é a peça atacada, ou vazio quando o alvo é criatura.
func (r Request) TargetObjectTokenID() string {
	if r.TargetObject == nil {
		return ""
	}
	return r.TargetObject.TokenID
}

// strikeSetup é o que o golpe sabe antes de qualquer dado: quem ataca, com que
// arma, contra o quê e sob quais linhas da Tabela 5-3. Os dois tempos o montam
// do mesmo jeito — o teste a partir do pedido, o dano a partir do provisório —,
// e montá-lo em dois lugares faria a próxima correção acertar um e esquecer o
// outro.
type strikeSetup struct {
	attacker live.InitiativeEntry
	striker  Combatant
	weapon   engine.WeaponCard
	boost    paidBoost
	// victim é a CRIATURA atacada, e é vazio quando o alvo é peça: um objeto
	// não tem ficha, bloco nem verbete (p239).
	victim     Combatant
	target     engine.AttackTarget
	targetID   string
	situations []engine.SpecialSituation
}

// setUp confere quem ataca e o alvo, e monta a arma.
//
// O OBJETO é um ramo e não um caso de uso irmão pelo que ele NÃO faz: não há
// linha da fila a achar nem `Combatant` a montar — a Defesa e a RD vêm das
// escadas da p239. O que ele divide é a vez, a posse, a ação padrão e a arma,
// que vêm do `strikerFor`.
func (s Strike) setUp(ctx context.Context, role string, req Request) (strikeSetup, error) {
	attackerEntry, striker, err := s.strikerFor(ctx, role, req)
	if err != nil {
		return strikeSetup{}, err
	}
	if req.Maneuver != "" && req.Boost.PowerID != "" {
		// O IMPULSO É DO GOLPE: a manobra não rola dano, e o Golpe Divino e o
		// Ataque Especial vendem bônus num ataque que fere (p65, p82).
		return strikeSetup{}, fmt.Errorf("o impulso de PM é do golpe, e não da manobra: %w", app.ErrRefused)
	}
	weapon, boost, err := weaponFor(striker, req)
	if err != nil {
		return strikeSetup{}, err
	}
	if weapon, err = chargedWeapon(striker, req, weapon); err != nil {
		return strikeSetup{}, err
	}
	setup := strikeSetup{attacker: attackerEntry, striker: striker, weapon: weapon, boost: boost}
	if req.TargetObject != nil {
		if setup.target, err = engine.ObjectTarget(req.TargetObject.Size, req.TargetObject.Material); err != nil {
			return strikeSetup{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
		}
	} else {
		state, err := s.tables.State(ctx, req.SessionID)
		if err != nil {
			return strikeSetup{}, err
		}
		targetEntry, err := entryOf(state, req.TargetEntryID)
		if err != nil {
			return strikeSetup{}, err
		}
		if attackerEntry.ID == targetEntry.ID {
			return strikeSetup{}, fmt.Errorf("ninguém ataca a si mesmo: %w", app.ErrRefused)
		}
		if setup.victim, err = s.combatants.Of(ctx, req.CampaignID, targetEntry); err != nil {
			return strikeSetup{}, err
		}
		setup.targetID = targetEntry.ID
		setup.target = engine.AttackTarget{
			Defense:               setup.victim.Defense,
			DamageReduction:       setup.victim.DamageReduction,
			DamageReductionByType: setup.victim.DamageReductionByType,
			CritExtraIgnoreChance: setup.victim.CritExtraIgnoreChance,
			CritImmune:            setup.victim.CritImmune,
		}
	}
	if req.Maneuver == "" {
		if setup.situations, err = s.specialSituations(ctx, req); err != nil {
			return strikeSetup{}, err
		}
	}
	return setup, nil
}

// pendingOf é o provisório do golpe, com o que o pedido escolheu e o teste
// devolveu. O dano fica de fora: ele é rolado depois da decisão do mestre.
func (setup strikeSetup) pendingOf(who app.Caller, req Request, out engine.AttackOutcome) live.PendingAttack {
	margin, multiplier := engine.CriticalOf(setup.weapon)
	pending := live.PendingAttack{
		AttackerEntryID: setup.attacker.ID, TargetEntryID: setup.targetID, Weapon: setup.weapon.Name,
		// A DEFESA do provisório é a que o ataque ENFRENTOU, já com a Tabela
		// 5-3 — não a da ficha. "Errei por 1" e "errei por 1 porque ele está
		// atrás da carroça" são leituras diferentes do mesmo número.
		Roll: out.Roll, Total: out.Total, Defense: out.Defense, Situations: out.Situations,
		Hit: out.Hit, Critical: out.Critical, CritRange: margin, CritMult: multiplier,
		Melee: setup.weapon.IsMelee(), ByUserID: who.ID,
		BoostPowerID: setup.boost.powerID, BoostPM: setup.boost.pm, Charge: req.Charge,
		Replay: req.replayOf(setup.weapon),
	}
	if req.TargetObject != nil {
		pending.TargetTokenID, pending.TargetLabel = req.TargetObject.TokenID, req.TargetObject.Label
	}
	return pending
}

// Propose rola o TESTE do ataque e guarda o provisório. Ninguém perde PV aqui,
// e o dano ainda não existe: ele vem depois da decisão do mestre.
func (s Strike) Propose(ctx context.Context, who app.Caller, role string, req Request) (live.PendingAttack, error) {
	pending, err := s.roll(ctx, who, role, req)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if _, err := s.tables.ProposeAttack(ctx, req.SessionID, pending); err != nil {
		return live.PendingAttack{}, err
	}
	return pending, nil
}

// roll rola o teste do golpe ou a manobra do pedido, sem guardar: quem guarda
// é o `Propose`, e o `Reroll` junta antes o que o primeiro dado deixou.
func (s Strike) roll(ctx context.Context, who app.Caller, role string, req Request) (live.PendingAttack, error) {
	setup, err := s.setUp(ctx, role, req)
	if err != nil {
		return live.PendingAttack{}, err
	}
	d20, err := s.d20Of(req.D20)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if req.Maneuver != "" {
		return s.rollManeuver(who, req, setup, d20)
	}
	out, err := engine.JudgeAttack(setup.weapon, setup.target, setup.situations, req.attackDie(d20), s.rollDie)
	if err != nil {
		return live.PendingAttack{}, err
	}
	return setup.pendingOf(who, req, out), nil
}

// rollManeuver resolve o teste OPOSTO da manobra e guarda o provisório.
//
// Ele divide com o golpe tudo que vem antes: a vez, a posse, a ação padrão e as
// duas pontas da fila. O que muda é só a REGRA no meio — e é por isso que ele é
// um ramo aqui, e não um caso de uso irmão que repetiria sete conferências.
func (s Strike) rollManeuver(who app.Caller, req Request, setup strikeSetup, d20 int) (live.PendingAttack, error) {
	if req.TargetObject != nil {
		return live.PendingAttack{}, fmt.Errorf("manobra é contra criatura, e não contra peça de cenário: %w", app.ErrRefused)
	}
	striker, victim, weapon := setup.striker, setup.victim, setup.weapon
	// O LUTA DO DEFENSOR, e não a Defesa dele: "mesmo que ela esteja usando uma
	// arma de ataque à distância, deve fazer o teste usando seu valor de Luta"
	// (p234). Quem não tem arma empunhada não tem carta, e aí o Luta dele é
	// zero — um NPC de bloco escrito à mão não traz perícia.
	defesa := engine.ManeuverSide{Bonus: victim.Melee + victim.ManeuverDefense[req.Maneuver]}
	ataque := engine.ManeuverSide{
		Bonus:  weapon.Attack + striker.ManeuverOffense[req.Maneuver],
		Ranged: !weapon.IsMelee(),
	}

	opposed, err := s.d20Of(req.OpposedD20)
	if err != nil {
		return live.PendingAttack{}, err
	}
	out := engine.ResolveManeuver(req.Maneuver, ataque, defesa, d20, opposed)
	if out.Refused != "" {
		return live.PendingAttack{}, fmt.Errorf("%s: %w", out.Refused, app.ErrRefused)
	}

	pending := live.PendingAttack{
		AttackerEntryID: setup.attacker.ID, TargetEntryID: setup.targetID, Weapon: weapon.Name,
		Roll: out.AttackerRoll, Total: out.AttackerTotal, ByUserID: who.ID, Replay: req.replayOf(weapon),
		Maneuver: &live.ManeuverRoll{
			Kind: out.Kind, OpposedRoll: out.DefenderRoll, Opposed: out.DefenderTotal,
			Margin: out.Margin, ConditionOnAWin: out.ConditionOnAWin,
		},
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

// foreseenOf copia as condições que o livro reserva para a forma que o regime
// entende — ele não conhece o motor, e o DADO é o que atravessa.
//
// O `OnCritical` fica para trás de propósito: o resultado já veio filtrado pelo
// desfecho, então uma condição que chegou aqui é uma que vale para ESTE golpe.
func foreseenOf(out engine.AttackOutcome) []live.ForeseenCondition {
	ditas := make([]live.ForeseenCondition, 0, len(out.Announces))
	for _, a := range out.Announces {
		ditas = append(ditas, live.ForeseenCondition{Condition: a.Condition, Note: a.Note})
	}
	if len(ditas) == 0 {
		return nil
	}
	return ditas
}
