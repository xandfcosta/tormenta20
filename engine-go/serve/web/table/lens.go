package table

import (
	"fmt"
	"sync"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/board"
)

// VER COMO JOGADOR: a lente do mestre sobre a própria cena — ele confere a
// emboscada SEM parar de montá-la. Sem ela, saber o que a mesa está vendo exige
// abrir dois navegadores com dois logins.
//
// A CÓPIA VEM DO SERVIDOR e nunca de uma segunda regra na tela: é literalmente o
// que o `BoardForRole("player", …)` manda à mesa, o mesmo gargalo por papel que
// já redige o mapa. Uma lente que reimplementasse a redação mediria a
// reimplementação — e mostraria "está escondido" sobre uma peça que a mesa vê.
//
// # Por que ela é ESTADO DO SERVIDOR, e não um sinal do navegador
//
// Porque o STREAM não pergunta nada a ninguém: ele redesenha as regiões a cada
// mudança, e um sinal do navegador não tem como participar dessa decisão. Uma
// lente ligada em `data-show` seria desfeita pelo primeiro quadro do SSE — a
// peça escondida voltaria à tela do mestre sozinha, no meio da conferência, e a
// resposta que ele estava buscando seria a errada.
//
// Ela é EFÊMERA de propósito: mora em memória e morre com o processo. É um modo
// de conferência de meia dúzia de segundos, não uma preferência. E é por PESSOA
// na sessão, não por aba — duas abas do mesmo mestre acendem juntas.

// lenses guarda quem está vendo a cena como a mesa.
//
// Tipo próprio e não um `sync.Map` solto no `Server` porque a chave é composta e
// a regra de leitura tem um caso ("não sou mestre, não há lente") que precisa
// morar junto do dado.
type lenses struct {
	mu sync.RWMutex
	on map[lensKey]bool
}

type lensKey struct {
	SessionID int64
	UserID    int64
}

func newLenses() *lenses {
	return &lenses{on: map[lensKey]bool{}}
}

// Toggle liga ou desliga, e devolve como ficou.
//
// ALTERNA (`Toggle`) e não recebe o estado desejado, ao contrário do pincel de terreno: o
// botão é UM, com `aria-pressed`, e mandar o valor faria a tela ser a fonte da
// verdade de um estado que é do servidor — dois cliques rápidos com a resposta
// atrasada apagariam um ao outro.
func (l *lenses) Toggle(sessionID, userID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := lensKey{SessionID: sessionID, UserID: userID}
	if l.on[key] {
		// APAGA a entrada em vez de gravar `false`: o mapa vive enquanto o
		// processo viver, e uma sessão que acumulasse um `false` por pessoa nunca
		// devolveria a memória.
		delete(l.on, key)
		return false
	}
	l.on[key] = true
	return true
}

func (l *lenses) On(sessionID, userID int64) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.on[lensKey{SessionID: sessionID, UserID: userID}]
}

// Erase desliga a lente de todo mundo naquela sessão.
//
// Chamado quando a CENA ACABA: uma lente ligada sobre um tabuleiro que não
// existe mais mostraria "você está vendo como a mesa" sobre uma tela vazia, e o
// mestre concluiria que o próprio mapa sumiu para os jogadores.
// HowMany conta as lentes acesas desta sessão — ver o `Watching`, que soma as
// duas metades da memória efêmera.
func (l *lenses) HowMany(sessionID int64) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	lit := 0
	for key := range l.on {
		if key.SessionID == sessionID {
			lit++
		}
	}
	return lit
}

func (l *lenses) Erase(sessionID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for key := range l.on {
		if key.SessionID == sessionID {
			delete(l.on, key)
		}
	}
}

func (s Scene) LensRoutes(r chi.Router) {
	r.Post(sessionPattern+"/tabuleiro/lente",
		s.gmBoardCommand(toggleLens))
}

// toggleLens acende ou apaga a lente de quem clicou.
//
// Devolve o tabuleiro SEM MUDÁ-LO — a lente não é mutação da cena, e publicá-la
// acordaria a mesa inteira para um modo que é de uma pessoa só. O que redesenha
// a tela de quem clicou é a resposta do próprio comando.
func toggleLens(st Scene, c commandCtx) (*board.BoardState, error) {
	st.lenses.Toggle(c.SessionID, c.User)
	return nil, nil
}

// seesTableHowScene devolve o tabuleiro redigido quando a lente está ligada.
//
// Devolve TAMBÉM quantas peças sumiram, porque essa é a pergunta que trouxe o
// mestre até aqui — "a emboscada está mesmo invisível?". Contar o que sobrou na
// tela não responde: ele não sabe o que não está vendo.
//
// A conta é a DIFERENÇA entre os dois retratos, e não uma varredura por `Hidden`:
// assim ela cobre tudo o que a redação tira, inclusive o que ela vier a tirar
// depois — a cortina esvazia a cena inteira, e uma contagem por campo diria zero
// escondidas sobre um mapa que a mesa não vê.
func seesTableHowScene(forGM *board.BoardState) (fromTable *board.BoardState, hidden int) {
	fromTable = board.BoardForRole("player", forGM)
	if forGM == nil {
		return fromTable, 0
	}
	seen := 0
	if fromTable != nil {
		seen = len(fromTable.Tokens)
	}
	return fromTable, len(forGM.Tokens) - seen
}

// lensCommand escreve o gesto que acende ou apaga.
func lensCommand(v BoardView) string {
	return fmt.Sprintf("@post('%s/lente')", v.Base)
}

// lensPhrase diz o modo E o número, e nunca só o modo.
//
// Um modo que se esquece é pior que nenhum: o mestre que não percebe que está na
// vista da mesa não vê a peça que ele mesmo escondeu, e vai concluir que ela
// sumiu. Por isso a tira é PERSISTENTE, nomeia o modo em texto e carrega a
// própria saída.
func lensPhrase(hidden int) string {
	switch {
	case hidden <= 0:
		return "Você está vendo a cena como a mesa. Nenhuma peça escondida nesta cena."
	case hidden == 1:
		return "Você está vendo a cena como a mesa. 1 peça escondida não aparece."
	default:
		return fmt.Sprintf("Você está vendo a cena como a mesa. %d peças escondidas não aparecem.", hidden)
	}
}
