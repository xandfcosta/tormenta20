package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// As rotas da FICHA no piloto (ALE-272, fatia 1).
//
// A ficha nova mora em `/piloto/personagens/{id}` — filha do endereço do elenco,
// e não numa raiz própria: ela é o que se abre DE dentro da lista, e o endereço
// dizer isso é o que faz o ‹ Voltar ter para onde voltar.
//
// NENHUM link aponta para cá ainda, e é deliberado: enquanto os painéis não
// existirem, mandar quem clica num herói para uma casca vazia seria uma
// regressão. A virada dos `href` é a última fatia desta issue, como foi na
// ALE-269 — até lá esta cena se alcança por URL, que é o que a bancada e os
// guardas usam.

func (s *Server) rotasDaFicha(r chi.Router) {
	r.Get("/personagens/{id}", s.handleFicha)
	// O PASSO no caminho e não no corpo, como o quadrado do movimento no
	// tabuleiro: o valor é do botão que foi clicado, e não de um sinal da página
	// que quatro botões disputariam.
	r.Post("/personagens/{id}/vitais/{qual}/{passo}", s.comandoDaFicha(mexeNoVital))
	// A CLASSE vai no caminho porque o nível é dela: o do personagem é a SOMA.
	r.Post("/personagens/{id}/nivel/{classe}/{passo}", s.comandoDaFicha(mudaONivel))
	// DOIS caminhos e não um com "padrao" no lugar da categoria: aí o dia em que
	// uma proficiência se chamasse `padrao` calaria o restaurar, e o `alterna`
	// no meio é o que impede a colisão de existir.
	r.Post("/personagens/{id}/proficiencias/alterna/{categoria}", s.comandoDaFicha(alternaAProficiencia))
	r.Post("/personagens/{id}/proficiencias/padrao", s.comandoDaFicha(restauraOPadraoDaClasse))
	// AS PERÍCIAS (fatia 4). O NOME vai no CAMINHO, escapado, e não num sinal:
	// é a mesma decisão do `oComandoDoDegrau` com a classe, e o `handleDelete`
	// da API JSON já endereça a perícia assim. Um sinal aqui seria disputado por
	// 29 botões idênticos.
	r.Post("/personagens/{id}/pericias/treino/{nome}", s.comandoDaFicha(alternaOTreino))
	r.Post("/personagens/{id}/pericias/atributo/{nome}/{atributo}", s.comandoDaFicha(trocaOAtributo))
	r.Post("/personagens/{id}/pericias/remove/{nome}", s.comandoDaFicha(removeOOficio))
	// A CRIAÇÃO é a única que lê SINAL, porque o nome é texto que a pessoa acabou
	// de digitar e ainda não existe em lugar nenhum para virar caminho.
	r.Post("/personagens/{id}/pericias/nova", s.comandoDaFicha(criaOOficio))
	// OS EFEITOS (fatia 5). Quatro donos de estado, quatro caminhos.
	r.Post("/personagens/{id}/efeitos/condicao/{cond}", s.comandoDaFicha(toggleBookCondition))
	r.Post("/personagens/{id}/efeitos/aplica/{magia}", s.comandoDaFicha(applySpellBuff))
	r.Post("/personagens/{id}/efeitos/encerra/{efeito}", s.comandoDaFicha(endAppliedEffect))
	r.Post("/personagens/{id}/efeitos/postura/{flag}", s.comandoDaFicha(endStance))
	// O SITUACIONAL manda a CHAVE do condicional, que é um encadeado com `::` e
	// texto livre do catálogo dentro — por isso ela vai por SINAL e não no
	// caminho. É a exceção da ficha, e a razão dela é o formato da chave.
	r.Post("/personagens/{id}/efeitos/situacao", s.comandoDaFicha(toggleSituational))
	// AS MAGIAS (fatia 6).
	r.Post("/personagens/{id}/magias/aprende/{magia}", s.comandoDaFicha(learnSpell))
	r.Post("/personagens/{id}/magias/esquece/{magia}", s.comandoDaFicha(forgetSpell))
	r.Post("/personagens/{id}/magias/prepara/{magia}", s.comandoDaFicha(togglePrepared))
	r.Post("/personagens/{id}/magias/conjura/{magia}", s.comandoDaFicha(castSpellFromSheet))
	// A MOCHILA (fatia 7).
	r.Post("/personagens/{id}/itens/{item}/guarda", s.comandoDaFicha(stowItem))
	r.Post("/personagens/{id}/itens/{item}/equipa/{slot}", s.comandoDaFicha(equipItemFromSheet))
	// O DINHEIRO manda o modo e o valor por SINAL: a conta do saldo depende dos
	// dois juntos, e um caminho com o valor dentro daria um endereço diferente
	// a cada tecla digitada no campo.
	r.Post("/personagens/{id}/dinheiro", s.comandoDaFicha(changeMoney))
	r.Post("/personagens/{id}/itens/adiciona/{catalogo}", s.comandoDaFicha(addCatalogItem))
	r.Post("/personagens/{id}/itens/custom", s.comandoDaFicha(addCustomItem))
	r.Post("/personagens/{id}/itens/{item}/edita", s.comandoDaFicha(editItem))
	r.Post("/personagens/{id}/itens/{item}/remove", s.comandoDaFicha(removeItemFromSheet))
	r.Post("/personagens/{id}/itens/{item}/usa", s.comandoDaFicha(useItem))
	r.Post("/personagens/{id}/itens/{item}/melhorias", s.comandoDaFicha(applyOverlays))
	// OS PODERES (fatia 8).
	r.Post("/personagens/{id}/poderes/usa/{poder}", s.comandoDaFicha(usePower))
	r.Post("/personagens/{id}/poderes/postura/{flag}/entra", s.comandoDaFicha(enterStance))
	// AS ESCOLHAS do diálogo (fatia 8).
	r.Post("/personagens/{id}/poderes/escolhe/{poder}", s.comandoDaFicha(pickPower))
	r.Post("/personagens/{id}/poderes/origem/{beneficio}", s.comandoDaFicha(pickOriginBenefit))
	r.Post("/personagens/{id}/poderes/variante/{variante}", s.comandoDaFicha(pickRaceVariant))
	r.Post("/personagens/{id}/poderes/classe/{classe}/{escolha}/{valor}", s.comandoDaFicha(pickClassChoice))
	r.Post("/personagens/{id}/poderes/atributos", s.comandoDaFicha(pickRaceAttributes))
	r.Post("/personagens/{id}/poderes/ascendencia/{ascendencia}", s.comandoDaFicha(pickRaceAscendencia))
}

