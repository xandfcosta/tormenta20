package session

import (
	"context"
	"fmt"

	"t20engine/domain/live"
	"t20engine/infra/events"
)

// O QUE A FILA ESPELHA DA FICHA, e a divisa que decide quem é a fonte.
//
// É a segunda razão de mudar do `store.go`, e ela tem nome: toda mutação aqui
// faz a MESMA pergunta antes de agir — "há ficha atrás desta linha?". Com
// ficha, quem manda é a ficha e a fila ESPELHA o resultado; sem, a linha é a
// fonte, porque ficha ele não tem.
//
// Escrever só num dos dois daria o caso do NPC certo e o do PC errado EM
// SILÊNCIO, que é exatamente a divergência que a fila espelhada existe para não
// ter — e é a razão de o regime não aplicar nada por conta própria (ALE-364).
//
// O resto do `store.go` é a FILA e a CENA: quem entra, de quem é a vez, que
// cena está no ar. Aquilo não pergunta pela ficha em lugar nenhum.

// PatchVitals fixa os vitais de uma entrada. Mesma regra do delta sobre quem é a
// fonte; valor absoluto NÃO drena pool temporário, porque é uma afirmação sobre
// o total e não uma pancada.
func (st *Store) PatchVitals(ctx context.Context, sessionID int64, entryID string, hpCurrent, mpCurrent *int64) (*live.SessionRuntimeState, error) {
	charID := st.CharacterIDOf(sessionID, entryID)
	if charID == nil {
		return st.apply(ctx, sessionID, vitalsEvent(sessionID, entryID, nil),
			patchEntryVitals(entryID, hpCurrent, mpCurrent))
	}
	hp, mp, err := st.sheet.ApplyAbsolute(ctx, *charID, hpCurrent, mpCurrent)
	if err != nil {
		return nil, err
	}
	return st.apply(ctx, sessionID, vitalsEvent(sessionID, entryID, charID),
		patchEntryVitals(entryID, hp, mp))
}

// DeltaCharacterVitals move os vitais de um PERSONAGEM, esteja ele na fila ou
// não.
//
// O irmão dele, o `DeltaVitals`, entra pela ENTRADA da fila, e é o caminho do
// COMBATE. Este entra pelo personagem, e é o caminho do ELENCO — onde metade da
// gente não tem linha na iniciativa durante a maior parte da sessão, que é a
// razão de o elenco existir separado da fila.
//
// Quem manda é a FICHA nos dois, e é isso que impede as duas telas de
// divergirem sobre o mesmo herói. A fila ESPELHA quando existe linha; quando não
// existe, não há o que espelhar e a ficha é a única a mudar — devolver o estado
// como está é a resposta certa, e não um erro, porque "não está na fila" é o
// caso comum aqui e não uma falha.
func (st *Store) DeltaCharacterVitals(ctx context.Context, sessionID, characterID int64, hpDelta, mpDelta *int64, nonLethal int64) (*live.SessionRuntimeState, error) {
	hp, mp, err := st.sheet.ApplyDelta(ctx, characterID, hpDelta, mpDelta, nonLethal)
	if err != nil {
		return nil, err
	}
	entryID, err := st.entryIDForCharacter(ctx, sessionID, characterID)
	if err != nil {
		return nil, err
	}
	if entryID == "" {
		return st.State(ctx, sessionID)
	}
	return st.apply(ctx, sessionID, vitalsEvent(sessionID, entryID, &characterID),
		patchEntryVitals(entryID, hp, mp))
}

// entryIDForCharacter é o inverso do `CharacterIDOf`, e devolve "" para quem não
// está na fila.
//
// Vazio e não erro: no elenco, estar FORA da iniciativa é o estado normal — o
// mestre cura a Arwen entre duas brigas —, e tratar isso como falha faria o
// gesto recusar exatamente o caso que ele veio atender.
func (st *Store) entryIDForCharacter(ctx context.Context, sessionID, characterID int64) (string, error) {
	state, err := st.State(ctx, sessionID)
	if err != nil {
		return "", err
	}
	for _, e := range state.Initiative {
		if e.CharacterID != nil && *e.CharacterID == characterID {
			return e.ID, nil
		}
	}
	return "", nil
}

