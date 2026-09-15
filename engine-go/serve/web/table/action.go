package table

import (
	"fmt"
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

// A única mutação da Mesa: o jogador registra a PRÓPRIA iniciativa. O cliente
// manda o d20 e MAIS NADA — a soma é do motor.

// tableSignals é o que o Datastar manda: os sinais da página. Quem os lê é o
// `datastar.ReadSignals` do SDK, que sabe de onde tirá-los — query string no
// GET, corpo JSON nos outros métodos.
//
// Só o d20 é lido. Os outros sinais viajam junto porque o Datastar manda todos,
// e ignorá-los aqui é o mesmo cuidado do `selfInitiativeEntry`, que monta um
// payload NOVO em vez de escrever no do cliente: um `initiative` que a página
// mandasse junto não pode vencer a conta do servidor.
type tableSignals struct {
	D20 int64 `json:"d20"`
}

func (s Scene) handleTableInitiative(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	// LER OS SINAIS PRIMEIRO. O `NewSSE` assume a resposta e fecha o corpo do
	// pedido, então um `ReadSignals` depois dele encontra o corpo fechado e o
	// próprio SDK devolve "are you sure you created the SSE ***AFTER*** the
	// ReadSignals?". A ordem inversa passa VERDE em teste de handler e só falha no
	// servidor de verdade: o `httptest.NewRequest` não reproduz esse ciclo de
	// vida.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // teto de 1 MB, como o `httpio.DecodeJSON` da casa
	var sinais tableSignals
	erroDeLeitura := datastar.ReadSignals(r, &sinais)

	// Responde SEMPRE em SSE, inclusive na recusa: aqui a resposta É a tela. Um
	// caminho de volta que o cliente não escuta faz a recusa sumir em silêncio, e
	// o jogador clica olhando para uma tela que não muda.
	sse := datastar.NewSSE(w, r)
	erro := ""
	if erroDeLeitura != nil {
		erro = fmt.Sprintf("não entendi o dado enviado: %v", erroDeLeitura)
	} else if err := s.registerInitiativeTable(r, campaignID, sessionID, sinais.D20); err != nil {
		erro = err.Error()
	}
	// Sai o sinal nos DOIS caminhos: no do erro para acender a frase, e no do
	// acerto para APAGAR a frase anterior. Quem redesenha a fila é o stream, que
	// já está aberto — mandar o fragmento aqui também a desenharia por dois
	// caminhos que podem discordar.
	_ = sse.MarshalAndPatchSignals(map[string]string{"error": erro})
}

// registerInitiativeTable é o caminho inteiro da escrita: autoriza, acha o
// personagem de quem pediu, chama a REGRA e avisa as duas telas.
//
// Recebe o `d20` já lido em vez de ler o corpo: quem lê é o handler, ANTES de
// abrir o SSE, e a ordem é obrigatória (ver o comentário lá em cima).
func (s Scene) registerInitiativeTable(r *http.Request, campaignID, sessionID, d20 int64) error {
	userID := s.deps.CurrentUserID(r)
	if _, _, _, err := s.deps.SessionForCaller(r.Context(), userID, campaignID, sessionID); err != nil {
		return err
	}
	_, _, eu := s.tableRoster(r.Context(), userID, campaignID)
	if eu == nil {
		return fmt.Errorf("você não tem personagem nesta mesa")
	}
	// A REGRA: confere o d20 de 1 a 20, pergunta o bônus ao motor e soma. O app
	// não tem uma segunda — se tivesse, mediria a cópia.
	entry, err := s.deps.SelfInitiativeEntry(userID, campaignID, eu.CharacterID, d20)
	if err != nil {
		return err
	}
	estado, err := s.deps.Sessions().UpsertInitiativeEntry(sessionID, entry)
	if err != nil {
		return err
	}
	// Sem este aviso a linha nova só apareceria para o mestre no próximo F5.
	s.deps.PublishSessionState(sessionID, estado)
	return nil
}
