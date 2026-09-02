package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-chi/chi/v5"

	"t20engine/tabuleiro"
)

// AS ABAS DE TABULEIRO (ALE-205): o grupo se separou e a cena é uma só.
//
// O mestre abre a cripta sem guardar a taverna, e cada pessoa na mesa escolhe
// qual das duas está olhando. Até aqui mostrar o outro lado significava ARQUIVAR
// este — a mesa via a grade sumir e voltar —, porque a sessão tinha um tabuleiro
// e o schema dizia isso com todas as letras (`sessionId INTEGER PRIMARY KEY`).
//
// # Por que a aba ativa é ESTADO DO SERVIDOR, e não um sinal do navegador
//
// A issue tinha decidido o contrário em 2026-08-22, e aquilo foi escrito quando
// o alvo era o SolidJS, onde o componente segura o próprio estado. Em Datastar
// quem desenha o mapa é o SERVIDOR, e uma aba local custaria duas coisas que
// não se pagam:
//
//   - o stream teria de mandar TODOS os tabuleiros abertos em TODO quadro,
//     para o cliente poder trocar sem ir ao servidor (o mapa sozinho já é
//     ~41,7 KB por pintura, medido na ALE-203);
//   - e o jogador receberia no HTML a cena que ele não está olhando, que é
//     exatamente o que o `BoardForRole` existe para não fazer.
//
// A forma certa já estava no projeto: a LENTE (`asLentes`) é estado do servidor
// por `(sessão, pessoa)` pelo mesmo motivo, e está escrito lá — "o stream não
// pergunta nada a ninguém". Esta é a irmã dela, e paga os mesmos preços: duas
// abas do navegador da mesma pessoa compartilham a escolha, e a escolha morre
// com o processo (todo mundo volta para a aba padrão, que é a mais antiga).

// asAbasEscolhidas guarda qual tabuleiro cada pessoa está olhando, e o PUXÃO do
// mestre por cima disso.
//
// Tipo próprio e não um `sync.Map` solto no `Server` pela mesma razão da lente:
// a chave é composta e a regra de leitura tem um caso — "a aba que eu escolhi
// foi fechada" — que precisa morar junto do dado.
//
// # O puxão é um CONTADOR da sessão, e não uma escrita na escolha de cada um
//
// "Mostrar esta à mesa" (ALE-205, fatia 2) tinha de alcançar TODO MUNDO, e o
// mapa de escolhas só conhece quem já escolheu: quem entrou e ficou na aba
// padrão não tem entrada nenhuma, e um laço sobre o mapa passaria por cima
// justamente de quem nunca mexeu em nada.
//
// Então o puxão mora na SESSÃO, com um número que só sobe, e cada pessoa guarda
// qual puxão ela já viu. Quem tem `ForcaVista` menor está sendo puxado agora —
// inclusive quem nunca apareceu no mapa, cujo zero é menor que qualquer puxão.
//
// E ele **não sobrescreve a escolha de ninguém**, o que dá o "voltar para onde
// eu estava" de graça: a escolha anterior continua lá, intacta, e é ela que a
// tira do jogador oferece como saída. Sobrescrever seria apagar a informação de
// que a tira precisa.
type asAbasEscolhidas struct {
	mu        sync.RWMutex
	escolhida map[chaveDaAba]escolhaDeAba
	// puxao é o "parem tudo e olhem isto" de cada sessão.
	puxao map[int64]oPuxaoDaMesa
}

type chaveDaAba struct {
	SessionID int64
	UserID    int64
}

// escolhaDeAba é o que uma pessoa escolheu, mais o puxão que ela já consumiu.
type escolhaDeAba struct {
	Tabuleiro string
	// ForcaVista é o número do último puxão que esta pessoa já viu. É o que faz
	// o puxão ser UM EMPURRÃO e não uma trava (decisão do dono): assim que ela
	// escolhe qualquer aba, ela consome o puxão e volta a decidir sozinha.
	ForcaVista int64
}

type oPuxaoDaMesa struct {
	Tabuleiro string
	Seq       int64
}

func novasAbas() *asAbasEscolhidas {
	return &asAbasEscolhidas{
		escolhida: map[chaveDaAba]escolhaDeAba{},
		puxao:     map[int64]oPuxaoDaMesa{},
	}
}

