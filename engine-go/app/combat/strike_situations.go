package combat

import (
	"context"
	"slices"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O QUE A CENA EM VOLTA FAZ COM O GOLPE: o que o mapa diz, o que o gesto pede e o que a mira anula.

// settleSituations põe no golpe as linhas da Tabela 5-3 que valem para ele.
func (s Strike) settleSituations(ctx context.Context, req Request, setup *strikeSetup) error {
	// O ESTILO DESARMADO troca o tipo "sem penalidades" (p128): sem o -5 da Tabela 5-3, o que
	// sobra da troca é a natureza do golpe.
	if req.SwitchesDamageType && setup.weapon.SwitchesDamageFreely {
		setup.weapon.NonLethal = !setup.weapon.NonLethal
		req.SwitchesDamageType = false
	}
	situations, err := s.specialSituations(ctx, req)
	if err != nil {
		return err
	}
	state, err := s.tables.State(ctx, req.SessionID)
	if err != nil {
		return err
	}
	setup.situations = withoutWhatTheAimCancels(state, setup.attacker.ID, req.TargetEntryID, situations)
	return nil
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

// withoutWhatTheAimCancels tira o -5 de atirar em quem está em combate corpo a corpo quando quem ataca
// mirou nesse alvo nesta vez (p234). O motor nem precisa saber: a linha não chega a ele.
func withoutWhatTheAimCancels(
	state *live.SessionRuntimeState, attacker, target string, situations []engine.SpecialSituation,
) []engine.SpecialSituation {
	if state == nil || state.Scene == nil || state.Scene.AimedBy != attacker || state.Scene.AimedAt != target {
		return situations
	}
	return slices.DeleteFunc(situations, func(situation engine.SpecialSituation) bool {
		return situation == engine.TargetInMeleeCombat
	})
}
