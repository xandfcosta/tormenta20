package session

import (
	"context"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/events"
)

// AS TRÊS MUTAÇÕES DO ATAQUE PROPOSTO (ALE-364).
//
// Elas moram fora do `store.go` porque têm outra razão para mudar: o resto do
// store é a fila e os vitais, e isto é o ciclo propor-confirmar-cancelar de um
// gesto de combate. O que as junta a ele é o mesmo estado de sessão, e não o
// mesmo assunto.

// ProposeAttack guarda o ataque rolado. Ninguém perde PV aqui.
func (st *Store) ProposeAttack(ctx context.Context, sessionID int64, attack live.PendingAttack) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.AttackRolled{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { return live.ProposeAttack(s, attack) })
}

// CommitAttack aplica o dano do provisório e o tira da mesa.
//
// O DANO PASSA PELO `DeltaVitals`, e é aí que mora a razão de este método
// existir em vez de o regime aplicar sozinho: quando há FICHA atrás da linha,
// quem manda é a ficha — o dano drena PV temporário e a fila espelha o
// resultado; quando é NPC, o rastreador é o registro. O regime não alcança a
// porta da ficha, então aplicar lá daria o caso do NPC certo e o do PC errado
// em silêncio (ALE-364).
//
// São DUAS mutações, e a ordem importa: o dano primeiro, o provisório depois.
// Invertida, uma falha ao gravar o PV deixaria a mesa sem o provisório e sem o
// dano — e ninguém saberia que o ataque existiu.
func (st *Store) CommitAttack(ctx context.Context, sessionID int64, who live.Attacker) (*live.SessionRuntimeState, error) {
	state, err := st.State(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	attack, err := live.AttackToCommit(state, who)
	if err != nil {
		return nil, err
	}
	// AGREDIR É AÇÃO PADRÃO (p233), cobrada de quem ataca SE ele ainda está na
	// vez — a mesma regra do movimento: o mestre confirma fora de hora o tempo
	// todo, e cobrar do turno de outro tiraria a ação de quem não atacou. A
	// CONFERÊNCIA vem antes do dano e a COBRANÇA depois, pela razão do
	// `ActionFits`: entre propor e confirmar o jogador pode ter conjurado, e o
	// dano de um ataque que não cabia mais não pode pousar.
	onTurn := attackerIsOnTurn(state, attack)
	if onTurn {
		if err := st.ActionFits(ctx, sessionID, engine.ActionStandard); err != nil {
			return nil, err
		}
	}
	// O ALVO-PEÇA NÃO PASSA POR AQUI, e não é esquecimento: o PV de um objeto
	// mora no TABULEIRO, que é outro agregado com trava própria. Quem o escreve
	// é a cena, antes de chamar este método — aqui o provisório só é liquidado.
	// Deixar o `DeltaVitals` rodar com um `TargetEntryID` vazio tiraria PV de
	// uma linha que não existe, ou de nenhuma, em silêncio.
	if attack.Damage > 0 && attack.TargetTokenID == "" {
		loss := int64(-attack.Damage)
		// O NÃO LETAL VIAJA COM A PANCADA: a arma Piedosa (p336) causa dano que
		// derruba e não mata, e é aqui que a parcela entra na ficha. O `min` é
		// a rede: a RD pode ter comido parte do dano, e a parcela não pode
		// passar do que o alvo de fato perdeu.
		if _, err := st.DeltaVitals(ctx, sessionID, attack.TargetEntryID, &loss, nil,
			int64(min(attack.NonLethal, attack.Damage))); err != nil {
			return nil, err
		}
	}
	// A CONDIÇÃO DA MANOBRA NÃO POUSA AQUI, e a ausência é a regra e não um
	// esquecimento: o sistema INFORMA, o mestre DECIDE (ver o `CLAUDE.md` da
	// raiz). Este bloco aplicava `m.Imposes` sozinho — um embate de dados em que
	// o motor anunciava o vencedor e punha a condição no alvo.
	//
	// O que o livro prevê continua chegando à mesa: o provisório leva o
	// `ConditionOnAWin`, a faixa o mostra, e o mestre marca a condição no gesto
	// que já existe para isso. O conhecimento não sumiu; mudou de dono.
	return st.apply(ctx, sessionID, events.AttackSettled{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error {
			// Conferido DE NOVO sob a trava: entre a leitura lá em cima e esta
			// linha, outro pedido pode ter cancelado o mesmo provisório.
			if _, err := live.AttackToCommit(s, who); err != nil {
				return err
			}
			live.ClearPendingAttack(s)
			if !onTurn {
				return nil
			}
			return chargeTurn(s, engine.ActionStandard)
		})
}

// attackerIsOnTurn diz se quem atacou é a linha da vez em curso.
func attackerIsOnTurn(s *live.SessionRuntimeState, attack live.PendingAttack) bool {
	if s == nil || s.TurnIndex < 0 || s.TurnIndex >= len(s.Initiative) {
		return false
	}
	return s.Initiative[s.TurnIndex].ID == attack.AttackerEntryID
}

// CancelAttack descarta o provisório sem mexer em ninguém.
func (st *Store) CancelAttack(ctx context.Context, sessionID int64, who live.Attacker) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.AttackSettled{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { return live.CancelAttack(s, who) })
}