// Escolhe grava a aba que esta pessoa está olhando, e CONSOME o puxão em curso.
//
// Consumir aqui é o que solta a pessoa: ela foi trazida, olhou, e escolheu outra
// coisa — a partir daí a decisão é dela de novo, e a tira do puxão some. Vale
// também quando ela escolhe a própria aba para onde foi trazida: ficar é uma
// escolha, e a tira que continuasse acesa depois dela seria um modo sem gesto.
func (a *asAbasEscolhidas) Escolhe(sessionID, userID int64, tabuleiroID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	chave := chaveDaAba{SessionID: sessionID, UserID: userID}
	visto := a.puxao[sessionID].Seq
	if tabuleiroID == "" && visto == 0 {
		// APAGA em vez de gravar vazio, como o `Alterna` da lente: o mapa vive
		// enquanto o processo viver, e uma sessão que acumulasse uma entrada
		// morta por pessoa nunca devolveria a memória. Com puxão em curso a
		// entrada TEM de existir, mesmo apontando para a padrão — ela é o
		// registro de que esta pessoa já o consumiu.
		delete(a.escolhida, chave)
		return
	}
	a.escolhida[chave] = escolhaDeAba{Tabuleiro: tabuleiroID, ForcaVista: visto}
}

// Puxa traz a mesa para uma aba, e devolve o número do puxão.
//
// Quem puxa já CONSUMIU o próprio puxão: ele está olhando aquela aba — foi por
// isso que a mostrou —, e a tira "o mestre trouxe você para cá" na tela do
// próprio mestre seria a cena contando a ele o que ele acabou de fazer.
func (a *asAbasEscolhidas) Puxa(sessionID, userID int64, tabuleiroID string) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	seq := a.puxao[sessionID].Seq + 1
	a.puxao[sessionID] = oPuxaoDaMesa{Tabuleiro: tabuleiroID, Seq: seq}
	a.escolhida[chaveDaAba{SessionID: sessionID, UserID: userID}] = escolhaDeAba{
		Tabuleiro: tabuleiroID, ForcaVista: seq,
	}
	return seq
}

// Resolve diz qual aba vale para esta pessoa AGORA, se ela está sendo puxada, e
// de onde ela veio.
//
// A ordem importa e é a regra inteira: o puxão ainda não consumido VENCE a
// escolha, e a escolha vence o padrão.
func (a *asAbasEscolhidas) Resolve(sessionID, userID int64) (tabuleiroID string, puxado bool, deOnde string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	minha := a.escolhida[chaveDaAba{SessionID: sessionID, UserID: userID}]
	if puxao := a.puxao[sessionID]; puxao.Seq > minha.ForcaVista {
		return puxao.Tabuleiro, true, minha.Tabuleiro
	}
	return minha.Tabuleiro, false, ""
}

// PuxaoEmCurso é o número do puxão que esta pessoa ainda NÃO consumiu (0 = nenhum).
//
// Existe para o stream, que precisa empurrar a SUPERFÍCIE uma vez por puxão e
// não a cada quadro: empurrar sempre seria uma trava — a pessoa mandada para o
// tabuleiro não conseguiria voltar para a Mesa, porque o quadro seguinte a
// traria de volta 200ms depois, e ela concluiria que o botão está quebrado.
func (a *asAbasEscolhidas) PuxaoEmCurso(sessionID, userID int64) int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	minha := a.escolhida[chaveDaAba{SessionID: sessionID, UserID: userID}]
	if puxao := a.puxao[sessionID]; puxao.Seq > minha.ForcaVista {
		return puxao.Seq
	}
	return 0
}

// Apaga esquece as escolhas e o puxão de uma sessão inteira.
//
// Chamado quando a última cena morre, pelo mesmo motivo da lente: uma escolha
// apontando para um tabuleiro que não existe mais é lixo que sobrevive à sessão.
func (a *asAbasEscolhidas) Apaga(sessionID int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for chave := range a.escolhida {
		if chave.SessionID == sessionID {
			delete(a.escolhida, chave)
		}
	}
	delete(a.puxao, sessionID)
}

