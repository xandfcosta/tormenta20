package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/aovivo"
	"t20engine/db/sqlcgen"
)

// O CICLO DA SESSÃO na Mesa em Datastar (ALE-269, superfícies 3, 4 e 11).
//
// Iniciar, encerrar, renomear, reiniciar o combate e sair. A REGRA de cada um
// mora no `sessao_ciclo.go` — aqui é só o caminho até ela.
//
// EXCLUIR é o único que não passa por `@post`: ele NAVEGA para fora da sessão, e
// um comando de Datastar responde com remendo, não com destino. Ele é um `form`
// que termina em `http.Redirect`, que é o mesmo padrão das ações destrutivas da
// cena de campanhas.

func (s *Server) SessionRoutes(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/sessao"
	r.Post(base+"/iniciar", s.comandoDoMestre(iniciaAPartida))
	r.Post(base+"/encerrar", s.comandoDoMestre(encerraAPartida))
	r.Post(base+"/titulo", s.comandoDoMestre(renomeiaAPartida))
	r.Post(base+"/reiniciar", s.comandoDoMestre(reiniciaAFila))
	// O `form` de excluir, fora do fluxo de comandos: ele navega.
	r.Post(base+"/excluir", s.excluiAPartida)
}

// aSessaoDoComando relê a linha, que é onde o ciclo mora.
func aSessaoDoComando(st *Server, c mesaComando) (sqlcgen.Session, error) {
	return st.queries.GetSession(c.R.Context(), c.SessionID)
}

func iniciaAPartida(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	sess, err := aSessaoDoComando(st, c)
	if err != nil {
		return nil, fmt.Errorf("não deu para ler a sessão %d", c.SessionID)
	}
	if _, err := st.StartSession(c.R.Context(), sess); err != nil {
		return nil, err
	}
	return st.sessions.GetState(c.SessionID), nil
}

func encerraAPartida(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	sess, err := aSessaoDoComando(st, c)
	if err != nil {
		return nil, fmt.Errorf("não deu para ler a sessão %d", c.SessionID)
	}
	if _, err := st.EndSession(c.R.Context(), sess); err != nil {
		return nil, err
	}
	return st.sessions.GetState(c.SessionID), nil
}

// renomeiaAPartida troca o título.
//
// Título VAZIO é legítimo e vira nulo: a sessão tem NÚMERO, que é a identidade
// dela, e o título é o apelido da noite. Obrigar a um faria o mestre inventar
// texto para poder salvar.
func renomeiaAPartida(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	var sinais struct {
		Titulo string `json:"titulodasessao"`
	}
	c.R.Body = http.MaxBytesReader(nil, c.R.Body, 1<<20)
	if err := datastar.ReadSignals(c.R, &sinais); err != nil {
		return nil, fmt.Errorf("não entendi o título enviado: %v", err)
	}
	titulo := strings.TrimSpace(sinais.Titulo)
	// O MESMO `setBuilder` do handler JSON, e não uma query nova: o `title` desta
	// tabela não tem query própria no sqlc — quem escreve é um SET montado, e
	// uma segunda forma de gravar a mesma coluna divergiria no dia em que o
	// `execTouched` mudar (é ele quem carimba o `updatedAt`).
	var set setBuilder
	set.Add("title = ?", nullableArg(trimOrNull(&titulo)))
	if err := set.execTouched(c.R.Context(), st.db, "UPDATE sessions", c.SessionID); err != nil {
		return nil, fmt.Errorf("não deu para salvar o título: %v", err)
	}
	return st.sessions.GetState(c.SessionID), nil
}

// reiniciaAFila esvazia o combate SEM tirar a partida do ar.
//
// Os dois verbos moram na mesma tela, um perto do outro, e é por isso que a
// frase de cada um diz o que ACONTECE em vez de repetir o nome do botão:
// "encerrar" tira a sessão do ar, "reiniciar" só apaga a ordem e os turnos.
func reiniciaAFila(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	if err := st.RestartCombat(c.R.Context(), c.SessionID); err != nil {
		return nil, fmt.Errorf("não deu para reiniciar o combate: %v", err)
	}
	// O `Forget` da regra derrubou o cache; o `Load` traz a linha limpa de volta,
	// senão o `GetState` recria um estado vazio sem passar pelo banco e a
	// próxima carga fria discordaria desta.
	if _, err := st.sessions.Load(c.R.Context(), c.SessionID); err != nil {
		return nil, fmt.Errorf("não deu para reler a fila: %v", err)
	}
	return st.sessions.GetState(c.SessionID), nil
}

// excluiAPartida apaga a sessão e MANDA O MESTRE PARA A CRÔNICA.
//
// Ele é `form` e não `@post` porque navega: um comando de Datastar responde com
// remendo, e remendo não tem para onde levar. O destino é a crônica da campanha,
// que é de onde se entra numa sessão — voltar para a mesa apagada seria mandar
// o mestre para uma porta que não existe mais.
func (s *Server) excluiAPartida(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := mesaParams(w, r)
	if !ok {
		return
	}
	user := currentUser(r)
	_, papel, status, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	if papel != "gm" {
		http.Error(w, "só o mestre apaga a sessão", http.StatusForbidden)
		return
	}
	if err := s.queries.DeleteSession(r.Context(), sessionID); err != nil {
		http.Error(w, "não deu para apagar a sessão", http.StatusInternalServerError)
		return
	}
	// O cache da fila morre junto: sem isto a sessão apagada continuaria
	// respondendo em memória até o processo reiniciar.
	s.sessions.Forget(sessionID)
	http.Redirect(w, r, "/campanhas/"+strconv.FormatInt(campaignID, 10), http.StatusSeeOther)
}
