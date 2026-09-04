package forge

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"

	"github.com/starfederation/datastar-go/datastar"
	"t20engine/web/ui"
)

// AS ROTAS DA FORJA (ALE-272, fatia 9).
//
// São três, e as três leem o MESMO formulário: desenhar a folha, redesenhá-la
// quando a classe ou a origem muda, e forjar. Nenhuma delas guarda rascunho —
// enquanto o herói não nasce, quem carrega as respostas é o próprio formulário
// no navegador, e depois que ele nasce quem carrega é o banco.

// handleForge desenha a folha em branco.
func (s Scene) handleForge(w http.ResponseWriter, r *http.Request) {
	s.writeForge(w, r, http.StatusOK, blankForgeSheet(forgeAnswers{}, nil))
}

// handleForgeDraft redesenha a folha com o que já foi respondido.
//
// É o único pedido do Datastar desta cena, e ele existe por uma razão só: o
// equipamento de p140 depende da classe, e essa regra fica no servidor. A
// resposta é 200 com a cena inteira — inclusive quando há campo em branco —,
// porque remendo de resposta que não é 2xx o Datastar DESCARTA, e a folha
// ficaria parada sem uma palavra na tela.
func (s Scene) handleForgeDraft(w http.ResponseWriter, r *http.Request) {
	folha, err := answersFromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fragmento, err := ui.RenderFragment(r.Context(), forgeBody(blankForgeSheet(folha, nil)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = datastar.NewSSE(w, r).PatchElements(fragmento)
}

// handleForgePost cria o herói e leva para a distribuição de atributos.
//
// A recusa REDESENHA a folha com o que foi respondido e o erro no campo, em 422
// — a mesma decisão da folha em branco da campanha (ALE-246). Aqui o 422 é
// seguro (e a recusa 200 do Datastar não se aplica) porque este caminho é o
// `submit` de um formulário de verdade: quem desenha a resposta é o navegador,
// não um remendo.
func (s Scene) handleForgePost(w http.ResponseWriter, r *http.Request) {
	folha, err := answersFromForm(r)
	if err != nil {
		s.writeForge(w, r, http.StatusBadRequest, blankForgeSheet(forgeAnswers{}, nil))
		return
	}
	if erros := forgeRefusals(folha); len(erros) > 0 {
		s.writeForge(w, r, http.StatusUnprocessableEntity, blankForgeSheet(folha, erros))
		return
	}
	id, err := s.birthHero(r, s.deps.CurrentUserID(r), folha)
	if err != nil {
		http.Error(w, "não foi possível forjar o herói", http.StatusInternalServerError)
		return
	}
	// 303 e não 302: depois de um POST, o `See Other` é o que garante que o
	// navegador siga com GET — sem ele, recarregar a cena dos atributos forjaria
	// um segundo herói igual.
	http.Redirect(w, r, "/personagens/"+strconv.FormatInt(id, 10)+"/atributos", http.StatusSeeOther)
}

// answersFromForm lê as respostas do formulário.
//
// Serve os dois caminhos — o `submit` do navegador e o `@post` do Datastar com
// `contentType: 'form'` — porque os dois mandam `application/x-www-form-urlencoded`.
// É por isso que esta cena não tem sinal nenhum: o formulário É o estado.
func answersFromForm(r *http.Request) (forgeAnswers, error) {
	if err := r.ParseForm(); err != nil {
		return forgeAnswers{}, err
	}
	return forgeAnswers{
		Name:          r.PostFormValue("name"),
		Race:          r.PostFormValue("race"),
		Class:         r.PostFormValue("class"),
		Origin:        r.PostFormValue("origin"),
		SimpleWeapon:  r.PostFormValue("weaponSimple"),
		MartialWeapon: r.PostFormValue("weaponMartial"),
		Armor:         r.PostFormValue("armor"),
		Shield:        r.PostFormValue("shield") != "",
	}, nil
}

func (s Scene) writeForge(w http.ResponseWriter, r *http.Request, status int, v forgeView) {
	s.deps.WritePage(w, r, status, ui.Page{
		Titulo: "Forja · Tormenta 20",
		// `ui.ShellDense`: o cabeçalho compacto com o "‹ Voltar", como a folha em
		// branco da campanha. Sem ele a folha nasce sem saída visível.
		Forma:        ui.ShellDense,
		Voltar:       "/personagens",
		VoltarRotulo: "Personagens",
	}, forgeSheet(v))
}

// Routes monta a forja no roteador de quem a hospeda.
//
// Os endereços moram AQUI e não em quem monta (ALE-278): a cena é a dona do que
// ela atende, e quem a hospeda escolhe só onde encaixá-la. As três primeiras
// rotas são a folha em branco; as duas últimas vivem sob o id porque o herói JÁ
// existe — o nascimento é o `POST /personagens/nova`, e daqui em diante tudo é
// comando sobre uma linha do banco.
func Routes(r chi.Router, s Scene) {
	r.Get("/personagens/nova", s.handleForge)
	r.Post("/personagens/nova", s.handleForgePost)
	r.Post("/personagens/nova/esboco", s.handleForgeDraft)
	r.Get("/personagens/{id}/atributos", s.handleAttributes)
	r.Post("/personagens/{id}/atributos/{atributo}/{passo}", s.handleAttributeStep)
}