// aAbaDe resolve qual tabuleiro esta pessoa está olhando AGORA, conferindo
// contra os que existem.
//
// A conferência é o coração da função e não uma precaução: **a aba que a pessoa
// escolheu pode ter sido fechada pelo mestre enquanto ela olhava.** Sem cair no
// padrão, a tela dela ficaria dizendo "esta sessão não tem tabuleiro" com duas
// cenas abertas na mesa ao lado — e o gesto que causou isso foi de outra pessoa,
// então ela não teria como ligar uma coisa à outra.
//
// Devolve o id VAZIO quando a escolha ainda vale por ser a padrão, e é
// deliberado: vazio é a palavra do store para "a primeira aberta", e reescrevê-la
// aqui como um id concreto faria a resposta envelhecer no instante em que a
// primeira aba trocasse.
func (s *Server) aAbaDe(ctx context.Context, sessionID, userID int64) string {
	aba, _, _ := s.aAbaComOPuxao(ctx, sessionID, userID)
	return aba
}

// aAbaComOPuxao é a resolução INTEIRA: a aba que vale, se ela veio de um puxão
// do mestre, e de onde a pessoa foi trazida.
//
// As rotas usam o `aAbaDe`, que joga fora as duas últimas — para um comando só
// importa ONDE ele age. Quem precisa do resto é a CENA, que tem de dizer à
// pessoa que ela foi trazida e como voltar: um puxão silencioso trocaria o mapa
// debaixo dela no meio de um turno, e ela procuraria o defeito na própria tela.
//
// O puxão para uma cena JÁ ENCERRADA cai na escolha de quem olha, e não numa
// tela morta: o mestre mostra a cripta, encerra a cripta, e quem foi trazido
// volta para onde estava em vez de ficar olhando um tabuleiro que não existe.
func (s *Server) aAbaComOPuxao(ctx context.Context, sessionID, userID int64) (aba string, puxado bool, deOnde string) {
	abertos := s.boards.OpenBoards(ctx, sessionID)
	aberta := func(id string) bool {
		if id == "" {
			return false
		}
		for _, b := range abertos {
			if b.ID == id {
				return true
			}
		}
		return false
	}
	alvo, puxado, deOnde := s.abas.Resolve(sessionID, userID)
	if puxado && !aberta(alvo) {
		alvo, puxado, deOnde = deOnde, false, ""
	}
	if !aberta(alvo) {
		// A aba que a pessoa escolheu pode ter sido fechada pelo mestre enquanto
		// ela olhava. Sem cair no padrão, a tela dela ficaria dizendo "esta sessão
		// não tem tabuleiro" com duas cenas abertas na mesa ao lado — e o gesto
		// que causou isso foi de outra pessoa, então ela não teria como ligar uma
		// coisa à outra.
		alvo = ""
	}
	return alvo, puxado, deOnde
}

func (s *Server) TabRoutes(r chi.Router) {
	// A troca é de TODO MUNDO, e é a metade da issue que o jogador ganha: ele
	// não fica preso ao que o mestre está olhando — quem está na cripta abre a
	// aba da cripta porque quer.
	r.Post("/mesa/{campaignId}/{sessionId}/tabuleiro/aba/{tabuleiroId}",
		s.comandoDaMesa(trocaDeTabuleiro))
	// MOSTRAR À MESA é só do mestre, e a trava é do servidor: um jogador que
	// puxasse a mesa para a aba dele tiraria dos outros cinco o que esta issue
	// acabou de lhes dar.
	r.Post("/mesa/{campaignId}/{sessionId}/tabuleiro/aba/{tabuleiroId}/mostrar",
		s.comandoDoMestreNoTabuleiro(mostraAMesaEstaAba))
}

// mostraAMesaEstaAba é o "parem tudo e olhem isto".
//
// Devolve `nil` como a lente e a troca, e é a MESMA razão apesar de o alcance
// ser outro: puxar não muda a CENA. Nenhuma peça andou, nenhum terreno foi
// pintado — o que mudou foi para onde cada pessoa está olhando, e isso não é um
// quadro do tabuleiro. Quem leva o puxão às outras telas é o batimento do
// stream, que redesenha a cena de cada um a cada 200ms e já pergunta ao
// `aAbaComOPuxao` qual aba vale.
func mostraAMesaEstaAba(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	alvo := chi.URLParam(c.R, "tabuleiroId")
	// Puxar para uma aba que não existe deixaria a mesa inteira caindo no padrão
	// sem nada dizendo por quê. O id vem do caminho, então isto é a conferência
	// de sempre: o que o cliente manda não é a verdade.
	for _, aberto := range st.boards.OpenBoards(c.R.Context(), c.SessionID) {
		if aberto.ID == alvo {
			st.abas.Puxa(c.SessionID, c.User.ID, alvo)
			return nil, nil
		}
	}
	return nil, fmt.Errorf("o tabuleiro %q não está aberto nesta sessão", alvo)
}

