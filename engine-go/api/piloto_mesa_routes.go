package api

import (
	"context"
	"t20engine/web/hub"

	"fmt"
	"github.com/a-h/templ"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"t20engine/aovivo"
	"t20engine/tabuleiro"
	"time"

	"github.com/go-chi/chi/v5"
	"t20engine/web/ui"
)

// O piloto Datastar (ALE-219): a superfície "Mesa" do jogador renderizada pelo
// SERVIDOR, ao lado da SPA e no mesmo binário.
//
// Ela mora fora do `Router()` de propósito. O `Router()` é montado sob `/api`
// em produção, e isto não é API — é uma PÁGINA, e ela precisa de uma URL que o
// jogador possa abrir e favoritar.
//
// Autenticação é a MESMA e sem uma linha nova: o `requireAuth` já lê o cookie
// `t20_session` antes do Bearer (middleware.go), e o cookie ignora porta, então
// a sessão criada pela SPA vale aqui.
//
// SAÍDA DO PILOTO: apagar `api/mesa*`, a linha do `buildMux` e a entrada do
// proxy no `vite.config.ts`.
func (s *Server) WebRouter() http.Handler {
	r := chi.NewRouter()
	// Os estáticos são anônimos: são o bundle do Datastar e a folha de estilo,
	// e exigir sessão para eles só quebraria o cache.
	r.Handle("/static/*", http.StripPrefix("/static/", pilotoStaticHandler()))
	// A PORTA (ALE-229) é anônima por necessidade: é ela que cria a sessão. Ela
	// tem de vir ANTES dos grupos com `requireAuth` — não por ordem de casamento
	// (o chi casa por rota, não por ordem), mas porque ficar dentro do grupo a
	// tornaria inalcançável para exatamente quem precisa dela.
	s.DoorRoutes(r)
	// O HUB (ALE-231): o menu principal, atrás de sessão como todo o resto.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePagina)
		hub.Routes(r, hub.New(s))
		s.CampaignRoutes(r)
		s.CharacterRoutes(r)
		// A FICHA (ALE-272) é filha do endereço do elenco: `/personagens/{id}`.
		s.SheetRoutes(r)
		s.GrimoireRoutes(r)
		s.GMToolRoutes(r)
		// O BUSCADOR (ALE-264) fica no grupo do Hub e não no do mestre: a caixa
		// abre em QUALQUER cena, inclusive na Mesa, e a rota tem de existir onde
		// quer que o ⌃K seja apertado.
		s.BookSearchRoutes(r)
		// O VERBETE citado por um elo (ALE-264), na casca pelo mesmo motivo do
		// buscador: a caixa abre em qualquer cena.
		s.EntryRoutes(r)
		// O LIVRO (ALE-264) é servido para quem ENTROU e não anonimamente como
		// os estáticos: os estáticos são o bundle do Datastar, e isto é um
		// arquivo do dono da mesa. Sem `LIVRO_PDF` a rota devolve 404 — o botão
		// que a levaria também não é desenhado.
		r.Handle(rotaDoLivro, s.LivroDoPiloto())
		// O LEITOR é uma PÁGINA e o `/livro` é o arquivo. Rotas irmãs de
		// propósito: quem quiser o PDF cru (imprimir, buscar no visualizador do
		// navegador) tem o endereço de sempre.
		r.Get(rotaDoLivro+"/ler", s.handleLeitorDoLivro)
	})
	r.Group(func(r chi.Router) {
		r.Use(s.requirePagina)
		r.Get("/mesa/{campaignId}/{sessionId}", s.handleMesaPage)
		r.Get("/mesa/{campaignId}/{sessionId}/stream", s.handleMesaStream)
		r.Post("/mesa/{campaignId}/{sessionId}/iniciativa", s.handleMesaInitiative)
		s.TableCommandRoutes(r)
		s.TableBestiaryRoutes(r)
		s.MoveRoutes(r)
		s.MovePreviewRoutes(r)
		s.RulerRoutes(r)
		s.SceneRoutes(r)
		s.PartyRoutes(r)
		s.MarkerRoutes(r)
		s.CurtainRoutes(r)
		s.TabRoutes(r)
		s.LensRoutes(r)
		s.TokenActionRoutes(r)
		s.ConditionRoutes(r)
		s.SessionRoutes(r)
		s.NoteRoutes(r)
		s.CastRoutes(r)
		s.NPCRoutes(r)
		s.NPCEditorRoutes(r)
	})
	// A SEGUNDA superfície (ALE-219): a administração. Mesmo `requireAdmin` da
	// API — a tela não decide quem pode ver, ela só deixa de oferecer o que o
	// servidor recusaria.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePagina)
		r.Use(s.requireAdmin)
		r.Get("/admin", s.handleAdminPiloto)
		r.Post("/admin/usuarios/{id}/apagar", s.handleAdminPilotoApagar)
		r.Post("/admin/backup", s.handleAdminPilotoBackup)
		r.Post("/admin/usuarios/{id}/redefinir", s.handleAdminPilotoRedefinir)
		r.Post("/admin/convites", s.handleAdminPilotoConvite)
	})
	return r
}

