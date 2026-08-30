package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// A FICHA como dado (ALE-272, fatia 1) — a casca, as abas e o crachá.
//
// É a última tela sem contraparte em Datastar, e é ela que segura a SPA inteira
// de pé. Esta fatia entrega o que a ficha tem de ENVOLTÓRIO: o endereço, as sete
// abas e o crachá do jogador. Os painéis vêm um por fatia, do menor para o
// maior — e enquanto um deles não chegou, a aba dele DIZ isso e leva para a
// ficha antiga, em vez de mostrar um vazio que parece defeito.
//
// A ficha chega COMPUTADA DO SERVIDOR, pela mesma `ComputeSheetV2` que a Mesa e
// a cena de personagens já usam. É a decisão 2 da ALE-225 em ação: sem o
// requisito de offline, não há por que a conta acontecer no navegador — e com
// ela some o WASM, que é fatia própria no fim desta issue.

// fichaView é a ficha de um personagem pronta para desenhar.
type fichaView struct {
	ID   int64
	Nome string
	// Iniciais e Gradiente são o retrato derivado do nome, como no palco e no
	// cartão da campanha: o app não guarda imagem de personagem.
	Iniciais  string
	Gradiente string
	// Papel é "GUERREIRO 10" — a mesma placa do palco de personagens.
	Papel  string
	Resumo string
	Nivel  int64
	// Defesa vem do MOTOR e é travessão quando não há catálogo primado: a tela
	// inteira não pode cair por causa de um número, e um zero seria pior — 0 é
	// um valor plausível, e o jogador agiria sobre ele.
	Defesa  string
	PV      vitalDaFicha
	PM      vitalDaFicha
	SemMana bool
	// Classes é "Guerreiro 3 / Ladino 2" — a mesma linha do palco, e string e
	// não lista porque é assim que a mesa lê e é assim que o cartão já a monta.
	Classes string
	// AsClasses são as classes com o nível de cada uma, e elas existem para o
	// DEGRAU DE NÍVEL: o nível de um personagem é a SOMA dos níveis de classe, e
	// subir um nível é escolher QUAL classe o recebe.
	AsClasses []classeDaFicha
	// Abas são as sete da ficha, com a ativa marcada. Ver `asAbasDaFicha`.
	Abas []abaDaFicha
	// AbaAtiva é o valor resolvido — nunca o que veio na URL cru, que pode ser
	// um endereço antigo ou lixo digitado.
	AbaAtiva string
}

// classeDaFicha é uma classe do personagem, com o que o degrau precisa saber.
type classeDaFicha struct {
	Nome  string
	Nivel int64
	// PodeSubir e PodeDescer são a elegibilidade do livro, e elas são POR CLASSE
	// e não do personagem: descer uma classe de nível 1 a apagaria, e subir com
	// o total em 20 (p32) passaria do teto.
	PodeSubir  bool
	PodeDescer bool
}

type vitalDaFicha struct {
	Atual int64
	Max   int64
	// Fracao é "12/20", que é como a mesa fala.
	Fracao string
	// Porcento é a largura da barra, entre 0 e 100.
	Porcento int
}

// abaDaFicha é uma das sete seções da ficha.
type abaDaFicha struct {
	// Valor é o que vai na URL, e ele é o MESMO da SPA — ver `asAbasDaFicha`.
	Valor  string
	Rotulo string
	Icone  string
	Ativa  bool
	// Portada diz se o painel já existe em Datastar. Enquanto for falso, a aba
	// leva para a ficha antiga em vez de mostrar um vazio.
	Portada bool
}

// asAbasDaFicha são as sete seções, na ORDEM da SPA.
//
// # Os valores são endereço guardado, e não se "arrumam"
//
// `?tab=abilities` continua sendo Poderes, e o comentário da SPA diz por quê: o
// valor sobreviveu de propósito ao renome Habilidades→Poderes, porque link
// compartilhado e favorito apontam para ele. O mesmo vale para a chave `tab`
// estar em inglês enquanto a tela fala português — ela é FRONTEIRA (GLOSSARIO
// §F), e trocá-la quebraria endereços para ganhar estética.
//
// A ordem é a do `SHEET_PANELS`, e a primeira é o padrão de quem chega sem
// `?tab=`.
func asAbasDaFicha() []abaDaFicha {
	return []abaDaFicha{
		{Valor: "expertises", Rotulo: "Perícias", Icone: "Scroll"},
		{Valor: "combat", Rotulo: "Combate", Icone: "Swords"},
		{Valor: "bag", Rotulo: "Mochila", Icone: "Backpack"},
		{Valor: "proficiencies", Rotulo: "Proficiências", Icone: "ShieldCheck"},
		{Valor: "conditionals", Rotulo: "Efeitos", Icone: "Zap"},
		{Valor: "abilities", Rotulo: "Poderes", Icone: "Star"},
		{Valor: "spells", Rotulo: "Magias", Icone: "BookMarked"},
	}
}

