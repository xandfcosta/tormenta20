package api

import (
	"fmt"
	"sync"

	"github.com/go-chi/chi/v5"

	"t20engine/tabuleiro"
)

// VER COMO JOGADOR (ALE-193, portado na ALE-269 como superfície 7).
//
// A lente do mestre sobre a própria cena: ele confere a emboscada SEM parar de
// montá-la. Até existir, a única forma de saber o que a mesa está vendo era
// abrir dois navegadores com dois logins — foi assim que a ALE-178 foi
// verificada, e é caro demais para se fazer no meio de uma sessão.
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
// de conferência de meia dúzia de segundos, não uma preferência.
//
// Divergência anotada da SPA: lá a lente é por ABA (um sinal do componente), e
// aqui é por PESSOA na sessão — duas abas do mesmo mestre acendem juntas. É
// consequência de o modo morar no servidor, e o caso ("o mestre com duas abas da
// mesma mesa") não é o que a lente existe para servir.

// lenses guarda quem está vendo a cena como a mesa.
//
// Tipo próprio e não um `sync.Map` solto no `Server` porque a chave é composta e
// a regra de leitura tem um caso ("não sou mestre, não há lente") que precisa
// morar junto do dado.
type lenses struct {
	mu     sync.RWMutex
	ligada map[lensKey]bool
}

type lensKey struct {
	SessionID int64
	UserID    int64
}

func newLenses() *lenses {
	return &lenses{ligada: map[lensKey]bool{}}
}

// Alterna liga ou desliga, e devolve como ficou.
//
// ALTERNA e não recebe o estado desejado, ao contrário do pincel de terreno: o
// botão é UM, com `aria-pressed`, e mandar o valor faria a tela ser a fonte da
// verdade de um estado que é do servidor — dois cliques rápidos com a resposta
// atrasada apagariam um ao outro.
func (l *lenses) Toggle(sessionID, userID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	chave := lensKey{SessionID: sessionID, UserID: userID}
	if l.ligada[chave] {
		// APAGA a entrada em vez de gravar `false`: o mapa vive enquanto o
		// processo viver, e uma sessão que acumulasse um `false` por pessoa nunca
		// devolveria a memória.
		delete(l.ligada, chave)
		return false
	}
	l.ligada[chave] = true
	return true
}

func (l *lenses) On(sessionID, userID int64) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.ligada[lensKey{SessionID: sessionID, UserID: userID}]
}

// Apaga desliga a lente de todo mundo naquela sessão.
//
// Chamado quando a CENA ACABA: uma lente ligada sobre um tabuleiro que não
// existe mais mostraria "você está vendo como a mesa" sobre uma tela vazia, e o
// mestre concluiria que o próprio mapa sumiu para os jogadores.
func (l *lenses) Erase(sessionID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for chave := range l.ligada {
		if chave.SessionID == sessionID {
			delete(l.ligada, chave)
		}
	}
}

func (s *Server) LensRoutes(r chi.Router) {
	r.Post("/mesa/{campaignId}/{sessionId}/tabuleiro/lente",
		s.gmBoardCommand(toggleLens))
}

// toggleLens acende ou apaga a lente de quem clicou.
//
// Devolve o tabuleiro SEM MUDÁ-LO — a lente não é mutação da cena, e publicá-la
// acordaria a mesa inteira para um modo que é de uma pessoa só. O que redesenha
// a tela de quem clicou é a resposta do próprio comando.
func toggleLens(st *Server, c commandCtx) (*tabuleiro.BoardState, error) {
	st.lentes.Toggle(c.SessionID, c.User.ID)
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
func seesTableHowScene(doMestre *tabuleiro.BoardState) (daMesa *tabuleiro.BoardState, escondidas int) {
	daMesa = tabuleiro.BoardForRole("player", doMestre)
	if doMestre == nil {
		return daMesa, 0
	}
	vistas := 0
	if daMesa != nil {
		vistas = len(daMesa.Tokens)
	}
	return daMesa, len(doMestre.Tokens) - vistas
}

// lensCommand escreve o gesto que acende ou apaga.
func lensCommand(v boardView) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/lente')", v.CampaignID, v.SessionID)
}

// lensPhrase diz o modo E o número, e nunca só o modo.
//
// Um modo que se esquece é pior que nenhum: o mestre que não percebe que está na
// vista da mesa não vê a peça que ele mesmo escondeu, e vai concluir que ela
// sumiu. Por isso a tira é PERSISTENTE, nomeia o modo em texto e carrega a
// própria saída.
func lensPhrase(escondidas int) string {
	switch {
	case escondidas <= 0:
		return "Você está vendo a cena como a mesa. Nenhuma peça escondida nesta cena."
	case escondidas == 1:
		return "Você está vendo a cena como a mesa. 1 peça escondida não aparece."
	default:
		return fmt.Sprintf("Você está vendo a cena como a mesa. %d peças escondidas não aparecem.", escondidas)
	}
}
