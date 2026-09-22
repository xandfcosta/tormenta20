package session

import (
	"context"
	"log"
	"sync"

	"t20engine/domain/live"
	"t20engine/infra/events"
)

// Store guarda o estado vivo de cada sessão em memória: protege as
// mutações puras (session_state.go) com um mutex e grava em
// `Session.runtimeState`. Reiniciar o servidor zera os rastreadores até o
// primeiro `Load()` reidratar do banco, e cada mutação devolve um instantâneo
// para quem chama serializar e transmitir fora da trava.
type Store struct {
	Mu sync.Mutex
	// sheet é a PORTA para o contexto da ficha: o regime escreve PV e PM de
	// personagem, mas as REGRAS dessa escrita são de lá. Nulo é caminho normal em
	// teste de regime puro — quem tem personagem na fila injeta o implementador.
	sheet live.SheetVitals
	// turnEffects é a porta do que o GIRO DA VEZ faz com os efeitos da ficha
	// (p227). Separada da `sheet` porque muda por outra razão; o mesmo
	// adaptador cumpre as duas.
	turnEffects live.SheetTurnEffects
	// States é o CACHE da mesa (ver `cache.go`): quem manda é o retrato gravado.
	States map[int64]*live.SessionRuntimeState
	// seqs numera as mutações de cada sessão, para o hub reconhecer quadro
	// atrasado. Mora aqui e não no estado: hidratar do banco troca o estado, e um
	// contador que vivesse nele voltaria a zero.
	seqs  map[int64]uint64
	newID func() string
	// snapshots é o RETRATO no banco, e é ele a fonte da verdade: toda mutação
	// passa por ele antes de existir para alguém (ALE-371).
	//
	// É o retrato SOLTO, para o gesto que só mexe na fila. O gesto que também
	// escreve na FICHA usa o da unidade (ver `unit.go`).
	snapshots SessionSnapshots
	// units abre a unidade de trabalho de um gesto: uma transação para a fila e
	// a ficha juntas (ALE-373).
	units Units
	// bus é por onde as mutações desta sessão viram notícia.
	//
	// Quem publica é o `apply`, DEPOIS de soltar a trava, e o evento diz o que
	// aconteceu em vez de só tocar o sino.
	//
	// Ponteiro e não valor porque ele é COMPARTILHADO com o tabuleiro e com o
	// servidor: um barramento por store devolveria a quem escuta o trabalho de
	// juntar as peças de novo. Nulo EXPLODE, e é para explodir: quem monta um store
	// à mão sem barramento descobre no primeiro `apply`, e não numa tela que não
	// atualiza.
	bus *events.Bus
}

// NewStore recebe a PORTA da ficha por parâmetro — injetada e não
// importada, que é o que impede o regime de conhecer as regras da ficha.
func NewStore(
	snapshots SessionSnapshots, units Units, newID func() string,
	sheet live.SheetVitals, turnEffects live.SheetTurnEffects, bus *events.Bus,
) *Store {
	return &Store{
		States:      map[int64]*live.SessionRuntimeState{},
		seqs:        map[int64]uint64{},
		newID:       newID,
		sheet:       sheet,
		turnEffects: turnEffects,
		snapshots:   snapshots,
		units:       units,
		bus:         bus,
	}
}

// nextSeqLocked devolve a ordem da próxima mutação desta sessão. Chamada SEMPRE
// com `st.Mu` seguro, que é o que faz a numeração coincidir com a ordem real
// das mutações.
//
// O contador mora na LOJA e não no estado: hidratar do banco substitui o estado
// e zeraria um contador que vivesse nele, e aí o hub descartaria todo quadro
// seguinte por achá-los atrasados. O contador só reinicia com o processo, que é
// quando o hub também esquece o que já mandou.
func (st *Store) nextSeqLocked(sessionID int64) uint64 {
	st.seqs[sessionID]++
	return st.seqs[sessionID]
}