// mesaStaticHandler serve o bundle e a folha embutidos.
func pilotoStaticHandler() http.Handler {
	sub, err := fs.Sub(pilotoFS, "piloto/static")
	if err != nil {
		panic("piloto: static embutido ausente: " + err.Error())
	}
	return comCacheVersionado(versaoDosEstaticos, "public", http.FileServer(http.FS(sub)))
}

// rotaDaMesa é PARA ONDE se entra numa sessão, e desde a ALE-269 ela é a Mesa em
// Datastar.
//
// Era `/campaigns/{id}/sessions/{sid}` — a rota da SPA — em quatro lugares: o
// Hub, o cartão da campanha e duas linhas da crônica. Enquanto o piloto apontava
// para lá, a Mesa nova só era alcançável por URL digitada, e é por isso que
// trocar estes quatro `href` É a virada: a partir daqui, entrar numa sessão é
// entrar nela.
//
// UMA função e não quatro `Sprintf`, e a razão vale para os dois sentidos: hoje
// ela é o que faz os quatro caminhos concordarem, e no dia do `git rm` ela é o
// único lugar que precisa ser lido para saber quem manda para onde.
//
// A tela antiga continua de pé e alcançável por URL (decisão do dono): apagar a
// `SessionTrackerPage` é fatia própria, depois de uma sessão de verdade rodar na
// Mesa nova. Enquanto isso, voltar atrás é um `git revert` deste commit.
//
// @example rotaDaMesa(1, 4) // "/mesa/1/4"
func rotaDaMesa(campanhaID, sessaoID int64) string {
	return fmt.Sprintf("/mesa/%d/%d", campanhaID, sessaoID)
}

// mesaParams lê os dois ids da URL. Erro aqui é URL digitada errada, e a
// resposta é uma frase e não um JSON: quem está do outro lado é um navegador
// mostrando uma página.
func mesaParams(w http.ResponseWriter, r *http.Request) (campaignID, sessionID int64, ok bool) {
	campaignID, err1 := strconv.ParseInt(chi.URLParam(r, "campaignId"), 10, 64)
	sessionID, err2 := strconv.ParseInt(chi.URLParam(r, "sessionId"), 10, 64)
	if err1 != nil || err2 != nil {
		http.Error(w, "campanha e sessão precisam ser números", http.StatusBadRequest)
		return 0, 0, false
	}
	return campaignID, sessionID, true
}

// handleMesaPage é a carga fria: o documento inteiro, já com a fila desenhada.
//
// Renderizar o estado JÁ na primeira resposta (em vez de mandar uma casca que
// espera o primeiro tique do SSE) é o que faz a página não piscar vazia — e é
// a mesma lição do `settledQuery` na SPA, um andar acima (ALE-96).
func (s *Server) handleMesaPage(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := mesaParams(w, r)
	if !ok {
		return
	}
	view, status, err := s.loadMesaView(r.Context(), currentUser(r), campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	view.MinhaFicha = s.aFichaDoJogadorNaMesa(r, view)
	// A página é um retrato de agora, e o `escrevePagina` já a manda `no-store`:
	// guardá-la serviria uma fila velha.
	s.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: fmt.Sprintf("Mesa · Sessão %d", view.SessionNum),
		Sinais: sinaisDaMesa(),
		Init:   fmt.Sprintf("@get('/mesa/%d/%d/stream')", campaignID, sessionID),
	}, s.corpoDaMesa(r, view, campaignID, sessionID))
}

