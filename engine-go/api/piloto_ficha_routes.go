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
}

func (s *Server) handleFicha(w http.ResponseWriter, r *http.Request) {
	id, ok := oPersonagemDaURL(w, r)
	if !ok {
		return
	}
	view, status, err := s.carregaFicha(r.Context(), currentUser(r), id, aAbaPedida(r.URL.Query().Get("tab")))
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo: view.Nome + " · Tormenta 20",
		// `cascaNua`: a cena desenha o próprio cabeçalho, com a volta e o nome.
		Forma: cascaNua,
	}, cenaDaFicha(view))
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
	mutar func(*Server, *http.Request, sqlcgen.Character) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := oPersonagemDaURL(w, r)
		if !ok {
			return
		}
		row, err := s.queries.GetCharacter(r.Context(), id)
		if err != nil {
			http.Error(w, "este personagem não existe", http.StatusNotFound)
			return
		}
		if row.Ownerid != currentUser(r).ID {
			http.Error(w, "esta ficha não é sua", http.StatusForbidden)
			return
		}
		if err := mutar(s, r, row); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, status, err := s.carregaFicha(
			r.Context(), currentUser(r), id, aAbaPedida(r.URL.Query().Get("tab")))
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
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
func mexeNoVital(s *Server, r *http.Request, row sqlcgen.Character) error {
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
func mudaONivel(s *Server, r *http.Request, row sqlcgen.Character) error {
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