// LiveSessionsWithCharacter devolve as sessões EM MEMÓRIA que têm este
// personagem na fila.
//
// Só as vivas, e isso é o ponto: o aviso serve para atualizar tela aberta. Mesa
// que ninguém está olhando não precisa ser avisada — quem entrar depois busca o
// estado do zero.
func (st *Store) LiveSessionsWithCharacter(characterID int64) []int64 {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	var out []int64
	for sessionID, state := range st.States {
		for _, entry := range state.Initiative {
			if entry.CharacterID != nil && *entry.CharacterID == characterID {
				out = append(out, sessionID)
				break
			}
		}
	}
	return out
}

// GetState devolve um instantâneo do estado atual, lido do banco na primeira
// vez. Se a leitura falha, o instantâneo é um rastreador VAZIO e NÃO fica em
// memória: a próxima pergunta tenta o banco de novo.
func (st *Store) GetState(sessionID int64) *live.SessionRuntimeState {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	s, err := st.cachedLocked(context.Background(), sessionID)
	if err != nil {
		log.Printf("session %d: %v", sessionID, err)
		return live.EmptyRuntimeState()
	}
	return live.CloneState(s)
}

// apply roda uma mutação pura sob a trava, publica o evento e devolve o
// instantâneo para o broadcast.
//
// O EVENTO É PARÂMETRO OBRIGATÓRIO, e é isso que substitui uma promessa por uma
// garantia: não dá para mutar sem dizer O QUE aconteceu, e o compilador cobra.
//
// A publicação sai FORA da trava. O barramento é folha e poderia ser chamado de
// dentro (ver `events.Bus.Publish`), mas quem acorda agora sabe o que houve e
// pode ler o estado na hora — publicar sob a trava faria esse leitor esperar
// pelo escritor no exato instante em que foi acordado para ler.
func (st *Store) apply(sessionID int64, ev events.Event, fn func(*live.SessionRuntimeState) error) (*live.SessionRuntimeState, error) {
	clone, err := st.applyLocked(sessionID, fn)
	if err != nil {
		return nil, err
	}
	st.bus.Publish(ev)
	return clone, nil
}

// applyLocked é a parte que precisa da trava: mutar e tirar o retrato.
//
// A `seq` nasce AQUI DENTRO e não na publicação: ela numera as mutações para o
// hub reconhecer quadro atrasado, e decidir a sequência e entregar têm de ser
// atômicos.
func (st *Store) applyLocked(sessionID int64, fn func(*live.SessionRuntimeState) error) (*live.SessionRuntimeState, error) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	// A GRAVAÇÃO É A MUTAÇÃO (ALE-371): o `Mutate` lê o retrato, deixa a regra
	// mudá-lo e grava, tudo numa transação. O que não gravou não aconteceu, e o
	// erro sobe para quem clicou.
	//
	// A trava continua aqui, e agora ela guarda uma coisa só: a ORDEM. A `seq`
	// numera as mutações para o hub reconhecer quadro atrasado, e numerar fora
	// da ordem em que o banco as aceitou entregaria à tela o quadro velho por
	// último.
	saved, err := st.snapshots.Mutate(context.Background(), sessionID, fn)
	if err != nil {
		return nil, err
	}
	st.States[sessionID] = saved
	clone := live.CloneState(saved)
	clone.Seq = st.nextSeqLocked(sessionID)
	return clone, nil
}

func (st *Store) AddInitiativeEntry(sessionID int64, e live.InitiativeEntry) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.CombatantJoined{SessionID: sessionID, EntryID: e.ID},
		func(s *live.SessionRuntimeState) error { return live.AddEntry(s, e, st.newID) })
}

func (st *Store) UpsertInitiativeEntry(sessionID int64, e live.InitiativeEntry) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.CombatantJoined{SessionID: sessionID, EntryID: e.ID},
		func(s *live.SessionRuntimeState) error { return live.UpsertCharacterEntry(s, e, st.newID) })
}

func (st *Store) UpdateInitiativeEntry(sessionID int64, entryID string, patch live.EntryPatch) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.CombatantChanged{SessionID: sessionID, EntryID: entryID},
		func(s *live.SessionRuntimeState) error { return live.UpdateEntry(s, entryID, patch) })
}

func (st *Store) RemoveInitiativeEntry(sessionID int64, entryID string) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.CombatantLeft{SessionID: sessionID, EntryID: entryID},
		func(s *live.SessionRuntimeState) error { return live.RemoveEntry(s, entryID) })
}