// aAbaPedida resolve o que veio na URL contra as sete que existem.
//
// DUAS traduções de endereço antigo, herdadas do `resolveSheetTab` da SPA:
// `inventory` e `equipment` viram `bag`. Elas existem porque a Mochila já se
// chamou assim, e um favorito daquela época não pode cair numa aba que não
// existe — cair no padrão seria abrir a ficha noutra seção sem dizer por quê.
//
// Valor desconhecido cai na PRIMEIRA, que é o mesmo que a SPA faz.
func aAbaPedida(bruto string) string {
	if bruto == "inventory" || bruto == "equipment" {
		return "bag"
	}
	for _, aba := range asAbasDaFicha() {
		if aba.Valor == bruto {
			return bruto
		}
	}
	return asAbasDaFicha()[0].Valor
}

// oPainelJaPortado diz quais abas já têm contraparte em Datastar.
//
// Ela é uma LISTA e não um `false` solto porque é o placar da migração: cada
// fatia desta issue acrescenta um nome aqui, e a aba deixa de mandar o jogador
// para a ficha antiga no mesmo commit em que o painel nasce. Enquanto o mapa
// estiver vazio, a casca é honesta — ela não finge ter o que não tem.
func oPainelJaPortado(valor string) bool {
	portados := map[string]bool{}
	return portados[valor]
}

// carregaFicha busca o personagem e computa a ficha.
//
// A POSSE é conferida como em toda rota de personagem: quem não é dono não
// abre. O `characterOwnedBy` é o mesmo gargalo que a API JSON usa — a cena não
// ganha uma segunda regra sobre quem pode ver a ficha de quem.
func (s *Server) carregaFicha(ctx context.Context, user AuthUser, id int64, aba string) (fichaView, int, error) {
	row, err := s.queries.GetCharacter(ctx, id)
	if err != nil {
		return fichaView{}, 404, fmt.Errorf("personagem %d não existe", id)
	}
	if row.Ownerid != user.ID {
		return fichaView{}, 403, fmt.Errorf("esta ficha não é sua")
	}
	dto, err := s.loadCharacter(ctx, row)
	if err != nil {
		return fichaView{}, 500, err
	}
	cartao := s.cartaoDoHeroi(dto)
	v := fichaView{
		ID:        dto.ID,
		Nome:      dto.Name,
		Iniciais:  cartao.Iniciais,
		Gradiente: cartao.Gradiente,
		Papel:     cartao.Papel,
		Resumo:    cartao.Resumo,
		Nivel:     dto.Level,
		Defesa:    cartao.Defesa,
		PV:        oVital(dto.HpCurrent, dto.HpMax),
		PM:        oVital(dto.MpCurrent, dto.MpMax),
		SemMana:   dto.MpMax == 0,
		Classes:   cartao.Classes,
		AsClasses: asClassesDoDegrau(dto),
		AbaAtiva:  aba,
	}
	for _, item := range asAbasDaFicha() {
		item.Ativa = item.Valor == aba
		item.Portada = oPainelJaPortado(item.Valor)
		v.Abas = append(v.Abas, item)
	}
	return v, 200, nil
}

// oVital monta a barra de PV ou PM.
//
// A FRAÇÃO é o que a mesa fala em voz alta ("doze de vinte"), e a porcentagem é
// só a largura da barra. Máximo ZERO não vira divisão por zero nem barra cheia:
// quem não tem mana tem a barra vazia e apagada, que é o que a SPA faz.
func oVital(atual, max int64) vitalDaFicha {
	v := vitalDaFicha{Atual: atual, Max: max, Fracao: strconv.FormatInt(atual, 10) + "/" + strconv.FormatInt(max, 10)}
	if max <= 0 {
		return v
	}
	pct := int(atual * 100 / max)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	v.Porcento = pct
	return v
}

// aRotaDaFicha é PARA ONDE se abre uma ficha no piloto.
//
// Uma função e não um `Sprintf` espalhado, pela mesma razão do `rotaDaMesa`: no
// dia da virada ela é o único lugar que precisa ser lido para saber quem manda
// para onde.
//
// @example aRotaDaFicha(7, "bag") // "/piloto/personagens/7?tab=bag"
func aRotaDaFicha(id int64, aba string) string {
	if aba == "" {
		return fmt.Sprintf("/piloto/personagens/%d", id)
	}
	return fmt.Sprintf("/piloto/personagens/%d?tab=%s", id, aba)
}

// aFichaAntiga é o endereço da SPA, para as abas que ainda não foram portadas.
//
// Ele existe para a casca ser honesta: uma aba vazia parece defeito, e mandar a
// pessoa procurar a ficha velha por conta é pior do que dar o link. Some no
// commit da última fatia, junto com a SPA.
func aFichaAntiga(id int64, aba string) string {
	return fmt.Sprintf("/characters/%d?tab=%s", id, aba)
}

