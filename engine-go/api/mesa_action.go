package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// A única mutação da Mesa: o jogador registra a PRÓPRIA iniciativa (ALE-219).
//
// Ela existe porque a ALE-213 já tinha reduzido esta superfície a leitura mais
// este gesto — e porque ele é a forma mais honesta de medir o Datastar num
// caminho de escrita: o cliente manda o d20 e MAIS NADA, e a soma é do motor.

// mesaSignals é o que o Datastar manda no corpo: os sinais da página, JSON cru
// e sem embrulho (confirmado lendo o bundle v1.0.2 — `Y.body = JSON.stringify(signals)`).
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
	// Respondemos SEMPRE em SSE, inclusive na recusa: é o caminho de volta que
	// o socket não tem. A ALE-213 deixou anotado que o cliente não escuta o
	// `exception`, então lá uma recusa some em silêncio e o jogador clica
	// olhando para uma tela que não muda. Aqui a resposta É a tela.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")

	if err := s.registraIniciativaDaMesa(r, campaignID, sessionID); err != nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(patchSignalsEvent(mesaErroJSON(err.Error()))))
		return
	}
	w.WriteHeader(http.StatusOK)
	// Limpa o erro anterior e mais nada: quem redesenha a fila é o stream, que
	// já está aberto. Mandar o fragmento aqui TAMBÉM o desenharia duas vezes,
	// por dois caminhos que podem discordar — o defeito que a ALE-122 consertou
	// fazendo os dois caminhos passarem pelo mesmo gargalo.
	_, _ = w.Write([]byte(patchSignalsEvent(mesaErroJSON(""))))
}

// registraIniciativaDaMesa é o caminho inteiro da escrita: autoriza, acha o
// personagem de quem pediu, chama a REGRA e avisa as duas telas.
func (s *Server) registraIniciativaDaMesa(r *http.Request, campaignID, sessionID int64) error {
	user := currentUser(r)
	if _, _, _, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID); err != nil {
		return err
	}
	var sinais mesaSignals
	// Teto de 1 MB como o `decodeJSON` da casa: um corpo sem limite é uma porta.
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&sinais); err != nil {
		return fmt.Errorf("não entendi o dado enviado: %w", err)
	}
	_, _, eu := s.mesaRoster(r.Context(), user, campaignID)
	if eu == nil {
		return fmt.Errorf("você não tem personagem nesta mesa")
	}
	// A REGRA, e ela é a mesma do socket: confere o d20 de 1 a 20, pergunta o
	// bônus ao motor e soma. O piloto não tem uma segunda — se tivesse, mediria
	// a cópia.
	entry, err := s.selfInitiativeEntry(user.ID, campaignID, eu.CharacterID, sinais.D20)
	if err != nil {
		return err
	}
	estado, err := s.sessions.upsertInitiativeEntry(sessionID, entry)
	if err != nil {
		return err
	}
	// O mestre está na SPA, ouvindo o socket. Sem este aviso a linha nova só
	// apareceria para ele no próximo F5 — e é exatamente aqui que o custo de
	// ter DOIS transportes aparece: cada escrita nova tem de lembrar dos dois.
	if s.rt == nil {
		return fmt.Errorf("o tempo real não subiu; sua iniciativa foi gravada mas a mesa não foi avisada")
	}
	s.rt.emitSessionState(sessionID, estado)
	return nil
}

// mesaErroJSON monta o patch do sinal `erro`. Passa pelo `json.Marshal` porque
// a mensagem carrega o valor ofendido ("d20 must be an integer from 1 to 20,
// got 47") e uma aspa solta ali quebraria o evento inteiro.
func mesaErroJSON(msg string) string {
	blob, err := json.Marshal(map[string]string{"erro": msg})
	if err != nil {
		return `{"erro":"erro inesperado"}`
	}
	return string(blob)
}
