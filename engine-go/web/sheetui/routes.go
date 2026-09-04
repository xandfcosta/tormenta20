package sheetui

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
	"t20engine/web/ui"
)

// As rotas da FICHA no piloto (ALE-272, fatia 1).
//
// A ficha nova mora em `/personagens/{id}` — filha do endereço do elenco,
// e não numa raiz própria: ela é o que se abre DE dentro da lista, e o endereço
// dizer isso é o que faz o ‹ Voltar ter para onde voltar.
//
// NENHUM link aponta para cá ainda, e é deliberado: enquanto os painéis não
// existirem, mandar quem clica num herói para uma casca vazia seria uma
// regressão. A virada dos `href` é a última fatia desta issue, como foi na
// ALE-269 — até lá esta cena se alcança por URL, que é o que a bancada e os
// guardas usam.

func Routes(r chi.Router, s Scene) {
	r.Get("/personagens/{id}", s.sheetHandle)
	// O PASSO no caminho e não no corpo, como o quadrado do movimento no
	// tabuleiro: o valor é do botão que foi clicado, e não de um sinal da página
	// que quatro botões disputariam.
	r.Post("/personagens/{id}/vitais/{qual}/{passo}", s.sheetCommand(touchesVital))
	// A CLASSE vai no caminho porque o nível é dela: o do personagem é a SOMA.
	r.Post("/personagens/{id}/nivel/{classe}/{passo}", s.sheetCommand(mudaONivel))
	// DOIS caminhos e não um com "padrao" no lugar da categoria: aí o dia em que
	// uma proficiência se chamasse `padrao` calaria o restaurar, e o `alterna`
	// no meio é o que impede a colisão de existir.
	r.Post("/personagens/{id}/proficiencias/alterna/{categoria}", s.sheetCommand(toggleProficiency))
	r.Post("/personagens/{id}/proficiencias/padrao", s.sheetCommand(restoresDefaultClass))
	// AS PERÍCIAS (fatia 4). O NOME vai no CAMINHO, escapado, e não num sinal:
	// é a mesma decisão do `stepCommand` com a classe, e o `handleDelete`
	// da API JSON já endereça a perícia assim. Um sinal aqui seria disputado por
	// 29 botões idênticos.
	r.Post("/personagens/{id}/pericias/treino/{nome}", s.sheetCommand(toggleTraining))
	r.Post("/personagens/{id}/pericias/atributo/{nome}/{atributo}", s.sheetCommand(swapAttribute))
	r.Post("/personagens/{id}/pericias/remove/{nome}", s.sheetCommand(removeCraft))
	// A CRIAÇÃO é a única que lê SINAL, porque o nome é texto que a pessoa acabou
	// de digitar e ainda não existe em lugar nenhum para virar caminho.
	r.Post("/personagens/{id}/pericias/nova", s.sheetCommand(criaOOficio))
	// OS EFEITOS (fatia 5). Quatro donos de estado, quatro caminhos.
	r.Post("/personagens/{id}/efeitos/condicao/{cond}", s.sheetCommand(toggleBookCondition))
	r.Post("/personagens/{id}/efeitos/aplica/{magia}", s.sheetCommand(applySpellBuff))
	r.Post("/personagens/{id}/efeitos/encerra/{efeito}", s.sheetCommand(endAppliedEffect))
	r.Post("/personagens/{id}/efeitos/postura/{flag}", s.sheetCommand(endStance))
	// O SITUACIONAL manda a CHAVE do condicional, que é um encadeado com `::` e
	// texto livre do catálogo dentro — por isso ela vai por SINAL e não no
	// caminho. É a exceção da ficha, e a razão dela é o formato da chave.
	r.Post("/personagens/{id}/efeitos/situacao", s.sheetCommand(toggleSituational))
	// AS MAGIAS (fatia 6).
	r.Post("/personagens/{id}/magias/aprende/{magia}", s.sheetCommand(learnSpell))
	r.Post("/personagens/{id}/magias/esquece/{magia}", s.sheetCommand(forgetSpell))
	r.Post("/personagens/{id}/magias/prepara/{magia}", s.sheetCommand(togglePrepared))
	r.Post("/personagens/{id}/magias/conjura/{magia}", s.sheetCommand(castSpellFromSheet))
	// A MOCHILA (fatia 7).
	r.Post("/personagens/{id}/itens/{item}/guarda", s.sheetCommand(stowItem))
	r.Post("/personagens/{id}/itens/{item}/equipa/{slot}", s.sheetCommand(equipItemFromSheet))
	// O DINHEIRO manda o modo e o valor por SINAL: a conta do saldo depende dos
	// dois juntos, e um caminho com o valor dentro daria um endereço diferente
	// a cada tecla digitada no campo.
	r.Post("/personagens/{id}/dinheiro", s.sheetCommand(changeMoney))
	r.Post("/personagens/{id}/itens/adiciona/{catalogo}", s.sheetCommand(addCatalogItem))
	r.Post("/personagens/{id}/itens/custom", s.sheetCommand(addCustomItem))
	r.Post("/personagens/{id}/itens/{item}/edita", s.sheetCommand(editItem))
	r.Post("/personagens/{id}/itens/{item}/remove", s.sheetCommand(removeItemFromSheet))
	r.Post("/personagens/{id}/itens/{item}/usa", s.sheetCommand(useItem))
	r.Post("/personagens/{id}/itens/{item}/melhorias", s.sheetCommand(applyOverlays))
	// OS PODERES (fatia 8).
	r.Post("/personagens/{id}/poderes/usa/{poder}", s.sheetCommand(usePower))
	r.Post("/personagens/{id}/poderes/postura/{flag}/entra", s.sheetCommand(enterStance))
	// AS ESCOLHAS do diálogo (fatia 8).
	r.Post("/personagens/{id}/poderes/escolhe/{poder}", s.sheetCommand(pickPower))
	r.Post("/personagens/{id}/poderes/origem/{beneficio}", s.sheetCommand(pickOriginBenefit))
	r.Post("/personagens/{id}/poderes/variante/{variante}", s.sheetCommand(pickRaceVariant))
	r.Post("/personagens/{id}/poderes/classe/{classe}/{escolha}/{valor}", s.sheetCommand(pickClassChoice))
	r.Post("/personagens/{id}/poderes/atributos", s.sheetCommand(pickRaceAttributes))
	r.Post("/personagens/{id}/poderes/ascendencia/{ascendencia}", s.sheetCommand(pickRaceAscendencia))
}