// oVitalNaRota traduz o rótulo da tela para o pedaço da URL.
//
// Duas palavras para a mesma coisa é o que o GLOSSARIO chama de colisão, e aqui
// ela é deliberada e contida: a TELA diz "PV" porque é o que a mesa fala, e a
// ROTA diz "pv" porque endereço é minúsculo. Esta função é a única costura.
func oVitalNaRota(rotulo string) string {
	if rotulo == "PM" {
		return "pm"
	}
	return "pv"
}

// oSinalDoPasso escreve o rótulo do botão: "+5", "−1".
//
// O MENOS É O SINAL TIPOGRÁFICO (U+2212) e não o hífen: no mesmo tamanho de
// fonte o hífen fica mais curto e mais alto que o traço do "+", e a fileira dos
// quatro botões desalinha. É a mesma escolha que o enquadramento do tabuleiro já
// faz.
func oSinalDoPasso(passo int) string {
	if passo < 0 {
		return "−" + strconv.Itoa(-passo)
	}
	return "+" + strconv.Itoa(passo)
}

// oRotuloDoPasso é o nome acessível: "Curar 5 de PV", "Ferir 1 de PV".
//
// O VERBO muda com o sinal em vez de "mais 5 PV", porque é o verbo que a mesa
// usa — e um leitor de tela lendo "menos cinco pê vê" obriga quem ouve a
// traduzir de volta para "levou cinco".
func oRotuloDoPasso(rotulo string, passo int) string {
	verbo := "Curar"
	if passo < 0 {
		verbo = "Ferir"
		passo = -passo
	}
	return fmt.Sprintf("%s %d de %s", verbo, passo, rotulo)
}

// asClassesDoDegrau monta as classes com a elegibilidade de cada uma.
//
// A REGRA É A DA SPA, e o comentário dela diz por que existe: *"a single-class
// character steps straight; a multiclass one is ASKED which class takes the
// level — guessing would silently put a Bardo level on the Guerreiro"*. Adivinhar
// é o defeito, e ele é silencioso: a ficha fecha certo no total e errado na
// classe, e só aparece quando alguém for usar um poder que não veio.
//
// SUBIR exige que o TOTAL caiba em 20 (p32) — o teto é do personagem, não da
// classe. DESCER exige que a classe tenha mais de um nível: levá-la a zero
// apagaria a classe, que é outra coisa e não tem gesto nesta tela.
func asClassesDoDegrau(dto CharacterDTO) []classeDaFicha {
	var total int64
	for _, cl := range dto.Classes {
		total += cl.Level
	}
	classes := make([]classeDaFicha, 0, len(dto.Classes))
	for _, cl := range dto.Classes {
		classes = append(classes, classeDaFicha{
			Nome:       cl.ClassName,
			Nivel:      cl.Level,
			PodeSubir:  total < 20,
			PodeDescer: cl.Level > 1,
		})
	}
	return classes
}

// asQuePodem filtra as classes elegíveis para um sentido do degrau.
func asQuePodem(classes []classeDaFicha, passo int) []classeDaFicha {
	elegiveis := make([]classeDaFicha, 0, len(classes))
	for _, cl := range classes {
		if (passo > 0 && cl.PodeSubir) || (passo < 0 && cl.PodeDescer) {
			elegiveis = append(elegiveis, cl)
		}
	}
	return elegiveis
}

// oDegrauDireto é o comando de quem só tem UMA classe elegível — o caso comum.
//
// Vazio quando há mais de uma: aí o gesto abre a escolha, porque adivinhar qual
// classe recebe o nível é o defeito que a SPA nomeia.
func oDegrauDireto(v fichaView, passo int) string {
	elegiveis := asQuePodem(v.AsClasses, passo)
	if len(elegiveis) != 1 {
		return ""
	}
	return oComandoDoDegrau(v.ID, elegiveis[0].Nome, passo)
}

// oComandoDoDegrau escreve o `@post` de subir ou descer uma classe.
//
// A CLASSE VAI NO CAMINHO, codificada: nome de classe é do catálogo e não tem
// espaço hoje, mas escrever a rota assumindo isso é o tipo de suposição que
// quebra no dia em que uma classe nova chegar.
func oComandoDoDegrau(id int64, classe string, passo int) string {
	return fmt.Sprintf("@post('/piloto/personagens/%d/nivel/%s/%d')", id, url.PathEscape(classe), passo)
}

// oDialogoDoDegrau é o id do diálogo de escolher a classe, por sentido.
//
// DOIS diálogos e não um, porque as listas são diferentes: subir oferece as que
// cabem no teto, descer oferece as que têm nível de sobra. Um diálogo só teria
// de ser reescrito no gesto que o abre — e o guia do pacote já registra o que
// acontece quando um nó compartilhado recebe escrita depois de renderizado.
func oDialogoDoDegrau(passo int) string {
	if passo > 0 {
		return "subir-de-nivel"
	}
	return "descer-de-nivel"
}