// sinaisDaMesa é o estado que mora no NAVEGADOR, agrupado por superfície.
//
// Ele era uma linha só de mil e duzentos caracteres, e a régua a empurrou para
// mil e quinhentos — a essa altura ninguém mais lia o que estava lá dentro, e
// sinal repetido ou esquecido não dá erro em lugar nenhum: ele nasce `undefined`
// no primeiro uso e a expressão que o lê fica muda.
//
// Nomes MINÚSCULOS nos que aparecem como CHAVE de atributo (`data-bind`,
// `data-signals`): o HTML minuscula a chave, e um `data-bind="gabaritoTamanho"`
// liga um sinal NOVO, vazio, ao lado do que se queria. Os que só vivem dentro de
// expressão (`$erroDoComando`) mantêm a caixa.
func sinaisDaMesa() string {
	return "{" + strings.Join([]string{
		// `erro` e `erroDoComando` são DOIS sinais e não um. Um só faria a
		// recusa de "Adicionar grupo" acender a frase vermelha dentro da caixa
		// "Registrar iniciativa" do mestre que também joga: a frase certa no
		// lugar errado, que é como se lê um defeito. Uma palavra por conceito
		// vale para sinal de página como vale para identificador.
		"d20: 10, erro: '', erroDoComando: '', erroDoMovimento: ''",
		// O chão padrão é DERIVADO e não digitado: escrever 'pedra' aqui seria a
		// terceira cópia da mesma escolha (a lista, o servidor e a página), e a
		// que fica para trás quando alguém trocar o padrão é justamente esta —
		// o formulário nasceria oferecendo um chão e o servidor abrindo outro.
		fmt.Sprintf("novolugar: '', novochao: '%s'", tabuleiro.ChaoPadrao()),
		// A SUPERFÍCIE do jogador (ALE-129): qual das duas ocupa a tela. Abre na
		// MESA (decisão do dono) — quem entra na sessão quer saber de quem é a vez
		// e quem está em cena, e o tabuleiro pode nem estar aberto.
		fmt.Sprintf("superficie: '%s'", superficieQueAbrePadrao),
		// A FICHA DENTRO DA SESSÃO (ALE-275). `fichatab` é a seção que a pessoa
		// está olhando — quem a escreve é o clique na aba, e quem a lê é o
		// repedido que o stream dispara; sem ela, um aviso do servidor
		// redesenharia a ficha na aba padrão e tiraria o jogador de onde ele
		// estava. `fichaversao` é o carimbo que o servidor empurra quando o
		// personagem muda no banco, e ele nasce vazio porque a página já chega
		// com a ficha de agora.
		fmt.Sprintf("fichatab: '%s', fichaversao: ''", aAbaPedida("")),
		// O TRILHO de ferramentas: um sinal só, e o valor É a ferramenta.
		"ferramenta: '', marcadorescolhido: '', escolhidosdomapa: ''",
		// O MENU DA PEÇA (ALE-206). `pecaescolhida` é qual menu está aberto e
		// `pecaeditada` é qual peça o diálogo está editando: são DOIS porque abrir
		// o diálogo FECHA o menu, e um sinal só faria o gesto de abrir apagar o
		// alvo do gesto de salvar.
		"pecaescolhida: '', pecaeditada: '', pecanome: '', pecatamanho: 1",
		// A FILA e os verbos da linha.
		"qualidadedodescanso: 'normal', formdecombatente: false",
		"linhadacondicao: '', condicoesdalinha: '', rotulodalinha: ''",
		"novonome: '', novainiciativa: 10, novopv: 0, novotipo: 'npc'",
		"edicaolinha: '', edicaonome: '', edicaoiniciativa: 0, edicaopv: 0, edicaopvmax: 0",
		// O BESTIÁRIO e o elenco.
		"rascunhode: '', pvdoverbete: 0, inidoverbete: 10, copiasdoverbete: 1, nomedonpc: ''",
		// O EDITOR DE BLOCO. O `rascunho` nasce com a FORMA inteira e não vazio,
		// e isso não é enfeite: `data-bind` num caminho que ainda não existe liga
		// um sinal NOVO em vez de escrever no de baixo, e o campo ficaria mudo
		// até o primeiro `@post`. A semente é a mesma do "criar do zero", vinda
		// do `blocoEmBranco` — escrevê-la aqui à mão seria o segundo branco.
		fmt.Sprintf("rascunhoaberto: false, rascunhoaba: %q, erroDoRascunho: '', rascunho: %s",
			abaDosNumeros, oRascunhoEmBranco()),
		// O ENQUADRAMENTO e o arrasto, que são do navegador de ponta a ponta.
		fmt.Sprintf("quadrado: %d", quadradoPadrao),
		"arrastando: '', arrastoinix: 0, arrastoiniy: 0, arrastox: 0, arrastoy: 0",
		// A JANELA sobre o plano infinito (ALE-203): ela substituiu a rolagem
		// nativa, que precisava de uma caixa com fim para ter até onde rolar.
		osSinaisDaJanela,
		// O TRAÇO do pincel e da borracha (ALE-203): o modo em curso e a última
		// casa que ele já mandou.
		osSinaisDoPincel,
		// O LAÇO do retângulo (ALE-203, item 10): o modo em curso e os dois cantos.
		osSinaisDoRetangulo,
		// AS PEÇAS MARCADAS pelo laço (ALE-203, item 10): ids separados por
		// vírgula, numa string só. Ver `sinalDasPecasMarcadas` para por que não é
		// uma lista.
		fmt.Sprintf("%s: '', %s: false", sinalDasPecasMarcadas, sinalDoCliqueEngolido),
		// A RÉGUA: as PARADAS em coordenada do PLANO (podem ser negativas), a mira
		// sob o ponteiro, a fase da máquina, os rótulos de cada perna e a frase do
		// total — as duas últimas escritas pelo servidor.
		osSinaisDaRegua,
		// A PRÉVIA DO ARRASTO: as três faixas da seta viva, os rótulos de cada
		// perna e a frase do custo, todos escritos pelo servidor a cada casa que o
		// dedo atravessa. Ver `piloto_mesa_movimento_previa.go`.
		osSinaisDaPrevia,
		// O GABARITO nasce na PRIMEIRA forma da lista, e ela é derivada e não
		// digitada pelo mesmo motivo do chão: a página que escreve 'esfera' à mão
		// é a que fica para trás no dia em que a ordem mudar, e o defeito seria a
		// barra marcando uma forma e o mapa desenhando outra. `aponta: false`
		// acompanha, porque a esfera vai para todos os lados (p225).
		fmt.Sprintf("gabarito: '%s', gabaritoaponta: %t, gabaritonaintersecao: %t, gabaritotamanho: 2",
			string(asFormasDoLivro[0]), apontaOGabarito(asFormasDoLivro[0]),
			aFormaNasceNaIntersecao(asFormasDoLivro[0])),
		"gabaritox: 0, gabaritoy: 0, gabaritomirax: 0, gabaritomiray: 0, gabaritofase: 0",
		fmt.Sprintf("gabaritopath: '', gabaritotexto: %q", aDicaDoGabaritoVazio),
		// As NOTAS da sessão.
		"notas: '', notassalvas: '', notasmodo: 'duplo', notasabertas: false",
		"notassalvando: false, erroDasNotas: ''",
	}, ", ") + "}"
}

