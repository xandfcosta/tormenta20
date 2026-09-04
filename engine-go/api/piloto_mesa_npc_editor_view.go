package api

import (
	"encoding/json"
	"fmt"
	"t20engine/book"

	"github.com/a-h/templ"
	"t20engine/creature"
)

// As expressões e as listas do EDITOR DE BLOCO (ALE-269).
//
// A divisão do trabalho é a mesma do resto do piloto, e aqui ela cai num lugar
// incomum: o RASCUNHO inteiro é do navegador — cada caixa escreve num pedaço de
// `$rascunho` —, e o servidor só entra onde o navegador não sabe ir sozinho, que
// é mudar o NÚMERO DE LINHAS de uma lista. Datastar não tem laço no cliente.

// Os nomes das três listas de tamanho variável. Constantes porque cada uma
// aparece na rota, no fragmento e no `data-bind` de cada campo: escritas à mão, a
// quarta ocorrência é a que erra a letra e liga uma linha a um sinal que ninguém
// lê.
const (
	listaDeAtaques     = "ataque"
	listaDePericias    = "pericia"
	listaDeHabilidades = "habilidade"
)

// As três abas do editor (decisão do dono).
//
// Uma coluna só com os ~25 campos mais as três listas rola demais num diálogo de
// 40rem; as abas cortam por PERGUNTA e não por tamanho — "quais são os números
// dele", "como ele bate", "o que ele sabe e o que ele carrega".
const (
	abaDosNumeros = "numeros"
	abaDosAtaques = "ataques"
	abaDasPosses  = "posses"
)

type editorTab struct {
	ID     string
	Rotulo string
}

var editorTabs = []editorTab{
	{abaDosNumeros, "Números"},
	{abaDosAtaques, "Ataques"},
	{abaDasPosses, "Perícias e posses"},
}

// blocoEmBranco é a semente de "criar do zero", e ela NÃO é o zero de tudo.
//
// Um bloco todo em zero seria recusado pelo `validateCreature` (PV precisa de 1)
// e, pior, ensinaria errado: quem escreve do zero está inventando um NPC, não
// preenchendo um formulário — os padrões do humano médio (Defesa 10, 9m de
// deslocamento, PV 10) são o ponto de partida do livro e poupam seis campos.
//
// Os mesmos números do `blankCreatureBlock` da SPA, portados e não inventados: as
// duas telas escrevem na MESMA coluna do MESMO banco, e dois brancos diferentes
// dariam dois NPCs diferentes para o mesmo gesto.
func blocoEmBranco() creature.Block {
	return creature.Block{
		ND: 1, Tipo: "humanoide", Size: "medio",
		Defesa: 10, HP: 10, Deslocamento: "9m (6q)",
		Attacks: []creature.Attack{}, Skills: []creature.Skill{}, SpecialAbilities: []string{},
	}
}

// blankDraft é a FORMA do rascunho para a semente da página.
//
// A forma inteira e não um objeto vazio, e a razão é uma armadilha do Datastar:
// `data-bind` num caminho que ainda não existe cria um sinal NOVO em vez de
// escrever no de baixo, e a caixa nasceria muda até o primeiro `@post`. Semear a
// forma completa é o que faz cada campo já ter onde escrever no primeiro
// caractere.
//
// Sai como JSON e não como texto escrito à mão pelo mesmo motivo de sempre: uma
// segunda grafia dos vinte e cinco campos seria a que envelhece.
func blankDraft() string {
	bruto, err := json.Marshal(paraOFormulario(0, "", blocoEmBranco()))
	if err != nil {
		// Um `CreatureBlock` de campos simples não tem como falhar aqui, e cair
		// num objeto vazio deixaria a página sem sinal em vez de sem servidor.
		return "{}"
	}
	return string(bruto)
}

// draftLists são os três fragmentos que o servidor redesenha.
//
// Uma lista e não três chamadas soltas: os três são a MESMA resposta a qualquer
// gesto de forma, e mandar só o que mudou exigiria saber qual mudou — informação
// que o handler tem e que não vale o acoplamento. Três fragmentos por clique é
// HTML de meia dúzia de linhas.
func draftLists(c commandCtx, rascunho npcDraft) []templ.Component {
	return []templ.Component{
		draftAttacks(c.CampaignID, c.SessionID, rascunho.Bloco.Attacks),
		draftExpertises(c.CampaignID, c.SessionID, rascunho.Bloco.Skills),
		draftAbilities(c.CampaignID, c.SessionID, rascunho.Bloco.SpecialAbilities),
	}
}

