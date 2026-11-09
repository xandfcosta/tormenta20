package live

import "slices"

// A VEZ PASSANDO, para a frente e para trás.

// AS MUTAÇÕES DA FILA SÃO PÚBLICAS, e isso é a fronteira e não conveniência
// (ALE-344).
//
// Elas eram privadas porque o único chamador morava no mesmo pacote: o STORE.
// Com ele em `app/session`, onde a orquestração pertence, o que sobra aqui é a
// regra — e a regra de um pacote de domínio existe para ser chamada de fora.
// O `domain/board` sempre foi assim: o `AddToken` e os irmãos dele já eram
// públicos, e a assimetria entre os dois era acidente de história.
//
// O que elas NÃO fazem continua sendo o que as define: nenhuma trava, nenhuma
// gravação, nenhum aviso. Recebem o estado, mudam o estado.
func AdvanceTurn(st *SessionRuntimeState) {
	// SÓ A CENA DE AÇÃO tem vez a passar (p252): numa conversa na corte não há
	// rodada, e "passar o turno" não quer dizer nada. Antes a condição era "a
	// cena está ligada", que era a mesma coisa só porque a única cena que o app
	// sabia abrir era o combate.
	if !st.CountsRounds() || len(st.Initiative) == 0 {
		return
	}
	dropTheSettledMiss(st)
	if st.TurnIndex < 0 {
		st.TurnIndex = 0
		if st.Round < 1 {
			st.Round = 1
		}
		st.TurnsTaken++
		return
	}
	leavesTheTurnBehind(st)
	st.TurnIndex++
	st.TurnsTaken++
	if st.TurnIndex >= len(st.Initiative) {
		st.TurnIndex = 0
		st.Round++
	}
	RefreshTurn(st)
}

// dropTheSettledMiss tira da mesa o ataque que o mestre já disse que ERROU. Ele
// fica para o jogador ler a decisão, e a vez mudando é o fim dessa leitura. O
// ataque ainda sem decisão fica: o mestre não falou, e mudar a vez não é falar.
func dropTheSettledMiss(st *SessionRuntimeState) {
	if st.PendingAttack != nil && st.PendingAttack.Ruling == RulingMiss {
		st.PendingAttack = nil
	}
}

// RewindTurn desfaz um "Próximo turno". Cruzar a virada de volta devolve a
// rodada; desfazer o primeiro turno devolve ao pré-combate (turnIndex -1) sem
// zerar a rodada, porque a rodada 1 JÁ começou e voltar não desfaz isso. Quem
// volta encontra o turno como o deixou (`takesTheTurnBack`).
func RewindTurn(st *SessionRuntimeState) {
	if len(st.Initiative) == 0 || st.TurnIndex < 0 {
		return
	}
	dropTheSettledMiss(st)
	// Desfazer um turno desconta um turno: o contador é o que JÁ aconteceu, e
	// voltar diz que não aconteceu.
	if st.TurnsTaken > 0 {
		st.TurnsTaken--
	}
	switch {
	case st.TurnIndex > 0:
		st.TurnIndex--
	case st.Round > 1:
		st.TurnIndex = len(st.Initiative) - 1
		st.Round--
	default:
		st.TurnIndex = -1
	}
	takesTheTurnBack(st)
}

// TurnLeftBehind é o que o "Próximo" guarda do turno que passou: o que sobrou de ação, o ataque a mais
// aberto e a mira.
type TurnLeftBehind struct {
	EntryID           string   `json:"entryId"`
	StandardLeft      bool     `json:"standardLeft"`
	MovementLeft      bool     `json:"movementLeft"`
	ExtraMovementLeft bool     `json:"extraMovementLeft,omitempty"`
	ExtraStrikeEntry  string   `json:"extraStrikeEntry,omitempty"`
	ExtraStrikeBarred string   `json:"extraStrikeBarred,omitempty"`
	ExtraStrikesUsed  []string `json:"extraStrikesUsed,omitempty"`
	ExtraStyle        string   `json:"extraStyle,omitempty"`
	ExtraStyleItem    int64    `json:"extraStyleItem,omitempty"`
	AimedBy           string   `json:"aimedBy,omitempty"`
	AimedAt           string   `json:"aimedAt,omitempty"`
}

// turnsKeptBehind é até onde o "voltar a vez" lembra. Ele é conserto de mesa, de uma ou duas vezes, e o
// retrato da sessão é gravado a cada gesto: lembrar a cena inteira só o faria crescer.
const turnsKeptBehind = 16

// leavesTheTurnBehind guarda o turno de quem está na vez, antes de ele passar.
func leavesTheTurnBehind(st *SessionRuntimeState) {
	if st.Scene == nil || st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return
	}
	s := st.Scene
	st.Scene.TurnsBehind = append(st.Scene.TurnsBehind, TurnLeftBehind{
		EntryID: st.Initiative[st.TurnIndex].ID, StandardLeft: s.StandardLeft, MovementLeft: s.MovementLeft,
		ExtraMovementLeft: s.ExtraMovementLeft, ExtraStrikeEntry: s.ExtraStrikeEntry, ExtraStrikeBarred: s.ExtraStrikeBarred,
		ExtraStrikesUsed: s.ExtraStrikesUsed, ExtraStyle: s.ExtraStyle, ExtraStyleItem: s.ExtraStyleItem,
		AimedBy: s.AimedBy, AimedAt: s.AimedAt,
	})
	if extra := len(s.TurnsBehind) - turnsKeptBehind; extra > 0 {
		s.TurnsBehind = s.TurnsBehind[extra:]
	}
}

// takesTheTurnBack devolve o turno a quem voltou a ser a vez. Sem retrato dele (voltou para antes do que se
// lembra, ou a fila mudou), o turno é cheio: o que era de quem estava na vez não passa para ele.
func takesTheTurnBack(st *SessionRuntimeState) {
	if st.Scene == nil {
		return
	}
	s := st.Scene
	last := len(s.TurnsBehind) - 1
	if last < 0 || st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) || s.TurnsBehind[last].EntryID != st.Initiative[st.TurnIndex].ID {
		s.TurnsBehind = nil
		RefreshTurn(st)
		return
	}
	turn := s.TurnsBehind[last]
	s.TurnsBehind = slices.Clip(s.TurnsBehind[:last])
	s.StandardLeft, s.MovementLeft, s.ExtraMovementLeft = turn.StandardLeft, turn.MovementLeft, turn.ExtraMovementLeft
	s.ExtraStrikeEntry, s.ExtraStrikeBarred, s.ExtraStrikesUsed = turn.ExtraStrikeEntry, turn.ExtraStrikeBarred, turn.ExtraStrikesUsed
	s.ExtraStyle, s.ExtraStyleItem = turn.ExtraStyle, turn.ExtraStyleItem
	s.AimedBy, s.AimedAt = turn.AimedBy, turn.AimedAt
}