// corpoDaMesa escolhe QUAL DAS DUAS FORMAS a página desenha (ALE-269).
//
// O jogador recebe a coluna — uma superfície que rola, com Grupo, mapa e fila
// empilhados. O mestre recebe o PALCO: trilhos nas bordas e o tabuleiro no
// centro, que é a geometria da `session-gm-view` desde a ALE-198.
//
// SÃO DUAS FORMAS E NÃO DUAS TELAS, e a distinção é o que mantém de pé o
// argumento da ALE-265 contra uma segunda cena: as REGIÕES são as mesmas, com
// os mesmos ids e os mesmos componentes, e o que muda é onde elas são
// penduradas. Duas listas de combatente para manter em dia continuaria sendo o
// defeito da ALE-122; duas ARRUMAÇÕES da mesma lista não é.
//
// O painel do bestiário só nasce para quem pode abri-lo, pela mesma trava do
// resto da fatia: não é a tela que esconde, é a página que não o tem. Mandá-lo
// para todo mundo e escondê-lo por CSS entregaria as 80 criaturas com PV e
// defesa a quem abrisse o inspetor — e esconder PV de NPC é literalmente o que o
// olho da linha faz.
func (s *Server) corpoDaMesa(r *http.Request, view mesaView, campaignID, sessionID int64) templ.Component {
	if view.Mestre == nil {
		return mesa(view)
	}
	return palcoDoMestre(view, s.bestiarioDaMesaPara(r, campaignID, sessionID))
}