func (s Scene) sheetHandle(w http.ResponseWriter, r *http.Request) {
	id, ok := uRLCharacter(w, r)
	if !ok {
		return
	}
	sinaisDaPagina := sheetSignals(r)
	view, status, err := s.Load(
		r.Context(), s.deps.CurrentUserID(r), id, AskedTab(r.URL.Query().Get("tab")),
		sinaisDaPagina.term(), sinaisDaPagina)
	view.Embutida = insideSessionAsked(r)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	// A FICHA DENTRO DA SESSÃO responde só o pedaço (ALE-272, fatia 10b): quem
	// pediu foi a aba "Minha ficha" da Mesa, e mandar a página inteira faria o
	// Datastar remendar a cena da sessão com uma ficha de corpo inteiro.
	if view.Embutida {
		fragmento, err := ui.RenderFragment(r.Context(), SceneBody(view))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = datastar.NewSSE(w, r).PatchElements(fragmento)
		return
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: view.Nome + " · Tormenta 20",
		// `cascaNua`: a cena desenha o próprio cabeçalho, com a volta e o nome.
		Forma: ui.ShellBare,
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
	}, SceneBody(view))
}

// insideSessionAsked diz se este pedido veio da superfície "Minha ficha" da
// Mesa. A marca viaja na query pela mesma razão que o `?tab=`: o handler
// descobre o que desenhar lendo a requisição, e sinal do cliente sumiria num F5.
func insideSessionAsked(r *http.Request) bool {
	return r.URL.Query().Get("embutida") == "1"
}

