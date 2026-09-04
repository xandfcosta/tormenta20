package table

import (
	"context"
	"t20engine/markdown"
	"t20engine/web/sheetui"

	"fmt"
	"net/http"
	"strconv"
	"strings"
	"t20engine/aovivo"
	"t20engine/book"
	"t20engine/tabuleiro"
	"time"

	"github.com/a-h/templ"

	"t20engine/web/ui"

	"github.com/go-chi/chi/v5"
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
// Routes registra as vinte rotas da Mesa (ALE-278).
//
// Elas moravam dentro do `WebRouter` — o roteador do app INTEIRO —, que também
// vivia neste arquivo por acidente de história: a Mesa foi a primeira cena do
// piloto, então o mux nasceu no arquivo dela e as onze cenas seguintes foram
// sendo penduradas ali. O roteador ficou no `api`, que é quem monta o app; o
// que veio para cá é só o grupo da Mesa.
//
// O `requirePage` NÃO está aqui: quem decide que esta cena exige sessão é o
// hospedeiro, no grupo em que ela é montada. Uma cena que se autoprotegesse
// daria a impressão de que a fronteira é dela, e ela não é.
func Routes(r chi.Router, s Scene) {
	r.Get("/mesa/{campaignId}/{sessionId}", s.handleTablePage)
	r.Get("/mesa/{campaignId}/{sessionId}/stream", s.handleTableStream)
	r.Post("/mesa/{campaignId}/{sessionId}/iniciativa", s.handleTableInitiative)
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
	s.RoutesSession(r)
	s.RoutesNote(r)
	s.CastRoutes(r)
	s.RoutesNpc(r)
	s.RoutesEditorNpc(r)
	// O RASCUNHO DE LUGAR (ALE-292) entra por aqui apesar de o endereço dele ser
	// `/campanhas/…`: quem o desenha é o TABULEIRO, que é desta cena, e montá-lo
	// no grupo das campanhas exigiria que aquele pacote alcançasse este.
	s.DraftRoutes(r)
}

// O endereço da Mesa mora em `web/routes` desde a ALE-278 (`routes.Table`): a
// cena das campanhas o cita, e depois de virar pacote ela não alcança mais uma
// função daqui. É o critério de lá reclassificando pela terceira vez.

// tableParams lê os dois ids da URL. Erro aqui é URL digitada errada, e a
// resposta é uma frase e não um JSON: quem está do outro lado é um navegador
// mostrando uma página.
func tableParams(w http.ResponseWriter, r *http.Request) (campaignID, sessionID int64, ok bool) {
	campaignID, err1 := strconv.ParseInt(chi.URLParam(r, "campaignId"), 10, 64)
	sessionID, err2 := strconv.ParseInt(chi.URLParam(r, "sessionId"), 10, 64)
	if err1 != nil || err2 != nil {
		http.Error(w, "campanha e sessão precisam ser números", http.StatusBadRequest)
		return 0, 0, false
	}
	return campaignID, sessionID, true
}

// handleTablePage é a carga fria: o documento inteiro, já com a fila desenhada.
//
// Renderizar o estado JÁ na primeira resposta (em vez de mandar uma casca que
// espera o primeiro tique do SSE) é o que faz a página não piscar vazia — e é
// a mesma lição do `settledQuery` na SPA, um andar acima (ALE-96).
func (s Scene) handleTablePage(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	view, status, err := s.LoadView(r.Context(), s.deps.CurrentUserID(r), campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	view.MinhaFicha = s.tablePlayerSheet(r, view)
	// A página é um retrato de agora, e o `WritePage` já a manda `no-store`:
	// guardá-la serviria uma fila velha.
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: fmt.Sprintf("Mesa · Sessão %d", view.SessionNum),
		Sinais: tableSignalsExpr(),
		Init:   fmt.Sprintf("@get('/mesa/%d/%d/stream')", campaignID, sessionID),
	}, s.tableBody(r, view, campaignID, sessionID))
}