// draftField é o caminho de um pedaço do rascunho, para o `data-bind`.
//
// O caminho é montado aqui e nunca escrito à mão no `.templ`: `rascunho.bloco.hp`
// digitado errado não dá erro em lugar nenhum — a caixa liga um sinal NOVO, o
// servidor lê o antigo para sempre vazio, e o número some ao salvar.
//
// @example draftField("hp") // "rascunho.bloco.hp"
func draftField(campo string) string { return "rascunho.bloco." + campo }

// rowField é o mesmo para um item de lista, com o índice no meio.
//
// @example rowField(listaDeAtaques, 0, "name") // "rascunho.bloco.attacks.0.name"
func rowField(lista string, indice int, campo string) string {
	caminho := fmt.Sprintf("rascunho.bloco.%s.%d", blockName(lista), indice)
	if campo == "" {
		return caminho
	}
	return caminho + "." + campo
}

// blockName traduz o nome da ROTA para o nome do campo no JSON do bloco.
//
// Os dois divergem e é deliberado: a rota fala a língua do usuário ("ataque") e
// o bloco fala a do wire, que veio da SPA e é inglês ("attacks"). Traduzir num
// lugar só é o que impede a terceira grafia de aparecer num `data-bind`.
func blockName(lista string) string {
	switch lista {
	case listaDeAtaques:
		return "attacks"
	case listaDePericias:
		return "skills"
	default:
		return "specialAbilities"
	}
}

// openEditor é o gesto que troca a lista do elenco pelo formulário.
//
// O `npcId` vazio abre em branco; com id, o servidor semeia do banco. Quem faz a
// troca é o SERVIDOR devolvendo `rascunhoaberto`, e não o clique: assim o
// formulário nunca aparece antes de ter conteúdo — um editor que abre vazio e
// preenche meio segundo depois mostra os números de outro NPC no meio do caminho.
func openEditor(v tableView, npcID int64) string {
	if npcID == 0 {
		return fmt.Sprintf("@post('/mesa/%d/%d/elenco/npc/novo')", v.CampaignID, v.SessionID)
	}
	return fmt.Sprintf("@post('/mesa/%d/%d/elenco/npc/%d/editar')", v.CampaignID, v.SessionID, npcID)
}

// closeEditor é o Cancelar, e ele não fala com o servidor.
//
// Não precisa: o rascunho mora no navegador e NADA foi escrito. É a metade que
// paga a decisão do dono — "Cancelar desfaz de verdade" é grátis quando não há
// nada a desfazer.
const closeEditor = "$rascunhoaberto = false; $erroDoRascunho = ''"

// listCommand escreve o gesto que acrescenta ou tira uma linha.
func listCommand(campanha, sessao int64, lista string, indice int) string {
	base := fmt.Sprintf("/mesa/%d/%d/elenco/npc/rascunho/%s", campanha, sessao, lista)
	if indice < 0 {
		return fmt.Sprintf("@post('%s/nova')", base)
	}
	return fmt.Sprintf("@post('%s/%d/remover')", base, indice)
}

// salvaOBloco é o único gesto desta tela que grava.
func salvaOBloco(v tableView) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/elenco/npc/rascunho/salvar')", v.CampaignID, v.SessionID)
}

// onTabExpr é a condição que mostra uma aba. Escrita aqui e não no `.templ` porque o
// id da aba tem de casar com o do botão que a liga, e dois literais divergem.
func onTabExpr(aba string) string { return fmt.Sprintf("$rascunhoaba === %q", aba) }

// editorTitle diz o que está aberto, e ele vem do SINAL e não do servidor.
//
// Do sinal porque o nome é editável: o cabeçalho tem de acompanhar a caixa
// enquanto o mestre digita "Ogro Capitão", senão ele lê o nome antigo sobre o
// formulário novo. Nome vazio cai em "NPC sem nome", que é o que a validação vai
// recusar — dizê-lo antes é mais barato que recusar depois.
const editorTitle = "$rascunho.nome || 'NPC sem nome'"

