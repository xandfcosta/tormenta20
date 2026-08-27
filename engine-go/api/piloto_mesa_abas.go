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

// asAbasEscolhidas guarda qual tabuleiro cada pessoa está olhando.
//
// Tipo próprio e não um `sync.Map` solto no `Server` pela mesma razão da lente:
// a chave é composta e a regra de leitura tem um caso — "a aba que eu escolhi
// foi fechada" — que precisa morar junto do dado.
type asAbasEscolhidas struct {
	mu        sync.RWMutex
	escolhida map[chaveDaAba]string
}

type chaveDaAba struct {
	SessionID int64
	UserID    int64
}

func novasAbas() *asAbasEscolhidas {
	return &asAbasEscolhidas{escolhida: map[chaveDaAba]string{}}
}

// Escolhe grava a aba que esta pessoa está olhando.
func (a *asAbasEscolhidas) Escolhe(sessionID, userID int64, tabuleiroID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	chave := chaveDaAba{SessionID: sessionID, UserID: userID}
	if tabuleiroID == "" {
		// APAGA em vez de gravar vazio, como o `Alterna` da lente: o mapa vive
		// enquanto o processo viver, e uma sessão que acumulasse uma entrada
		// morta por pessoa nunca devolveria a memória.
		delete(a.escolhida, chave)
		return
	}
	a.escolhida[chave] = tabuleiroID
}

func (a *asAbasEscolhidas) Escolhida(sessionID, userID int64) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.escolhida[chaveDaAba{SessionID: sessionID, UserID: userID}]
}

// Apaga esquece as escolhas de uma sessão inteira.
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
	escolhida := s.abas.Escolhida(sessionID, userID)
	if escolhida == "" {
		return ""
	}
	for _, aberto := range s.boards.Abertos(ctx, sessionID) {
		if aberto.ID == escolhida {
			return escolhida
		}
	}
	return ""
}

func (s *Server) rotasDasAbas(r chi.Router) {
	// A troca é de TODO MUNDO, e é a metade da issue que o jogador ganha: ele
	// não fica preso ao que o mestre está olhando — quem está na cripta abre a
	// aba da cripta porque quer.
	r.Post("/mesa/{campaignId}/{sessionId}/tabuleiro/aba/{tabuleiroId}",
		s.comandoDaMesa(trocaDeTabuleiro))
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
		barra = append(barra, abaDoTabuleiro{
			ID:      daMesa.ID,
			Nome:    nomeDaAba(daMesa, i),
			Ativa:   ehAAbaAtiva(daMesa.ID, ativa, i),
			Cortina: daMesa.Curtained,
			Comando: fmt.Sprintf("@post('/piloto/mesa/%d/%d/tabuleiro/aba/%s')",
				campaignID, sessionID, daMesa.ID),
		})
	}
	return barra
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
