package table

import (
	"encoding/json"
	"fmt"
	"strings"
	"t20engine/domain/live"
	"t20engine/domain/markdown"
	"t20engine/serve/web/routes"
	"t20engine/serve/web/sheetui"
	"t20engine/serve/web/ui"
)

// A Mesa como DADO: o handler busca, este arquivo decide, o template só
// desenha. Nenhuma regra NOVA mora aqui — o estado já chega redigido por
// `stateForRole`/`redactForPlayers`, que é o gargalo único.

// View é uma tela inteira da Mesa. Campos exportados porque `html/template`
// não enxerga os minúsculos — a única razão, e ela é do pacote de template.
type View struct {
	// Status é o ciclo da sessão — `planned`, `active` ou `ended`. Ele decide
	// QUAIS verbos a tela oferece: o servidor recusa encerrar o que nunca
	// começou, e um botão que existe para levar recusa é um erro desenhado.
	Status string
	// Titulo é o apelido da noite, e pode ser VAZIO: a identidade da sessão é o
	// NÚMERO. Obrigar a um título faria o mestre inventar texto para salvar.
	Titulo     string
	CampaignID int64
	SessionID  int64
	SessionNum int64
	// SceneActive vem do estado JÁ REDIGIDO: fora de cena o `redactForPlayers`
	// devolve fila limpa, então o falso aqui É a trava e não uma segunda
	// decisão tomada na tela.
	SceneActive bool
	// Scene é a cena EM CURSO, com o tipo e o número — nil fora de cena. O
	// `SceneActive` continua ao lado porque a tela pergunta as duas coisas, e
	// "há cena" é a pergunta de dezoito sítios.
	Scene *sceneView
	Round int
	Turn  tableTurn
	// Proximos é a faixa de quem vem depois: a vez e as duas seguintes, dando a
	// volta. Vazia fora de combate.
	Proximos []turnAhead
	Grupo    []Member
	Fila     []tableRow
	Eu       *tableMe
	// MinhaFicha é a ficha do personagem DESTE jogador, desenhada dentro da
	// sessão. Nil para o mestre e para quem não tem personagem na campanha — é a
	// mesma trava do `Mestre`: o que a view não tem, a cena não desenha.
	//
	// Ela NÃO é região do stream, e isso é decisão: a ficha é cara de computar
	// e muda pelos comandos DELA, não pelo que acontece na mesa. Pendurá-la em
	// `TableRegions` faria cada tique do stream recomputar sete painéis para
	// descobrir que nada mudou.
	MinhaFicha *sheetui.View
	// Tabuleiro é o mapa da cena. `Aberto` falso é o estado normal — a maior
	// parte de uma sessão não tem mapa —, e ele desenha a frase e nenhuma grade.
	Tabuleiro BoardView
	// Mestre é nil para o jogador, e essa é a trava na CENA: não há como
	// desenhar controle que não existe na view. Esconder por classe deixaria o
	// HTML na página para quem abrisse o inspetor.
	Mestre *viewGm
	// Notas é o caderno da noite, e ele é DO MESTRE. Vazio para quem não é
	// mestre, pela mesma trava do resto — a view não tem o que desenhar, em vez
	// de a tela esconder.
	Notas string
	// NotasBlocos é a mesma nota já em ÁRVORE, para o templ montar elementos em
	// vez de cuspir HTML. Nasce aqui e não no template porque parsear em
	// template é regra escondida onde ninguém a testa.
	NotasBlocos []markdown.Block
	// NPCs é o elenco da CAMPANHA — o taverneiro que não briga e o chefe da
	// semana que vem. Do mestre, como as notas.
	NPCs []castNpc
}

// tableTurn é de quem é a vez, do ponto de vista de quem olha.
type tableTurn struct {
	Kind  string // "mine" | "other" | "idle"
	Label string
}

