package api

import (
	"context"
	"t20engine/web/ui"

	"crypto/sha256"
	"github.com/a-h/templ"
	"net/http"
	"strconv"
	"t20engine/events"
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

	// A ASSINATURA vem ANTES do primeiro quadro, senão uma mutação que caia entre
	// render e assinatura se perde e a tela fica velha até o batimento.
	//
	// UMA, com os interesses de quem está olhando (ALE-279). Eram TRÊS canais —
	// a sessão, o tabuleiro e a ficha —, um por store, e o `select` abaixo tinha
	// um `case` para cada um só para juntar de novo o que estava separado por
	// acidente de onde o estado mora. O tabuleiro precisou do segundo porque sem
	// ele mover uma peça só aparecia no BATIMENTO: 1310ms para andar um quadrado,
	// medido no navegador com o mestre arrastando na frente de seis pessoas.
	//
	// Quem não tem ficha nesta mesa (o mestre, e quem só assiste) simplesmente
	// não pede o interesse dela — antes isso era um canal NULO devolvido por uma
	// função à parte, e agora é um item a menos numa lista.
	sub, parar := s.bus.Subscribe(readerInterests(view)...)
	defer parar()

	sse := datastar.NewSSE(w, r, datastar.WithCompression())
	ultimo := escreveMesa(r.Context(), sse, view, nil)
	// O PUXÃO já empurrado NESTA conexão (ALE-205, fatia 2). Ver `empurraParaOMapa`.
	var puxaoEmpurrado int64
	puxaoEmpurrado = empurraParaOMapa(s, sse, sessionID, user.ID, puxaoEmpurrado)
	// A ficha do jogador SEMEADA e não empurrada: o valor de agora entra como
	// "já avisado" para o primeiro AVISO não disparar um repedido do que a
	// página acabou de desenhar — e o primeiro aviso pode chegar no mesmo
	// instante, porque o jogador mexe na própria ficha. Ver `announceSheetChange`.
	sheetAnnounced := sheetVersion(s, r.Context(), view)
	sheetTouched := false

	batimento := time.NewTicker(mesaBatimento)
	defer batimento.Stop()
	for {
		select {
		case <-r.Context().Done():
			// A aba fechou, o jogador trocou de superfície ou a rede caiu. Sair
			// aqui é o que impede a goroutine de sobreviver ao leitor.
			return
		case ev := <-sub.C:
			// A ficha só é relida quando o evento diz que ELA mudou. Antes a
			// leitura acontecia a cada tique — uma linha por segundo por jogador
			// conectado, quase sempre para descobrir que nada mudou.
			sheetTouched = sheetTouched || sheetChanged(ev)
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
		puxaoEmpurrado = empurraParaOMapa(s, sse, sessionID, user.ID, puxaoEmpurrado)
		if sheetTouched {
			sheetAnnounced = announceSheetChange(s, r.Context(), sse, view, sheetAnnounced)
			sheetTouched = false
		}
	}
}

// empurraParaOMapa leva quem foi puxado para a superfície do TABULEIRO, UMA vez
// por puxão (ALE-205, fatia 2).
//
// Sem isto o "parem tudo e olhem isto" falha justamente no caso mais comum, e
// falha em silêncio: o jogador abre a sessão na superfície **Mesa**, que é a
// padrão, e o tabuleiro dele está num bloco com `display:none`. Medido na
// bancada — o puxão chegou, a região do mapa foi remendada, a tira foi escrita
// no HTML, e a pessoa não viu nada disso porque estava olhando a outra
// superfície.
//
// UMA VEZ POR PUXÃO, e é o que separa o empurrão da trava (decisão do dono):
// empurrar a cada quadro faria a pessoa que tenta voltar para a Mesa ser
// devolvida ao mapa 200ms depois, para sempre. A memória do que já foi empurrado
// é da CONEXÃO, e não do servidor: duas abas da mesma pessoa merecem o empurrão
// cada uma, e um contador compartilhado daria o empurrão a uma só.
//
// A SUPERFÍCIE é sinal do navegador — é o cliente que decide o que aparece —,
// então o que vai daqui é um remendo de SINAL e não de HTML. É o único lugar
// desta cena em que o servidor escreve num sinal pelo stream.
func empurraParaOMapa(s *Server, sse *datastar.ServerSentEventGenerator, sessionID, userID, jaEmpurrado int64) int64 {
	seq := s.abas.PuxaoEmCurso(sessionID, userID)
	if seq == 0 || seq == jaEmpurrado {
		return jaEmpurrado
	}
	if err := sse.PatchSignals([]byte(`{"superficie":"` + superficieDoTabuleiro + `"}`)); err != nil {
		// O leitor foi embora; o laço do stream descobre isso no próximo ciclo.
		// NÃO grava o número: gravar faria este puxão ser considerado empurrado
		// numa tela que nunca o recebeu.
		return jaEmpurrado
	}
	return seq
}

// readerInterests diz o que este stream quer receber: a mesa, sempre; a
// ficha, só de quem tem uma aqui.
//
// Uma LISTA e não um canal nulo. Aqui morava o `assinaAFichaDoLeitor`, que
// devolvia `nil` para o mestre porque um canal nulo num `select` nunca dispara —
// era o truque certo para a forma antiga, e ele deixou de ser necessário: não
// pedir o interesse é dizer a mesma coisa sem precisar que o leitor conheça o
// truque.
func readerInterests(view mesaView) []events.Interest {
	interesses := []events.Interest{events.OfSession(view.SessionID)}
	if view.Mestre == nil && view.Eu != nil {
		interesses = append(interesses, events.OfCharacter(view.Eu.CharacterID))
	}
	return interesses
}

// sheetChanged diz se este evento mexeu na ficha de quem está olhando.
//
// São DOIS, e o segundo é o que a ALE-275 existiu para cobrir: a ficha salva
// pela própria tela do jogador, e o dano que o mestre aplica pela fila — que
// chega como vital de um combatente COM personagem atrás. O `Subscribe` já
// garantiu que o evento é de quem interessa; aqui a pergunta é só que tipo é.
func sheetChanged(ev events.Event) bool {
	switch e := ev.(type) {
	case events.CharacterChanged:
		return true
	case events.VitalsChanged:
		return e.CharacterID != 0
	}
	return false
}

// announceSheetChange acorda a superfície "Minha ficha" quando o personagem
// DESTE jogador mudou no banco (ALE-275).
//
// # Por que um AVISO e não a ficha remendada
//
// A ficha é sete painéis computados e ela NÃO é região do stream (ver
// `mesaView.MinhaFicha`): pendurá-la em `regioesDaMesa` faria cada tique
// recomputá-la para descobrir que nada mudou, e o tabuleiro sozinho produz um
// aviso por quadrado arrastado. O que vai daqui é um SINAL de uma linha; quem
// repede a ficha é o cliente, e ele repede na ABA em que a pessoa está — coisa
// que o servidor não tem como saber daqui, porque a aba viaja na query dos
// comandos DELA e este stream abriu antes de qualquer clique.
//
// # A cadência é a do puxão, e pela mesma razão
//
// Remendo de HTML é idempotente; remendo de sinal não é. Mandar a versão em
// TODO quadro faria o cliente repedir a ficha sem parar — o gasto exato que
// esta função existe para evitar. A memória do que já foi avisado é da
// CONEXÃO, e não do servidor: duas abas da mesma pessoa merecem o aviso cada
// uma.
//
// # Quem chama é o CANAL, não o relógio
//
// Ela só roda quando o `CharacterWatch` cutuca, e é por isso que a leitura do
// banco aqui é barata: uma linha por MUDANÇA. A primeira versão desta função
// lia a cada tique do stream — uma linha por segundo por jogador conectado,
// quase sempre para descobrir que nada mudou —, e o dono cortou isso na
// revisão: toda ação dentro de uma sessão já é um evento, e o servidor tem os
// dois outros canais para provar que essa é a forma da casa.
func announceSheetChange(
	s *Server, ctx context.Context, sse *datastar.ServerSentEventGenerator,
	view mesaView, jaAvisada string,
) string {
	versao := sheetVersion(s, ctx, view)
	if versao == "" || versao == jaAvisada {
		return jaAvisada
	}
	if err := sse.PatchSignals([]byte(`{"fichaversao":` + strconv.Quote(versao) + `}`)); err != nil {
		// O leitor foi embora. NÃO grava: gravar faria esta mudança ser
		// considerada avisada numa tela que nunca a recebeu.
		return jaAvisada
	}
	return versao
}

// sheetVersion é o `updatedAt` do personagem de quem está olhando, ou "" para
// quem não tem ficha nesta mesa (o mestre, e o jogador sem personagem).
//
// O `updatedAt` da LINHA do personagem, e não um hash da ficha inteira: ele é
// uma leitura de uma linha, e toda mutação que o mestre alcança de fora — dano,
// cura, condição do motor, nível — passa por um `UPDATE characters` que o
// carimba. O que ele NÃO pega são as tabelas filhas (perícias, itens, magias),
// e isso é aceitável porque elas só mudam pelas mãos do próprio dono, que já
// está remendando a ficha ao mexer nelas.
func sheetVersion(s *Server, ctx context.Context, view mesaView) string {
	if view.Mestre != nil || view.Eu == nil {
		return ""
	}
	row, err := s.queries.GetCharacter(ctx, view.Eu.CharacterID)
	if err != nil {
		return ""
	}
	return row.Updatedat
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
		fragmento, err := ui.RenderFragment(ctx, r.No)
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
	regioes := []regiaoDaMesa{
		{"mesa-cabecalho", mesaCabecalho(v)},
		{"mesa-registrar", mesaRegistrarRegiao(v)},
		{"mesa-grupo", mesaGrupo(v)},
		{"mesa-tabuleiro", mesaTabuleiro(v)},
		// REGIÃO PRÓPRIA, e ela nasceu de um guarda: o diálogo de pôr no mapa muda
		// com a FILA (quem existe) e com o MAPA (quem já tem peça), então pendurá-lo
		// em qualquer uma das duas faz a outra ser remendada de graça. Dentro do
		// mapa, `TestUmaMudancaNaFilaNaoRemendaOMapa` acusou na hora — a peça
		// debaixo do dedo do mestre seria trocada no meio do arrasto.
		{"mesa-por-no-mapa", mesaPorNoMapa(v)},
		// O ACERVO é região pela MESMA razão, e ela foi medida (ALE-203): a lista
		// de 147 lugares guardados era 236 dos 282 KB da região do tabuleiro, e
		// ela muda duas vezes por sessão enquanto o mapa muda a cada peça que
		// anda. Ver `mesaAcervoDeLugares`.
		{"mesa-acervo", mesaAcervoDeLugares(v)},
		{"mesa-config-da-sessao", mesaConfigDaSessao(v)},
		{"mesa-fila", mesaFila(v)},
		{"mesa-comandos", mesaComandos(v)},
	}
	// O TRILHO DA FILA só existe no palco do mestre (ALE-269), e por isso ele
	// entra na lista pela MESMA condição que o desenha. Mandar um remendo para
	// um id que não está no documento é escrever no vazio — e a lista e a
	// página não podem discordar sobre quais regiões existem, o que só se
	// garante fazendo as duas perguntarem à mesma `view`.
	if v.Mestre != nil {
		regioes = append(regioes, regiaoDaMesa{"mesa-trilho-fila", mesaTrilhoDaFila(v)})
		regioes = append(regioes, regiaoDaMesa{"mesa-npcs", mesaListaDeNPCs(v)})
	}
	return regioes
}
