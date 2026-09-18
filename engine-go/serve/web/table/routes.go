package table

import (
	"context"
	"t20engine/app"
	"t20engine/domain/markdown"
	"t20engine/domain/sheet"
	"t20engine/serve/web/sheetui"

	"fmt"
	"net/http"
	"strconv"
	"strings"
	"t20engine/domain/board"
	"t20engine/domain/book"
	"t20engine/domain/live"

	"github.com/a-h/templ"

	"t20engine/serve/web/ui"

	"github.com/go-chi/chi/v5"
)

// Routes registra as rotas da Mesa e as do RASCUNHO DE LUGAR.
//
// O `requirePage` NÃO está aqui: quem decide que esta cena exige sessão é o
// hospedeiro, no grupo em que ela é montada. Uma cena que se autoprotegesse
// daria a impressão de que a fronteira é dela.
// sessionPattern é o padrão que o chi casa para esta cena — o endereço da
// sessão, com os dois parâmetros nomeados.
//
// Escrito UMA vez e não em cada `Routes*`: a ALE-345 trocou este prefixo em 35
// registros, e a única razão de terem sido 35 é ele estar copiado. O endereço
// RESOLVIDO (com os ids) é outra coisa e mora no `routes.Session`.
const sessionPattern = "/campanhas/{campaignId}/sessoes/{sessionId}"

func Routes(r chi.Router, s Scene) {
	r.Get(sessionPattern, s.handleTablePage)
	r.Get(sessionPattern+"/fluxo", s.handleTableStream)
	r.Post(sessionPattern+"/iniciativa", s.handleTableInitiative)
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
	// O RASCUNHO DE LUGAR entra por aqui apesar de o endereço dele ser
	// `/campanhas/…`: quem o desenha é o TABULEIRO, que é desta cena.
	s.DraftRoutes(r)
}

// O ENDEREÇO da cena mora em `web/routes` (`routes.Session`), e não aqui: a cena
// das campanhas o cita, e ela não alcança uma função deste pacote.

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
// Renderizar o estado JÁ na primeira resposta, em vez de mandar uma casca que
// espera o primeiro tique do SSE, é o que faz a página não piscar vazia.
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
		Init:   fmt.Sprintf("@get('/campanhas/%d/sessoes/%d/fluxo')", campaignID, sessionID),
		// A ILHA DA MESA: o que anima quando o estado chega pelo fio.
		//
		// Módulo PRÓPRIO e não `scene.js`, que carrega em toda página: um
		// observador de mutação sobre o tabuleiro não tem o que fazer na ficha
		// nem na porta. É a mesma divisão que o leitor do livro já usa.
		Scripts: []string{s.deps.Asset("table.js")},
	}, s.tableBody(r, view, campaignID, sessionID))
}