// tableBar é uma barra de vital já com a porcentagem e a COR resolvidas: o
// template não faz conta nem escolhe tom, porque conta em template é regra
// escondida onde ninguém a testa.
type tableBar struct {
	Current int64
	Max     int64
	Pct     int
	// Hidden é "o mestre está escondendo ESTE pool da mesa".
	//
	// Mora na BARRA e não na linha porque a decisão é por pool: o PV do grupo
	// aparece e o PM não, e uma flag por linha não teria como dizer as duas
	// coisas do mesmo combatente. É também o que põe o olho ao lado da barra que
	// ele esconde, em vez de numa tira onde ele não diz de qual número fala.
	//
	// Para o JOGADOR, uma barra escondida chega sem números e com isto ligado —
	// é assim que a tela distingue "não tem PV rastreado" de "o mestre escondeu".
	Hidden bool
	// Temp é a reserva de PV TEMPORÁRIO, e ela é parcela À PARTE do `Current`.
	//
	// O dano gasta a reserva ANTES do PV (p106), então o mestre que decide uma
	// pancada precisa dela para saber se o golpe chega na carne. Ela não entra
	// no `Pct` pela mesma razão que não entra na fração do crachá da ficha: a
	// barra responde "quanto apanhei", e somar faria um herói ferido desenhar
	// cheio.
	//
	// ZERO quer dizer "não há reserva", e aí a barra fica igual à de sempre.
	Temp int64
	// TempPct é a largura do filete da reserva, presa em 0..100 — o trilho da
	// fila não tem número para mostrar, só espaço para uma linha.
	TempPct int
	// Tom é a CLASSE do preenchimento — a cor diz "quão mal", não só a largura.
	//
	// Classe e não `var(--token)` inline por duas razões que se somam: o
	// `html/template` sanitiza contexto CSS e um `var(--hp-full)` interpolado
	// vira o sentinela ZgotmplZ, e classe é o que o scanner do Tailwind procura.
	// Como o nome nasce aqui e não no template, o scanner NÃO o vê — por isso
	// os quatro estão declarados no `@source inline(...)` do `app.src.css`.
	Tone string
}

// Member é um personagem do grupo no cartão "Grupo".
type Member struct {
	// CharacterID não é desenhado: é a chave que casa o cartão com a presença.
	CharacterID int64
	Nome        string
	// Iniciais é o monogram do elenco no trilho do mestre. O jogador continua
	// lendo o nome inteiro no cartão.
	Iniciais string
	Nivel    int64
	Classes  string
	PV       tableBar
	PM       tableBar
	// Presenca chega para os DOIS papéis: saber quem caiu é o que faz a mesa
	// ESPERAR em vez de continuar sem alguém, e isso vale mais que a discrição
	// de não dizer ao jogador quem está fora.
	//
	// É PONTEIRO e não `bool`: "não sei" e "está fora" são coisas diferentes, e
	// um cartão sem dado de presença não pode afirmar ausência. Nil é o que o
	// remendo desenha quando a cena ainda não resolveu quem está na mesa.
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
	// tela.
	Frase string
}

// marcaAPresenca escreve em cada cartão do Grupo se aquele personagem está na
// mesa agora. Chamada para os DOIS papéis — ver o campo `Presenca`.
func marcaAPresenca(grupo []Member, conectados map[int64]bool) {
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
	// EhFicha responde "esta linha é ficha ou é NPC?". O par na tela é
	// `Ficha`/`NPC`, porque `PC` é termo proibido pelo GLOSSARY.
	EhFicha bool
	Minha   bool
	NaVez   bool
	// PV nil = linha sem vida rastreada. Escondido é outra coisa: o mestre
	// escondeu de propósito, e a marca sobrevive à redação — "sem barra" e
	// "escondido" não são a mesma coisa, e a segunda é informação.
	PV *tableBar
	// PM é o mana da linha.
	PM        *tableBar
	Condicoes []string
	// Iniciais é o monogram do trilho de 80px. Nasce na view e não no template
	// porque duas letras NÃO são um nome: quem desenha o retrato precisa do
	// rótulo inteiro ao lado, no `aria-label`.
	Iniciais string
}

// portraitLabel é o nome INTEIRO de um combatente do trilho, com os vitais
// junto: o retrato de 80px desenha só o monogram, e quem usa leitor de tela —
// ou o ponteiro parado em cima — precisa ouvir "Ogro, PV 22 de 40" e não "OG".
//
// PV ausente e PV OCULTO dizem coisas diferentes e a frase separa as duas: a
// primeira é linha sem vida rastreada, a segunda é o mestre tendo escondido de
// propósito.
//
// @example portraitLabel(tableRow{Rotulo: "Ogro", PV: &tableBar{Current: 22, Max: 40}}) // "Ogro — PV 22 de 40"
func portraitLabel(l tableRow) string {
	if l.PV == nil {
		return l.Rotulo
	}
	if l.PV.Hidden {
		return fmt.Sprintf("%s — PV oculto", l.Rotulo)
	}
	return fmt.Sprintf("%s — %s", l.Rotulo, barLabel("PV", *l.PV))
}

