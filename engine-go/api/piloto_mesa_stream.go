package api

import (
	"context"

	"crypto/sha256"
	"github.com/a-h/templ"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

// O stream da Mesa — o piloto Datastar (ALE-219).
//
// Quem embala os eventos é o SDK oficial (`datastar-go`), e não uma cópia do
// formato escrita aqui: o `data: elements ` por linha, os dois "\n" finais e a
// negociação de compressão são contrato do FRAMEWORK, e contrato copiado é
// contrato que envelhece calado — um `data:` com quebra de linha chega truncado
// sem erro em lugar nenhum.
//
// COMPRESSÃO é o ganho dominante e ela é uma opção do SDK. Medido no stream
// deste piloto: 52.332 bytes crus de três remendos viram 2.513 em gzip e 1.827
// em brotli — 17KB por remendo caem para ~600. Quadros sucessivos são quase
// idênticos, e essa redundância é exatamente o que um compressor de stream come.
//
// A CADÊNCIA deixou de ser só relógio (o passo (b) da ordem combinada). Os
// stores avisam quem escuta a cada mutação, então o caminho comum acorda na hora
// em vez de esperar o próximo tique. O relógio continua existindo como BATIMENTO
// de reserva, e não é redundância: mudanças que a Mesa mostra nascem FORA dos
// stores — o PV do Grupo vem da ficha, alterado por HTTP —, e nenhum aviso as
// cobriria. Por isso ele afrouxou de 200ms para 1s: o aviso paga a latência, o
// batimento paga a abrangência.
//
// São DOIS avisos desde a ALE-264, e o segundo nasceu de uma medição: o
// tabuleiro é outro store, então mover uma peça não acordava ninguém e a
// mudança só chegava no batimento — 1310ms, cronometrados no navegador.
const mesaBatimento = time.Second

func (s *Server) handleMesaStream(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := mesaParams(w, r)
	if !ok {
		return
	}
	user := currentUser(r)
	// Uma primeira leitura ANTES de abrir o stream: sem acesso, o jogador merece
	// um 403 legível e não um stream que abre e nunca manda nada. Precisa vir
	// antes do `NewSSE`, que já escreve os cabeçalhos.
	view, status, err := s.loadMesaView(r.Context(), user, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	// O aviso é assinado ANTES do primeiro quadro, senão uma mutação que caia
	// entre render e assinatura se perde e a tela fica velha até o batimento.
	aviso, parar := s.sessions.Assinar(sessionID)
	defer parar()
	// DOIS canais porque são dois stores, e o tabuleiro não passa pelo da
	// sessão. Sem este segundo, mover uma peça só aparecia no BATIMENTO —
	// medido no navegador, 1310ms para a peça andar um quadrado, com o mestre
	// arrastando na frente de seis pessoas.
	avisoDoMapa, pararMapa := s.boards.Assinar(sessionID)
	defer pararMapa()

	sse := datastar.NewSSE(w, r, datastar.WithCompression())
	ultimo := escreveMesa(r.Context(), sse, view, nil)

	batimento := time.NewTicker(mesaBatimento)
	defer batimento.Stop()
	for {
		select {
		case <-r.Context().Done():
			// A aba fechou, o jogador trocou de superfície ou a rede caiu. Sair
			// aqui é o que impede a goroutine de sobreviver ao leitor.
			return
		case <-aviso:
		case <-avisoDoMapa:
		case <-batimento.C:
		}
		view, _, err := s.loadMesaView(r.Context(), user, campaignID, sessionID)
		if err != nil {
			// Um erro passageiro (banco ocupado) não derruba o stream: o próximo
			// ciclo tenta de novo, e a tela continua no último estado bom em vez
			// de piscar.
			continue
		}
		ultimo = escreveMesa(r.Context(), sse, view, ultimo)
	}
}

// escreveMesa manda o fragmento SÓ quando o HTML mudou, e devolve a impressão
// digital do que foi mandado.
//
// O hash continua sendo o árbitro mesmo com o aviso do store, e de propósito: o
// aviso diz "a sessão mudou", que não é a mesma pergunta que "a tela mudou".
// Um `hpHidden` ligado numa linha não muda nada para o jogador, e mandar por
// causa dele seria tráfego que não desenha nada.
//
// Comparar o HTML RENDERIZADO e não o estado também é escolha: o
// `refreshCharacterMaxes` devolve struct nova a cada leitura, então igualdade de
// estado mandaria tudo sempre; e comparar campo a campo seria a lista que
// envelhece — o defeito que o `cloneState` documenta ter tido com o `TurnsTaken`.
func escreveMesa(ctx context.Context, sse *datastar.ServerSentEventGenerator, view mesaView, anterior digitais) digitais {
	if anterior == nil {
		anterior = digitais{}
	}
	for _, r := range regioesDaMesa(view) {
		fragmento, err := renderFragmento(ctx, r.No)
		if err != nil {
			continue
		}
		digital := sha256.Sum256([]byte(fragmento))
		if digital == anterior[r.ID] {
			continue
		}
		if err := sse.PatchElements(fragmento); err != nil {
			// Falhar ao escrever é o leitor tendo ido embora. NÃO grava a
			// digital: gravar faria a região ser pulada no próximo ciclo, e a
			// tela ficaria com o estado velho para sempre.
			continue
		}
		anterior[r.ID] = digital
	}
	return anterior
}

// digitais é a impressão do que cada região tem NA TELA de um leitor. Uma por
// conexão, porque cada uma chegou num momento e viu coisas diferentes.
type digitais map[string][32]byte

// regiaoDaMesa é um pedaço da cena que muda por conta própria (ALE-264).
type regiaoDaMesa struct {
	ID string
	No templ.Component
}

// regioesDaMesa é a lista, e a ORDEM é a da tela.
//
// O corte é por QUEM MUDA a região, não por assunto: o cabeçalho e os comandos
// mudam com o turno, o grupo muda com a ficha, o mapa muda com o tabuleiro, a
// fila muda com a fila. Cortar por assunto juntaria coisas que mudam em ritmos
// diferentes, e a região só é útil enquanto ela é a unidade de mudança.
func regioesDaMesa(v mesaView) []regiaoDaMesa {
	return []regiaoDaMesa{
		{"mesa-cabecalho", mesaCabecalho(v)},
		{"mesa-registrar", mesaRegistrarRegiao(v)},
		{"mesa-grupo", mesaGrupo(v)},
		{"mesa-tabuleiro", mesaTabuleiro(v)},
		{"mesa-fila", mesaFila(v)},
		{"mesa-comandos", mesaComandos(v)},
	}
}