// tableSignalsExpr é o estado que mora no NAVEGADOR, agrupado por superfície.
//
// Agrupado e comentado porque sinal repetido ou esquecido não dá erro em lugar
// nenhum: ele nasce `undefined` no primeiro uso e a expressão que o lê fica
// muda.
//
// Nomes MINÚSCULOS nos que aparecem como CHAVE de atributo (`data-bind`,
// `data-signals`): o HTML minuscula a chave, e um `data-bind="gabaritoTamanho"`
// liga um sinal NOVO, vazio, ao lado do que se queria. Os que só vivem dentro de
// expressão (`$command_error`) mantêm a caixa.
func tableSignalsExpr() string {
	return "{" + strings.Join([]string{
		// `error` e `command_error` são DOIS sinais e não um. Um só faria a
		// recusa de "Adicionar grupo" acender a frase vermelha dentro da caixa
		// "Registrar iniciativa" do mestre que também joga: a frase certa no
		// lugar errado, que é como se lê um defeito.
		//
		// Renomear um destes é renomear a DECLARAÇÃO aqui junto com o leitor e o
		// escritor: esquecê-la não estoura nada, porque a expressão passa a ler
		// `undefined` e `undefined != ''` é verdadeiro — o `<p>` da recusa nasce
		// MOSTRADO (vazio, logo invisível). O
		// `TestEverySignalDeclaredByValueHasAReader` não alcança este canal: ele
		// lê o VALOR de atributo (`data-ref="x"`), e a declaração aqui é uma
		// string montada em Go.
		"d20: 10, error: '', command_error: '', move_error: ''",
		// O chão padrão é DERIVADO e não digitado: escrever 'pedra' aqui seria a
		// terceira cópia da mesma escolha (a lista, o servidor e a página), e a
		// que fica para trás quando alguém trocar o padrão é justamente esta —
		// o formulário nasceria oferecendo um chão e o servidor abrindo outro.
		fmt.Sprintf("new_place: '', new_ground: '%s'", board.DefaultGround()),
		// A SUPERFÍCIE do jogador: qual das duas ocupa a tela. Abre na
		// MESA (decisão do dono) — quem entra na sessão quer saber de quem é a vez
		// e quem está em cena, e o tabuleiro pode nem estar aberto.
		fmt.Sprintf("surface: '%s'", DefaultOpeningSurface),
		// A FICHA DENTRO DA SESSÃO. `sheet_tab` é a seção que a pessoa está
		// olhando — quem a escreve é o clique na aba, e quem a lê é o repedido
		// que o stream dispara; sem ela, um aviso do servidor redesenharia a
		// ficha na aba padrão e tiraria o jogador de onde ele estava.
		// `sheet_version` é o carimbo que o servidor empurra quando o personagem
		// muda no banco, e ele nasce vazio porque a página já chega com a ficha
		// de agora.
		fmt.Sprintf("sheet_tab: '%s', sheet_version: ''", sheetui.AskedTab("")),
		// O TRILHO de ferramentas: um sinal só, e o valor É a ferramenta.
		"tool: '', marker_chosen: '', map_selection: ''",
		// O MENU DA PEÇA. `token_chosen` é qual menu está aberto e `token_edited`
		// é qual peça o diálogo está editando: são DOIS porque abrir o diálogo
		// FECHA o menu, e um sinal só faria o gesto de abrir apagar o alvo do
		// gesto de salvar. A SEGUNDA CAMADA do menu não tem sinal: ela é popover
		// NATIVO, e quem guarda o aberto/fechado é o navegador.
		"token_chosen: '', token_edited: '', token_name: '', token_size: 1",
		// A ÁREA DE TRANSFERÊNCIA da peça, e ela é do CLIENTE de propósito: é a
		// área de QUEM COPIOU, não da mesa. No servidor ela seria um estado por
		// usuário e por sessão que ninguém pediu, e que o mestre encontraria
		// cheio no dia seguinte.
		//
		// `area_board` guarda de ONDE a peça veio, e é ele que faz o colar
		// atravessar as abas: a original pode não estar no tabuleiro em que se
		// cola, e sem a origem o servidor não teria onde procurá-la.
		//
		// `area_mode` é o valor que o SERVIDOR consome e `area_phrase` é o que a
		// pessoa lê: sem os dois, a faixa diria "Zumbi · sozinha", que é um
		// identificador de código na tela de alguém.
		"area_token: '', area_board: '', area_mode: '', area_label: '', area_phrase: ''",
		// A PEÇA AVULSA — a porta, o baú, o barril. Os nomes levam `new_` porque
		// `token_name` e `token_size` JÁ SÃO do diálogo de EDITAR peça, logo
		// acima, e vivem no mesmo documento: reusá-los faria o gesto de criar
		// escrever no alvo do gesto de salvar.
		"new_token_name: '', new_token_size: 1, new_token_look: 'object'",
		// A FILA e os verbos da linha.
		"rest_quality: 'normal', combatant_form: false",
		"condition_row: '', row_conditions: '', row_label: ''",
		"new_name: '', new_initiative: 10, new_hp: 0, new_type: 'npc'",
		"edit_row: '', edit_name: '', edit_initiative: 0, edit_hp: 0, edit_hp_max: 0",
		// O BESTIÁRIO e o elenco.
		"draft_of: '', entry_hp: 0, entry_initiative: 10, entry_copies: 1, npc_name: ''",
		// O EDITOR DE BLOCO. O `rascunho` nasce com a FORMA inteira e não vazio,
		// e isso não é enfeite: `data-bind` num caminho que ainda não existe liga
		// um sinal NOVO em vez de escrever no de baixo, e o campo ficaria mudo
		// até o primeiro `@post`. A semente é a mesma do "criar do zero", vinda
		// do `blocoEmBranco` — escrevê-la aqui à mão seria o segundo branco.
		fmt.Sprintf("draft_open: false, draft_tab: %q, draft_error: '', draft: %s",
			abaDosNumeros, blankDraft()),
		// O ENQUADRAMENTO e o arrasto, que são do navegador de ponta a ponta.
		fmt.Sprintf("square: %d", DefaultSquare),
		"dragging: '', drag_start_x: 0, drag_start_y: 0, drag_x: 0, drag_y: 0",
		// A JANELA sobre o plano infinito, no lugar da rolagem nativa: rolagem
		// precisa de uma caixa com fim para ter até onde rolar.
		viewportSignals,
		// O TRAÇO do pincel e da borracha: o modo em curso e a última casa que ele
		// já mandou.
		brushSignals,
		// O LAÇO do retângulo: o modo em curso e os dois cantos.
		rectSignals,
		// AS PEÇAS MARCADAS pelo laço: ids separados por vírgula, numa string só.
		// Ver `markedTokensSignal` para por que não é uma lista.
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
		fmt.Sprintf("template: '%s', template_aims: %t, template_at_intersection: %t, template_size: 2",
			string(bookShapes[0]), pointsTemplate(bookShapes[0]),
			shapeStartsAtIntersection(bookShapes[0])),
		"template_x: 0, template_y: 0, template_aim_x: 0, template_aim_y: 0, template_phase: 0",
		fmt.Sprintf("template_path: '', template_text: %q", emptyTemplateHint),
		// As NOTAS da sessão.
		"notes: '', notes_saved: '', notes_mode: 'duplo', notes_open: false, notes_width: 0, notes_dragging: false, notes_floating: false",
		// `notes_window` é o LUGAR das notas quando ele não é esta aba: verdadeiro
		// enquanto a janela própria estiver com elas. Quem o escreve é o
		// `storage`, e não o clique — ver `watchesTheNotesWindow`.
		"notes_saving: false, notes_error: '', notes_window: false",
	}, ", ") + "}"
}

