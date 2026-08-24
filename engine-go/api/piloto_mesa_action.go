package api

import (
	"fmt"
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

// A única mutação da Mesa: o jogador registra a PRÓPRIA iniciativa (ALE-219).
//
// Ela existe porque a ALE-213 já tinha reduzido esta superfície a leitura mais
// este gesto — e porque ele é a forma mais honesta de medir o Datastar num
// caminho de escrita: o cliente manda o d20 e MAIS NADA, e a soma é do motor.

// mesaSignals é o que o Datastar manda: os sinais da página. Quem os lê é o
// `datastar.ReadSignals` do SDK, que sabe de onde tirá-los — query string no
// GET, corpo JSON nos outros métodos.
//
// Só o d20 é lido. Os outros sinais viajam junto porque o Datastar manda todos,
// e ignorá-los aqui é o mesmo cuidado do `selfInitiativeEntry`, que monta um
// payload NOVO em vez de escrever no do cliente: um `initiative` que a página
// mandasse junto não pode vencer a conta do servidor.
type mesaSignals struct {
	D20 int64 `json:"d20"`
}

func (s *Server) handleMesaInitiative(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := mesaParams(w, r)
	if !ok {
		return
	}
	// LER OS SINAIS PRIMEIRO. O `NewSSE` assume a resposta e fecha o corpo do
	// pedido, então um `ReadSignals` depois dele encontra o corpo fechado e o
	// próprio SDK devolve "are you sure you created the SSE ***AFTER*** the
	// ReadSignals?". A ordem inversa passou VERDE em teste de handler e falhou no
	// servidor de verdade: `httptest.NewRequest` não reproduz esse ciclo de vida
	// (foi assim que o defeito apareceu — no navegador, não na suíte).
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // teto de 1 MB, como o `plataforma.DecodeJSON` da casa
	var sinais mesaSignals
	erroDeLeitura := datastar.ReadSignals(r, &sinais)

	// Respondemos SEMPRE em SSE, inclusive na recusa: é o caminho de volta que o
	// socket não tem. A ALE-213 deixou anotado que o cliente não escuta o
	// `exception`, então lá uma recusa some em silêncio e o jogador clica olhando
	// para uma tela que não muda. Aqui a resposta É a tela.
	sse := datastar.NewSSE(w, r)
	erro := ""
	if erroDeLeitura != nil {
		erro = fmt.Sprintf("não entendi o dado enviado: %v", erroDeLeitura)
	} else if err := s.registraIniciativaDaMesa(r, campaignID, sessionID, sinais.D20); err != nil {
		erro = err.Error()
	}
	// Sai o sinal nos DOIS caminhos: no do erro para acender a frase, e no do
	// acerto para APAGAR a frase anterior. Quem redesenha a fila é o stream, que
	// já está aberto — mandar o fragmento aqui também o desenharia por dois
	// caminhos que podem discordar, que é o defeito que a ALE-122 consertou.
	_ = sse.MarshalAndPatchSignals(map[string]string{"erro": erro})
}

// registraIniciativaDaMesa é o caminho inteiro da escrita: autoriza, acha o
// personagem de quem pediu, chama a REGRA e avisa as duas telas.
//
// Recebe o `d20` já lido em vez de ler o corpo: quem lê é o handler, ANTES de
// abrir o SSE, e a ordem é obrigatória (ver o comentário lá em cima).
func (s *Server) registraIniciativaDaMesa(r *http.Request, campaignID, sessionID, d20 int64) error {
	user := currentUser(r)
	if _, _, _, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID); err != nil {
		return err
	}
	_, _, eu := s.mesaRoster(r.Context(), user, campaignID)
	if eu == nil {
		return fmt.Errorf("você não tem personagem nesta mesa")
	}
	// A REGRA, e ela é a mesma do socket: confere o d20 de 1 a 20, pergunta o
	// bônus ao motor e soma. O piloto não tem uma segunda — se tivesse, mediria
	// a cópia.
	entry, err := s.selfInitiativeEntry(user.ID, campaignID, eu.CharacterID, d20)
	if err != nil {
		return err
	}
	estado, err := s.sessions.UpsertInitiativeEntry(sessionID, entry)
	if err != nil {
		return err
	}
	// O mestre está na SPA. Sem este aviso a linha nova só apareceria para ele
	// no próximo F5.
	//
	// Isto era `s.rt.emitSessionState`, com uma guarda de nil e uma mensagem de
	// erro para o caso de o socket não ter subido — e um comentário reclamando
	// que "cada escrita nova tem de lembrar dos dois transportes". A ALE-253
	// tirou o socket do projeto e o custo junto: há um caminho de publicação só,
	// ele existe desde o `newServer`, e não há mais o que estar nil.
	s.publishSessionState(sessionID, estado)
	return nil
}