// barLabel é o nome ACESSÍVEL de uma barra: "PV 22 de 40", mais a reserva
// quando ela existe.
//
// O filete da reserva é COR e largura, e nenhuma das duas existe para quem usa
// leitor de tela — o mesmo motivo que põe a presença no `castLabel` logo
// abaixo. Aqui ela é o canal ÚNICO no trilho da fila, que não tem número
// nenhum: sem esta frase, a reserva simplesmente não existiria para quem não vê
// a tela.
func barLabel(rotulo string, b tableBar) string {
	frase := fmt.Sprintf("%s %d de %d", rotulo, b.Current, b.Max)
	if b.Temp > 0 {
		frase += fmt.Sprintf(", mais %d temporários", b.Temp)
	}
	return frase
}

// castLabel é o nome de um personagem do elenco recolhido, com a presença
// junto — porque no trilho ela é um PONTO colorido, e cor não existe para quem
// usa leitor de tela.
//
// @example castLabel(Member{Nome: "Arwen", Nivel: 3}) // "Arwen, Nv 3"
func castLabel(m Member) string {
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

// tableTurnOf responde de quem é a vez para quem está olhando.
//
// Fora de combate ninguém está na vez. A linha na vez sendo de um personagem
// MEU é o único caso em que a faixa acende.
func tableTurnOf(st *live.SessionRuntimeState, meus map[int64]bool) tableTurn {
	if st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return tableTurn{Kind: "idle"}
	}
	naVez := st.Initiative[st.TurnIndex]
	if naVez.CharacterID != nil && meus[*naVez.CharacterID] {
		return tableTurn{Kind: "mine"}
	}
	return tableTurn{Kind: "other", Label: naVez.Label}
}

// A FAIXA DE QUEM VEM DEPOIS: a vez de agora e as duas seguintes, na ordem da
// mesa, dando a volta.
//
// Ler a lista não responde "chamo quem depois?" no ÚLTIMO da rodada — ali o
// próximo está no TOPO, e é justamente o turno em que a pergunta mais importa.
//
// Ela não substitui o "Próximo: Arwen" do botão, e a repetição é deliberada: a
// faixa é onde se LÊ a ordem, o botão é o que o CLIQUE vai fazer.

// turnAhead é um lugar na faixa.
type turnAhead struct {
	Rotulo string
	// Meu é o meu personagem, e é o que faz a faixa responder "quanto falta para
	// mim?" — a pergunta do JOGADOR, que é quem mais precisa dela.
	Meu bool
	// Agora é a vez em curso. Sempre a primeira, e escrito mesmo assim: o
	// desenho não deve depender da posição no laço para saber o que destacar.
	Agora bool
	// ViraARodada marca o lugar onde a rodada seguinte começa — o ponto em que a
	// lista deu a volta. Sem ele a faixa mentiria por omissão: "Zumbi 2 › Ogro"
	// parece a mesma rodada, e não é.
	ViraARodada bool
	// DizVoce troca o nome pela palavra, e ela é uma decisão da faixa INTEIRA e
	// não desta linha: a palavra só desambigua enquanto for UMA. Com dois
	// personagens meus na faixa, "você › Fulano › ⟲ você" deixa de responder
	// quanto falta para mim.
	DizVoce bool
}

// turnStripOf traduz a janela circular para a tela.
//
// A REGRA continua no `live`: quem escolhe os três e dá a volta é o
// `UpcomingTurns`. O que se decide aqui são dois fatos de APRESENTAÇÃO: qual
// dos três é de quem está olhando, e onde a rodada vira.
//
// A VOLTA é recalculada do índice em vez de vir do `UpcomingTurns`, que devolve
// entradas e não posições — a regra é *quem* vem; isto é *onde desenhar o
// símbolo*.
//
// A fila que chega aqui é a que o `StateForRole` já redigiu: um segundo caminho
// até os nomes seria um segundo lugar por onde vazar o que a mesa não vê.
func turnStripOf(st *live.SessionRuntimeState, meus map[int64]bool) []turnAhead {
	janela := live.UpcomingTurns(st.Initiative, st.TurnIndex, turnsAhead)
	faixa := make([]turnAhead, 0, len(janela))
	for passo, entrada := range janela {
		faixa = append(faixa, turnAhead{
			Rotulo:      entrada.Label,
			Meu:         entrada.CharacterID != nil && meus[*entrada.CharacterID],
			Agora:       passo == 0,
			ViraARodada: st.TurnIndex+passo == len(st.Initiative),
		})
	}
	// "VOCÊ" é decisão da faixa inteira, então ela é tomada com ela pronta: a
	// palavra só desambigua enquanto for UMA. Ver o `DizVoce`.
	if meusNaFaixa(faixa) == 1 {
		for i := range faixa {
			faixa[i].DizVoce = faixa[i].Meu
		}
	}
	return faixa
}

