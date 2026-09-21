package live

// A CENA, como o livro a define (T20 p252).
//
// "O tempo narrativo de uma aventura de Tormenta20 é medido em CENAS. Uma cena
// não é uma unidade de tempo fixa, mas um pedaço distinto da história. Um
// combate é uma cena. Uma discussão noite adentro na corte da Rainha-Imperatriz
// é uma cena."
//
// Aqui morava um BOOLEANO — `SceneActive` —, e ele dizia outra coisa: "o
// combate está ligado". Duas consequências, e as duas apareciam como falta de
// recurso e não como erro:
//
//   - uma sessão tinha no máximo UMA cena, e o livro diz que uma sessão típica
//     tem três;
//   - para um poder "1/cena" zerar numa conversa, o mestre precisava abrir um
//     COMBATE — porque a única cena que o app sabia abrir contava rodadas.

// SceneKind é um dos três tipos do livro. "A maioria pode ser classificada em
// três tipos: cenas de ação, cenas de exploração e cenas de interpretação."
type SceneKind string

const (
	// SceneAction — "têm como objetivo conquistar algo fisicamente… o jogo é
	// dividido em rodadas, com cada personagem tendo sua vez de agir". É a
	// ÚNICA que mede tempo em rodadas.
	SceneAction SceneKind = "acao"
	// SceneExploration — atravessar o pântano, vasculhar a masmorra.
	SceneExploration SceneKind = "exploracao"
	// SceneRoleplay — a discussão na corte.
	SceneRoleplay SceneKind = "interpretacao"
)

// Name é o nome que a mesa lê.
func (k SceneKind) Name() string {
	switch k {
	case SceneAction:
		return "Ação"
	case SceneExploration:
		return "Exploração"
	case SceneRoleplay:
		return "Interpretação"
	}
	return ""
}

// Scene é a cena EM CURSO. Nil no estado quer dizer fora de cena.
type Scene struct {
	Kind SceneKind `json:"kind"`
	// StandardLeft e MovementLeft são a economia de ação do turno EM CURSO
	// (p233): uma padrão e uma de movimento, com uma troca de mão única entre
	// elas. Elas são DADO aqui e a regra mora no `engine.TurnBudget` — o regime
	// não alcança o motor (ver `boundary_test.go`), e duplicar a troca daria
	// duas versões da mesma assimetria para divergir.
	//
	// Só a cena de AÇÃO tem turno, então só ela as usa.
	StandardLeft bool `json:"standardLeft"`
	MovementLeft bool `json:"movementLeft"`
	// Number é a ordem dentro da sessão, começando em 1. Ele existe porque a
	// sessão tem uma SEQUÊNCIA de cenas — sem número, "a terceira cena da noite"
	// não tem como ser dita.
	Number int `json:"number"`
	// Upkeep é o extrato da manutenção que o início desta vez cobrou (p227) —
	// nil quando quem entrou não sustenta nada. Ele mora na cena, e não numa
	// notícia à parte, porque é estado do turno EM CURSO: quem recarrega a
	// página no meio do turno tem de ler a mesma coisa.
	Upkeep *TurnUpkeep `json:"upkeep,omitempty"`
}

// CountsRounds diz se esta cena mede tempo em rodadas. Só a de ação (p252).
func (s *Scene) CountsRounds() bool { return s != nil && s.Kind == SceneAction }

// InScene diz se há cena em curso.
func (st *SessionRuntimeState) InScene() bool { return st.Scene != nil }

// CountsRounds diz se a cena em curso mede tempo em rodadas.
func (st *SessionRuntimeState) CountsRounds() bool { return st.Scene.CountsRounds() }

// StartScene abre uma cena do tipo pedido.
//
// COM UMA CENA EM CURSO, ela INTERROMPE — e isso não é um erro, é o livro:
// "uma cena pode ser interrompida para dar lugar a outra… se os personagens
// estão discutindo na corte e de repente são atacados, a cena da discussão
// acaba e uma nova cena começa — um combate" (p252).
func StartScene(st *SessionRuntimeState, kind SceneKind) {
	if st.InScene() {
		EndScene(st)
	}
	st.ScenesSoFar++
	st.Scene = &Scene{Kind: kind, Number: st.ScenesSoFar, StandardLeft: true, MovementLeft: true}
}

// EndScene encerra a cena em curso e devolve o combate ao começo — mas GUARDA a
// fila.
//
// Encerrar não é reiniciar, e essa é a única diferença entre os dois: quem
// esvazia é o ResetInitiative. O mestre que encerra a briga do castelo não pode
// pagar oito goblins digitados de novo para recomeçá-la.
func EndScene(st *SessionRuntimeState) {
	st.Scene = nil
	st.Round = 0
	st.TurnIndex = -1
	st.TurnsTaken = 0
}

// TurnUpkeep é o que a manutenção das sustentadas cobrou ao entrar nesta vez.
//
// Os nomes são os que a MESA lê, e não os ids: quem olha a faixa precisa saber
// que "Velocidade" caiu, não que "velocidade" caiu.
type TurnUpkeep struct {
	Paid    []string `json:"paid,omitempty"`
	Dropped []string `json:"dropped,omitempty"`
	Cost    int      `json:"cost"`
	// Unconscious diz que tudo caiu porque quem entrou na vez está a 0 PV, e
	// não por falta de mana: manter é ação livre, e quem está no chão não age.
	Unconscious bool `json:"unconscious,omitempty"`
	// MpBefore e MpAfter são o poço ANTES e DEPOIS da manutenção. O gasto
	// sozinho diz o preço e não diz se dá para pagar no turno seguinte, que é a
	// decisão de quem sustenta.
	MpBefore int `json:"mpBefore"`
	MpAfter  int `json:"mpAfter"`
}

// RefreshTurn devolve o turno inteiro a quem acabou de entrar nele.
//
// Ela é chamada pelo `AdvanceTurn` e pelo irmão que volta: um turno que começa
// com a ação já gasta seria o turno de outra pessoa.
func RefreshTurn(st *SessionRuntimeState) {
	if st.Scene == nil {
		return
	}
	st.Scene.StandardLeft, st.Scene.MovementLeft = true, true
	// A MANUTENÇÃO DA VEZ ANTERIOR SAI JUNTO: ela é o extrato deste turno, e
	// deixá-la faria a faixa dizer que a Velocidade de outra pessoa caiu agora.
	st.Scene.Upkeep = nil
}

// actionsLeft escreve o que ainda cabe no turno.
//
// A frase diz a CONSEQUÊNCIA e não o mecanismo: com a padrão de pé e a de
// movimento gasta, ainda dá para andar — "você pode trocar sua ação padrão por
// uma ação de movimento" (p233) —, e um "padrão" seco leria como "só atacar"
// para quem está decidindo se a criatura escapa do fogo.
func (s *Scene) actionsLeft() string {
	switch {
	case s.StandardLeft && s.MovementLeft:
		return "padrão e movimento"
	case s.StandardLeft:
		return "padrão (dá para mover)"
	case s.MovementLeft:
		return "movimento"
	}
	return "sem ação"
}
