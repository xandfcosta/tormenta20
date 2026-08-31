package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// OS COMANDOS DAS ABAS PERÍCIAS E PROFICIÊNCIAS (ALE-272, fatias 2 e 4).

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
