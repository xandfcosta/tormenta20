package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"t20engine/aovivo"
	"t20engine/markdown"
	"t20engine/web/sheetui"
	"t20engine/web/ui"
)

// A Mesa do jogador como DADO — o piloto Datastar (ALE-219).
//
// Puro de propósito: o handler busca, este arquivo decide, o template só
// desenha. É o que deixa a regra provável sem HTTP nenhum, pela mesma razão que
// o `selfInitiativeEntry` é transport-agnostic (ALE-213) — o que importa não é
// o transporte.
//
// Nenhuma regra NOVA mora aqui. O estado já chega redigido por
// `stateForRole`/`redactForPlayers`, que é o gargalo único da ALE-210, e a
// derivação do turno é a tradução literal do `playerTurnState` da SPA. Um
// piloto que reescrevesse a regra mediria a reescrita, não o Datastar.

// tableView é uma tela inteira da Mesa. Campos exportados porque `html/template`
// não enxerga os minúsculos — a única razão, e ela é do pacote de template.
type tableView struct {
	// Status é o ciclo da sessão — `planned`, `active` ou `ended`. Ele decide
	// QUAIS verbos a tela oferece, e não só como ela os pinta: o servidor recusa
	// encerrar o que nunca começou, e um botão que existe para levar recusa é um
	// erro desenhado.
	Status string
	// Titulo é o apelido da noite, e pode ser VAZIO: a identidade da sessão é o
	// NÚMERO. Obrigar a um título faria o mestre inventar texto para salvar.
	Titulo     string
	CampaignID int64
	SessionID  int64
	SessionNum int64
	// SceneActive vem do estado JÁ REDIGIDO: fora de cena o `redactForPlayers`
	// devolve fila limpa, então o falso aqui É a trava da ALE-210 e não
	// uma segunda decisão tomada na tela.
	SceneActive bool
	Round       int
	Turn        tableTurn
	Grupo       []tableMember
	Fila        []tableRow
	Eu          *tableMe
	// MinhaFicha é a ficha do personagem DESTE jogador, desenhada dentro da
	// sessão (ALE-272, fatia 10b). Nil para o mestre e para quem não tem
	// personagem na campanha — é a mesma trava do `Mestre`: o que a view não
	// tem, a cena não desenha.
	//
	// Ela NÃO é região do stream, e isso é decisão: a ficha é cara de computar
	// e muda pelos comandos DELA, não pelo que acontece na mesa. Pendurá-la em
	// `tableRegions` faria cada tique do stream recomputar sete painéis para
	// descobrir que nada mudou.
	MinhaFicha *sheetui.View
	// Tabuleiro é o mapa da cena. `Aberto` falso é o estado normal — a maior
	// parte de uma sessão não tem mapa —, e ele desenha a frase e nenhuma grade.
	Tabuleiro boardView
	// Mestre é nil para o jogador, e essa é a trava na CENA: não há como
	// desenhar controle que não existe na view. Esconder por classe deixaria o
	// HTML na página para quem abrisse o inspetor.
	Mestre *viewGm
	// Notas é o caderno da noite (ALE-269, superfície 5), e ele é DO MESTRE: na
	// SPA o painel vive na `session-gm-view` e o jogador nunca o recebe. Vazio
	// para quem não é mestre, pela mesma trava do resto — a view não tem o que
	// desenhar, em vez de a tela esconder.
	Notas string
	// NotasBlocos é a mesma nota já em ÁRVORE, para o templ montar elementos em
	// vez de cuspir HTML. Nasce aqui e não no template porque parsear em
	// template é regra escondida onde ninguém a testa.
	NotasBlocos []markdown.Block
	// NPCs é o elenco da CAMPANHA (ALE-269, superfície 6b) — o taverneiro que
	// não briga e o chefe da semana que vem. Do mestre, como as notas: a view do
	// jogador não o tem, e por isso não há o que a tela dele esconder.
	NPCs []castNpc
}

// tableTurn é de quem é a vez, do ponto de vista de quem olha. Espelha o
// `LiveTurnState` da SPA.
type tableTurn struct {
	Kind  string // "mine" | "other" | "idle"
	Label string
}