// trocaDeTabuleiro põe outra aba na tela de quem clicou.
//
// O nome não é `escolheAAba` porque o EDITOR DE NPC já tem uma função com esse
// nome — as abas dele são outra coisa (Números / Ataques / Perícias). Duas abas
// no mesmo pacote é uma colisão de palavra que o GLOSSARIO registra; o que
// resolve aqui é o verbo dizer o que se troca.
//
// Devolve `nil` como a lente, e pelo mesmo motivo: trocar de aba NÃO é mutação
// da cena. Publicar aqui acordaria a mesa inteira para uma escolha que é de uma
// pessoa só. Quem redesenha a tela de quem clicou é a resposta do comando.
//
// Não confere se o id existe, e isso não é descuido: quem confere é o `aAbaDe`,
// a cada leitura, porque a aba pode morrer DEPOIS da escolha. Uma conferência
// aqui daria a mesma resposta e ainda deixaria a outra necessária.
func trocaDeTabuleiro(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	st.abas.Escolhe(c.SessionID, c.User.ID, chi.URLParam(c.R, "tabuleiroId"))
	return nil, nil
}

// abaDoTabuleiro é uma ficha da barra de abas.
type abaDoTabuleiro struct {
	ID   string
	Nome string
	// Ativa é a que esta pessoa está olhando. Ela não vira botão: é o `<h2>` que
	// nomeia a região.
	Ativa bool
	// Cortina diz que esta aba está sob cortina PARA QUEM OLHA. Para o mestre é
	// a marca de que ele está montando escondido; para o jogador é tudo o que
	// existe daquela aba — o nome não atravessa (ver `BoardForRole`).
	Cortina bool
	Comando string
	// MostraAMesa é o gesto do mestre "parem tudo e olhem isto", e ele só é
	// escrito na aba ATIVA (decisão do dono): ele já está olhando a cena que quer
	// mostrar — foi por isso que trocou para ela —, e um alvo de clique por ficha
	// encheria uma barra que é estreita por natureza.
	MostraAMesa string
}

// oPuxaoNaTela é a tira "o mestre trouxe você para cá", com a saída dela.
//
// Ela existe porque o puxão é a única coisa nesta cena que muda o que a pessoa
// está vendo SEM ela ter feito nada. A cortina e a lente são estados que o
// próprio dono da tela ligou; este não — e um mapa que troca sozinho, no meio de
// um turno, é lido como defeito.
type oPuxaoNaTela struct {
	// Cena é para onde a mesa foi trazida, e Volta é de onde esta pessoa veio.
	//
	// `Volta` VAZIO é quem já estava nesta aba: para essa pessoa o puxão mudou a
	// superfície e não a cena, então não há para onde voltar — o `ComandoDeVolta`
	// aponta para a aba atual, e clicar nele é dizer "vi", que é o mesmo gesto
	// que solta qualquer um do puxão.
	Cena           string
	Volta          string
	ComandoDeVolta string
}

// asAbasDaMesa monta a barra a partir dos tabuleiros abertos, JÁ REDIGIDOS pelo
// papel de quem olha.
//
// A redação acontece antes de o nome ser lido, e é por isso que a aba sob
// cortina chega aqui sem `Place`: o rótulo dela é escrito pelo `nomeDaAba`, e
// não pelo que o servidor guardou. Ler o nome do estado CRU e "esconder na tela"
// seria pôr "Cripta do Rei" no HTML de quem não pode saber que há uma cripta —
// o vazamento que não aparece na tela, só no ver-código-fonte.
func asAbasDaMesa(abertos []*tabuleiro.BoardState, papel, ativa string, campaignID, sessionID int64) []abaDoTabuleiro {
	// UMA aba não é uma barra: com um tabuleiro só não há o que trocar, e a
	// tira de fichas seria enfeite ocupando mapa. A tela cai no `<h2>` de sempre.
	if len(abertos) < 2 {
		return nil
	}
	barra := make([]abaDoTabuleiro, 0, len(abertos))
	for i, aberto := range abertos {
		daMesa := tabuleiro.BoardForRole(papel, aberto)
		ficha := abaDoTabuleiro{
			ID:      daMesa.ID,
			Nome:    nomeDaAba(daMesa, i),
			Ativa:   ehAAbaAtiva(daMesa.ID, ativa, i),
			Cortina: daMesa.Curtained,
			Comando: fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/aba/%s')",
				campaignID, sessionID, daMesa.ID),
		}
		if ficha.Ativa && papel == "gm" {
			ficha.MostraAMesa = fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/aba/%s/mostrar')",
				campaignID, sessionID, daMesa.ID)
		}
		barra = append(barra, ficha)
	}
	return barra
}