// tableSignalsExpr é o estado que mora no NAVEGADOR, agrupado por superfície.
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
func tableSignalsExpr() string {
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
		fmt.Sprintf("novolugar: '', novochao: '%s'", tabuleiro.DefaultGround()),
		// A SUPERFÍCIE do jogador (ALE-129): qual das duas ocupa a tela. Abre na
		// MESA (decisão do dono) — quem entra na sessão quer saber de quem é a vez
		// e quem está em cena, e o tabuleiro pode nem estar aberto.
		fmt.Sprintf("superficie: '%s'", DefaultOpeningSurface),
		// A FICHA DENTRO DA SESSÃO (ALE-275). `fichatab` é a seção que a pessoa
		// está olhando — quem a escreve é o clique na aba, e quem a lê é o
		// repedido que o stream dispara; sem ela, um aviso do servidor
		// redesenharia a ficha na aba padrão e tiraria o jogador de onde ele
		// estava. `fichaversao` é o carimbo que o servidor empurra quando o
		// personagem muda no banco, e ele nasce vazio porque a página já chega
		// com a ficha de agora.
		fmt.Sprintf("fichatab: '%s', fichaversao: ''", sheetui.AskedTab("")),
		// O TRILHO de ferramentas: um sinal só, e o valor É a ferramenta.
		"ferramenta: '', marcadorescolhido: '', escolhidosdomapa: ''",
		// O MENU DA PEÇA (ALE-206). `pecaescolhida` é qual menu está aberto e
		// `pecaeditada` é qual peça o diálogo está editando: são DOIS porque abrir
		// o diálogo FECHA o menu, e um sinal só faria o gesto de abrir apagar o
		// alvo do gesto de salvar.
		"pecaescolhida: '', pecaeditada: '', pecanome: '', pecatamanho: 1",
		// A PEÇA AVULSA (ALE-291) — a porta, o baú, o barril. Os nomes levam
		// `nova` porque `pecanome` e `pecatamanho` JÁ SÃO do diálogo de EDITAR
		// peça, logo acima, e vivem no mesmo documento: reusá-los faria o gesto
		// de criar escrever no alvo do gesto de salvar, que é o defeito que a
		// linha do `buscador` no GLOSSARIO existe para impedir.
		"novapecanome: '', novapecatamanho: 1, novapecaaparencia: 'object'",
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
			abaDosNumeros, blankDraft()),
		// O ENQUADRAMENTO e o arrasto, que são do navegador de ponta a ponta.
		fmt.Sprintf("quadrado: %d", DefaultSquare),
		"arrastando: '', arrastoinix: 0, arrastoiniy: 0, arrastox: 0, arrastoy: 0",
		// A JANELA sobre o plano infinito (ALE-203): ela substituiu a rolagem
		// nativa, que precisava de uma caixa com fim para ter até onde rolar.
		viewportSignals,
		// O TRAÇO do pincel e da borracha (ALE-203): o modo em curso e a última
		// casa que ele já mandou.
		brushSignals,
		// O LAÇO do retângulo (ALE-203, item 10): o modo em curso e os dois cantos.
		rectSignals,
		// AS PEÇAS MARCADAS pelo laço (ALE-203, item 10): ids separados por
		// vírgula, numa string só. Ver `markedTokensSignal` para por que não é
		// uma lista.
		fmt.Sprintf("%s: '', %s: false", markedTokensSignal, sinalDoCliqueEngolido),
		// A RÉGUA: as PARADAS em Coordinate do PLANO (podem ser negativas), a mira
		// sob o ponteiro, a fase da máquina, os rótulos de cada perna e a frase do
		// total — as duas últimas escritas pelo servidor.
		rulerSignals,
		// A PRÉVIA DO ARRASTO: as três faixas da seta viva, os rótulos de cada
		// perna e a frase do custo, todos escritos pelo servidor a cada casa que o
		// dedo atravessa. Ver `move_preview.go`.
		previewSignals,
		// O GABARITO nasce na PRIMEIRA forma da lista, e ela é derivada e não
		// digitada pelo mesmo motivo do chão: a página que escreve 'esfera' à mão
		// é a que fica para trás no dia em que a ordem mudar, e o defeito seria a
		// barra marcando uma forma e o mapa desenhando outra. `aponta: false`
		// acompanha, porque a esfera vai para todos os lados (p225).
		fmt.Sprintf("gabarito: '%s', gabaritoaponta: %t, gabaritonaintersecao: %t, gabaritotamanho: 2",
			string(bookShapes[0]), pointsTemplate(bookShapes[0]),
			shapeStartsAtIntersection(bookShapes[0])),
		"gabaritox: 0, gabaritoy: 0, gabaritomirax: 0, gabaritomiray: 0, gabaritofase: 0",
		fmt.Sprintf("gabaritopath: '', gabaritotexto: %q", emptyTemplateHint),
		// As NOTAS da sessão.
		"notas: '', notassalvas: '', notasmodo: 'duplo', notasabertas: false",
		"notassalvando: false, erroDasNotas: ''",
	}, ", ") + "}"
}