// NextTurn passa a vez, e é UM gesto: a fila, os efeitos que acabam com a vez,
// a manutenção das sustentadas e o teste de sangramento gravam na MESMA
// transação (ALE-373).
//
// TUDO-OU-NADA, e isto é decisão do dono que inverteu a de antes. Aqui as
// escritas da ficha eram engolidas com `_ =`, e a razão escrita era boa: "uma
// mesa travada no turno de alguém é pior que uma manutenção não cobrada". O
// preço dela era pior — a vez passava com o efeito ainda ligado na ficha, sem
// ninguém saber (ALE-372). Agora a escrita que falha RECUSA o clique.
//
// LER TAMBÉM: qualquer erro do banco recusa, e não só os de escrita. Quem não
// consegue ler a ficha não sabe o que expirar nem o que cobrar, e seguir é
// afirmar que não havia nada — decisão do dono, e é o princípio que sustenta a
// fatia inteira: sem o banco não temos certeza de nada.
func (st *Store) NextTurn(sessionID int64) (*live.SessionRuntimeState, error) {
	return st.turnGesture(sessionID, func(u Unit, s *live.SessionRuntimeState) error {
		// O FIM DA VEZ vem ANTES do começo da próxima: o que durava a vez que
		// acaba tem de sair antes de alguém entrar na sua.
		if err := st.expireTurnEffects(u, s); err != nil {
			return err
		}
		live.AdvanceTurn(s)
		return st.payUpkeep(u, s)
	})
}

// turnGesture é o corpo comum de quem gira a vez: abre a unidade, muta o
// retrato dentro dela e só publica DEPOIS de a transação fechar — anunciar
// antes seria contar à mesa um turno que o disco ainda pode recusar.
func (st *Store) turnGesture(
	sessionID int64, change func(Unit, *live.SessionRuntimeState) error,
) (*live.SessionRuntimeState, error) {
	st.Mu.Lock()
	var clone *live.SessionRuntimeState
	err := st.units.Do(context.Background(), func(u Unit) error {
		saved, err := u.Snapshots.Mutate(context.Background(), sessionID,
			func(s *live.SessionRuntimeState) error {
				if err := change(u, s); err != nil {
					return err
				}
				return st.openBleedingCheck(u, s)
			})
		if err != nil {
			return err
		}
		st.States[sessionID] = saved
		clone = live.CloneState(saved)
		clone.Seq = st.nextSeqLocked(sessionID)
		return nil
	})
	st.Mu.Unlock()
	if err != nil {
		return nil, err
	}
	st.bus.Publish(events.TurnAdvanced{SessionID: sessionID})
	return clone, nil
}

// PreviousTurn volta a vez. Ele também é gesto de turno — a mesma unidade —,
// mas não desfaz o que a vez que passou consumiu: voltar é conserto de mesa, e
// ressuscitar efeito expirado exigiria guardar o que foi derrubado.
func (st *Store) PreviousTurn(sessionID int64) (*live.SessionRuntimeState, error) {
	return st.turnGesture(sessionID, func(_ Unit, s *live.SessionRuntimeState) error {
		live.RewindTurn(s)
		return nil
	})
}

func (st *Store) Reset(sessionID int64) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.InitiativeReset{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { live.ResetInitiative(s); return nil })
}

func (st *Store) StartScene(sessionID int64, kind live.SceneKind) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.SceneStarted{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { live.StartScene(s, kind); return nil })
}

func (st *Store) EndScene(sessionID int64) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.SceneEnded{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { live.EndScene(s); return nil })
}

