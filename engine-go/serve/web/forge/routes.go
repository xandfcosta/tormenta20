package forge

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
	"t20engine/domain/sheet"
	"t20engine/serve/web/ui"
)

// AS ROTAS DA FORJA.
//
// Todas leem o MESMO formulário: desenhar a folha, redesenhá-la quando uma escolha
// muda, mover o nível de uma classe, e forjar. Nenhuma delas guarda rascunho —
// enquanto o herói não nasce, quem carrega as respostas é o próprio formulário
// no navegador, e depois que ele nasce quem carrega é o banco.

// handleForge desenha a folha em branco.
func (s Scene) handleForge(w http.ResponseWriter, r *http.Request) {
	s.writeForge(w, r, http.StatusOK, blankForgeSheet(forgeAnswers{}, nil))
}

// handleForgeDraft redesenha a folha com o que já foi respondido.
//
// Ele existe por uma razão só: o equipamento de p140 e a lista de deuses
// dependem das classes, e essas regras ficam no servidor. A
// resposta é 200 com a cena inteira — inclusive quando há campo em branco —,
// porque remendo de resposta que não é 2xx o Datastar DESCARTA, e a folha
// ficaria parada sem uma palavra na tela.
func (s Scene) handleForgeDraft(w http.ResponseWriter, r *http.Request) {
	answers, err := answersFromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redrawForge(w, r, answers)
}

// handleForgeLevelStep é o `−` e o `+` de uma classe. Só redesenha: o nível mora no formulário, e o
// mesmo desenho de sempre devolve a soma, o kit da nova primeira classe e a lista de deuses.
func (s Scene) handleForgeLevelStep(w http.ResponseWriter, r *http.Request) {
	answers, err := answersFromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	class, err := url.PathUnescape(chi.URLParam(r, "classe"))
	if err != nil {
		http.Error(w, fmt.Sprintf("classe %q não é um nome válido", chi.URLParam(r, "classe")), http.StatusBadRequest)
		return
	}
	step, err := strconv.ParseInt(chi.URLParam(r, "passo"), 10, 64)
	if err != nil || (step != 1 && step != -1) {
		http.Error(w, fmt.Sprintf("o passo é 1 ou -1, e veio %q", chi.URLParam(r, "passo")), http.StatusBadRequest)
		return
	}
	answers.Classes = steppedClasses(answers.Classes, class, step)
	redrawForge(w, r, answers)
}

func redrawForge(w http.ResponseWriter, r *http.Request, answers forgeAnswers) {
	fragment, err := ui.RenderFragment(r.Context(), forgeBody(blankForgeSheet(answers, nil)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = datastar.NewSSE(w, r).PatchElements(fragment)
}

// handleForgePost cria o herói e leva para a distribuição de atributos.
//
// A recusa REDESENHA a folha com o que foi respondido e o erro no campo, em 422
// — a mesma decisão da folha em branco da campanha. Aqui o 422 é
// seguro (e a recusa 200 do Datastar não se aplica) porque este caminho é o
// `submit` de um formulário de verdade: quem desenha a resposta é o navegador,
// não um remendo.
func (s Scene) handleForgePost(w http.ResponseWriter, r *http.Request) {
	answers, err := answersFromForm(r)
	if err != nil {
		s.writeForge(w, r, http.StatusBadRequest, blankForgeSheet(forgeAnswers{}, nil))
		return
	}
	if errs := forgeRefusals(answers); len(errs) > 0 {
		s.writeForge(w, r, http.StatusUnprocessableEntity, blankForgeSheet(answers, errs))
		return
	}
	id, err := s.birthHero(r, s.deps.CurrentUserID(r), answers)
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
	classes, broken := classLinesRead(r.PostForm["classes"])
	return forgeAnswers{
		Name:             r.PostFormValue("name"),
		Race:             r.PostFormValue("race"),
		Classes:          withNewClass(classes, r.PostFormValue("class")),
		BrokenClassLines: broken,
		Origin:           r.PostFormValue("origin"),
		SimpleWeapon:     r.PostFormValue("weaponSimple"),
		MartialWeapon:    r.PostFormValue("weaponMartial"),
		Armor:            r.PostFormValue("armor"),
		Shield:           r.PostFormValue("shield") != "",
	}, nil
}

// classLinesRead lê as linhas `Classe:nível`, na ordem do formulário. A ilegível volta à parte, para a
// recusa citá-la em vez de ela sumir em silêncio.
func classLinesRead(lines []string) (classes []sheet.ClassEntry, broken []string) {
	for _, line := range lines {
		name, written, found := strings.Cut(line, ":")
		level, err := strconv.ParseInt(written, 10, 64)
		if !found || err != nil || level < 1 || level > heroLevelMax {
			broken = append(broken, line)
			continue
		}
		classes = append(classes, sheet.ClassEntry{ClassName: name, Level: level})
	}
	return classes, broken
}

// withNewClass põe a classe que o rádio acabou de escolher no fim da lista, no 1º nível. O rádio some da
// folha assim que ela entra, então ele nunca repete uma classe sozinho.
func withNewClass(classes []sheet.ClassEntry, class string) []sheet.ClassEntry {
	if class == "" || slices.ContainsFunc(classes, func(c sheet.ClassEntry) bool { return c.ClassName == class }) {
		return classes
	}
	return append(classes, sheet.ClassEntry{ClassName: class, Level: 1})
}

// steppedClasses move um nível de uma classe. O `−` no 1º nível tira a classe, e é assim que se troca a
// primeira; o `+` no 20º nível do herói não faz nada, porque o botão já vem apagado.
func steppedClasses(classes []sheet.ClassEntry, class string, step int64) []sheet.ClassEntry {
	if step > 0 && totalLevelOf(classes) >= heroLevelMax {
		return classes
	}
	moved := make([]sheet.ClassEntry, 0, len(classes))
	for _, entry := range classes {
		if entry.ClassName == class {
			entry.Level += step
		}
		if entry.Level > 0 {
			moved = append(moved, entry)
		}
	}
	return moved
}

func (s Scene) writeForge(w http.ResponseWriter, r *http.Request, status int, v forgeView) {
	s.deps.WritePage(w, r, status, ui.Page{
		Title: "Forja · Tormenta 20",
		// `ui.ShellDense`: o cabeçalho compacto com a seta de voltar, como a
		// campanha nova. Sem ele a folha nasce sem saída visível.
		Shape:        ui.ShellDense,
		Back:         "/personagens",
		BackLabel:    "Personagens",
		VisibleTitle: "Forjar um herói",
	}, forgeSheet(v))
}

// Routes monta a forja no roteador de quem a hospeda.
//
// Os endereços moram AQUI e não em quem monta: a cena é a dona do que
// ela atende, e quem a hospeda escolhe só onde encaixá-la. As três primeiras
// rotas são a folha em branco; as duas últimas vivem sob o id porque o herói JÁ
// existe — o nascimento é o `POST /personagens/nova`, e daqui em diante tudo é
// comando sobre uma linha do banco.
func Routes(r chi.Router, s Scene) {
	r.Get("/personagens/nova", s.handleForge)
	r.Post("/personagens/nova", s.handleForgePost)
	r.Post("/personagens/nova/esboco", s.handleForgeDraft)
	r.Post("/personagens/nova/esboco/nivel/{classe}/{passo}", s.handleForgeLevelStep)
	r.Get("/personagens/{id}/atributos", s.handleAttributes)
	r.Post("/personagens/{id}/atributos/{atributo}/{passo}", s.handleAttributeStep)
}