// tableBody escolhe QUAL DAS DUAS FORMAS a página desenha (ALE-269).
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
func (s Scene) tableBody(r *http.Request, view View, campaignID, sessionID int64) templ.Component {
	if view.Mestre == nil {
		return tableScene(view)
	}
	return gmStage(view, s.forTableBestiary(r, campaignID, sessionID))
}

// tablePlayerSheet carrega a ficha que a superfície "Minha ficha" desenha.
//
// Só na CARGA FRIA e não no stream (ALE-272, fatia 10b): a ficha é sete painéis
// computados, e ela muda pelos comandos DELA — que remendam o `#cena-ficha`
// direto. Recomputá-la a cada tique da sessão seria pagar o preço mais caro da
// página para descobrir que nada mudou.
//
// Falha em silêncio de propósito: uma ficha que não carrega tira a aba da tela,
// e não derruba a sessão. Estar numa mesa é mais importante que ver a própria
// ficha dentro dela, e o jogador continua tendo o elenco.
func (s Scene) tablePlayerSheet(r *http.Request, view View) *sheetui.View {
	if view.Mestre != nil || view.Eu == nil {
		return nil
	}
	return s.deps.PlayerSheet(r, view.Eu.CharacterID)
}

// LoadView busca tudo o que a tela precisa e delega a DECISÃO ao
// `tableViewOf`. Impuro aqui, puro lá.
// LoadView monta a Mesa inteira para desenhar.
//
// Ela é exportada porque a BANCADA do hospedeiro a chama: nove casos provam o
// caminho BANCO → PALCO — quem está na fila sai no elenco, com a Defesa e a
// marca de "já está no mapa" —, e este pacote não tem banco. Importar o
// `db/testdb` junto com um `*api.Server` seria o ciclo que a divisão existe
// para evitar.
//
// É a mesma direção do `characters.Load` e do `master.LoadBestiaryFrom`: a cena
// diz como montar a si mesma, e o hospedeiro prova que o que está no banco
// chega até lá.
func (s Scene) LoadView(ctx context.Context, userID int64, campaignID, sessionID int64) (View, int, error) {
	sess, role, status, err := s.deps.SessionForCaller(ctx, userID, campaignID, sessionID)
	if err != nil {
		return View{}, status, err
	}
	// Hidrata do banco na primeira leitura, como o `onGetState` faz — sem isto
	// um servidor recém-subido serve fila vazia para um combate em andamento.
	if _, err := s.deps.Sessions().Load(ctx, sessionID); err != nil {
		return View{}, http.StatusInternalServerError, err
	}
	// `stateForRole` e não `redactForPlayers` direto: é o mesmo gargalo que o
	// socket usa (ALE-122/ALE-210), e papel desconhecido cai em jogador. O
	// piloto não ganha uma segunda decisão sobre quem vê o quê.
	st := aovivo.StateForRole(role, s.deps.Sessions().RefreshCharacterMaxes(ctx, sessionID))
	grupo, meus, eu := s.tableRoster(ctx, userID, campaignID)
	view := tableViewOf(st, campaignID, sessionID, sess.Sessionnumber, grupo, meus, eu)
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
	quemOlha := tabuleiro.Mover{UserID: userID, Role: role}
	// A ABA que ESTA pessoa está olhando (ALE-205), e não "o tabuleiro da
	// sessão", que deixou de existir como coisa única. Ela é resolvida contra os
	// abertos, então a aba que o mestre fechou não deixa ninguém numa tela morta.
	aba, puxado, deOnde := s.pullTab(ctx, sessionID, userID)
	scene := tabuleiro.BoardForRole(role, s.deps.Boards().Get(ctx, sessionID, aba))
	// A LENTE DO MESTRE (ALE-193): com ela ligada, o que se desenha é a cena
	// REDIGIDA — a mesma que a mesa recebe. Só a CENA muda; o `quemOlha` continua
	// dizendo "mestre", porque a lente é sobre o que ele vê e não sobre o que ele
	// pode: ele confere a emboscada sem parar de montá-la.
	escondidas := 0
	naLente := role == "gm" && s.lenses.On(sessionID, userID)
	if naLente {
		scene, escondidas = seesTableHowScene(scene)
	}
	view.Tabuleiro = boardViewOf(
		scene, st, saudeDaFila(st), turnCombatant(st), quemOlha, meus, campaignID, sessionID,
	)
	view.Tabuleiro.Lente = naLente
	view.Tabuleiro.PecasEscondidas = escondidas
	// A BARRA DE ABAS (ALE-205) é a lista dos abertos, redigida pelo papel de
	// quem olha — e ela vem depois da lente de propósito: a lente é sobre a CENA
	// que o mestre está vendo, não sobre quais cenas existem. Um mestre na lente
	// que perdesse as abas não teria como sair da que está olhando.
	view.Tabuleiro.Abas = tableTabs(s.deps.Boards().OpenBoards(ctx, sessionID), role, aba, campaignID, sessionID)
	// A TIRA DO PUXÃO vem depois da barra porque ela é feita DELA: os nomes já
	// passaram pelo papel de quem olha, e ler o estado cru aqui contaria o nome
	// de uma cena sob cortina a quem não pode sabê-lo.
	if puxado {
		view.Tabuleiro.Puxado = removePull(view.Tabuleiro.Abas, deOnde)
	}
	// O ACERVO é do mestre, pela mesma razão do rastreador: a mesa não escolhe
	// onde joga. A trava é a view não ter o que desenhar, e não a tela esconder.
	if role == "gm" {
		view.Tabuleiro.Acervo = campaignCollection(s.deps.Boards().Places(ctx, campaignID), s.deps.Boards().OpenBoards(ctx, sessionID))
	}
	// O rastreador só é MONTADO para o mestre. A trava não é a tela esconder o
	// bloco: é a view não ter o que desenhar, pelo mesmo `role` que o
	// `stateForRole` já usou para redigir o estado.
	// AS NOTAS são do mestre e chegam JÁ EM ÁRVORE (ALE-269). Elas não entram no
	// `tableViewOf` porque não vêm do estado ao vivo: moram na linha da sessão,
	// que é o mesmo lugar do título e do ciclo.
	if role == "gm" && sess.Notes.Valid {
		view.Notas = sess.Notes.String
		view.NotasBlocos = markdown.Parse(sess.Notes.String)
	}
	if role == "gm" {
		view.NPCs = s.CampaignCast(ctx, campaignID)
	}
	if role == "gm" {
		membros, presentes := s.membrosEPresenca(ctx, campaignID, sessionID)
		r := ofViewGm(st, membros, presentes, true, s.deps.SaveFailed(sessionID))
		// A presença é escrita nos cartões DEPOIS de o papel ser resolvido, e é
		// isso que a mantém fora da tela do jogador sem uma segunda decisão na
		// cena: quem não é mestre não chega aqui, e lá o campo continua nil.
		marcaAPresenca(view.Grupo, r.Conectados)
		view.Mestre = &r
	}
	return view, http.StatusOK, nil
}

