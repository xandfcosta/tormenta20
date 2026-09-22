package session

import (
	"context"
	"fmt"

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
// TODO ERRO DO BANCO SOBE — ler e escrever, sem distinção (ALE-373).
//
// Aqui o erro era engolido, com uma razão que parecia boa: "uma mesa travada no
// turno de alguém é pior que uma manutenção não cobrada". Ela cai por cima:
// quem não consegue LER a ficha não sabe o que cobrar, e seguir é afirmar que
// não havia nada a cobrar. O banco é a fonte da verdade — sem ele não temos
// certeza de nada, e uma mesa que segue jogando sobre uma incerteza é pior que
// uma mesa parada, porque ninguém fica sabendo (decisão do dono).
func (st *Store) payUpkeep(u Unit, s *live.SessionRuntimeState) error {
	if !s.Scene.CountsRounds() || u.TurnEffects == nil {
		return nil
	}
	if s.TurnIndex < 0 || s.TurnIndex >= len(s.Initiative) {
		return nil
	}
	entry := s.Initiative[s.TurnIndex]
	if entry.CharacterID == nil {
		return nil
	}
	effects, err := u.TurnEffects.SustainedOf(context.Background(), *entry.CharacterID)
	if err != nil {
		return fmt.Errorf("ler as sustentadas de %s: %w", entry.Label, err)
	}
	if len(effects) == 0 {
		return nil
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
	pools, err := u.Sheet.PoolsOf(context.Background(), []int64{*entry.CharacterID})
	if err != nil {
		return fmt.Errorf("ler o mana de %s: %w", entry.Label, err)
	}
	pool := pools[*entry.CharacterID]
	mana := int(pool.MpCurrent)
	// O INSTANTE é a vez de quem entrou, e PODER AGIR é ter PV: a 0 "você cai
	// inconsciente" (p236), e o poço do app tem piso em zero, então é aqui que
	// morrer e sangrar se encontram. O que uma ação LIVRE exige do instante
	// quem sabe é o motor.
	upkeep := engine.PaySustained(ids, mana, engine.MomentFor(true, pool.HpCurrent, nil))
	for _, id := range upkeep.Dropped {
		if err := u.TurnEffects.EndSustained(context.Background(), *entry.CharacterID, id); err != nil {
			return fmt.Errorf("derrubar %s de %s: %w", name[id], entry.Label, err)
		}
	}
	s.Scene.Upkeep = &live.TurnUpkeep{
		Paid: labelsOf(upkeep.Paid, name), Dropped: labelsOf(upkeep.Dropped, name), Cost: upkeep.Cost,
		MpBefore: mana, MpAfter: mana - upkeep.Cost, Unconscious: upkeep.Unconscious,
	}
	if upkeep.Cost == 0 {
		return nil
	}
	// O MANA SAI AQUI DENTRO, na mesma transação e na mesma mutação.
	//
	// Ele saía depois, por um `DeltaVitals` de fora: chamá-lo daqui tomaria o
	// mesmo cadeado e travaria a sessão contra si mesma. Com a unidade não há
	// segundo cadeado nem segunda transação — a ficha é escrita pela porta da
	// unidade e a linha espelha o resultado, tudo antes de o retrato ser
	// gravado (ALE-373).
	spent := int64(-upkeep.Cost)
	hp, mp, err := u.Sheet.ApplyDelta(context.Background(), *entry.CharacterID, nil, &spent)
	if err != nil {
		return fmt.Errorf("cobrar a manutenção de %s: %w", entry.Label, err)
	}
	return live.PatchEntryVitals(s, entry.ID, hp, mp, hpFloorOf(s, entry.ID))
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

// expireTurnEffects derruba, de TODA a fila, os efeitos que duravam a vez que
// está acabando.
//
// A VEZ É A EM CURSO, e não a de quem carrega o efeito. Uma duração de "1
// turno" chega por REAÇÃO — o Escudo da Fé é o único caso do catálogo (p192) —,
// e "uma reação pode ocorrer mesmo fora do seu turno" (p233): ancorá-la em quem
// a recebeu daria a um alvo que ainda não jogou uma rodada inteira de escudo.
// Por isso a varredura é da fila toda e não do combatente que entra.
//
// O erro SOBE e desfaz a vez (ALE-373): um efeito que devia ter acabado e
// continua ligado na ficha é pior que um clique recusado, porque a mesa segue
// jogando sem saber.
func (st *Store) expireTurnEffects(u Unit, s *live.SessionRuntimeState) error {
	if !s.Scene.CountsRounds() || u.TurnEffects == nil {
		return nil
	}
	for _, entry := range s.Initiative {
		if entry.CharacterID == nil {
			continue
		}
		if err := u.TurnEffects.ExpireTurnEffects(context.Background(), *entry.CharacterID); err != nil {
			return fmt.Errorf("expirar os efeitos de vez de %s: %w", entry.Label, err)
		}
	}
	return nil
}