// tableBar é uma barra de vital já com a porcentagem e a COR resolvidas: o
// template não faz conta nem escolhe tom, porque conta em template é regra
// escondida onde ninguém a testa.
type tableBar struct {
	Atual int64
	Max   int64
	Pct   int
	// Tom é a CLASSE do preenchimento — a cor diz "quão mal", não só a largura
	// (espelha `hpFillVar` do `vital-bar.tsx`).
	//
	// Classe e não `var(--token)` inline por duas razões que se somam: o
	// `html/template` sanitiza contexto CSS e um `var(--hp-full)` interpolado
	// vira `ZgotmplZ`, e classe é o que o scanner do Tailwind sabe procurar.
	// Como o nome nasce aqui e não no template, o scanner NÃO o vê — por isso
	// os quatro estão declarados no `@source inline(...)` do `table.css`.
	Tom string
}

// tableMember é um personagem do grupo no cartão "Grupo".
type tableMember struct {
	// CharacterID não é desenhado: é a chave que casa o cartão com a presença.
	CharacterID int64
	Nome        string
	// Iniciais é o monograma do ELENCO no trilho do mestre (ALE-269) — a vaga
	// do `CastRail` da SPA. O jogador continua lendo o nome inteiro no cartão.
	Iniciais string
	Nivel    int64
	Classes  string
	PV       tableBar
	PM       tableBar
	// Presenca é nil para o JOGADOR, e isso segue o precedente da SPA: presença
	// POR PERSONAGEM é do mestre (o trilho do elenco vive na `session-gm-view`),
	// enquanto os crachás de NOME são de todo mundo. Nil e não "false" porque
	// "não mostrar" e "está fora" são coisas diferentes — um anel apagado diria
	// "fora da mesa" a quem não tem por que saber.
	Presenca *presencaDoMembro
	// Defesa é TEXTO e nunca número, pela mesma razão do cartão de personagem:
	// sem motor ela é desconhecida, e um ZERO é um valor de Defesa plausível e
	// errado. Travessão diz "não sei"; zero mente com cara de dado.
	Defesa string
	// NaFila responde "este já está no combate?", e é o que decide se o elenco
	// OFERECE pô-lo na fila. Oferecer o que só pode dar linha repetida é
	// desenhar um erro — a mesma regra que trava os verbos do ciclo da sessão.
	NaFila bool
}

// presencaDoMembro é "está com a aba aberta agora?", já com a frase pronta.
type presencaDoMembro struct {
	NaMesa bool
	// Frase é o nome acessível, porque anel colorido não existe para leitor de
	// tela (ALE-212). As palavras são as MESMAS da gaveta do elenco na SPA: duas
	// telas não devem inventar dois vocabulários para o mesmo estado.
	Frase string
}

// marcaAPresenca escreve em cada cartão do Grupo se aquele personagem está na
// mesa agora. Só é chamada para o MESTRE — ver o comentário do campo.
func marcaAPresenca(grupo []tableMember, conectados map[int64]bool) {
	for i := range grupo {
		naMesa := conectados[grupo[i].CharacterID]
		frase := "fora da mesa"
		if naMesa {
			frase = "na mesa"
		}
		grupo[i].Presenca = &presencaDoMembro{NaMesa: naMesa, Frase: frase}
	}
}

// tableRow é uma linha da fila de iniciativa como o jogador a vê.
type tableRow struct {
	ID         string
	Rotulo     string
	Iniciativa int
	// EhFicha responde a pergunta que o GLOSSARIO faz do `type === "character"`:
	// "esta linha é ficha ou é NPC?". O campo se chamava `PC`, que é termo
	// PROIBIDO — e o par na tela é `Ficha`/`NPC` pelo mesmo motivo.
	EhFicha bool
	Minha   bool
	NaVez   bool
	// PV nil = linha sem vida rastreada. `Oculto` é outra coisa: o mestre
	// escondeu, e a flag sobrevive à redação de propósito (ALE-210) — "sem
	// barra" e "escondido" não são a mesma coisa, e a segunda é informação.
	PV        *tableBar
	Oculto    bool
	Condicoes []string
	// Iniciais é o monograma do trilho de 80px (ALE-269). Nasce na view e não
	// no template pela convenção da casa — `web/campaigns/list_view.go` faz o
	// mesmo —, e porque duas letras NÃO são um nome: quem desenha o retrato
	// precisa do rótulo inteiro ao lado, no `aria-label`.
	Iniciais string
}