// DeltaVitals move os vitais de uma entrada. Se há personagem atrás dela, quem
// manda é a FICHA: o delta é aplicado na linha do personagem (dano drenando PV
// temporários, como o endpoint de dano) e a entrada espelha o resultado. NPC não
// tem ficha — ali o rastreador é o registro.
func (st *Store) DeltaVitals(ctx context.Context, sessionID int64, entryID string, hpDelta, mpDelta *int64, nonLethal int64) (*live.SessionRuntimeState, error) {
	charID := st.CharacterIDOf(sessionID, entryID)
	if charID == nil {
		return st.apply(ctx, sessionID, vitalsEvent(sessionID, entryID, nil),
			deltaEntryVitals(entryID, hpDelta, mpDelta))
	}
	hp, mp, err := st.sheet.ApplyDelta(ctx, *charID, hpDelta, mpDelta, nonLethal)
	if err != nil {
		return nil, err
	}
	return st.apply(ctx, sessionID, vitalsEvent(sessionID, entryID, charID),
		patchEntryVitals(entryID, hp, mp))
}

// RefreshCharacterVitals repergunta o POÇO INTEIRO — máximo e atual — de toda
// entrada que tem personagem atrás.
//
// ELA DEVOLVE ERRO desde a ALE-373: era "melhor esforço", logava a piscada do
// banco e devolvia o instantâneo que tinha. O instantâneo que ela devolvia
// nessa hora era o da memória, com os poços de antes — a mesa lia números
// velhos achando que eram os de agora.
//
// # O atual TAMBÉM, e é isso que a ALE-358 consertou
//
// Ela chamava-se `RefreshCharacterMaxes` e refrescava só os tetos, com o atual
// declarado intocável. O efeito: sete gestos mudam o poço de um personagem e só
// DOIS contavam à fila — o dano pela própria fila e o mestre descansando o
// grupo. Os outros cinco são os da FICHA, e um jogador que bebesse uma poção,
// apanhasse pelo próprio crachá, conjurasse ou entrasse em Fúria deixava o
// mestre escolhendo alvo por um PV que não existia mais.
//
// # Por que sobrescrever é seguro
//
// Porque a entrada com personagem atrás NUNCA foi a autoridade sobre o atual: o
// `DeltaVitals` e o `PatchVitals` dizem, com todas as letras, que *"quem manda é
// a FICHA"* — eles escrevem nela e a linha espelha. A linha é o espelho, e
// espelho se redesenha.
//
// Quem NÃO tem personagem atrás — o NPC digitado — é pulado, e ali o rastreador
// continua sendo o registro. É a mesma fronteira que os dois gestos acima usam.
//
// # E o aparo some junto
//
// Havia um `clampCurrentTo` aqui para o caso de o máximo ENCOLHER com o atual
// acima dele. Ele deixou de ter caso: o atual vem do poço derivado, que é
// `máximo − dano` e já nasce na faixa (ALE-355).
func (st *Store) RefreshCharacterVitals(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	st.Mu.Lock()
	s, err := st.cachedLocked(ctx, sessionID)
	var ids []int64
	if err == nil {
		ids = uniqueCharacterIDs(s)
	}
	st.Mu.Unlock()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return st.State(ctx, sessionID)
	}
	pools, err := st.sheet.PoolsOf(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("reler os poços da sessão %d: %w", sessionID, err)
	}
	st.Mu.Lock()
	defer st.Mu.Unlock()
	saved, err := st.snapshotsFor(ctx).Mutate(ctx, sessionID, func(s *live.SessionRuntimeState) error {
		for i := range s.Initiative {
			e := &s.Initiative[i]
			if e.CharacterID == nil {
				continue
			}
			if fresh, ok := pools[*e.CharacterID]; ok {
				e.HpMax, e.HpCurrent = live.PtrInt64(fresh.HpMax), live.PtrInt64(fresh.HpCurrent)
				e.MpMax, e.MpCurrent = live.PtrInt64(fresh.MpMax), live.PtrInt64(fresh.MpCurrent)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("gravar os poços refrescados da sessão %d: %w", sessionID, err)
	}
	st.States[sessionID] = saved
	return live.CloneState(saved), nil
}

func uniqueCharacterIDs(s *live.SessionRuntimeState) []int64 {
	seen := map[int64]bool{}
	ids := []int64{}
	for _, e := range s.Initiative {
		if e.CharacterID != nil && !seen[*e.CharacterID] {
			seen[*e.CharacterID] = true
			ids = append(ids, *e.CharacterID)
		}
	}
	return ids
}

func vitalsEvent(sessionID int64, entryID string, charID *int64) events.VitalsChanged {
	ev := events.VitalsChanged{SessionID: sessionID, EntryID: entryID}
	if charID != nil {
		ev.CharacterID = *charID
	}
	return ev
}