// tableBody escolhe QUAL DAS DUAS FORMAS a página desenha.
//
// O jogador recebe a coluna — uma superfície que rola, com Grupo, mapa e fila
// empilhados. O mestre recebe o PALCO: trilhos nas bordas e o tabuleiro no
// centro.
//
// SÃO DUAS FORMAS E NÃO DUAS TELAS, e a distinção é o que segura o argumento
// contra uma segunda cena: as REGIÕES são as mesmas, com os mesmos ids e os
// mesmos componentes, e o que muda é onde elas são penduradas. Duas listas de
// combatente para manter em dia seria o defeito; duas ARRUMAÇÕES da mesma lista
// não é.
//
// O painel do bestiário só nasce para quem pode abri-lo: não é a tela que
// esconde, é a página que não o tem. Mandá-lo para todo mundo e escondê-lo por
// CSS entregaria as criaturas com PV e defesa a quem abrisse o inspetor.
func (s Scene) tableBody(r *http.Request, view View, campaignID, sessionID int64) templ.Component {
	if view.Mestre == nil {
		return tableScene(view)
	}
	return gmStage(view, s.forTableBestiary(r, campaignID, sessionID))
}

// tablePlayerSheet carrega a ficha que a superfície "Minha ficha" desenha.
//
// Só na CARGA FRIA e não no stream: a ficha é sete painéis
// computados, e ela muda pelos comandos DELA — que remendam o `#sheet-scene`
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

// LoadView monta a Mesa inteira para desenhar: busca tudo o que a tela precisa
// e delega a DECISÃO ao `tableViewOf`. Impuro aqui, puro lá.
//
// Ela é exportada porque a BANCADA do hospedeiro a chama, e este pacote não tem
// banco: importar o `db/testdb` junto com um `*api.Server` seria o ciclo que a
// divisão existe para evitar. É a mesma direção do `characters.Load` e do
// `master.LoadBestiaryFrom` — a cena diz como montar a si mesma, e o hospedeiro
// prova que o que está no banco chega até lá.
func (s Scene) LoadView(ctx context.Context, userID int64, campaignID, sessionID int64) (View, int, error) {
	sess, role, err := s.access.Session(ctx, app.Caller{ID: userID}, campaignID, sessionID)
	status := statusOf(err)
	if err != nil {
		return View{}, status, err
	}
	// Hidrata do banco na primeira leitura, como o `onGetState` faz — sem isto
	// um servidor recém-subido serve fila vazia para um combate em andamento.
	if _, err := s.deps.Sessions().Load(ctx, sessionID); err != nil {
		return View{}, http.StatusInternalServerError, err
	}
	// `stateForRole` e não `redactForPlayers` direto: é o mesmo gargalo que o
	// socket usa, e papel desconhecido cai em jogador. Esta cena não ganha uma
	// segunda decisão sobre quem vê o quê.
	st := live.StateForRole(role, s.deps.Sessions().RefreshCharacterMaxes(ctx, sessionID))
	grupo, meus, eu := s.tableRoster(ctx, userID, campaignID)
	view := tableViewOf(st, campaignID, sessionID, sess.Sessionnumber, grupo, meus, eu)
	// O CICLO da sessão chega à tela porque, sem ele, os verbos teriam de ser
	// oferecidos todos — e "encerrar" numa sessão que nunca começou é o gesto
	// que o servidor recusa. Oferecer o que será recusado é desenhar um erro.
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
	// contra o banco (o `meus` do roster) e nunca contra o cliente.
	quemOlha := board.Mover{UserID: userID, Role: role}
	// A ABA que ESTA pessoa está olhando, e não "o tabuleiro da sessão": não há
	// tabuleiro único. Ela é resolvida contra os abertos, então a aba que o
	// mestre fechou não deixa ninguém numa tela morta.
	aba, puxado, deOnde := s.pullTab(ctx, sessionID, userID)
	scene := board.BoardForRole(role, s.deps.Boards().Get(ctx, sessionID, aba))
	// A LENTE DO MESTRE: com ela ligada, o que se desenha é a cena
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
	// A BARRA DE ABAS é a lista dos abertos, redigida pelo papel de
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
	// AS NOTAS são do mestre e chegam JÁ EM ÁRVORE. Elas não entram no
	// `tableViewOf` porque não vêm do estado ao vivo: moram na linha da sessão,
	// que é o mesmo lugar do título e do ciclo. A trava não é a tela esconder o
	// bloco, é a view não ter o que desenhar — pelo mesmo `role` que o
	// `stateForRole` já usou para redigir o estado.
	if role == "gm" && sess.Notes.Valid {
		view.Notas = sess.Notes.String
		view.NotasBlocos = markdown.Parse(sess.Notes.String)
	}
	if role == "gm" {
		view.NPCs = s.CampaignCast(ctx, campaignID)
	}
	// A PRESENÇA É DOS DOIS PAPÉIS, e não só do mestre (decisão do dono): saber
	// quem caiu é o que faz a mesa ESPERAR em vez de continuar sem alguém. Por
	// isso ela é calculada FORA do ramo do mestre — dentro dele, o `cardsParty`
	// do jogador desenharia o ponto de presença que nunca chegaria.
	membros, presentes := s.membrosEPresenca(ctx, campaignID, sessionID)
	conectados := live.ConnectedCharacters(membros, presentes)
	marcaAPresenca(view.Grupo, conectados)
	if role == "gm" {
		r := ofViewGm(st, membros, presentes, true, s.saveFailed(sessionID))
		view.Mestre = &r
	}
	return view, http.StatusOK, nil
}

