package session

import (
	"context"

	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// A MANUTENÇÃO DO TURNO: quem entra na vez paga 1 PM por habilidade sustentada,
// e a que não for paga termina (T20 p227).
//
// Ela é AÇÃO LIVRE, então não toca no orçamento do turno — quem sustenta
// Velocidade entra na vez com a padrão e a de movimento inteiras (p233).

// payUpkeep cobra as sustentadas de quem acabou de entrar na vez.
//
// O ERRO É ENGOLIDO de propósito, e é a única saída defensável: a alternativa é
// o turno não virar porque a ficha não pôde ser lida, e uma mesa travada no
// turno de alguém é pior que uma manutenção não cobrada. O extrato fica nil, a
// faixa não diz nada, e o mestre segue jogando.
func (st *Store) payUpkeep(s *live.SessionRuntimeState) upkeepCharge {
	if !s.Scene.CountsRounds() || st.turnEffects == nil {
		return upkeepCharge{}
	}
	if s.TurnIndex < 0 || s.TurnIndex >= len(s.Initiative) {
		return upkeepCharge{}
	}
	entry := s.Initiative[s.TurnIndex]
	if entry.CharacterID == nil {
		return upkeepCharge{}
	}
	effects, err := st.turnEffects.SustainedOf(context.Background(), *entry.CharacterID)
	if err != nil || len(effects) == 0 {
		return upkeepCharge{}
	}
	ids := make([]string, 0, len(effects))
	name := make(map[string]string, len(effects))
	for _, e := range effects {
		ids = append(ids, e.CatalogID)
		name[e.CatalogID] = e.Label
	}
	// O MANA VEM DA FICHA, e não da linha da fila.
	//
	// A linha ESPELHA a ficha, e o espelho pode estar vazio: um combatente
	// acabado de entrar na fila tem `MpCurrent` nulo até a primeira operação de
	// vitais. Ler dali derrubaria a Velocidade de quem está com o mana cheio, e
	// o teste que prende isto começou vermelho exatamente assim.
	pools, err := st.ficha.PoolsOf(context.Background(), []int64{*entry.CharacterID})
	if err != nil {
		return upkeepCharge{}
	}
	pool := pools[*entry.CharacterID]
	mana := int(pool.MpCurrent)
	// O INSTANTE é a vez de quem entrou, e PODER AGIR é ter PV: a 0 "você cai
	// inconsciente" (p236), e o poço do app tem piso em zero, então é aqui que
	// morrer e sangrar se encontram. O que uma ação LIVRE exige do instante
	// quem sabe é o motor.
	upkeep := engine.PaySustained(ids, mana, engine.ActionMoment{OnTurn: true, CanAct: pool.HpCurrent > 0})
	for _, id := range upkeep.Dropped {
		_ = st.turnEffects.EndSustained(context.Background(), *entry.CharacterID, id)
	}
	s.Scene.Upkeep = &live.TurnUpkeep{
		Paid: labelsOf(upkeep.Paid, name), Dropped: labelsOf(upkeep.Dropped, name), Cost: upkeep.Cost,
		MpBefore: mana, MpAfter: mana - upkeep.Cost, Unconscious: upkeep.Unconscious,
	}
	// O MANA SAI DEPOIS, e a cobrança VOLTA em vez de ficar guardada: quem
	// grava na ficha e espelha na fila é o `DeltaVitals`, que toma o mesmo
	// cadeado — chamá-lo daqui de dentro travaria a sessão contra si mesma. Um
	// campo no `Store` seria pior ainda: ele é de TODAS as sessões, e duas mesas
	// virando o turno ao mesmo tempo trocariam de cobrança.
	return upkeepCharge{entryID: entry.ID, pm: upkeep.Cost}
}

// labelsOf troca os ids pelos nomes que a mesa lê.
func labelsOf(ids []string, name map[string]string) []string {
	if len(ids) == 0 {
		return nil
	}
	labels := make([]string, 0, len(ids))
	for _, id := range ids {
		labels = append(labels, name[id])
	}
	return labels
}

// upkeepCharge é o que a manutenção decidiu tirar de quem entrou na vez.
type upkeepCharge struct {
	entryID string
	pm      int
}

// expireTurnEffects derruba, de TODA a fila, os efeitos que duravam a vez que
// está acabando.
//
// A VEZ É A EM CURSO, e não a de quem carrega o efeito. Uma duração de "1
// turno" chega por REAÇÃO — o Escudo da Fé é o único caso do catálogo (p192) —,
// e "uma reação pode ocorrer mesmo fora do seu turno" (p233): ancorá-la em quem
// a recebeu daria a um alvo que ainda não jogou uma rodada inteira de escudo.
// Por isso a varredura é da fila toda e não do combatente que entra.
//
// O erro é engolido pela mesma razão que o `payUpkeep` explica: uma mesa travada
// no turno de alguém é pior que um efeito que sobrou.
func (st *Store) expireTurnEffects(s *live.SessionRuntimeState) {
	if !s.Scene.CountsRounds() || st.turnEffects == nil {
		return
	}
	for _, entry := range s.Initiative {
		if entry.CharacterID == nil {
			continue
		}
		_ = st.turnEffects.ExpireTurnEffects(context.Background(), *entry.CharacterID)
	}
}
