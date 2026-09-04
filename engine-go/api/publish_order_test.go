package api

import "t20engine/aovivo"

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// O QUADRO QUE DESCE TEM DE SEGUIR A ORDEM DA MUTAÇÃO (ALE-238 #1).
//
// Este teste nasceu de uma medição, e a medição contradisse a previsão de quem
// a fez — vale contar, porque é o motivo de ele existir nesta camada.
//
// A ALE-238 registrou "uma condição some no meio de três". A leitura era escrita
// perdida, e a previsão era que a troca do socket por SSE (ALE-253) matasse o
// defeito, porque cada escrita passaria a ser uma requisição com resposta
// própria. Oito corridas do `session.spec.ts` em cada lado, mesma máquina e
// mesmo banco, deram 3 vermelhos em 8 no socket e 1 em 8 no SSE — e a assinatura
// da condição perdida saiu UMA vez na base. Fisher exato: p = 0,57. Nada.
// **O experimento foi inconclusivo, e a estatística disse por quê: com base de
// 1 em 8, oito corridas não distinguem conserto de sorte.**
//
// Perseguir isso por loteria de suíte custaria dezenas de corridas de 3 minutos.
// O mecanismo, porém, é determinístico e mora aqui:
//
//	apply()  → trava, muta, CLONA, destrava
//	publish  → emite o clone  ← FORA da trava
//
// Duas mutações concorrentes serializam a MUTAÇÃO e não a EMISSÃO. A que mutou
// primeiro pode emitir por último, e o cliente faz `setState(next)` com o
// estado inteiro, sem número de versão e sem guarda de ordem
// (`realtime.ts`, `live.on('session-state', …)`). O quadro velho vence, e a
// condição que estava no quadro novo desaparece da tela.
//
// Isso é indiferente ao transporte: existia igual no socket e sobreviveu à
// ALE-253, porque a forma foi copiada do `mutateAndBroadcast` sem ser revista.
// O verde de 8 corridas foi sorte, exatamente como o p = 0,57 avisava.
//
// A garantia é do SERVIDOR e não do cliente: SSE preserva a ordem POR CONEXÃO,
// então basta emitir na ordem certa para o navegador receber na ordem certa.
// Pedir ao cliente que descarte quadro velho exigiria versão no fio e uma
// segunda cópia da regra em cada tela que escuta.
func TestTheFrameFollowsTheOrderOfTheMutation(t *testing.T) {
	const sessionID = int64(7)
	// Repetição porque a corrida é de agendamento: uma passada só não a
	// visita. Sem o conserto isto fica vermelho em poucas dezenas.
	const tentativas = 200

	for tentativa := range tentativas {
		s := newTestServer(t)
		conn := s.sse.Add(sessionID, "c1", "gm")

		var wg sync.WaitGroup
		for _, nome := range []string{"Abalado", "Agarrado", "Cego", "Surdo", "Lento", "Fraco"} {
			wg.Add(1)
			go func(rotulo string) {
				defer wg.Done()
				estado, err := s.sessions.AddInitiativeEntry(sessionID, aovivo.InitiativeEntry{
					Label: rotulo, Type: "npc", Initiative: 10,
				})
				if err != nil {
					return
				}
				s.tableRules().publishSessionState(sessionID, estado)
			}(nome)
		}
		wg.Wait()

		ultimo := ultimoQuadro(conn)
		if ultimo == "" {
			t.Fatalf("tentativa %d: nenhum quadro desceu — o canal não existe, e ausência aqui não é resultado", tentativa)
		}
		// O ÚLTIMO quadro é o que fica na tela. Ele tem de conter as duas
		// entradas: as duas mutações já aconteceram quando o `wg.Wait` voltou.
		faltando := ""
		for _, nome := range []string{"Abalado", "Agarrado", "Cego", "Surdo", "Lento", "Fraco"} {
			if !strings.Contains(ultimo, nome) {
				faltando = nome
				break
			}
		}
		if faltando != "" {
			t.Fatalf("tentativa %d: o último quadro perdeu %q — a tela fica com ela sumida.\n%s",
				tentativa, faltando, resumoDoQuadro(t, ultimo))
		}
		s.sse.Remove(sessionID, "c1")
	}
}

// ultimoQuadro drena a fila e devolve o último quadro, que é o que sobrevive na
// tela — os anteriores são sobrescritos por ele.
func ultimoQuadro(conn *aovivo.SSEConn) string {
	var ultimo string
	for {
		select {
		case frame := <-conn.Frames:
			if strings.Contains(string(frame), "event: session-state") {
				ultimo = string(frame)
			}
		default:
			return ultimo
		}
	}
}

// resumoDoQuadro lista os rótulos que o quadro carrega, para a falha dizer o
// que sobrou em vez de despejar o JSON inteiro.
func resumoDoQuadro(t *testing.T, frame string) string {
	t.Helper()
	_, corpo, _ := strings.Cut(frame, "data: ")
	var estado struct {
		Initiative []struct {
			Label string `json:"label"`
		} `json:"initiative"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(corpo)), &estado); err != nil {
		return frame
	}
	rotulos := make([]string, 0, len(estado.Initiative))
	for _, e := range estado.Initiative {
		rotulos = append(rotulos, e.Label)
	}
	return "sobrou: [" + strings.Join(rotulos, " ") + "]"
}