func meusNaFaixa(faixa []turnAhead) int {
	quantos := 0
	for _, p := range faixa {
		if p.Meu {
			quantos++
		}
	}
	return quantos
}

// turnsAhead é o tamanho da faixa: a vez e as duas seguintes.
//
// TRÊS é escolhido e não medido — o Go não sabe quanto cabe na caixa. Três
// nomes cabem no cabeçalho a 390px sem truncar a ponto de não identificar
// ninguém. Quem quer a ordem inteira tem o trilho e a gaveta; crescer a faixa
// até competir com eles é como ela deixa de caber.
const turnsAhead = 3

// tableBarOf resolve a porcentagem (presa em 0..100) e o tom.
//
// Máximo ausente ou zero devolve 0% em vez de dividir: uma linha sem pool não
// tem barra cheia nem vazia, ela não tem barra — e é quem chama que decide não
// desenhar.
func tableBarOf(current, max int64, arcane bool) tableBar {
	bar := tableBar{Current: current, Max: max, Pct: ui.VitalPercent(current, max), Tone: "bg-mp-arcane"}
	if !arcane {
		bar.Tone = ui.HpFillTone(bar.Pct)
	}
	return bar
}

// tableTrackerOf desenha a fila que o jogador recebeu — já redigida.
func tableTrackerOf(
	st *live.SessionRuntimeState, meus map[int64]bool, reservas map[int64]int64,
) []tableRow {
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
			Condicoes:  e.Conditions,
			Iniciais:   ui.Monogram(e.Label),
		}
		// O `HpMax` nil depois da redação é como o servidor DIZ "isto não é seu
		// para ver". Desenhar barra aqui inventaria um número — mas a MARCA que
		// veio junto ainda tem de virar tela, senão "não tem PV" e "o mestre
		// escondeu" ficam iguais.
		linha.PV = poolBar(e.HpCurrent, e.HpMax, e.HpHidden, false)
		linha.PM = poolBar(e.MpCurrent, e.MpMax, e.MpHidden, true)
		// A RESERVA pega carona no gargalo da REDAÇÃO, e o que a segura é o
		// MÁXIMO e não a marca do olho.
		//
		// Para a mesa, o pool escondido volta do `StateForRole` sem números —
		// o `poolBar` devolve uma barra só com a marca, `Max` zero —, e o
		// `withTempHp` para aí. Para o MESTRE os números vêm, porque esconder é
		// decisão sobre o que os OUTROS veem: uma trava pela marca cegaria
		// justamente quem a acionou. Medido: com `!Hidden`, o mestre perdia a
		// própria reserva de vista.
		//
		// Só linha com personagem atrás tem reserva — NPC não tem ficha.
		if e.CharacterID != nil {
			withTempHp(linha.PV, reservas[*e.CharacterID])
		}
		fila = append(fila, linha)
	}
	return fila
}

// poolBar traduz UM pool da linha em barra, ou em nada.
//
// Três estados e não dois, e é por isso que ela existe: com número, é barra; sem
// número mas com a marca, é a barra ESCONDIDA que a tela desenha como frase; sem
// número e sem marca, é linha que não rastreia aquele pool e não desenha nada.
// O segundo caso é o que a redação produz para a mesa, e juntá-lo ao
// terceiro faria o jogador ler "este capanga não tem PV" sobre um ogro de 130.
func poolBar(current, max *int64, hidden *bool, arcane bool) *tableBar {
	if max != nil {
		bar := tableBarOf(live.DerefOr(current, 0), *max, arcane)
		bar.Hidden = hidden != nil && *hidden
		return &bar
	}
	if hidden != nil && *hidden {
		return &tableBar{Hidden: true}
	}
	return nil
}

