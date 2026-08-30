package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
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
}

// oNomeDaPericia lê o nome do caminho, desescapando como a API JSON faz.
func oNomeDaPericia(r *http.Request) string {
	nome := chi.URLParam(r, "nome")
	if decodificado, err := url.PathUnescape(nome); err == nil {
		return decodificado
	}
	return nome
}

// alternaOTreino liga ou desliga o treino de UMA perícia.
//
// O comando manda a PERÍCIA e não o estado desejado, pela mesma razão da
// proficiência: mandar "treinada" perde para o clique repetido e para a segunda
// aba aberta no mesmo personagem. Quem clica quer INVERTER o que está na tela, e
// o servidor sabe o que está na tela melhor que o botão.
func alternaOTreino(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	nome := oNomeDaPericia(r)
	// O estado ATUAL vem da lista e não do `GetExpertiseMeta`, que devolve só o
	// id e o `custom` — inverter exige saber o que está lá.
	todas, err := s.queries.ListExpertisesByCharacter(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, e := range todas {
		if e.Name != nome {
			continue
		}
		depois := e.Trained == 0
		_, err := s.queries.UpdateExpertise(r.Context(), sqlcgen.UpdateExpertiseParams{
			Trained: nullBool(&depois), CharacterId: row.ID, Name: nome,
		})
		return err
	}
	return fmt.Errorf("a perícia %q não é desta ficha", nome)
}

// trocaOAtributo repõe a perícia em outro atributo.
//
// O atributo vai no CAMINHO junto do nome: é o valor do `<option>` escolhido, e
// mandá-lo por sinal faria seis opções de 29 linhas disputarem a mesma chave.
func trocaOAtributo(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	atributo := chi.URLParam(r, "atributo")
	if !attributeKeys[atributo] {
		return fmt.Errorf("%q não é um atributo: são %v", atributo, engine.AttributeKeys)
	}
	nome := oNomeDaPericia(r)
	_, err := s.queries.UpdateExpertise(r.Context(), sqlcgen.UpdateExpertiseParams{
		Attribute: nullString(&atributo), CharacterId: row.ID, Name: nome,
	})
	if err != nil {
		return fmt.Errorf("a perícia %q não é desta ficha", nome)
	}
	return nil
}

// removeOOficio apaga uma perícia INVENTADA pelo jogador.
//
// As 29 do livro não se apagam, e a recusa é do servidor e não da tela: a ficha
// nova não desenha a lixeira numa perícia do livro, mas travar só na UI deixaria
// a regra sem fronteira — quem montar o `@post` à mão apagaria a Fortitude.
func removeOOficio(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	nome := oNomeDaPericia(r)
	meta, err := s.queries.GetExpertiseMeta(r.Context(), sqlcgen.GetExpertiseMetaParams{
		Characterid: row.ID, Name: nome,
	})
	if err != nil {
		return fmt.Errorf("a perícia %q não é desta ficha", nome)
	}
	// A COLUNA decide, e não a lista das 29: `custom` é o que o banco guarda
	// sobre esta linha, enquanto a lista é uma opinião do código sobre o nome. As
	// duas concordam hoje; no dia em que uma perícia nova entrar no livro, a
	// coluna continua certa e a lista fica velha.
	if meta.Custom == 0 {
		return fmt.Errorf("%q é uma perícia do livro e não se remove da ficha", nome)
	}
	return s.queries.DeleteExpertiseByID(r.Context(), meta.ID)
}

// criaOOficio acrescenta uma perícia que o livro não tem — o saber de um ferreiro,
// a arte de um marinheiro.
//
// Ela nasce TREINADA, porque inventar um ofício e não tê-lo treinado não é um
// estado que signifique alguma coisa. A validação é a MESMA da API JSON
// (`guardaOOficioNovo`), extraída na fatia 4: duas validações divergiriam no dia
// em que uma regra nova chegasse, e a esquecida aceitaria o que a outra recusa.
func criaOOficio(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	nome, atributo := "", "intelligence"
	if sinais.NovaPericia != nil {
		nome = strings.TrimSpace(*sinais.NovaPericia)
	}
	if sinais.NovoAtributo != nil && attributeKeys[*sinais.NovoAtributo] {
		atributo = *sinais.NovoAtributo
	}
	if err := s.guardaOOficioNovo(r.Context(), row.ID, nome); err != nil {
		return err
	}
	_, err := s.queries.CreateExpertise(r.Context(), sqlcgen.CreateExpertiseParams{
		Characterid: row.ID, Name: nome, Attribute: atributo, Trained: 1, Custom: 1,
	})
	return err
}

// alternaAProficiencia liga ou desliga UMA categoria.
//
// O comando não manda o estado desejado, manda a categoria: mandar "ligada"
// perderia para o clique repetido e para a segunda aba aberta no mesmo
// personagem — quem clica quer INVERTER o que está na tela, e o servidor sabe o
// que está na tela melhor do que o botão sabe.
func alternaAProficiencia(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	depois, err := aTrocaDaProficiencia(dto, chi.URLParam(r, "categoria"))
	if err != nil {
		return err
	}
	return s.gravaAsProficienciasDaFicha(r, row.ID, depois)
}

// restauraOPadraoDaClasse joga fora os ajustes manuais.
func restauraOPadraoDaClasse(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.gravaAsProficienciasDaFicha(r, row.ID, oPadraoDaClasse(dto))
}

// gravaAsProficienciasDaFicha usa a MESMA gravação da API JSON.
//
// A lista de desconhecidas vira frase porque quem está do outro lado é um
// navegador mostrando página, e não um cliente lendo `FieldErrorMap`. Ela só
// dispara se o servidor montar uma categoria que ele próprio não conhece — é o
// guarda contra a tela e a validação divergirem, não contra o jogador.
func (s *Server) gravaAsProficienciasDaFicha(r *http.Request, id int64, categorias []string) error {
	_, desconhecidas, err := s.guardaAsProficiencias(r.Context(), id, categorias)
	if len(desconhecidas) > 0 {
		return fmt.Errorf("proficiência fora do catálogo: %s", strings.Join(desconhecidas, "; "))
	}
	return err
}

func (s *Server) handleFicha(w http.ResponseWriter, r *http.Request) {
	id, ok := oPersonagemDaURL(w, r)
	if !ok {
		return
	}
	view, status, err := s.carregaFicha(
		r.Context(), currentUser(r), id, aAbaPedida(r.URL.Query().Get("tab")),
		osSinaisDaFicha(r).aBusca())
	if err != nil {
		http.Error(w, err.Error(), status)
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
		Sinais: "{detalhe: '', oficio: false, novapericia: '', novoatributo: 'intelligence'}",
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
		if err := mutar(s, r, row, sinais); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, status, err := s.carregaFicha(
			r.Context(), currentUser(r), id, aAbaPedida(r.URL.Query().Get("tab")), sinais.aBusca())
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