// tableRoster traduz o roster da campanha nas três coisas que a tela quer: os
// cartões do Grupo, o conjunto dos MEUS personagens, e qual deles registra
// iniciativa.
//
// A ponte até "quem está olhando" é o `ownerId` do roster, e não o id do
// personagem: a ficha de um membro é o SNAPSHOT da campanha (ALE-33), então o
// dono registrado é o único fio de volta até a pessoa. Mesmo caminho do
// `myCharacterIdsOf` na SPA, pelo mesmo motivo.
func (s Scene) tableRoster(ctx context.Context, userID int64, campaignID int64) ([]Member, map[int64]bool, *tableMe) {
	rows, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		// Roster indisponível não derruba a fila: a iniciativa é o assunto da
		// tela, e o Grupo é o cartão ao lado.
		return nil, map[int64]bool{}, nil
	}
	grupo := make([]Member, 0, len(rows))
	meus := make(map[int64]bool, len(rows))
	var eu *tableMe
	for _, m := range rows {
		if dono, err := s.deps.Queries().GetCharacterOwner(ctx, m.Characterid); err == nil && dono == userID {
			meus[m.Characterid] = true
			if eu == nil {
				eu = &tableMe{CharacterID: m.Characterid, Nome: m.Charname}
			}
		}
		// Aqui morava um filtro de PAPEL, e ele NUNCA excluiu ninguém.
		//
		// A intenção estava escrita (ALE-212): "o mestre costuma ter um PC
		// próprio no roster, e listá-lo aqui faria duas telas discordarem sobre
		// quem é o grupo". Mas a condição era `m.Role != "player"` sobre uma
		// coluna que valia `'player'` em toda linha, então ela nunca excluiu
		// nada — e a coluna saiu na ALE-287.
		//
		// **Ele não volta, e a razão é do dono da mesa:** o mestre NÃO tem
		// personagem próprio. O que ele tem é um elenco de NPCs que entram na
		// história ou não — e essa decisão é por CENA, tomada na hora de pôr a
		// linha na fila. NPC nem é membro da campanha: ele entra na iniciativa
		// por `label` e `initiative`, sem `characterId` (ver `materializeEntry`).
		//
		// Ou seja: `campaign_members` só tem personagem de jogador, e o grupo é
		// o grupo. Não havia o que filtrar.
		grupo = append(grupo, Member{
			CharacterID: m.Characterid,
			Nome:        m.Charname,
			Iniciais:    ui.Monogram(m.Charname),
			Defesa:      s.memberDefense(ctx, m.Characterid),
			Nivel:       m.Charlevel,
			Classes:     s.tableClasses(ctx, m.Characterid),
			PV:          tableBarOf(m.Charhpcurrent, m.Charhpmax, false),
			PM:          tableBarOf(m.Charmpcurrent, m.Charmpmax, true),
		})
	}
	if eu != nil {
		// O bônus é do MOTOR, nunca do template: é a mesma `ComputeSheetV2` que
		// a ficha inteira usa (ALE-213).
		if bonus, err := s.deps.InitiativeBonus(ctx, eu.CharacterID); err == nil {
			eu.Bonus = bonus
		}
	}
	return grupo, meus, eu
}