// sheetCommand é o gateway das mutações da ficha.
//
// Ele existe pela mesma razão do `comandoDoTabuleiro`: resolver a posse, mutar e
// redesenhar são três passos que toda mutação da ficha faz, e escrevê-los em
// cada handler é como um deles esquece de redesenhar — a pessoa clica, o banco
// muda e a tela fica igual.
//
// A POSSE é conferida aqui, uma vez. A ficha é do dono e de mais ninguém: a
// regra é a mesma da API JSON (`characterFor`), e a cena não ganha uma segunda.
func (s Scene) sheetCommand(
	mutar func(Scene, *http.Request, sqlcgen.Character, Signals) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := uRLCharacter(w, r)
		if !ok {
			return
		}
		// UMA leitura de sinais por requisição, e ela vem ANTES de tudo: o
		// `ReadSignals` consome o corpo do POST, então a segunda chamada
		// receberia vazio sem erro nenhum. Ver `sinais.go`.
		sinais := sheetSignals(r)
		row, err := s.deps.Queries().GetCharacter(r.Context(), id)
		if err != nil {
			http.Error(w, "este personagem não existe", http.StatusNotFound)
			return
		}
		if row.Ownerid != s.deps.CurrentUserID(r) {
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
		} else {
			// A FICHA MUDOU, e quem avisa é o GATEWAY e não cada comando
			// (ALE-275). São mais de trinta mutações passando por aqui, e uma
			// que esquecesse a linha nasceria sem aviso — a mesma forma de
			// defeito que o `characterChanged` já documenta: recurso desligado
			// porque alguém tinha de lembrar de ligá-lo.
			//
			// Só no caminho SEM recusa: uma regra que barrou o gesto não mudou
			// nada, e avisar ali faria toda tela que escuta refazer a ficha por
			// causa de um clique que não aconteceu.
			s.deps.CharacterChanged(row.ID)
		}
		view, status, err := s.Load(
			r.Context(), s.deps.CurrentUserID(r), id, AskedTab(r.URL.Query().Get("tab")), sinais.term(), sinais)
		view.Embutida = insideSessionAsked(r)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		view.Recusa = recusa
		sse := datastar.NewSSE(w, r)
		fragmento, err := ui.RenderFragment(r.Context(), SceneBody(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
	}
}

// touchesVital soma o passo ao PV ou ao PM, PRENDENDO na faixa.
//
// PRENDER e não recusar, e essa é a diferença entre o gesto e a API: o corpo do
// `PATCH /vitals` manda o valor ABSOLUTO e é recusado fora da faixa, o que está
// certo para um cliente que calculou. Aqui o gesto é "levou seis" — com 4 de PV
// o resultado é zero, e uma recusa faria o mestre clicar quatro vezes de um em
// um para chegar no mesmo lugar.
//
// O TETO é o máximo do personagem: curar além do máximo não é PV temporário, que
// é outra regra e tem dono no motor (`TempHpFuria`).
func touchesVital(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	passo, err := uRLStep(r)
	if err != nil {
		return err
	}
	qual := chi.URLParam(r, "qual")
	hp, mp := row.Hpcurrent, row.Mpcurrent
	switch qual {
	case "pv":
		hp = rangePinned(hp+int64(passo), row.Hpmax)
	case "pm":
		mp = rangePinned(mp+int64(passo), row.Mpmax)
	default:
		return fmt.Errorf("vital %q não existe: são 'pv' e 'pm'", qual)
	}
	return s.deps.Queries().SetVitalsCurrent(r.Context(), sqlcgen.SetVitalsCurrentParams{
		HpCurrent: hp, MpCurrent: mp, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	})
}

// rangePinned mantém o vital entre zero e o máximo.
func rangePinned(valor, max int64) int64 {
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
// `applyClassLevel`): a classe tem de ser do personagem, o total é limitado
// a 20, e os pools sincronizam.
func mudaONivel(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	passo, err := uRLStep(r)
	if err != nil {
		return err
	}
	classe, err := url.PathUnescape(chi.URLParam(r, "classe"))
	if err != nil {
		return fmt.Errorf("classe %q não é um nome válido", chi.URLParam(r, "classe"))
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
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
		return s.deps.ApplyClassLevel(r, row.ID, classe, alvo)
	}
	return fmt.Errorf("%s não é uma classe deste personagem", classe)
}

// uRLCharacter lê o id do caminho. Erro aqui é URL digitada errada, e a
// resposta é uma frase: quem está do outro lado é um navegador mostrando página.
func uRLCharacter(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "o personagem precisa ser um número", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// uRLStep aceita o sinal de menos: o passo é para os dois lados.
func uRLStep(r *http.Request) (int, error) {
	bruto := chi.URLParam(r, "passo")
	passo, err := strconv.Atoi(bruto)
	if err != nil || passo == 0 {
		return 0, fmt.Errorf("passo %q não é um número diferente de zero", bruto)
	}
	return passo, nil
}