func (s *Server) handleFicha(w http.ResponseWriter, r *http.Request) {
	id, ok := oPersonagemDaURL(w, r)
	if !ok {
		return
	}
	sinaisDaPagina := osSinaisDaFicha(r)
	view, status, err := s.carregaFicha(
		r.Context(), currentUser(r), id, aAbaPedida(r.URL.Query().Get("tab")),
		sinaisDaPagina.aBusca(), sinaisDaPagina)
	view.Embutida = pedidaDeDentroDaSessao(r)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	// A FICHA DENTRO DA SESSÃO responde só o pedaço (ALE-272, fatia 10b): quem
	// pediu foi a aba "Minha ficha" da Mesa, e mandar a página inteira faria o
	// Datastar remendar a cena da sessão com uma ficha de corpo inteiro.
	if view.Embutida {
		fragmento, err := renderFragmento(r.Context(), cenaDaFicha(view))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = datastar.NewSSE(w, r).PatchElements(fragmento)
		return
	}
	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo: view.Nome + " · Tormenta 20",
		// `cascaNua`: a cena desenha o próprio cabeçalho, com a volta e o nome.
		Forma: cascaNua,
		// `detalhe` é a caixa do Combate cujo diálogo está aberto, e ele mora no
		// <body> porque o <body> nunca é remendado: declarado dentro do painel, o
		// `@post` do PV o redeclararia a cada toque e fecharia o diálogo que o
		// jogador acabou de abrir. Mesma armadilha do `fichaAberta` do bestiário.
		//
		// Minúsculo de propósito: chave de atributo é minusculada pelo HTML, e um
		// `detalheAberto` ligaria um sinal NOVO em vez do que a expressão lê.
		// `oficio` é o diálogo de criar perícia, e `novapericia`/`novoatributo`
		// são os dois campos dele. Tudo MINÚSCULO: chave de atributo é minusculada
		// pelo HTML, e um `data-bind:novaPericia` ligaria um sinal `novapericia`
		// que o servidor lê — mas o `data-bind` teria escrito noutro, e o campo
		// chegaria sempre vazio.
		Sinais: "{detalhe: '', oficio: false, novapericia: '', novoatributo: 'intelligence'," +
			" condicao: false, buff: false, situacao: ''," +
			" aprender: false, aug0: 0, aug1: 0, aug2: 0, aug3: 0, aug4: 0, aug5: 0," +
			" magiabusca: '', magiacirculo: '', magiaescola: ''," +
			" itembusca: '', itemcategoria: '', tibarmodo: 'receber', tibarvalor: 0," +
			" catalogobusca: '', catalogocategoria: '', itemqtd: 1, itemnome: '', itemespacos: 1," +
			" itemrolagempv: 0, itemrolagempm: 0, itemmelhorias: [], itemmaterial: ''," +
			" poderbusca: '', poderdegraus: 0, poder: '', passivas: false," +
			" fonte: 'raca', racaatributos: []}",
	}, cenaDaFicha(view))
}

