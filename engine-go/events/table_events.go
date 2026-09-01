package events

// O VOCABULÁRIO DA MESA (ALE-279).
//
// Cada evento é nomeado pelo ATO — o que o mestre ou o jogador fez —, e nunca
// pela tabela que mudou. "O turno passou" é o que aconteceu; "o
// SessionRuntimeState foi gravado" é como aconteceu, e é a segunda forma que
// envelhece junto com o armazenamento.
//
// Eles moram todos aqui, e não cada um no pacote que o publica, e isso é
// deliberado: o `aovivo` e o `tabuleiro` publicam, o `api` escuta, e nenhum dos
// três precisa importar os outros. Ler este arquivo é ler tudo que pode
// acontecer numa mesa — que é a pergunta que ninguém conseguia responder quando
// os avisos eram três `chan struct{}` em três pacotes.
//
// Todos carregam o `SessionID` porque toda notícia de mesa é sobre uma mesa. O
// `CharacterID` só aparece onde há ficha atrás do combatente: NPC não tem, e
// preencher zero ali é o que faz o `Target` casar com quem não devia.

// CombatantJoined — alguém entrou na fila da iniciativa.
type CombatantJoined struct {
	SessionID int64
	EntryID   string
}

func (e CombatantJoined) Target() Target { return Target{SessionID: e.SessionID} }

// CombatantChanged — a linha de um combatente mudou (nome, iniciativa, PV
// oculto, condição).
type CombatantChanged struct {
	SessionID int64
	EntryID   string
}

func (e CombatantChanged) Target() Target { return Target{SessionID: e.SessionID} }

// CombatantLeft — o mestre tirou alguém da fila.
type CombatantLeft struct {
	SessionID int64
	EntryID   string
}

func (e CombatantLeft) Target() Target { return Target{SessionID: e.SessionID} }

// VitalsChanged — PV ou PM de um combatente mudou.
//
// É o único evento da fila que carrega `CharacterID`, e é o que faz a ficha
// dentro da sessão se atualizar quando o mestre fere um jogador: quando há ficha
// atrás da linha, quem levou o dano foi o PERSONAGEM e não o rastreador. NPC
// entra aqui com `CharacterID` zero, e o alvo então é só a mesa.
type VitalsChanged struct {
	SessionID   int64
	EntryID     string
	CharacterID int64
}

func (e VitalsChanged) Target() Target {
	return Target{SessionID: e.SessionID, CharacterID: e.CharacterID}
}

// TurnAdvanced — a vez passou para outro combatente, para a frente ou para trás.
type TurnAdvanced struct {
	SessionID int64
}

func (e TurnAdvanced) Target() Target { return Target{SessionID: e.SessionID} }

// InitiativeReset — o mestre limpou a fila.
type InitiativeReset struct {
	SessionID int64
}

func (e InitiativeReset) Target() Target { return Target{SessionID: e.SessionID} }

// SceneStarted — a cena começou, e a mesa passa a ver a fila (ALE-210).
type SceneStarted struct {
	SessionID int64
}

func (e SceneStarted) Target() Target { return Target{SessionID: e.SessionID} }

// SceneEnded — a cena acabou, e com ela expiram os efeitos de duração "cena".
type SceneEnded struct {
	SessionID int64
}

func (e SceneEnded) Target() Target { return Target{SessionID: e.SessionID} }

// BoardOpened — o mestre abriu um tabuleiro. Nascer e sumir são as mudanças mais
// visíveis do mapa, e é por isso que elas têm evento próprio em vez de caírem no
// `BoardChanged`.
type BoardOpened struct {
	SessionID int64
}

func (e BoardOpened) Target() Target { return Target{SessionID: e.SessionID} }

// BoardClosed — o mestre fechou o tabuleiro.
type BoardClosed struct {
	SessionID int64
}

func (e BoardClosed) Target() Target { return Target{SessionID: e.SessionID} }

// BoardChanged — uma peça andou, o terreno foi pintado, a cortina abriu.
//
// Este é o evento de ALTA FREQUÊNCIA da casa: o mestre arrastando uma peça
// produz um por quadrado atravessado. É a razão de a fila do ouvinte ter
// dezesseis lugares e de o descarte ser contado em vez de fatal.
type BoardChanged struct {
	SessionID int64
}

func (e BoardChanged) Target() Target { return Target{SessionID: e.SessionID} }

// CharacterChanged — a ficha de um personagem mudou no banco.
//
// Publicado pelo GATEWAY e não por cada comando: passam mais de trinta mutações
// pelo `comandoDaFicha`, e a linha esquecida numa delas seria uma ficha que não
// atualiza só naquele gesto (ALE-275).
//
// Sem `SessionID`, e isso não é esquecimento: a ficha muda por caminhos que não
// passam por mesa nenhuma — o dono mexendo no PV pela ficha solta —, e o
// interesse que a alcança é o `OfCharacter`.
type CharacterChanged struct {
	CharacterID int64
}

func (e CharacterChanged) Target() Target { return Target{CharacterID: e.CharacterID} }