// tableClasses monta "Guerreiro 3 / Ladino 2".
func (s Scene) tableClasses(ctx context.Context, characterID int64) string {
	classes, err := s.deps.Queries().ListClassesByCharacter(ctx, characterID)
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

// tableTick é a cadência do stream. 200ms é a medida que a comunidade do
// Datastar pratica (o Game of Life multiplayer do Anders Murphy re-renderiza o
// <main> inteiro nessa cadência), e ela é folgada para uma mesa de RPG: o que
// muda é um turno por vez, não um quadro por vez.
//
// Só sai byte quando o HTML MUDA — ver `handleTableStream`.
const tableTick = 200 * time.Millisecond

// membrosEPresenca junta o que as regras de presença precisam.
//
// Roster indisponível não derruba a cena: a presença é enfeite ao lado dos
// nomes, e a fila é o assunto da tela.
func (s Scene) membrosEPresenca(ctx context.Context, campaignID, sessionID int64) ([]aovivo.TableMember, []int64) {
	rows, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		return nil, nil
	}
	membros := make([]aovivo.TableMember, 0, len(rows))
	for _, m := range rows {
		dono, err := s.deps.Queries().GetCharacterOwner(ctx, m.Characterid)
		if err != nil {
			continue
		}
		membros = append(membros, aovivo.TableMember{CharacterID: m.Characterid, OwnerID: dono})
	}
	var presentes []int64
	for _, u := range s.deps.Presence().Roster(sessionID) {
		presentes = append(presentes, u.UserID)
	}
	return membros, presentes
}

// memberDefense pergunta ao MOTOR, que é a mesma `ComputeSheetV2` da ficha.
//
// Travessão quando o motor não está de pé: a cena inteira não pode cair por
// causa de um número, e um zero seria pior — Defesa 0 é um valor plausível, e o
// mestre agiria sobre ele. É a mesma escolha que o cartão de personagem faz.
func (s Scene) memberDefense(ctx context.Context, characterID int64) string {
	if s.deps.Catalogs() == nil {
		return "—"
	}
	row, err := s.deps.Queries().GetCharacter(ctx, characterID)
	if err != nil {
		return "—"
	}
	ficha, err := s.deps.ComputedSheet(ctx, row)
	if err != nil {
		return "—"
	}
	// A MESMA frase da ficha, e da mesma função: o mestre confere aqui a Defesa
	// de um jogador para decidir se o ataque acerta, e com o alvo caído o total
	// é o único número que não responde essa pergunta (ALE-274).
	return book.DefenseLabel(ficha.Defense)
}