// saveFailed junta os DOIS stores numa pergunta só.
//
// Para quem está mestrando não existe "o tabuleiro não salvou" e "a fila não
// salvou": existe "a mesa não está sendo salva". Separar daria à tela uma
// decisão que ela não tem o que fazer com — os dois têm a mesma causa (o disco)
// e o mesmo remédio (parar e chamar alguém).
func (s Scene) saveFailed(sessionID int64) bool {
	return s.deps.Boards().SaveFailed(sessionID) || s.deps.Sessions().SaveFailed(sessionID)
}

// tableRoster traduz o roster da campanha nas três coisas que a tela quer: os
// cartões do Grupo, o conjunto dos MEUS personagens, e qual deles registra
// iniciativa.
//
// A ponte até "quem está olhando" é o `ownerId` do roster, e não o id do
// personagem: a ficha de um membro é o SNAPSHOT da campanha, então o dono
// registrado é o único fio de volta até a pessoa.
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
		// NÃO entra filtro de PAPEL aqui, e a razão é do dono da mesa: o mestre
		// NÃO tem personagem próprio. O que ele tem é um elenco de NPCs que
		// entram na história ou não — decisão por CENA, tomada na hora de pôr a
		// linha na fila —, e NPC nem é membro da campanha: ele entra na
		// iniciativa por `label` e `initiative`, sem `characterId` (ver
		// `Roster.Entry`). `campaign_members` só tem personagem de jogador, e
		// o grupo é o grupo.
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
		// a ficha inteira usa.
		if bonus, err := s.queue.Roster().Bonus(ctx, eu.CharacterID); err == nil {
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

// membrosEPresenca junta o que as regras de presença precisam.
//
// Roster indisponível não derruba a cena: a presença é enfeite ao lado dos
// nomes, e a fila é o assunto da tela.
func (s Scene) membrosEPresenca(ctx context.Context, campaignID, sessionID int64) ([]live.TableMember, []int64) {
	rows, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		return nil, nil
	}
	membros := make([]live.TableMember, 0, len(rows))
	for _, m := range rows {
		dono, err := s.deps.Queries().GetCharacterOwner(ctx, m.Characterid)
		if err != nil {
			continue
		}
		membros = append(membros, live.TableMember{CharacterID: m.Characterid, OwnerID: dono})
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
	ficha, err := sheet.LoadAndCompute(ctx, s.deps.Queries(), s.deps.Catalogs(), row)
	if err != nil {
		return "—"
	}
	// A MESMA frase da ficha, e da mesma função: o mestre confere aqui a Defesa
	// de um jogador para decidir se o ataque acerta, e com o alvo caído o total
	// é o único número que não responde essa pergunta.
	return book.DefenseLabel(ficha.Defense)
}