// withTempHp põe a reserva numa barra, e a devolve.
//
// A reserva SÓ CABE onde há máximo: uma linha sem PV rastreado não ganha filete
// nem número, porque não há de que o filete ser uma fração.
func withTempHp(b *tableBar, temp int64) {
	if b == nil || temp <= 0 || b.Max <= 0 {
		return
	}
	b.Temp = temp
	b.TempPct = ui.VitalPercent(temp, b.Max)
}

// tableViewOf monta a tela a partir das partes já buscadas. Tudo o que decide
// mora aqui; o handler ao lado só sabe buscar.
func tableViewOf(
	st *live.SessionRuntimeState,
	campaignID, sessionID, sessionNum int64,
	grupo []Member,
	meus map[int64]bool,
	eu *tableMe,
	reservas map[int64]int64,
) View {
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
	return View{
		CampaignID:  campaignID,
		SessionID:   sessionID,
		SessionNum:  sessionNum,
		SceneActive: st.InScene(), Scene: sceneOf(st),
		Round:    st.Round,
		Turn:     tableTurnOf(st, meus),
		Proximos: turnStripOf(st, meus),
		Grupo:    grupo,
		Fila:     tableTrackerOf(st, meus, reservas),
		Eu:       eu,
	}
}

// ── o rastreador do MESTRE ───────────────────────────────────────────────────
//
// A tela do mestre é a do jogador mais o que ele COMANDA, e não uma segunda
// cena: duas cenas seriam duas listas de combatente para manter em dia. Quem
// decide entre as duas é o PAPEL, resolvido no servidor pelo `stateForRole`.

// viewGm é o acréscimo do mestre sobre a `View`.
type viewGm struct {
	// Contador é a frase que diz ONDE a sessão está.
	Contador string
	// Upkeep é o que sustentar cobrou de quem entrou na vez (p227). Vazia
	// quando não há sustentada, e aí a linha não é desenhada.
	Upkeep string
	// Avanco é o rótulo do botão mais clicado da sessão, e ele diz PARA ONDE vai
	// em vez de o que faz.
	Avanco live.NextTurnTarget
	// VeVitais decide se a fila mostra PV de NPC. A pergunta é sobre a FILA e
	// não sobre o papel: numa fila só de PCs não há o que reservar.
	VeVitais bool
	// Conectados são os personagens de quem está com a aba aberta agora.
	Conectados map[int64]bool
	// PodeAvancar separa "não há para onde ir" de "o botão está quebrado": sem
	// cena aberta o avanço não existe, e um botão aceso que recusa é pior que um
	// apagado que explica.
	PodeAvancar bool
	// GravacaoFalhando: a mesa está rodando de MEMÓRIA e o disco não recebeu a
	// última escrita.
	//
	// Ele vive no bloco do MESTRE e não na `View` porque quem pode parar a sessão
	// e chamar alguém é ele; para o jogador seria um alarme sobre o qual não há o
	// que fazer.
	GravacaoFalhando bool
}

func ofViewGm(
	st *live.SessionRuntimeState,
	membros []live.TableMember,
	presentes []int64,
	ehMestre bool,
	gravacaoFalhando bool,
) viewGm {
	return viewGm{
		GravacaoFalhando: gravacaoFalhando,
		Contador:         live.TurnCounter(st.Scene, st.Round, st.TurnIndex, len(st.Initiative)),
		Upkeep:           live.UpkeepLine(upkeepOf(st)),
		Avanco:           live.NextTurnButton(st.Initiative, st.TurnIndex),
		VeVitais:         live.GmSeesVitals(st.Initiative, ehMestre),
		Conectados:       live.ConnectedCharacters(membros, presentes),
		PodeAvancar:      st.CountsRounds() && len(st.Initiative) > 0,
	}
}

// SessionBase é O ENDEREÇO desta sessão, e os gestos da cena penduram o verbo
// nele: `v.SessionBase() + "/notas"`.
//
// Ela é MÉTODO e não campo, e a diferença tem um defeito atrás. O `BoardView.Base`
// é campo porque ali o prefixo é uma ESCOLHA — o mesmo tabuleiro posta na mesa
// ou no rascunho do acervo, e quem monta a view decide qual. Aqui não há
// escolha: o endereço é função dos dois ids e de mais nada. Um campo abriria a
// única falha que este desenho não tem — a view sintética que alguém constrói
// sem preenchê-lo, e que passa a postar em lugar nenhum. É a mesma família do
// `/campanhas/0/sessoes/0/notas` que a prévia das notas já produziu.
//
// O literal mora no `routes.Session`, e em mais lugar nenhum: quem cobra é o
// `TestNoHandwrittenSessionAddress`.
func (v View) SessionBase() string { return routes.Session(v.CampaignID, v.SessionID) }