// rotuloDoRetrato é o nome INTEIRO de um combatente do trilho, com os vitais
// junto (ALE-269).
//
// Ele existe porque duas letras não são um nome: o retrato de 80px desenha o
// monograma, e quem usa leitor de tela — ou o ponteiro parado em cima — precisa
// ouvir "Ogro, PV 22 de 40" e não "OG".
//
// PV ausente e PV OCULTO dizem coisas diferentes e a frase separa as duas: a
// primeira é linha sem vida rastreada, a segunda é o mestre tendo escondido de
// propósito (ALE-210), e ele é o único que lê este trilho.
//
// @example rotuloDoRetrato(tableRow{Rotulo: "Ogro", PV: &tableBar{22, 40, 55, ""}}) // "Ogro — PV 22 de 40"
func rotuloDoRetrato(l tableRow) string {
	if l.Oculto {
		return fmt.Sprintf("%s — PV oculto", l.Rotulo)
	}
	if l.PV == nil {
		return l.Rotulo
	}
	return fmt.Sprintf("%s — PV %d de %d", l.Rotulo, l.PV.Atual, l.PV.Max)
}

// castLabel é o nome de um personagem do elenco recolhido, com a presença
// junto — porque no trilho ela é um PONTO colorido, e cor não existe para quem
// usa leitor de tela (ALE-212).
//
// @example castLabel(tableMember{Nome: "Arwen", Nivel: 3}) // "Arwen, Nv 3"
func castLabel(m tableMember) string {
	rotulo := fmt.Sprintf("%s, Nv %d", m.Nome, m.Nivel)
	if m.Presenca == nil {
		return rotulo
	}
	return fmt.Sprintf("%s — %s", rotulo, m.Presenca.Frase)
}

// tableMe é o personagem de quem olha, quando ele tem um nesta mesa. Nil é um
// estado normal: o convidado que assiste não registra iniciativa.
type tableMe struct {
	CharacterID int64
	Nome        string
	Bonus       int64
	NaFila      bool
}

// tableTurnOf traduz `playerTurnState` (session-player-view.tsx) para o Go.
//
// Fora de combate ninguém está na vez. A linha na vez sendo de um personagem
// MEU é o único caso em que a faixa acende.
func tableTurnOf(st *aovivo.SessionRuntimeState, meus map[int64]bool) tableTurn {
	if st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return tableTurn{Kind: "idle"}
	}
	naVez := st.Initiative[st.TurnIndex]
	if naVez.CharacterID != nil && meus[*naVez.CharacterID] {
		return tableTurn{Kind: "mine"}
	}
	return tableTurn{Kind: "other", Label: naVez.Label}
}

// tableBarOf resolve a porcentagem (presa em 0..100) e o tom.
//
// Máximo ausente ou zero devolve 0% em vez de dividir: uma linha sem pool não
// tem barra cheia nem vazia, ela não tem barra — e é quem chama que decide não
// desenhar.
func tableBarOf(atual, max int64, arcano bool) tableBar {
	barra := tableBar{Atual: atual, Max: max, Tom: "bg-mp-arcane"}
	if max > 0 {
		pct := int(atual * 100 / max)
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		barra.Pct = pct
	}
	if !arcano {
		barra.Tom = hpTomDe(barra.Pct)
	}
	return barra
}

// hpTomDe é a tradução literal do `hpFillVar` (vital-bar.tsx): a COR da barra de
// PV diz "quão mal", e os limiares são os mesmos dos dois lados de propósito —
// duas escadas divergiriam em silêncio, cada tela chamando de "ferido" uma
// coisa diferente.
func hpTomDe(pct int) string {
	if pct <= 25 {
		return "bg-hp-critical"
	}
	if pct <= 50 {
		return "bg-hp-hurt"
	}
	return "bg-hp-full"
}