// aTiraDoPuxao monta o aviso a partir da barra JÁ REDIGIDA, ou devolve nil.
//
// Ela lê a barra e não o estado cru de propósito: os nomes já passaram pelo
// papel de quem olha, então a tira de um jogador trazido para uma cena sob
// cortina diz "Cena 2" — o mesmo rótulo da ficha dele — em vez de contar o nome
// que a cortina esconde.
//
// Devolve nil quando não há PARA ONDE VOLTAR, e isso não é economia: um aviso
// que diz "você foi trazido" sem oferecer a saída é um modo que a pessoa não
// tem como desfazer, e a casa já decidiu que cada tira carrega a própria saída.
// Acontece de verdade — o mestre puxa e depois encerra a cena de onde a pessoa
// veio — e o caminho de volta passa a ser a barra, que está ali do lado.
func aTiraDoPuxao(barra []abaDoTabuleiro, deOnde string) *oPuxaoNaTela {
	if len(barra) == 0 {
		return nil
	}
	var atual, volta *abaDoTabuleiro
	for i := range barra {
		if barra[i].Ativa {
			atual = &barra[i]
		}
		// `deOnde` vazio é a aba PADRÃO, que é a primeira da barra: quem nunca
		// escolheu estava nela, e é para lá que "voltar" o leva.
		if barra[i].ID == deOnde || (deOnde == "" && i == 0) {
			volta = &barra[i]
		}
	}
	if atual == nil {
		return nil
	}
	// QUEM JÁ ESTAVA NA CENA também recebe a tira, e isso não é ruído: o puxão
	// traz a pessoa para a superfície do TABULEIRO, e o caso mais comum da mesa é
	// exatamente este — o jogador abre na Mesa (a superfície padrão) e fica na
	// aba padrão. Sem a tira, a tela dele trocaria de superfície sozinha e sem
	// explicação, que é a leitura de defeito.
	//
	// O que muda é a SAÍDA: quem veio de outra aba tem para onde voltar; quem já
	// estava aqui só precisa de um jeito de dizer "vi" — e dizer "vi" é escolher
	// esta aba, que é o mesmo gesto que solta qualquer um do puxão.
	if volta == nil || volta.ID == atual.ID {
		return &oPuxaoNaTela{Cena: atual.Nome, ComandoDeVolta: atual.Comando}
	}
	return &oPuxaoNaTela{Cena: atual.Nome, Volta: volta.Nome, ComandoDeVolta: volta.Comando}
}

// ehAAbaAtiva resolve o id vazio, que é "a primeira aberta".
func ehAAbaAtiva(id, ativa string, posicao int) bool {
	if ativa == "" {
		return posicao == 0
	}
	return id == ativa
}

// nomeDaAba escreve o rótulo da ficha.
//
// A aba SOB CORTINA não tem nome para mostrar a quem está do outro lado dela, e
// mesmo assim precisa de uma palavra: a decisão do dono foi que ela APARECE para
// o jogador — sumir e voltar conforme o mestre corre a cortina trocaria a aba
// debaixo do dedo de quem estava olhando. Então ela se chama pela POSIÇÃO, que
// é o que se pode dizer sem contar nada: "Cena 2".
//
// Para o MESTRE o nome atravessa, porque a cortina não é sobre ele.
func nomeDaAba(daMesa *tabuleiro.BoardState, posicao int) string {
	if daMesa.Place != "" {
		return daMesa.Place
	}
	return fmt.Sprintf("Cena %d", posicao+1)
}