// aFichaDoJogadorNaMesa carrega a ficha que a superfície "Minha ficha" desenha.
//
// Só na CARGA FRIA e não no stream (ALE-272, fatia 10b): a ficha é sete painéis
// computados, e ela muda pelos comandos DELA — que remendam o `#cena-ficha`
// direto. Recomputá-la a cada tique da sessão seria pagar o preço mais caro da
// página para descobrir que nada mudou.
//
// Falha em silêncio de propósito: uma ficha que não carrega tira a aba da tela,
// e não derruba a sessão. Estar numa mesa é mais importante que ver a própria
// ficha dentro dela, e o jogador continua tendo o elenco.
func (s *Server) aFichaDoJogadorNaMesa(r *http.Request, view mesaView) *fichaView {
	if view.Mestre != nil || view.Eu == nil {
		return nil
	}
	ficha, _, err := s.carregaFicha(
		r.Context(), currentUser(r), view.Eu.CharacterID, aAbaPedida(""), "", fichaSignals{})
	if err != nil {
		return nil
	}
	ficha.Embutida = true
	return &ficha
}

// loadMesaView busca tudo o que a tela precisa e delega a DECISÃO ao
// `mesaViewOf`. Impuro aqui, puro lá.
func (s *Server) loadMesaView(ctx context.Context, user AuthUser, campaignID, sessionID int64) (mesaView, int, error) {
	sess, role, status, err := s.sessionForCaller(ctx, user, campaignID, sessionID)
	if err != nil {
		return mesaView{}, status, err
	}
	// Hidrata do banco na primeira leitura, como o `onGetState` faz — sem isto
	// um servidor recém-subido serve fila vazia para um combate em andamento.
	if _, err := s.sessions.Load(ctx, sessionID); err != nil {
		return mesaView{}, http.StatusInternalServerError, err
	}
	// `stateForRole` e não `redactForPlayers` direto: é o mesmo gargalo que o
	// socket usa (ALE-122/ALE-210), e papel desconhecido cai em jogador. O
	// piloto não ganha uma segunda decisão sobre quem vê o quê.
	st := aovivo.StateForRole(role, s.sessions.RefreshCharacterMaxes(ctx, sessionID))
	grupo, meus, eu := s.mesaRoster(ctx, user, campaignID)
	view := mesaViewOf(st, campaignID, sessionID, sess.Sessionnumber, grupo, meus, eu)
	// O CICLO da sessão chega à tela (ALE-269): sem o estado, os verbos teriam de
	// ser oferecidos todos, e "encerrar" numa sessão que nunca começou é o gesto
	// que o servidor recusa — oferecer o que será recusado é desenhar um erro.
	view.Status = sess.Status
	if sess.Title.Valid {
		view.Titulo = sess.Title.String
	}
	// O tabuleiro passa pelo MESMO gargalo por papel que a fila: o `BoardForRole`
	// é para o mapa o que o `StateForRole` é para a lista, e é ele que tira as
	// peças escondidas antes de a cena existir. A saúde vem do estado JÁ
	// REDIGIDO, então o combatente cujo PV o mestre ocultou chega sem `HpMax` e a
	// peça dele sai sem barra — a redação alcança o mapa sem uma segunda decisão.
	// O `Mover` diz de quem é a vez e de quem é a peça, e a POSSE é resolvida
	// contra o banco (o `meus` do roster) e nunca contra o cliente — é o mesmo
	// fio de volta até a pessoa que a ALE-33 fixou.
	quemOlha := tabuleiro.Mover{UserID: user.ID, Role: role}
	// A ABA que ESTA pessoa está olhando (ALE-205), e não "o tabuleiro da
	// sessão", que deixou de existir como coisa única. Ela é resolvida contra os
	// abertos, então a aba que o mestre fechou não deixa ninguém numa tela morta.
	aba, puxado, deOnde := s.aAbaComOPuxao(ctx, sessionID, user.ID)
	cena := tabuleiro.BoardForRole(role, s.boards.Get(ctx, sessionID, aba))
	// A LENTE DO MESTRE (ALE-193): com ela ligada, o que se desenha é a cena
	// REDIGIDA — a mesma que a mesa recebe. Só a CENA muda; o `quemOlha` continua
	// dizendo "mestre", porque a lente é sobre o que ele vê e não sobre o que ele
	// pode: ele confere a emboscada sem parar de montá-la.
	escondidas := 0
	naLente := role == "gm" && s.lentes.Ligada(sessionID, user.ID)
	if naLente {
		cena, escondidas = aCenaComoAMesaVe(cena)
	}
	view.Tabuleiro = tabuleiroViewOf(
		cena, st, saudeDaFila(st), combatenteDaVez(st), quemOlha, meus, campaignID, sessionID,
	)
	view.Tabuleiro.Lente = naLente
	view.Tabuleiro.PecasEscondidas = escondidas
	// A BARRA DE ABAS (ALE-205) é a lista dos abertos, redigida pelo papel de
	// quem olha — e ela vem depois da lente de propósito: a lente é sobre a CENA
	// que o mestre está vendo, não sobre quais cenas existem. Um mestre na lente
	// que perdesse as abas não teria como sair da que está olhando.
	view.Tabuleiro.Abas = asAbasDaMesa(s.boards.Abertos(ctx, sessionID), role, aba, campaignID, sessionID)
	// A TIRA DO PUXÃO vem depois da barra porque ela é feita DELA: os nomes já
	// passaram pelo papel de quem olha, e ler o estado cru aqui contaria o nome
	// de uma cena sob cortina a quem não pode sabê-lo.
	if puxado {
		view.Tabuleiro.Puxado = aTiraDoPuxao(view.Tabuleiro.Abas, deOnde)
	}
	// O ACERVO é do mestre, pela mesma razão do rastreador: a mesa não escolhe
	// onde joga. A trava é a view não ter o que desenhar, e não a tela esconder.
	if role == "gm" {
		view.Tabuleiro.Acervo = acervoDaCampanha(s.boards.Places(ctx, campaignID), s.boards.Abertos(ctx, sessionID))
	}
	// O rastreador só é MONTADO para o mestre. A trava não é a tela esconder o
	// bloco: é a view não ter o que desenhar, pelo mesmo `role` que o
	// `stateForRole` já usou para redigir o estado.
	// AS NOTAS são do mestre e chegam JÁ EM ÁRVORE (ALE-269). Elas não entram no
	// `mesaViewOf` porque não vêm do estado ao vivo: moram na linha da sessão,
	// que é o mesmo lugar do título e do ciclo.
	if role == "gm" && sess.Notes.Valid {
		view.Notas = sess.Notes.String
		view.NotasBlocos = parseNota(sess.Notes.String)
	}
	if role == "gm" {
		view.NPCs = s.oElencoDaCampanha(ctx, campaignID)
	}
	if role == "gm" {
		membros, presentes := s.membrosEPresenca(ctx, campaignID, sessionID)
		r := mestreViewOf(st, membros, presentes, true)
		// A presença é escrita nos cartões DEPOIS de o papel ser resolvido, e é
		// isso que a mantém fora da tela do jogador sem uma segunda decisão na
		// cena: quem não é mestre não chega aqui, e lá o campo continua nil.
		marcaAPresenca(view.Grupo, r.Conectados)
		view.Mestre = &r
	}
	return view, http.StatusOK, nil
}

