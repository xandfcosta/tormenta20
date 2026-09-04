package sheetui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// OS COMANDOS DAS ABAS PERÍCIAS E PROFICIÊNCIAS (ALE-272, fatias 2 e 4).

// oNomeDaPericia lê o nome do caminho, desescapando como a API JSON faz.
func expertiseName(r *http.Request) string {
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
func toggleTraining(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	nome := expertiseName(r)
	// O estado ATUAL vem da lista e não do `GetExpertiseMeta`, que devolve só o
	// id e o `custom` — inverter exige saber o que está lá.
	todas, err := s.deps.Queries().ListExpertisesByCharacter(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, e := range todas {
		if e.Name != nome {
			continue
		}
		depois := e.Trained == 0
		_, err := s.deps.Queries().UpdateExpertise(r.Context(), sqlcgen.UpdateExpertiseParams{
			Trained: plataforma.NullBool(&depois), CharacterId: row.ID, Name: nome,
		})
		return err
	}
	return fmt.Errorf("a perícia %q não é desta ficha", nome)
}

// trocaOAtributo repõe a perícia em outro atributo.
//
// O atributo vai no CAMINHO junto do nome: é o valor do `<option>` escolhido, e
// mandá-lo por sinal faria seis opções de 29 linhas disputarem a mesma chave.
func swapAttribute(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	atributo := chi.URLParam(r, "atributo")
	if !engine.IsAttributeKey(atributo) {
		return fmt.Errorf("%q não é um atributo: são %v", atributo, engine.AttributeKeys)
	}
	nome := expertiseName(r)
	_, err := s.deps.Queries().UpdateExpertise(r.Context(), sqlcgen.UpdateExpertiseParams{
		Attribute: plataforma.NullString(&atributo), CharacterId: row.ID, Name: nome,
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
func removeCraft(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	nome := expertiseName(r)
	meta, err := s.deps.Queries().GetExpertiseMeta(r.Context(), sqlcgen.GetExpertiseMetaParams{
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
	return s.deps.Queries().DeleteExpertiseByID(r.Context(), meta.ID)
}

// criaOOficio acrescenta uma perícia que o livro não tem — o saber de um ferreiro,
// a arte de um marinheiro.
//
// Ela nasce TREINADA, porque inventar um ofício e não tê-lo treinado não é um
// estado que signifique alguma coisa. A validação é a MESMA da API JSON
// (`saveNewCraft`), extraída na fatia 4: duas validações divergiriam no dia
// em que uma regra nova chegasse, e a esquecida aceitaria o que a outra recusa.
func criaOOficio(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	nome, atributo := "", "intelligence"
	if sinais.NovaPericia != nil {
		nome = strings.TrimSpace(*sinais.NovaPericia)
	}
	if sinais.NovoAtributo != nil && engine.IsAttributeKey(*sinais.NovoAtributo) {
		atributo = *sinais.NovoAtributo
	}
	if err := s.deps.SaveNewCraft(r.Context(), row.ID, nome); err != nil {
		return err
	}
	_, err := s.deps.Queries().CreateExpertise(r.Context(), sqlcgen.CreateExpertiseParams{
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
func toggleProficiency(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	depois, err := proficiencySwap(dto, chi.URLParam(r, "categoria"))
	if err != nil {
		return err
	}
	return s.saveTheProficienciesSheet(r, row.ID, depois)
}

// restauraOPadraoDaClasse joga fora os ajustes manuais.
func restoresDefaultClass(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.saveTheProficienciesSheet(r, row.ID, classDefault(dto))
}

// gravaAsProficienciasDaFicha usa a MESMA gravação da API JSON.
//
// A lista de desconhecidas vira frase porque quem está do outro lado é um
// navegador mostrando página, e não um cliente lendo `FieldErrorMap`. Ela só
// dispara se o servidor montar uma categoria que ele próprio não conhece — é o
// guarda contra a tela e a validação divergirem, não contra o jogador.
func (s Scene) saveTheProficienciesSheet(r *http.Request, id int64, categorias []string) error {
	_, desconhecidas, err := s.deps.SaveProficiencies(r.Context(), id, categorias)
	if len(desconhecidas) > 0 {
		return fmt.Errorf("proficiência fora do catálogo: %s", strings.Join(desconhecidas, "; "))
	}
	return err
}