// tableCommand escreve a chamada Datastar de um comando do mestre.
//
// O caminho é o da CENA e não o da API JSON: as rotas próprias chamam as MESMAS
// regras extraídas, e o que impede as duas telas de divergirem é compartilhar a
// REGRA, não a rota.
func tableCommand(v View, metodo, acao string) string {
	caminho := v.SessionBase() + "/" + acao
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
func rowCommand(v View, l tableRow, acao string) string {
	return fmt.Sprintf("@post('%s/iniciativa/%s/%s')", v.SessionBase(), l.ID, acao)
}

// rowVital escreve o ferir/curar com os DOIS passos já resolvidos em duas
// URLs, e o `evt.shiftKey` escolhendo entre elas.
//
// O `@post` recebe uma expressão como ARGUMENTO, e não é um `@post` dentro de
// cada braço de um ternário: a chamada é uma só, e o que varia é a string. Assim
// o que o Datastar precisa reescrever é uma ação, não duas dentro de um desvio.
//
// healVerb e harmVerb dão a CADA pool o verbo da mesa, e não é enfeite: com os
// dois passos na mesma linha, "Ferir Arwen" duas vezes são dois controles com o
// mesmo nome acessível e destinos diferentes — quem usa leitor de tela ouve a
// mesma frase e tira mana achando que tira vida.
//
// A frase é a da mesa: PV se fere e se cura, PM se gasta e se recupera. O "de"
// mora no verbo do mana porque "Gastar PM Arwen" não é português.
func healVerb(pool string) string {
	if pool == "mp" {
		return "Recuperar PM de"
	}
	return "Curar"
}

func harmVerb(pool string) string {
	if pool == "mp" {
		return "Gastar PM de"
	}
	return "Ferir"
}

func rowVital(v View, l tableRow, pool, verb string) string {
	base := fmt.Sprintf("%s/iniciativa/%s/vitais/%s/%s/", v.SessionBase(), l.ID, pool, verb)
	return fmt.Sprintf("@post(evt.shiftKey ? '%s5' : '%s1')", base, base)
}

// openEdit semeia o diálogo com os valores de AGORA e o abre.
//
// Semear é obrigatório e não é conveniência: o diálogo é UM para a fila inteira,
// então sem isto ele abriria com o que sobrou da linha anterior — e o mestre
// salvaria o PV do Ogro em cima do Goblin sem ver nada de errado.
func openEdit(v View, l tableRow) string {
	pv, pvMax := int64(0), int64(0)
	if l.PV != nil {
		pv, pvMax = l.PV.Current, l.PV.Max
	}
	return fmt.Sprintf(
		"$edit_row = '%s'; $edit_name = %s; $edit_initiative = %d; $edit_hp = %d; $edit_hp_max = %d; document.getElementById('edit-combatant').showModal()",
		l.ID, jsTextHow(l.Rotulo), l.Iniciativa, pv, pvMax,
	)
}

// saveEdit monta o caminho com o id que o número semeou.
func saveEdit(v View) string {
	return fmt.Sprintf(
		"document.getElementById('edit-combatant').close(); @post('%s/iniciativa/' + $edit_row + '/editar')",
		v.SessionBase(),
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

// ── As CONDIÇÕES do combatente na tela ──────────────────────────────────────

// openConditions escolhe a linha e abre o diálogo.
//
// Ele reescreve OS DOIS sinais, e é quem TROCA de item que limpa: quem gera
// não sabe que haverá um próximo; quem troca sabe que houve um anterior.
func openConditions(l tableRow) string {
	return fmt.Sprintf(
		"$condition_row = %q; $row_conditions = %q; $row_label = %q;"+
			" document.getElementById('combatant-conditions').showModal()",
		l.ID, strings.Join(l.Condicoes, ","), l.Rotulo,
	)
}

// onCondition é a pergunta que pinta o crachá do diálogo.
func onCondition(id string) string {
	return fmt.Sprintf("$row_conditions.split(',').includes(%q)", id)
}

// toggleConditionRow posta o clique na linha ESCOLHIDA.
//
// O `entryId` sai do sinal e não do caminho escrito pelo servidor porque o
// diálogo é UM só para todas as linhas — é o preço de não desenhar 35 crachás
// por combatente, e o sinal é reescrito a cada abertura.
func toggleConditionRow(v View, id string) string {
	return fmt.Sprintf(
		"@post('%s/iniciativa/' + $condition_row + '/condicao/%s')",
		v.SessionBase(), id,
	)
}

// ── O CICLO da sessão na tela ───────────────────────────────────────────────

// sessionAddress é o endereço do RECURSO, e os três verbos do ciclo agem sobre
// ele com método diferente. Uma função e não três `Sprintf`: o endereço é o
// mesmo, e é o método que diz o que se quer.
func sessionAddress(v View) string {
	return routes.Session(v.CampaignID, v.SessionID)
}

// sessionPatch escreve o remendo do ciclo — o que muda, e só isso.
//
// # O `payload` não é enfeite: sem ele o gesto renomeia de carona
//
// O `@patch` sem payload manda os SINAIS DA PÁGINA, e `session_title` é um
// deles — está ligado ao campo de texto das configurações. Um pedido de status
// levaria o título junto e gravaria o que estivesse digitado (ou apagado) no
// campo, sem ninguém ter clicado em Salvar.
//
// O `payload` SUBSTITUI os sinais, então o que vai é exatamente o que está
// escrito aqui. É a mesma razão que faz o handler ler ponteiros: só o campo
// presente muda.
//
// # A chave é LITERAL nas duas, e não um parâmetro
//
// Uma função só, com a chave vindo por argumento, produziria `{payload: {%s: …}}`
// no código — e aí a chave não existe em lugar nenhum que se possa ler. O
// `TestEveryPayloadKeyMatchesTheSignalItReads` confere cada chave contra a tag
// `json` que a lê, e ele não tem como conferir uma que só existe em tempo de
// execução. Duas funções custam uma linha e mantêm as duas chaves no `grep`.
//
//	sessionStatusPatch(v, "active") // @patch('…', {payload: {status: 'active'}})
func sessionStatusPatch(v View, status string) string {
	return fmt.Sprintf("@patch('%s', {payload: {status: '%s'}})", sessionAddress(v), status)
}

// sessionTitlePatch manda o título que está no campo, e só ele.
func sessionTitlePatch(v View) string {
	return fmt.Sprintf("@patch('%s', {payload: {session_title: $session_title}})", sessionAddress(v))
}

// restartCombatCommand é `POST` num sub-recurso e não um remendo na sessão:
// reiniciar o combate NÃO muda o status — a partida continua no ar, e o que
// esvazia é a fila.
func restartCombatCommand(v View) string {
	return fmt.Sprintf("@post('%s/combate/reiniciar')", sessionAddress(v))
}

// deleteSessionCommand apaga a sessão. A NAVEGAÇÃO vem pelo fio (`sse.Redirect`)
// porque `DELETE` não cabe num `form` de HTML — ver o handler.
func deleteSessionCommand(v View) string {
	return fmt.Sprintf("@delete('%s')", sessionAddress(v))
}

// campaignChronicle é para onde se sai da sessão.
//
// A crônica e não o Hub: é de lá que se entra numa sessão, então é lá que o
// mestre continua o que estava fazendo. Sair para a raiz obrigaria a refazer
// dois cliques para voltar à mesa que ele acabou de deixar.
func campaignChronicle(v View) string {
	// Pelo `routes` e não à mão: a crônica é cena de OUTRO pacote, e o critério
	// de entrada daquele arquivo é exatamente este — endereço que uma cena cita
	// de outra.
	return routes.CampaignTab(v.CampaignID, "")
}

// portugueseCycle é o que o crachá do cabeçalho diz.
//
// Traduzido AQUI e não no banco: `planned`/`active`/`ended` são a forma de fio,
// e a tela não deve imprimir identificador.
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
func openConfigSession(v View) string {
	return fmt.Sprintf("$session_title = %q; document.getElementById('session-config').showModal()", v.Titulo)
}

// upkeepOf lê o extrato da manutenção da cena em curso, se houver cena.
func upkeepOf(st *live.SessionRuntimeState) *live.TurnUpkeep {
	if st == nil || st.Scene == nil {
		return nil
	}
	return st.Scene.Upkeep
}