// mesaRoster traduz o roster da campanha nas três coisas que a tela quer: os
// cartões do Grupo, o conjunto dos MEUS personagens, e qual deles registra
// iniciativa.
//
// A ponte até "quem está olhando" é o `ownerId` do roster, e não o id do
// personagem: a ficha de um membro é o SNAPSHOT da campanha (ALE-33), então o
// dono registrado é o único fio de volta até a pessoa. Mesmo caminho do
// `myCharacterIdsOf` na SPA, pelo mesmo motivo.
func (s *Server) mesaRoster(ctx context.Context, user AuthUser, campaignID int64) ([]mesaMembro, map[int64]bool, *mesaEu) {
	rows, err := s.queries.ListMembers(ctx, campaignID)
	if err != nil {
		// Roster indisponível não derruba a fila: a iniciativa é o assunto da
		// tela, e o Grupo é o cartão ao lado.
		return nil, map[int64]bool{}, nil
	}
	grupo := make([]mesaMembro, 0, len(rows))
	meus := make(map[int64]bool, len(rows))
	var eu *mesaEu
	for _, m := range rows {
		if dono, err := s.queries.GetCharacterOwner(ctx, m.Characterid); err == nil && dono == user.ID {
			meus[m.Characterid] = true
			if eu == nil {
				eu = &mesaEu{CharacterID: m.Characterid, Nome: m.Charname}
			}
		}
		// O filtro de PAPEL é o mesmo do "Adicionar grupo" no servidor
		// (ALE-212): o mestre costuma ter um PC próprio no roster, e listá-lo
		// aqui faria duas telas discordarem sobre quem é o grupo.
		if m.Role != "player" {
			continue
		}
		grupo = append(grupo, mesaMembro{
			CharacterID: m.Characterid,
			Nome:        m.Charname,
			Iniciais:    iniciais(m.Charname),
			Defesa:      s.defesaDoMembro(ctx, m.Characterid),
			Nivel:       m.Charlevel,
			Classes:     s.mesaClasses(ctx, m.Characterid),
			PV:          mesaBarraDe(m.Charhpcurrent, m.Charhpmax, false),
			PM:          mesaBarraDe(m.Charmpcurrent, m.Charmpmax, true),
		})
	}
	if eu != nil {
		// O bônus é do MOTOR, nunca do template: é a mesma `ComputeSheetV2` que
		// a ficha inteira usa (ALE-213).
		if bonus, err := s.initiativeBonus(ctx, eu.CharacterID); err == nil {
			eu.Bonus = bonus
		}
	}
	return grupo, meus, eu
}