// pedidaDeDentroDaSessao diz se este pedido veio da superfície "Minha ficha" da
// Mesa. A marca viaja na query pela mesma razão que o `?tab=`: o handler
// descobre o que desenhar lendo a requisição, e sinal do cliente sumiria num F5.
func pedidaDeDentroDaSessao(r *http.Request) bool {
	return r.URL.Query().Get("embutida") == "1"
}

// comandoDaFicha é o gateway das mutações da ficha.
//
// Ele existe pela mesma razão do `comandoDoTabuleiro`: resolver a posse, mutar e
// redesenhar são três passos que toda mutação da ficha faz, e escrevê-los em
// cada handler é como um deles esquece de redesenhar — a pessoa clica, o banco
// muda e a tela fica igual.
//
// A POSSE é conferida aqui, uma vez. A ficha é do dono e de mais ninguém: a
// regra é a mesma da API JSON (`characterFor`), e a cena não ganha uma segunda.
func (s *Server) comandoDaFicha(
	mutar func(*Server, *http.Request, sqlcgen.Character, fichaSignals) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := oPersonagemDaURL(w, r)
		if !ok {
			return
		}
		// UMA leitura de sinais por requisição, e ela vem ANTES de tudo: o
		// `ReadSignals` consome o corpo do POST, então a segunda chamada
		// receberia vazio sem erro nenhum. Ver `piloto_ficha_sinais.go`.
		sinais := osSinaisDaFicha(r)
		row, err := s.queries.GetCharacter(r.Context(), id)
		if err != nil {
			http.Error(w, "este personagem não existe", http.StatusNotFound)
			return
		}
		if row.Ownerid != currentUser(r).ID {
			http.Error(w, "esta ficha não é sua", http.StatusForbidden)
			return
		}
		// A RECUSA VOLTA PELA CENA, e não por um status de erro.
		//
		// Medido na fatia 7 (ALE-272): o `http.Error(400)` que morava aqui não
		// chegava a lugar nenhum. O cliente do Datastar não aplica remendo de
		// resposta que não é 2xx, então a única marca da recusa era uma linha
		// vermelha no CONSOLE — "Failed to load resource: 400" — e na tela o
		// gesto simplesmente não acontecia. Gastar mais dinheiro do que se tem
		// fechava o diálogo e deixava o saldo igual, sem uma palavra.
		//
		// Todas as recusas da ficha são de REGRA — o teto de duas mãos, o PM que
		// falta, a magia que não está preparada —, e regra recusada é informação
		// para quem joga. Ela sobe com a cena inteira redesenhada, que é o que
		// prova que nada mudou, mais a frase. A API JSON continua respondendo os
		// status dela; esta rota desenha página.
		recusa := ""
		if err := mutar(s, r, row, sinais); err != nil {
			recusa = err.Error()
		}
		view, status, err := s.carregaFicha(
			r.Context(), currentUser(r), id, aAbaPedida(r.URL.Query().Get("tab")), sinais.aBusca(), sinais)
		view.Embutida = pedidaDeDentroDaSessao(r)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		view.Recusa = recusa
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDaFicha(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
	}
}

