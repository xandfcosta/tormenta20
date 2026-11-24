package table

import (
	"context"
	"t20engine/domain/live"
	"t20engine/serve/web/ui"

	"crypto/sha256"
	"net/http"
	"strconv"
	"t20engine/infra/events"

	"github.com/a-h/templ"

	"github.com/starfederation/datastar-go/datastar"
)

// O stream da Mesa.
//
// Quem embala os eventos é o SDK oficial (`datastar-go`), e não uma cópia do
// formato escrita aqui: o `data: elements ` por linha e os dois "\n" finais são
// contrato do FRAMEWORK, e contrato copiado é
// contrato que envelhece calado — um `data:` com quebra de linha chega truncado
// sem erro em lugar nenhum.
//
// COMPRESSÃO é o ganho dominante, e quem comprime é o `httpio.Gzip` do
// roteador, não o SDK: quadros sucessivos são quase idênticos, e essa
// redundância é exatamente o que um compressor de stream come.
//
// A CADÊNCIA é o EVENTO, e só ele: toda gravação que a Mesa mostra publica o
// do alvo dela, e o que muda só em memória (a presença, a tela da mesa) também.
// Não há relógio. Aqui morava um batimento de 1s que redesenhava a cada tique,
// para cobrir o que mudava sem avisar; cada Mesa ociosa pagava ~7 ms de CPU por
// segundo para descobrir que nada tinha mudado, e o que ele cobria virou evento
// (ALE-639). Nem para a queda ele servia: ela vem do `r.Context()` cancelado.

