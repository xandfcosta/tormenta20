package live

import "encoding/json"

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
	// Number é a ordem dentro da sessão, começando em 1. Ele existe porque a
	// sessão tem uma SEQUÊNCIA de cenas — sem número, "a terceira cena da noite"
	// não tem como ser dita.
	Number int `json:"number"`
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
func StartScene(st *SessionRuntimeState, tipo SceneKind) {
	if st.InScene() {
		EndScene(st)
	}
	st.ScenesSoFar++
	st.Scene = &Scene{Kind: tipo, Number: st.ScenesSoFar}
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

// O BLOB GRAVADO de antes da ALE-365 tem `sceneActive: true|false` e não tem
// cena. Ler o campo novo num blob velho devolveria "fora de cena" — e apagaria
// da tela o combate de quem estivesse jogando no dia em que isto subisse.
//
// O `UnmarshalJSON` é o lugar: ele é o ÚNICO caminho por onde um blob entra, e
// consertar no chamador deixaria de fora o próximo chamador.
func (st *SessionRuntimeState) UnmarshalJSON(bruto []byte) error {
	// O apelido corta a recursão: sem ele, o `Unmarshal` chamaria este mesmo
	// método para sempre.
	type comoEstaNoDisco SessionRuntimeState
	var lido struct {
		comoEstaNoDisco
		SceneActive *bool `json:"sceneActive"`
	}
	if err := json.Unmarshal(bruto, &lido); err != nil {
		return err
	}
	*st = SessionRuntimeState(lido.comoEstaNoDisco)
	// UM TURNO EM CURSO É PROVA de que havia cena, e esta linha é mais velha que
	// a cena tipada: sessão gravada antes de `sceneActive` existir volta sem ele,
	// e o zero de um bool é `false` — a mesa que parou na rodada 3 reabriria
	// "fora de cena" e a fila sumiria para os jogadores até o mestre clicar em
	// iniciar. Não existe turno sem cena (ver `AdvanceTurn`), então não é
	// remendo de migração: é a invariante afirmada onde o estado ENTRA.
	//
	// Nos dois casos a cena que se abre é a de AÇÃO, porque era a única que o
	// app sabia abrir quando aqueles blobs foram gravados.
	naCena := (lido.SceneActive != nil && *lido.SceneActive) || st.TurnIndex >= 0
	if st.Scene == nil && naCena {
		if st.ScenesSoFar == 0 {
			st.ScenesSoFar = 1
		}
		st.Scene = &Scene{Kind: SceneAction, Number: st.ScenesSoFar}
	}
	return nil
}
