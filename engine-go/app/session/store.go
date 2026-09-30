package session

import (
	"context"
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

// installWhenItCounts põe a mesa mudada no cache — AGORA, ou depois do commit
// quando há transação aberta. Ver o irmão em `boards.Store`, e a razão é a
// mesma: a memória não participa do `rollback` (ALE-376).
func (st *Store) installWhenItCounts(ctx context.Context, sessionID int64, saved *live.SessionRuntimeState) {
	if open, ok := UnitFrom(ctx); ok && open.AfterCommit != nil {
		open.AfterCommit(func() { st.States[sessionID] = saved })
		return
	}
	st.States[sessionID] = saved
}

// snapshotsFor devolve o retrato que ESTE gesto tem de usar: o da unidade
// aberta, quando o contexto carrega uma, e o do store no resto das vezes.
//
// Sem isto o `WithUnit` era meia rede (ALE-376). Ele já fazia um `Do` aninhado
// REUSAR a transação, mas o store seguia gravando pelo retrato do construtor —
// que é outra conexão do pool. O gesto que abria a unidade e chamava um método
// de store lá de dentro batia na própria trava e esperava o `busy_timeout`
// inteiro, que foi como este caso apareceu: 14s num teste de 0,1s.
//
// É a MESMA forma que o `boards.Store.snapshotsFor` usa, e pela mesma razão.
func (st *Store) snapshotsFor(ctx context.Context) SessionSnapshots {
	if open, ok := ctx.Value(unitKey{}).(Unit); ok && open.Snapshots != nil {
		return open.Snapshots
	}
	return st.snapshots
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
func (st *Store) apply(ctx context.Context, sessionID int64, ev events.Event, fn func(*live.SessionRuntimeState) error) (*live.SessionRuntimeState, error) {
	clone, err := st.applyLocked(ctx, sessionID, fn)
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
func (st *Store) applyLocked(ctx context.Context, sessionID int64, fn func(*live.SessionRuntimeState) error) (*live.SessionRuntimeState, error) {
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
	saved, err := st.snapshotsFor(ctx).Mutate(ctx, sessionID, fn)
	if err != nil {
		return nil, err
	}
	st.installWhenItCounts(ctx, sessionID, saved)
	clone := live.CloneState(saved)
	clone.Seq = st.nextSeqLocked(sessionID)
	return clone, nil
}

func (st *Store) AddInitiativeEntry(ctx context.Context, sessionID int64, e live.InitiativeEntry) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.CombatantJoined{SessionID: sessionID, EntryID: e.ID},
		func(s *live.SessionRuntimeState) error { return live.AddEntry(s, e, st.newID) })
}

func (st *Store) UpsertInitiativeEntry(ctx context.Context, sessionID int64, e live.InitiativeEntry) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.CombatantJoined{SessionID: sessionID, EntryID: e.ID},
		func(s *live.SessionRuntimeState) error { return live.UpsertCharacterEntry(s, e, st.newID) })
}

func (st *Store) UpdateInitiativeEntry(ctx context.Context, sessionID int64, entryID string, patch live.EntryPatch) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.CombatantChanged{SessionID: sessionID, EntryID: entryID},
		func(s *live.SessionRuntimeState) error { return live.UpdateEntry(s, entryID, patch) })
}

func (st *Store) RemoveInitiativeEntry(ctx context.Context, sessionID int64, entryID string) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.CombatantLeft{SessionID: sessionID, EntryID: entryID},
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
func (st *Store) NextTurn(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	return st.turnGesture(ctx, sessionID, func(ctx context.Context, u Unit, s *live.SessionRuntimeState) error {
		// O FIM DA VEZ vem ANTES do começo da próxima: o que durava a vez que
		// acaba tem de sair antes de alguém entrar na sua.
		if err := st.expireTurnEffects(ctx, u, s); err != nil {
			return err
		}
		live.AdvanceTurn(s)
		return st.payUpkeep(ctx, u, s)
	})
}

// turnGesture é o corpo comum de quem gira a vez: abre a unidade, muta o
// retrato dentro dela e só publica DEPOIS de a transação fechar — anunciar
// antes seria contar à mesa um turno que o disco ainda pode recusar.
//
// O CONTEXTO QUE DESCE DAQUI CARREGA A UNIDADE (ALE-374), e é isso que arma a
// rede do `Units.Do`: quem for chamado lá de dentro e abrir um `Do` REUSA a
// transação aberta em vez de pedir a segunda conexão do pool — que é o impasse
// da ALE-371, `SQLITE_BUSY` depois do `busy_timeout` inteiro. Sem esta linha a
// reentrância existe escrita e nunca dispara, porque nenhum contexto carrega
// unidade nenhuma.
func (st *Store) turnGesture(
	ctx context.Context, sessionID int64,
	change func(context.Context, Unit, *live.SessionRuntimeState) error,
) (*live.SessionRuntimeState, error) {
	st.Mu.Lock()
	var clone *live.SessionRuntimeState
	err := st.units.Do(ctx, func(u Unit) error {
		inUnit := WithUnit(ctx, u)
		saved, err := u.Snapshots.Mutate(inUnit, sessionID,
			func(s *live.SessionRuntimeState) error {
				if err := change(inUnit, u, s); err != nil {
					return err
				}
				return st.openBleedingCheck(inUnit, u, s)
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
func (st *Store) PreviousTurn(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	return st.turnGesture(ctx, sessionID, func(_ context.Context, _ Unit, s *live.SessionRuntimeState) error {
		live.RewindTurn(s)
		return nil
	})
}

// RestartCombat esvazia a fila e os turnos, pelo caminho das MUTAÇÕES.
//
// Ele escrevia direto na coluna (`queries.ResetSessionTracker`) e depois dava
// `Forget` no cache, e o comentário do `Lifecycle` explicava por que as duas
// metades eram obrigatórias. Passando pelo `apply`, as duas somem: o `Mutate`
// grava e o store instala o estado novo no cache — não há cache velho a
// esquecer (ALE-377).
//
// E o que isso compra de verdade é a UNIDADE: o `apply` usa o
// `snapshotsFor(ctx)`, então um gesto que abra a transação leva o esvaziamento
// junto. Escrevendo pelo caderno cru, o mesmo gesto bateria na própria trava e
// esperaria o `busy_timeout` (ALE-371).
func (st *Store) RestartCombat(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.InitiativeReset{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { *s = *live.EmptyRuntimeState(); return nil })
}

func (st *Store) Reset(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.InitiativeReset{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { live.ResetInitiative(s); return nil })
}

func (st *Store) StartScene(ctx context.Context, sessionID int64, kind live.SceneKind) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.SceneStarted{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { live.StartScene(s, kind); return nil })
}

func (st *Store) EndScene(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.SceneEnded{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { live.EndScene(s); return nil })
}