// tableTrackerOf desenha a fila que o jogador recebeu — já redigida.
func tableTrackerOf(st *aovivo.SessionRuntimeState, meus map[int64]bool) []tableRow {
	fila := make([]tableRow, 0, len(st.Initiative))
	for i := range st.Initiative {
		e := &st.Initiative[i]
		linha := tableRow{
			ID:         e.ID,
			Rotulo:     e.Label,
			Iniciativa: e.Initiative,
			EhFicha:    e.Type == "character",
			Minha:      e.CharacterID != nil && meus[*e.CharacterID],
			NaVez:      i == st.TurnIndex,
			Oculto:     e.HpHidden != nil && *e.HpHidden,
			Condicoes:  e.Conditions,
			Iniciais:   ui.Monogram(e.Label),
		}
		// O `HpMax` nil depois da redação é como o servidor DIZ "isto não é seu
		// para ver". Desenhar barra aqui inventaria um número.
		if e.HpMax != nil {
			barra := tableBarOf(aovivo.DerefOr(e.HpCurrent, 0), *e.HpMax, false)
			linha.PV = &barra
		}
		fila = append(fila, linha)
	}
	return fila
}

// tableViewOf monta a tela a partir das partes já buscadas. Tudo o que decide
// mora aqui; o handler ao lado só sabe buscar.
func tableViewOf(
	st *aovivo.SessionRuntimeState,
	campaignID, sessionID, sessionNum int64,
	grupo []tableMember,
	meus map[int64]bool,
	eu *tableMe,
) tableView {
	if eu != nil {
		eu.NaFila = false
		for i := range st.Initiative {
			if id := st.Initiative[i].CharacterID; id != nil && *id == eu.CharacterID {
				eu.NaFila = true
				break
			}
		}
	}
	// QUEM JÁ ESTÁ NA FILA, marcado no elenco. A pergunta é da FILA e não do
	// roster, então ela é respondida aqui, onde as duas estão à mão — o
	// `tableRoster` monta os cartões sem saber que existe combate.
	tracker := map[int64]bool{}
	for i := range st.Initiative {
		if id := st.Initiative[i].CharacterID; id != nil {
			tracker[*id] = true
		}
	}
	for i := range grupo {
		grupo[i].NaFila = tracker[grupo[i].CharacterID]
	}
	return tableView{
		CampaignID:  campaignID,
		SessionID:   sessionID,
		SessionNum:  sessionNum,
		SceneActive: st.SceneActive,
		Round:       st.Round,
		Turn:        tableTurnOf(st, meus),
		Grupo:       grupo,
		Fila:        tableTrackerOf(st, meus),
		Eu:          eu,
	}
}

// ── o rastreador do MESTRE (ALE-265) ─────────────────────────────────────────
//
// A cena de hoje é a superfície do JOGADOR: ela mostra a fila, o grupo e a vez.
// O mestre precisa das mesmas coisas mais o que ele COMANDA — avançar o turno,
// abrir e encerrar a cena, descansar, e mexer nos vitais de quem está na fila.
//
// O que decide entre as duas não é uma tela diferente: é o PAPEL, resolvido no
// servidor pelo mesmo `stateForRole` que o resto da casa usa. A tela do mestre
// é a do jogador mais os controles, e não uma segunda cena — duas cenas seriam
// duas listas de combatente para manter em dia.

// viewGm é o acréscimo do mestre sobre a `tableView`.
type viewGm struct {
	// Contador é a frase que diz ONDE a sessão está (ALE-210).
	Contador string
	// Avanco é o rótulo do botão mais clicado da sessão, e ele diz PARA ONDE vai
	// em vez de o que faz (ALE-184).
	Avanco aovivo.NextTurnTarget
	// VeVitais decide se a fila mostra PV de NPC. A pergunta é sobre a FILA e
	// não sobre o papel: numa fila só de PCs não há o que reservar.
	VeVitais bool
	// Conectados são os personagens de quem está com a aba aberta agora.
	Conectados map[int64]bool
	// PodeAvancar separa "não há para onde ir" de "o botão está quebrado": sem
	// cena aberta o avanço não existe, e um botão aceso que recusa é pior que um
	// apagado que explica.
	PodeAvancar bool
}