// mesaClasses monta "Guerreiro 3 / Ladino 2".
func (s *Server) mesaClasses(ctx context.Context, characterID int64) string {
	classes, err := s.queries.ListClassesByCharacter(ctx, characterID)
	if err != nil || len(classes) == 0 {
		return ""
	}
	out := ""
	for i, c := range classes {
		if i > 0 {
			out += " / "
		}
		out += c.Classname + " " + strconv.FormatInt(c.Level, 10)
	}
	return out
}

// mesaTick é a cadência do stream. 200ms é a medida que a comunidade do
// Datastar pratica (o Game of Life multiplayer do Anders Murphy re-renderiza o
// <main> inteiro nessa cadência), e ela é folgada para uma mesa de RPG: o que
// muda é um turno por vez, não um quadro por vez.
//
// Só sai byte quando o HTML MUDA — ver `handleMesaStream`.
const mesaTick = 200 * time.Millisecond

// membrosEPresenca junta o que as regras de presença precisam.
//
// Roster indisponível não derruba a cena: a presença é enfeite ao lado dos
// nomes, e a fila é o assunto da tela.
func (s *Server) membrosEPresenca(ctx context.Context, campaignID, sessionID int64) ([]aovivo.MembroDaMesa, []int64) {
	rows, err := s.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, nil
	}
	membros := make([]aovivo.MembroDaMesa, 0, len(rows))
	for _, m := range rows {
		dono, err := s.queries.GetCharacterOwner(ctx, m.Characterid)
		if err != nil {
			continue
		}
		membros = append(membros, aovivo.MembroDaMesa{CharacterID: m.Characterid, DonoID: dono})
	}
	var presentes []int64
	for _, u := range s.presence.Roster(sessionID) {
		presentes = append(presentes, u.UserID)
	}
	return membros, presentes
}

// defesaDoMembro pergunta ao MOTOR, que é a mesma `ComputeSheetV2` da ficha.
//
// Travessão quando o motor não está de pé: a cena inteira não pode cair por
// causa de um número, e um zero seria pior — Defesa 0 é um valor plausível, e o
// mestre agiria sobre ele. É a mesma escolha que o cartão de personagem faz.
func (s *Server) defesaDoMembro(ctx context.Context, characterID int64) string {
	if s.catalogs == nil {
		return "—"
	}
	row, err := s.queries.GetCharacter(ctx, characterID)
	if err != nil {
		return "—"
	}
	ficha, err := s.ComputeSheet(ctx, row)
	if err != nil {
		return "—"
	}
	return strconv.Itoa(ficha.Defense.Total)
}