// PatchVitals fixa os vitais de uma entrada. Mesma regra do delta sobre quem é a
// fonte; valor absoluto NÃO drena pool temporário, porque é uma afirmação sobre
// o total e não uma pancada.
func (st *Store) PatchVitals(sessionID int64, entryID string, hpCurrent, mpCurrent *int64) (*live.SessionRuntimeState, error) {
	charID := st.CharacterIDOf(sessionID, entryID)
	if charID == nil {
		return st.apply(sessionID, vitalsEvent(sessionID, entryID, nil),
			patchEntryVitals(entryID, hpCurrent, mpCurrent))
	}
	hp, mp, err := st.sheet.ApplyAbsolute(context.Background(), *charID, hpCurrent, mpCurrent)
	if err != nil {
		return nil, err
	}
	return st.apply(sessionID, vitalsEvent(sessionID, entryID, charID),
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
func (st *Store) DeltaCharacterVitals(sessionID, characterID int64, hpDelta, mpDelta *int64) (*live.SessionRuntimeState, error) {
	hp, mp, err := st.sheet.ApplyDelta(context.Background(), characterID, hpDelta, mpDelta)
	if err != nil {
		return nil, err
	}
	entryID := st.entryIDForCharacter(sessionID, characterID)
	if entryID == "" {
		return st.GetState(sessionID), nil
	}
	return st.apply(sessionID, vitalsEvent(sessionID, entryID, &characterID),
		patchEntryVitals(entryID, hp, mp))
}

// entryIDForCharacter é o inverso do `CharacterIDOf`, e devolve "" para quem não
// está na fila.
//
// Vazio e não erro: no elenco, estar FORA da iniciativa é o estado normal — o
// mestre cura a Arwen entre duas brigas —, e tratar isso como falha faria o
// gesto recusar exatamente o caso que ele veio atender.
func (st *Store) entryIDForCharacter(sessionID, characterID int64) string {
	for _, e := range st.GetState(sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == characterID {
			return e.ID
		}
	}
	return ""
}

// DeltaVitals move os vitais de uma entrada. Se há personagem atrás dela, quem
// manda é a FICHA: o delta é aplicado na linha do personagem (dano drenando PV
// temporários, como o endpoint de dano) e a entrada espelha o resultado. NPC não
// tem ficha — ali o rastreador é o registro.
func (st *Store) DeltaVitals(sessionID int64, entryID string, hpDelta, mpDelta *int64) (*live.SessionRuntimeState, error) {
	charID := st.CharacterIDOf(sessionID, entryID)
	if charID == nil {
		return st.apply(sessionID, vitalsEvent(sessionID, entryID, nil),
			deltaEntryVitals(entryID, hpDelta, mpDelta))
	}
	hp, mp, err := st.sheet.ApplyDelta(context.Background(), *charID, hpDelta, mpDelta)
	if err != nil {
		return nil, err
	}
	return st.apply(sessionID, vitalsEvent(sessionID, entryID, charID),
		patchEntryVitals(entryID, hp, mp))
}

// RefreshCharacterVitals repergunta o POÇO INTEIRO — máximo e atual — de toda
// entrada que tem personagem atrás. Melhor esforço: uma piscada do banco vira
// log e devolve o instantâneo atual em vez de derrubar a leitura.
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
func (st *Store) RefreshCharacterVitals(ctx context.Context, sessionID int64) *live.SessionRuntimeState {
	st.Mu.Lock()
	s, err := st.cachedLocked(ctx, sessionID)
	var ids []int64
	if err == nil {
		ids = uniqueCharacterIDs(s)
	}
	st.Mu.Unlock()
	if len(ids) == 0 {
		return st.GetState(sessionID)
	}
	pools, err := st.sheet.PoolsOf(ctx, ids)
	if err != nil {
		log.Printf("session %d: hpMax refresh failed (%v)", sessionID, err)
		return st.GetState(sessionID)
	}
	st.Mu.Lock()
	defer st.Mu.Unlock()
	saved, err := st.snapshots.Mutate(ctx, sessionID, func(s *live.SessionRuntimeState) error {
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
		log.Printf("session %d: refrescar os poços falhou (%v)", sessionID, err)
		return live.EmptyRuntimeState()
	}
	st.States[sessionID] = saved
	return live.CloneState(saved)
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

// vitalsEvent monta o evento do dano ou da cura.
//
// O `CharacterID` só entra quando HÁ ficha atrás da linha, e o zero do NPC é
// significativo: ele é o que impede o dano num ogro de acordar toda ficha do
// processo, porque um alvo zero casaria com um interesse zero. Ver
// `TestNpcVitalsWakeNoSheet`.
func vitalsEvent(sessionID int64, entryID string, charID *int64) events.VitalsChanged {
	ev := events.VitalsChanged{SessionID: sessionID, EntryID: entryID}
	if charID != nil {
		ev.CharacterID = *charID
	}
	return ev
}