func (s Scene) handleTableStream(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	userID := s.deps.CurrentUserID(r)
	// Uma primeira leitura ANTES de abrir o stream: sem acesso, o jogador merece
	// um 403 legível e não um stream que abre e nunca manda nada. Precisa vir
	// antes do `NewSSE`, que já escreve os cabeçalhos.
	view, status, err := s.LoadView(r.Context(), userID, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	view.dressPage(r)

	// A PRESENÇA é registrada AQUI, e não numa rota própria.
	//
	// Este fluxo é o lugar certo por três coisas que ele já tem e uma rota nova
	// teria de recriar: QUEM está pedindo, EM QUE sessão, e o `r.Context()` que o
	// servidor cancela quando a aba fecha — que é a saída, e ela é a metade difícil
	// de acertar num transporte próprio.
	//
	// O PAPEL sai do `view.Mestre`, que é nil para o jogador: é a mesma leitura que
	// já decidiu o que desenhar, e não uma segunda pergunta ao banco. Ele importa
	// ao registro porque quem abre duas abas conta uma vez só, e a dedupe promove a
	// pessoa a mestre se QUALQUER conexão dela for de mestre.
	//
	// A CHEGADA E A SAÍDA são notícia: o Grupo do mestre desenha quem está na
	// mesa, e nada além deste fluxo sabe quando alguém abre ou fecha a aba.
	role := "player"
	if view.GM != nil {
		role = "gm"
	}
	connection := live.NewUUID()
	s.deps.Presence().Join(sessionID, connection, live.PresenceUser{UserID: userID, Role: role})
	s.deps.Bus().Publish(events.PresenceJoined{SessionID: sessionID})
	defer func() {
		s.deps.Presence().Leave(sessionID, connection)
		s.deps.Bus().Publish(events.PresenceLeft{SessionID: sessionID})
	}()

	// A ASSINATURA vem ANTES do primeiro quadro, senão uma mutação que caia entre
	// render e assinatura se perde e a tela fica velha até o próximo evento.
	//
	// UMA, com os interesses de quem está olhando, e ela ACOMPANHA a view: quem
	// senta à mesa depois de a página abrir entra no Grupo, e a ficha dele passa a
	// interessar (`streamSubscription.follow`).
	subscription := subscribeStream(s.deps.Bus(), readerInterests(view))
	defer subscription.close()

	sse := datastar.NewSSE(w, r)
	// A VERSÃO DO SERVIDOR, uma vez por conexão: o fluxo reconecta sozinho depois de um
	// deploy, e é assim que a página aberta descobre que ficou velha.
	_ = sse.MarshalAndPatchSignals(map[string]string{serverBuildSignal: s.deps.BuildRevision()})
	last := writeTable(r.Context(), sse, view, nil)
	// O PUXÃO já empurrado NESTA conexão. Ver `PushForMap`.
	var pushedPull int64
	pushedPull = PushForMap(s, sse, sessionID, userID, pushedPull)
	// A ficha do jogador SEMEADA e não empurrada: o valor de agora entra como
	// "já avisado" para o primeiro AVISO não disparar um repedido do que a
	// página acabou de desenhar — e o primeiro aviso pode chegar no mesmo
	// instante, porque o jogador mexe na própria ficha. Ver `announceSheetChange`.
	sheetAnnounced := sheetVersion(s, r.Context(), view)
	sheetTouched := false

	again := false
	for {
		if !again {
			select {
			case <-r.Context().Done():
				// A aba fechou, o jogador trocou de superfície ou a rede caiu. Sair
				// aqui é o que impede a goroutine de sobreviver ao leitor.
				return
			case <-s.deps.LiveStreamsClosing():
				// O processo vai encerrar. ABORTA em vez de `return`: fechar limpo é uma
				// resposta completa para o Datastar, que não reconectaria, e a mesa ficaria
				// parada depois do deploy. Conexão cortada é erro de rede, e dele ele se
				// recupera sozinho.
				panic(http.ErrAbortHandler)
			case ev := <-subscription.woke():
				// A ficha só é relida quando o evento diz que ELA mudou: reler a cada
				// evento é uma linha por jogador conectado, quase sempre para descobrir
				// que nada mudou.
				sheetTouched = sheetTouched || sheetChanged(ev, view)
			}
		}
		again = false
		next, _, err := s.LoadView(r.Context(), userID, campaignID, sessionID)
		if err != nil {
			// Um erro passageiro (banco ocupado) não derruba o stream: o próximo
			// evento tenta de novo, e a tela continua no último estado bom em vez
			// de piscar.
			continue
		}
		view = next
		view.dressPage(r)
		last = writeTable(r.Context(), sse, view, last)
		pushedPull = PushForMap(s, sse, sessionID, userID, pushedPull)
		if sheetTouched {
			sheetAnnounced = announceSheetChange(s, r.Context(), sse, view, sheetAnnounced)
			sheetTouched = false
		}
		again = subscription.follow(readerInterests(view))
	}
}

// PushForMap leva quem foi puxado para a superfície do TABULEIRO, UMA vez por
// puxão.
//
// Sem isto o "parem tudo e olhem isto" falha justamente no caso mais comum, e
// falha em silêncio: o jogador abre a sessão na superfície **Mesa**, que é a
// padrão, e o tabuleiro dele está num bloco com `display:none` — o puxão chega,
// a região do mapa é remendada, e a pessoa não vê nada disso.
//
// UMA VEZ POR PUXÃO, e é o que separa o empurrão da trava: empurrar a cada
// quadro faria quem tenta voltar para a Mesa ser devolvido ao mapa no quadro
// seguinte, para sempre. A memória do que já foi empurrado é da CONEXÃO, e não
// do servidor: duas abas da mesma pessoa merecem o empurrão cada uma.
//
// A SUPERFÍCIE é sinal do navegador — é o cliente que decide o que aparece —,
// então o que vai daqui é um remendo de SINAL e não de HTML. É o único lugar
// desta cena em que o servidor escreve num sinal pelo stream.
func PushForMap(s Scene, sse *datastar.ServerSentEventGenerator, sessionID, userID, alreadyPushed int64) int64 {
	seq := s.chosenTabs.PullProgress(sessionID, userID)
	if seq == 0 || seq == alreadyPushed {
		return alreadyPushed
	}
	if err := sse.PatchSignals([]byte(`{"surface":"` + superficieDoTabuleiro + `"}`)); err != nil {
		// O leitor foi embora; o laço do stream descobre isso no próximo ciclo.
		// NÃO grava o número: gravar faria este puxão ser considerado empurrado
		// numa tela que nunca o recebeu.
		return alreadyPushed
	}
	return seq
}

// readerInterests diz o que este stream quer receber: tudo o que a Mesa MOSTRA. A sessão (a fila, a cena,
// o tabuleiro); a campanha (as regras, os lugares, o elenco); e a ficha de cada um do Grupo, porque o PV
// do outro jogador muda pela ficha dele, por HTTP, e a Mesa de todos o desenha.
//
// Uma LISTA e não um canal nulo por store: não pedir o interesse diz a mesma
// coisa sem exigir que quem lê saiba que um canal nulo num `select` nunca
// dispara.
func readerInterests(view View) []events.Interest {
	interests := []events.Interest{events.OfSession(view.SessionID), events.OfCampaign(view.CampaignID)}
	for _, member := range view.Group {
		interests = append(interests, events.OfCharacter(member.CharacterID))
	}
	if view.GM == nil && view.Eu != nil {
		interests = append(interests, events.OfCharacter(view.Eu.CharacterID))
	}
	return interests
}

// sheetChanged diz se este evento mexeu na ficha de quem está olhando.
//
// São DOIS: a ficha salva pela própria tela do jogador, e o dano que o mestre
// aplica pela fila — que chega como vital de um combatente COM personagem atrás.
// O `Subscribe` já garantiu que o evento é de quem interessa; aqui a pergunta é
// só que tipo é.
//
// ERA TRÊS: a condição que uma manobra confirmada deixava tinha evento próprio
// (`ConditionApplied`, que não existe mais). Ela saiu com o automatismo — hoje quem marca condição é o
// mestre, pelo gesto da fila, e esse caminho acorda a ficha pelo evento de
// personagem mexido como qualquer outra edição.
func sheetChanged(ev events.Event, view View) bool {
	if view.Eu == nil {
		return false
	}
	switch e := ev.(type) {
	case events.CharacterChanged:
		return e.CharacterID == view.Eu.CharacterID
	case events.VitalsChanged:
		return e.CharacterID != 0 && e.CharacterID == view.Eu.CharacterID
	}
	return false
}

// announceSheetChange acorda a superfície "Minha ficha" quando o personagem
// DESTE jogador mudou no banco.
//
// Um AVISO e não a ficha remendada: a ficha é sete painéis computados e ela NÃO
// é região do stream (ver `View.MinhaFicha`). O que vai daqui é um SINAL de uma
// linha; quem repede a ficha é o cliente, e ele repede na ABA em que a pessoa
// está — coisa que o servidor não tem como saber daqui, porque a aba viaja na
// query dos comandos DELA e este stream abriu antes de qualquer clique.
//
// Remendo de HTML é idempotente; remendo de sinal NÃO É. Mandar a versão em
// todo quadro faria o cliente repedir a ficha sem parar — o gasto exato que
// esta função existe para evitar. A memória do que já foi avisado é da CONEXÃO,
// e não do servidor: duas abas da mesma pessoa merecem o aviso cada uma.
//
// Quem chama é o CANAL e não o relógio, e é por isso que a leitura do banco
// aqui é barata: uma linha por MUDANÇA.
func announceSheetChange(
	s Scene, ctx context.Context, sse *datastar.ServerSentEventGenerator,
	view View, alreadyWarned string,
) string {
	version := sheetVersion(s, ctx, view)
	if version == "" || version == alreadyWarned {
		return alreadyWarned
	}
	if err := sse.PatchSignals([]byte(`{"sheet_version":` + strconv.Quote(version) + `}`)); err != nil {
		// O leitor foi embora. NÃO grava: gravar faria esta mudança ser
		// considerada avisada numa tela que nunca a recebeu.
		return alreadyWarned
	}
	return version
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
func sheetVersion(s Scene, ctx context.Context, view View) string {
	if view.GM != nil || view.Eu == nil {
		return ""
	}
	row, err := s.deps.Queries().GetCharacter(ctx, view.Eu.CharacterID)
	if err != nil {
		return ""
	}
	return row.Updatedat
}

// writeTable manda o fragmento SÓ quando o HTML mudou, e devolve a impressão
// digital do que foi mandado.
//
// O hash continua sendo o árbitro mesmo com o aviso do store, e de propósito: o
// aviso diz "a sessão mudou", que não é a mesma pergunta que "a tela mudou".
// Um `hpHidden` ligado numa linha não muda nada para o jogador, e mandar por
// causa dele seria tráfego que não desenha nada.
//
// Comparar o HTML RENDERIZADO e não o estado também é escolha: o
// `RefreshCharacterVitals` devolve struct nova a cada leitura, então igualdade
// de estado mandaria tudo sempre; e comparar campo a campo seria a lista que
// envelhece.
func writeTable(ctx context.Context, sse *datastar.ServerSentEventGenerator, view View, anterior digitais) digitais {
	if anterior == nil {
		anterior = digitais{}
	}
	for _, r := range TableRegions(view) {
		fragment, err := ui.RenderFragment(ctx, r.No)
		if err != nil {
			continue
		}
		digital := sha256.Sum256([]byte(fragment))
		if digital == anterior[r.ID] {
			continue
		}
		if err := sse.PatchElements(fragment); err != nil {
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

// tableRegion é um pedaço da cena que muda por conta própria.
type tableRegion struct {
	ID string
	No templ.Component
}

// TableRegions é a lista, e a ORDEM é a da tela.
//
// O corte é por QUEM MUDA a região, não por assunto: o cabeçalho e os comandos
// mudam com o turno, o grupo muda com a ficha, o mapa muda com o tabuleiro, a
// fila muda com a fila. Cortar por assunto juntaria coisas que mudam em ritmos
// diferentes, e a região só é útil enquanto ela é a unidade de mudança.
// AS REGIÕES DA MESA: cada uma tem raiz com id ESTÁVEL e hash próprio, e o
// stream manda só as que mudaram. Remendar a cena inteira a cada mudança de
// qualquer um substituiria o elemento debaixo do dedo de quem arrasta uma peça,
// cancelando o gesto. A raiz existe SEMPRE, com o condicional por dentro: uma
// região que some leva o id junto, e o remendo seguinte não acha onde pousar.
//
// TableRegions é exportada porque a BANCADA do hospedeiro a chama: um caso
// pergunta ao MESMO cálculo da produção quais regiões o stream mandaria para
// cada papel, e este pacote não tem banco para montar a view de verdade.
func TableRegions(v View) []tableRegion {
	// UMA PÁGINA À PARTE tem só a região dela: mandar remendo para uma região
	// que o documento não tem é escrever no vazio, e a lista e a página
	// respondem à mesma `view`. O Bestiário não tem região viva — o livro não
	// muda no meio da sessão.
	switch v.Page {
	case pageGroup:
		return []tableRegion{{"table-group", gmGroup(v)}}
	case pageCast:
		return []tableRegion{{"table-npcs", gmCastList(v)}}
	case pageBestiary:
		return nil
	}
	if v.GM != nil {
		return []tableRegion{
			{"table-header", gmHeader(v)},
			{"table-gm-menu", gmMenu(v)},
			// O ATAQUE PROPOSTO muda no ritmo do COMBATE, que não é o do
			// cabeçalho nem o do mapa: ele nasce e morre várias vezes por turno.
			{"table-attack", gmAttack(v)},
			{"table-skill-test", tempestadeSkillTest(v)},
			{"table-blast", tempestadeBlast(v)},
			{"table-spell-cast", tempestadeSpellCast(v)},
			// O ESTANDARTE nasce com a cena e encolhe a cada aliado que o mestre toca.
			{"table-banner", gmBanner(v)},
			{"table-tracker", gmQueue(v)},
			{"table-commands", gmTurnBar(v)},
			{"table-board", gmBoardRegion(v)},
			// O PÔR NO MAPA é região própria porque muda com a FILA (quem existe)
			// e com o MAPA (quem já tem peça): pendurado em qualquer uma das duas,
			// a outra seria remendada de graça — e a peça debaixo do dedo do
			// mestre trocada no meio do arrasto (`TestATrackerChangeDoesNotPatchTheMap`).
			{"table-populate", gmPopulate(v)},
			// O ACERVO pela mesma razão: a lista de lugares guardados muda duas
			// vezes por sessão, e o mapa a cada peça que anda.
			{"table-archive", gmArchive(v)},
			// As CONFIGURAÇÕES mudam com o CICLO da sessão, que não é o ritmo
			// de nenhuma das outras.
			{"table-session-config", tableConfigSession(v)},
		}
	}
	// A SESSÃO DO JOGADOR tem as regiões das superfícies que ela desenha; a Ficha
	// o stream nunca toca.
	return []tableRegion{
		{"table-header", tempestadeHeader(v)},
		{"table-attack", tempestadeAttack(v)},
		{"table-skill-test", tempestadeSkillTest(v)},
		{"table-blast", tempestadeBlast(v)},
		{"table-spell-cast", tempestadeSpellCast(v)},
		{"table-turn-target", tempestadeTargetBar(v)},
		{"table-register", tempestadeRegister(v)},
		{"table-party", tempestadeParty(v)},
		{"table-tracker", tempestadeTracker(v)},
		{"table-board", tempestadeBoardRegion(v)},
	}
}
