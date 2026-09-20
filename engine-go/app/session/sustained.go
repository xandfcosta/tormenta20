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
	if !s.Scene.CountsRounds() || st.sustentadas == nil {
		return upkeepCharge{}
	}
	if s.TurnIndex < 0 || s.TurnIndex >= len(s.Initiative) {
		return upkeepCharge{}
	}
	entrada := s.Initiative[s.TurnIndex]
	if entrada.CharacterID == nil {
		return upkeepCharge{}
	}
	efeitos, err := st.sustentadas.SustainedOf(context.Background(), *entrada.CharacterID)
	if err != nil || len(efeitos) == 0 {
		return upkeepCharge{}
	}
	ids := make([]string, 0, len(efeitos))
	nome := make(map[string]string, len(efeitos))
	for _, e := range efeitos {
		ids = append(ids, e.CatalogID)
		nome[e.CatalogID] = e.Label
	}
	// O MANA VEM DA FICHA, e não da linha da fila.
	//
	// A linha ESPELHA a ficha, e o espelho pode estar vazio: um combatente
	// acabado de entrar na fila tem `MpCurrent` nulo até a primeira operação de
	// vitais. Ler dali derrubaria a Velocidade de quem está com o mana cheio, e
	// o teste que prende isto começou vermelho exatamente assim.
	pocos, err := st.ficha.PoolsOf(context.Background(), []int64{*entrada.CharacterID})
	if err != nil {
		return upkeepCharge{}
	}
	poco := pocos[*entrada.CharacterID]
	mana := int(poco.MpCurrent)
	// PODE AGIR é ter PV: a 0 "você cai inconsciente" (p236), e o poço do app
	// tem piso em zero, então é aqui que morrer e sangrar se encontram.
	feito := engine.PaySustained(ids, mana, poco.HpCurrent > 0)
	for _, id := range feito.Dropped {
		_ = st.sustentadas.EndSustained(context.Background(), *entrada.CharacterID, id)
	}
	s.Scene.Upkeep = &live.TurnUpkeep{
		Paid: labelsOf(feito.Paid, nome), Dropped: labelsOf(feito.Dropped, nome), Cost: feito.Cost,
		MpBefore: mana, MpAfter: mana - feito.Cost, Unconscious: feito.Unconscious,
	}
	// O MANA SAI DEPOIS, e a cobrança VOLTA em vez de ficar guardada: quem
	// grava na ficha e espelha na fila é o `DeltaVitals`, que toma o mesmo
	// cadeado — chamá-lo daqui de dentro travaria a sessão contra si mesma. Um
	// campo no `Store` seria pior ainda: ele é de TODAS as sessões, e duas mesas
	// virando o turno ao mesmo tempo trocariam de cobrança.
	return upkeepCharge{entryID: entrada.ID, pm: feito.Cost}
}

// labelsOf troca os ids pelos nomes que a mesa lê.
func labelsOf(ids []string, nome map[string]string) []string {
	if len(ids) == 0 {
		return nil
	}
	lidos := make([]string, 0, len(ids))
	for _, id := range ids {
		lidos = append(lidos, nome[id])
	}
	return lidos
}

// upkeepCharge é o que a manutenção decidiu tirar de quem entrou na vez.
type upkeepCharge struct {
	entryID string
	pm      int
}