func ofViewGm(
	st *aovivo.SessionRuntimeState,
	membros []aovivo.TableMember,
	presentes []int64,
	ehMestre bool,
) viewGm {
	return viewGm{
		Contador:    aovivo.TurnCounter(st.SceneActive, st.Round, st.TurnIndex, len(st.Initiative)),
		Avanco:      aovivo.NextTurnButton(st.Initiative, st.TurnIndex),
		VeVitais:    aovivo.GmSeesVitals(st.Initiative, ehMestre),
		Conectados:  aovivo.ConnectedCharacters(membros, presentes),
		PodeAvancar: st.SceneActive && len(st.Initiative) > 0,
	}
}

// tableCommand escreve a chamada Datastar de um comando do mestre.
//
// O caminho é o do PILOTO e não o da API JSON, e essa é a mesma escolha das
// catorze fatias anteriores: a cena tem rotas próprias que chamam as MESMAS
// regras extraídas. Apontar para `/api/...` acoplaria o piloto a uma superfície
// que a migração existe para aposentar, e o ganho — não escrever a rota — some
// no dia em que a API mudar de forma.
//
// O que impede as duas telas de divergirem não é compartilhar a rota: é
// compartilhar a REGRA. É por isso que a ALE-122 aconteceu com dois transportes
// chamando dois caminhos de escrita, e não acontece aqui.
func tableCommand(v tableView, metodo, acao string) string {
	caminho := fmt.Sprintf("/mesa/%d/%d/%s", v.CampaignID, v.SessionID, acao)
	if metodo == "POST" {
		return fmt.Sprintf("@post('%s')", caminho)
	}
	return fmt.Sprintf("@%s('%s')", strings.ToLower(metodo), caminho)
}

// rowCommand escreve a chamada de um verbo que age sobre UM combatente.
//
// O `entryId` entra no CAMINHO, ao lado dos outros dois ids, e não num sinal:
// sinal é da página inteira, e nove linhas escrevendo no mesmo sinal antes de
// postar é uma corrida esperando por um mestre de dedo rápido. Caminho é do
// botão que foi clicado, e não há segundo escritor.
func rowCommand(v tableView, l tableRow, acao string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/initiative/%s/%s')", v.CampaignID, v.SessionID, l.ID, acao)
}

// rowVital escreve o ferir/curar com os DOIS passos já resolvidos em duas
// URLs, e o `evt.shiftKey` escolhendo entre elas.
//
// O `@post` recebe uma expressão como ARGUMENTO, e não é um `@post` dentro de
// cada braço de um ternário: a chamada é uma só, e o que varia é a string. Assim
// o que o Datastar precisa reescrever é uma ação, não duas dentro de um desvio.
func rowVital(v tableView, l tableRow, verbo string) string {
	base := fmt.Sprintf("/mesa/%d/%d/initiative/%s/vitals/%s/", v.CampaignID, v.SessionID, l.ID, verbo)
	return fmt.Sprintf("@post(evt.shiftKey ? '%s5' : '%s1')", base, base)
}

// openEdit semeia o diálogo com os valores de AGORA e o abre.
//
// Semear é obrigatório e não é conveniência: o diálogo é UM para a fila inteira,
// então sem isto ele abriria com o que sobrou da linha anterior — e o mestre
// salvaria o PV do Ogro em cima do Goblin sem ver nada de errado.
func openEdit(v tableView, l tableRow) string {
	pv, pvMax := int64(0), int64(0)
	if l.PV != nil {
		pv, pvMax = l.PV.Atual, l.PV.Max
	}
	return fmt.Sprintf(
		"$edicaolinha = '%s'; $edicaonome = %s; $edicaoiniciativa = %d; $edicaopv = %d; $edicaopvmax = %d; document.getElementById('editar-combatente').showModal()",
		l.ID, jsTextHow(l.Rotulo), l.Iniciativa, pv, pvMax,
	)
}

// saveEdit monta o caminho com o id que o número semeou.
func saveEdit(v tableView) string {
	return fmt.Sprintf(
		"document.getElementById('editar-combatente').close(); @post('/mesa/%d/%d/initiative/' + $edicaolinha + '/edit')",
		v.CampaignID, v.SessionID,
	)
}