// mexeNoVital soma o passo ao PV ou ao PM, PRENDENDO na faixa.
//
// PRENDER e não recusar, e essa é a diferença entre o gesto e a API: o corpo do
// `PATCH /vitals` manda o valor ABSOLUTO e é recusado fora da faixa, o que está
// certo para um cliente que calculou. Aqui o gesto é "levou seis" — com 4 de PV
// o resultado é zero, e uma recusa faria o mestre clicar quatro vezes de um em
// um para chegar no mesmo lugar.
//
// O TETO é o máximo do personagem: curar além do máximo não é PV temporário, que
// é outra regra e tem dono no motor (`TempHpFuria`).
func mexeNoVital(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	passo, err := oPassoDaURL(r)
	if err != nil {
		return err
	}
	qual := chi.URLParam(r, "qual")
	hp, mp := row.Hpcurrent, row.Mpcurrent
	switch qual {
	case "pv":
		hp = presoNaFaixa(hp+int64(passo), row.Hpmax)
	case "pm":
		mp = presoNaFaixa(mp+int64(passo), row.Mpmax)
	default:
		return fmt.Errorf("vital %q não existe: são 'pv' e 'pm'", qual)
	}
	return s.queries.SetVitalsCurrent(r.Context(), sqlcgen.SetVitalsCurrentParams{
		HpCurrent: hp, MpCurrent: mp, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	})
}

// presoNaFaixa mantém o vital entre zero e o máximo.
func presoNaFaixa(valor, max int64) int64 {
	if valor < 0 {
		return 0
	}
	if valor > max {
		return max
	}
	return valor
}

// mudaONivel sobe ou desce UMA CLASSE, e o nível do personagem acompanha.
//
// A primeira versão desta função escrevia direto no nível do personagem, e
// **estava errada** — descobri comparando com a SPA no navegador, não lendo o
// código: lá o degrau chama `PATCH /classes/level`, porque o nível do
// personagem é a SOMA dos níveis de classe. Escrever o total direto deixa a
// ficha dizendo 13 com as classes somando 12, e os pools de PV e PM (que
// derivam das CLASSES) não se mexem — o número sobe e o personagem não fica
// mais forte.
//
// A regra é a MESMA do handler JSON, extraída para os dois usarem (ver
// `aplicaONivelDaClasse`): a classe tem de ser do personagem, o total é limitado
// a 20, e os pools sincronizam.
func mudaONivel(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	passo, err := oPassoDaURL(r)
	if err != nil {
		return err
	}
	classe, err := url.PathUnescape(chi.URLParam(r, "classe"))
	if err != nil {
		return fmt.Errorf("classe %q não é um nome válido", chi.URLParam(r, "classe"))
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	for _, cl := range dto.Classes {
		if cl.ClassName != classe {
			continue
		}
		alvo := cl.Level + int64(passo)
		if alvo < 1 {
			return fmt.Errorf("%s está no nível 1: descer apagaria a classe", classe)
		}
		_, _, _, _, err := s.aplicaONivelDaClasse(r, row, classe, alvo)
		return err
	}
	return fmt.Errorf("%s não é uma classe deste personagem", classe)
}

// oPersonagemDaURL lê o id do caminho. Erro aqui é URL digitada errada, e a
// resposta é uma frase: quem está do outro lado é um navegador mostrando página.
func oPersonagemDaURL(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "o personagem precisa ser um número", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// oPassoDaURL aceita o sinal de menos: o passo é para os dois lados.
func oPassoDaURL(r *http.Request) (int, error) {
	bruto := chi.URLParam(r, "passo")
	passo, err := strconv.Atoi(bruto)
	if err != nil || passo == 0 {
		return 0, fmt.Errorf("passo %q não é um número diferente de zero", bruto)
	}
	return passo, nil
}
