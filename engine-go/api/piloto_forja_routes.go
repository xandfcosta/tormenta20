package api

import (
	"net/http"
	"strconv"

	"github.com/starfederation/datastar-go/datastar"
)

// AS ROTAS DA FORJA (ALE-272, fatia 9).
//
// São três, e as três leem o MESMO formulário: desenhar a folha, redesenhá-la
// quando a classe ou a origem muda, e forjar. Nenhuma delas guarda rascunho —
// enquanto o herói não nasce, quem carrega as respostas é o próprio formulário
// no navegador, e depois que ele nasce quem carrega é o banco.

// handleForja desenha a folha em branco.
func (s *Server) handleForja(w http.ResponseWriter, r *http.Request) {
	s.escreveAForja(w, r, http.StatusOK, blankForgeSheet(forgeAnswers{}, nil))
}

// handleForjaEsboco redesenha a folha com o que já foi respondido.
//
// É o único pedido do Datastar desta cena, e ele existe por uma razão só: o
// equipamento de p140 depende da classe, e essa regra fica no servidor. A
// resposta é 200 com a cena inteira — inclusive quando há campo em branco —,
// porque remendo de resposta que não é 2xx o Datastar DESCARTA, e a folha
// ficaria parada sem uma palavra na tela.
func (s *Server) handleForjaEsboco(w http.ResponseWriter, r *http.Request) {
	folha, err := aForjaDoFormulario(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fragmento, err := renderFragmento(r.Context(), forgeBody(blankForgeSheet(folha, nil)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = datastar.NewSSE(w, r).PatchElements(fragmento)
}

// handleForjaPost cria o herói e leva para a distribuição de atributos.
//
// A recusa REDESENHA a folha com o que foi respondido e o erro no campo, em 422
// — a mesma decisão da folha em branco da campanha (ALE-246). Aqui o 422 é
// seguro (e a recusa 200 do Datastar não se aplica) porque este caminho é o
// `submit` de um formulário de verdade: quem desenha a resposta é o navegador,
// não um remendo.
func (s *Server) handleForjaPost(w http.ResponseWriter, r *http.Request) {
	folha, err := aForjaDoFormulario(r)
	if err != nil {
		s.escreveAForja(w, r, http.StatusBadRequest, blankForgeSheet(forgeAnswers{}, nil))
		return
	}
	if erros := forgeRefusals(folha); len(erros) > 0 {
		s.escreveAForja(w, r, http.StatusUnprocessableEntity, blankForgeSheet(folha, erros))
		return
	}
	id, err := s.birth(r, currentUser(r).ID, folha)
	if err != nil {
		http.Error(w, "não foi possível forjar o herói", http.StatusInternalServerError)
		return
	}
	// 303 e não 302: depois de um POST, o `See Other` é o que garante que o
	// navegador siga com GET — sem ele, recarregar a cena dos atributos forjaria
	// um segundo herói igual.
	http.Redirect(w, r, "/personagens/"+strconv.FormatInt(id, 10)+"/atributos", http.StatusSeeOther)
}

// aForjaDoFormulario lê as respostas do formulário.
//
// Serve os dois caminhos — o `submit` do navegador e o `@post` do Datastar com
// `contentType: 'form'` — porque os dois mandam `application/x-www-form-urlencoded`.
// É por isso que esta cena não tem sinal nenhum: o formulário É o estado.
func aForjaDoFormulario(r *http.Request) (forgeAnswers, error) {
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

func (s *Server) escreveAForja(w http.ResponseWriter, r *http.Request, status int, v forgeView) {
	s.escrevePagina(w, r, status, paginaPiloto{
		Titulo: "Forja · Tormenta 20",
		// `cascaDensa`: o cabeçalho compacto com o "‹ Voltar", como a folha em
		// branco da campanha. Sem ele a folha nasce sem saída visível.
		Forma:        cascaDensa,
		Voltar:       "/personagens",
		VoltarRotulo: "Personagens",
	}, forgeSheet(v))
}
