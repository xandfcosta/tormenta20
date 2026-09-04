package table

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
// mora no `session_lifecycle.go` — aqui é só o caminho até ela.
//
// EXCLUIR é o único que não passa por `@post`: ele NAVEGA para fora da sessão, e
// um comando de Datastar responde com remendo, não com destino. Ele é um `form`
// que termina em `http.Redirect`, que é o mesmo padrão das ações destrutivas da
// cena de campanhas.

func (s Scene) RoutesSession(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/sessao"
	r.Post(base+"/iniciar", s.gmCommand(iniciaAPartida))
	r.Post(base+"/encerrar", s.gmCommand(encerraAPartida))
	r.Post(base+"/titulo", s.gmCommand(renameStart))
	r.Post(base+"/reiniciar", s.gmCommand(reiniciaAFila))
	// O `form` de excluir, fora do fluxo de comandos: ele navega.
	r.Post(base+"/excluir", s.excluiAPartida)
}

// commandSession relê a linha, que é onde o ciclo mora.
func commandSession(st Scene, c commandCtx) (sqlcgen.Session, error) {
	return st.deps.Queries().GetSession(c.R.Context(), c.SessionID)
}

func iniciaAPartida(st Scene, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	if _, err := st.deps.StartSessionForTable(c.R.Context(), c.SessionID); err != nil {
		return nil, err
	}
	return st.deps.Sessions().GetState(c.SessionID), nil
}

func encerraAPartida(st Scene, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	if _, err := st.deps.EndSessionForTable(c.R.Context(), c.SessionID); err != nil {
		return nil, err
	}
	return st.deps.Sessions().GetState(c.SessionID), nil
}

// renameStart troca o título.
//
// Título VAZIO é legítimo e vira nulo: a sessão tem NÚMERO, que é a identidade
// dela, e o título é o apelido da noite. Obrigar a um faria o mestre inventar
// texto para poder salvar.
func renameStart(st Scene, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	var sinais struct {
		Titulo string `json:"titulodasessao"`
	}
	c.R.Body = http.MaxBytesReader(nil, c.R.Body, 1<<20)
	if err := datastar.ReadSignals(c.R, &sinais); err != nil {
		return nil, fmt.Errorf("não entendi o título enviado: %v", err)
	}
	titulo := strings.TrimSpace(sinais.Titulo)
	// A gravação é uma PERGUNTA e não um SET montado aqui (ALE-278): o `title`
	// desta tabela não tem query própria no sqlc, então quem escreve é um SET —
	// e quem sabe montá-lo, que vazio é NULL e que a linha tem um `updatedAt` a
	// carimbar é o hospedeiro. A rota JSON grava a mesma coluna pelo mesmo
	// caminho, que é o que impede as duas de divergirem.
	if err := st.deps.SaveSessionTitle(c.R.Context(), c.SessionID, titulo); err != nil {
		return nil, fmt.Errorf("não deu para salvar o título: %v", err)
	}
	return st.deps.Sessions().GetState(c.SessionID), nil
}

// reiniciaAFila esvazia o combate SEM tirar a partida do ar.
//
// Os dois verbos moram na mesma tela, um perto do outro, e é por isso que a
// frase de cada um diz o que ACONTECE em vez de repetir o nome do botão:
// "encerrar" tira a sessão do ar, "reiniciar" só apaga a ordem e os turnos.
func reiniciaAFila(st Scene, c commandCtx) (*aovivo.SessionRuntimeState, error) {
	if _, err := st.deps.RestartCombatForTable(c.R.Context(), c.SessionID); err != nil {
		return nil, fmt.Errorf("não deu para reiniciar o combate: %v", err)
	}
	// O `Forget` da regra derrubou o cache; o `Load` traz a linha limpa de volta,
	// senão o `GetState` recria um estado vazio sem passar pelo banco e a
	// próxima carga fria discordaria desta.
	if _, err := st.deps.Sessions().Load(c.R.Context(), c.SessionID); err != nil {
		return nil, fmt.Errorf("não deu para reler a fila: %v", err)
	}
	return st.deps.Sessions().GetState(c.SessionID), nil
}

// excluiAPartida apaga a sessão e MANDA O MESTRE PARA A CRÔNICA.
//
// Ele é `form` e não `@post` porque navega: um comando de Datastar responde com
// remendo, e remendo não tem para onde levar. O destino é a crônica da campanha,
// que é de onde se entra numa sessão — voltar para a mesa apagada seria mandar
// o mestre para uma porta que não existe mais.
func (s Scene) excluiAPartida(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	userID := s.deps.CurrentUserID(r)
	_, papel, status, err := s.deps.SessionForCaller(r.Context(), userID, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	if papel != "gm" {
		http.Error(w, "só o mestre apaga a sessão", http.StatusForbidden)
		return
	}
	if err := s.deps.Queries().DeleteSession(r.Context(), sessionID); err != nil {
		http.Error(w, "não deu para apagar a sessão", http.StatusInternalServerError)
		return
	}
	// O ESTADO EM MEMÓRIA morre junto — a fila E o tabuleiro (ALE-270).
	//
	// Aqui estava só o `Sessions().Forget(sessionID)`, e o comentário dele dizia
	// a verdade sobre metade do problema: sem ele a sessão apagada continuaria
	// respondendo em memória. O que ele não alcançava era o tabuleiro, que ficava
	// no mapa do `BoardStore` batendo na chave estrangeira a cada gravação e
	// deixando a mesa marcada como suja para sempre.
	s.deps.SessionDeleted(sessionID)
	http.Redirect(w, r, "/campanhas/"+strconv.FormatInt(campaignID, 10), http.StatusSeeOther)
}