// kindOptions e sizeOptions são as listas do livro, com os rótulos que o
// bestiário já usa — um segundo par faria o mesmo Ogro ser "Humanoide" numa tela
// e "humanoid" na outra.
func kindOptions() []blockOption { return options(creatureTiposNaOrdem, book.TypeName) }
func sizeOptions() []blockOption { return options(creatureSizesNaOrdem, book.SizeName) }

type blockOption struct {
	Valor  string
	Rotulo string
}

func options(valores []string, rotulo func(string) string) []blockOption {
	fora := make([]blockOption, 0, len(valores))
	for _, v := range valores {
		fora = append(fora, blockOption{Valor: v, Rotulo: rotulo(v)})
	}
	return fora
}

// tabSummary é o que o botão da aba diz além do nome: quantas linhas há dentro.
//
// Sem o número, "Ataques" não distingue um bloco com três ataques de um sem
// nenhum, e o mestre precisa abrir a aba para descobrir o que já preencheu — que
// é exatamente o custo que as abas introduzem e este contador devolve.
func tabSummary(aba string) string {
	switch aba {
	case abaDosAtaques:
		return "$rascunho.bloco.attacks.length"
	case abaDasPosses:
		return "$rascunho.bloco.skills.length + $rascunho.bloco.specialAbilities.length"
	default:
		return ""
	}
}

// pickTab liga a aba pedida. Não desliga ao reclicar, ao contrário do trilho
// de ferramentas do mapa: uma aba desligada não deixaria nada na tela.
func pickTab(aba string) string { return fmt.Sprintf("$rascunhoaba = %q", aba) }

// tabStyling liga UMA das duas aparências, e nunca deixa as duas na mesa.
//
// Os DOIS lados no `data-class` em vez de a cor apagada ficar no `class` fixo, e
// isso é uma armadilha de CASCATA que custou uma medição: a marca de aberta
// (`.aba-ligada`) mora em `@layer components` e o `text-muted-foreground` é uma
// utilidade do Tailwind, que vive numa camada POSTERIOR — camada vence
// especificidade, então o dourado perdia para o cinza sem nada acusar. A aba
// ficava com a classe certa e a cor errada.
//
// Alternar as duas resolve por construção e não por ordem: só uma existe no
// elemento a cada instante, e a regra deixa de depender de onde o Tailwind
// resolveu escrever a folha.
func tabStyling(aba string) string {
	return fmt.Sprintf("{'aba-ligada': %s, 'text-muted-foreground': !(%s)}", onTabExpr(aba), onTabExpr(aba))
}

// attackName e expertiseName dizem O QUE o botão de remover vai tirar.
//
// Com cinco linhas na tela, cinco botões chamados "Remover" são cinco botões
// idênticos para quem navega por leitor de tela — e o que se apaga por engano é
// justamente o que se acabou de escrever. A linha em branco (a que acabou de
// nascer) cai no número, que é o que ela tem.
func attackName(a creature.Attack, indice int) string {
	return orderOrName(a.Name, "o ataque", indice)
}

func expertiseName(p creature.Skill, indice int) string {
	return orderOrName(p.Name, "a perícia", indice)
}

func orderOrName(nome, oQue string, indice int) string {
	if nome == "" {
		return fmt.Sprintf("%s %d", oQue, indice+1)
	}
	return nome
}

// abilityPlaceholder lembra a FORMA da linha do livro (p289) sem preencher
// nada: habilidade especial é prosa, e uma caixa vazia sem exemplo faz o mestre
// escrever um título quando o livro escreve uma frase.
const abilityPlaceholder = "Faro apurado. Recebe +2 em testes de Percepção baseados em olfato."

// Os TIPOS na ordem do trilho do bestiário, e não a do mapa de validação: mapa
// não tem ordem, e uma lista de opções que se reordena a cada abertura é uma
// lista que ninguém consegue usar. Reusar a `tiposDeCriatura` do bestiário é o
// que faz o mestre encontrar "Morto-vivo" no mesmo lugar nas duas telas.
var creatureTiposNaOrdem = book.CreatureTypes

// Os TAMANHOS na ordem do livro — do menor para o maior, que é a única ordem que
// alguém procura.
var creatureSizesNaOrdem = []string{"minusculo", "pequeno", "medio", "grande", "enorme", "colossal"}