// jsTextHow escreve um literal de string de JavaScript seguro.
//
// O rótulo é digitado pelo MESTRE e vai parar dentro de uma expressão do
// Datastar, que é JavaScript: um combatente chamado `O'Brien` fecharia a aspa e
// o resto da expressão viraria sintaxe. O `templ` escapa o atributo (as aspas
// viram `&#39;`), mas o navegador as desescapa antes de o Datastar compilar — o
// escape de HTML não é o escape de JS, e confundir os dois é como se escreve uma
// injeção sem querer.
func jsTextHow(s string) string {
	cru, err := json.Marshal(s)
	if err != nil {
		return "''"
	}
	return string(cru)
}

// ── As CONDIÇÕES do combatente na tela (ALE-122, portadas na ALE-269) ────────

// openConditions escolhe a linha e abre o diálogo.
//
// Ele reescreve OS DOIS sinais, e é quem TROCA de item que limpa — a lição do
// diálogo de redefinir senha, onde o link da Ana aparecia sob o nome da Bia
// (está no guia do `engine-go/`). Quem gera não sabe que haverá um próximo;
// quem troca sabe que houve um anterior.
func openConditions(l tableRow) string {
	return fmt.Sprintf(
		"$linhadacondicao = %q; $condicoesdalinha = %q; $rotulodalinha = %q;"+
			" document.getElementById('condicoes-do-combatente').showModal()",
		l.ID, strings.Join(l.Condicoes, ","), l.Rotulo,
	)
}

// onCondition é a pergunta que pinta o crachá do diálogo.
func onCondition(id string) string {
	return fmt.Sprintf("$condicoesdalinha.split(',').includes(%q)", id)
}

// toggleConditionRow posta o clique na linha ESCOLHIDA.
//
// O `entryId` sai do sinal e não do caminho escrito pelo servidor porque o
// diálogo é UM só para todas as linhas — é o preço de não desenhar 35 crachás
// por combatente, e o sinal é reescrito a cada abertura.
func toggleConditionRow(v tableView, id string) string {
	return fmt.Sprintf(
		"@post('/mesa/%d/%d/initiative/' + $linhadacondicao + '/condicao/%s')",
		v.CampaignID, v.SessionID, id,
	)
}

// ── O CICLO da sessão na tela (ALE-269) ─────────────────────────────────────

// sessionCommand escreve a chamada de um verbo do ciclo.
func sessionCommand(v tableView, acao string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/sessao/%s')", v.CampaignID, v.SessionID, acao)
}

// caminhoDeExcluir é o `action` do form, e não uma expressão: excluir NAVEGA.
func caminhoDeExcluir(v tableView) string {
	return fmt.Sprintf("/mesa/%d/%d/sessao/excluir", v.CampaignID, v.SessionID)
}

// campaignChronicle é para onde se sai da sessão.
//
// A crônica e não o Hub: é de lá que se entra numa sessão, então é lá que o
// mestre continua o que estava fazendo. Sair para a raiz obrigaria a refazer
// dois cliques para voltar à mesa que ele acabou de deixar.
func campaignChronicle(v tableView) string {
	return fmt.Sprintf("/campanhas/%d", v.CampaignID)
}

// portugueseCycle é o que o crachá do cabeçalho diz.
//
// Traduzido AQUI e não no banco: `planned`/`active`/`ended` são a forma de fio,
// e a tela não deve imprimir identificador — é o mesmo defeito que o crachá de
// condição tinha, e que custou CAIDO e VULNERAVEL na tela.
func portugueseCycle(status string) string {
	switch status {
	case "active":
		return "Ao vivo"
	case "ended":
		return "Encerrada"
	default:
		return "Planejada"
	}
}

// openConfigSession semeia o título de AGORA e abre o diálogo.
//
// Quem abre é quem semeia, pela mesma razão do `openEdit`: o campo é ligado a
// um SINAL e nunca recebe `value` do servidor, senão o remendo da próxima troca
// de turno apagaria o que o mestre está digitando.
func openConfigSession(v tableView) string {
	return fmt.Sprintf("$titulodasessao = %q; document.getElementById('config-da-sessao').showModal()", v.Titulo)
}
