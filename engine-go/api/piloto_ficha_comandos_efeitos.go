package api

import (
	"fmt"
	"net/http"
	"strconv"
	"t20engine/sheet"

	"github.com/go-chi/chi/v5"

	"t20engine/catalog"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// OS COMANDOS DA ABA EFEITOS (ALE-272, fatia 5).

// toggleBookCondition liga ou desliga UMA condição do livro (p394-395).
//
// # Ela avisa a MESA, e isso não é enfeite
//
// O motor deriva Defesa e perícias da condição (ALE-28), então uma condição
// aplicada sem aviso faz o jogador e o mestre verem números DIFERENTES do mesmo
// personagem, sem nada na tela dizendo que discordam. Foi o defeito da ALE-245,
// e o `handleUpdateConditions` da API JSON é o único lugar que o conserta — a
// ficha em Datastar tinha de conquistar o mesmo, senão o porte REGREDIRIA.
//
// O aviso sai DEPOIS da escrita, nunca antes: avisar sobre algo que ainda pode
// falhar faria a mesa buscar o estado velho e acreditar nele.
func toggleBookCondition(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	cond := chi.URLParam(r, "cond")
	if !catalog.IsCondition(cond) {
		return fmt.Errorf("%q não é uma condição do livro", cond)
	}
	atuais := parseConditionBlob(row.Activeconditions)
	depois := []string{}
	tinha := false
	for _, c := range atuais {
		if c == cond {
			tinha = true
			continue
		}
		depois = append(depois, c)
	}
	if !tinha {
		depois = append(depois, cond)
	}
	blob := marshalStrings(&depois)
	if err := s.queries.UpdateConditions(r.Context(), sqlcgen.UpdateConditionsParams{
		ActiveConditions: blob, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	}); err != nil {
		return err
	}
	s.characterChanged(row.ID)
	return nil
}

// applySpellBuff aplica uma magia de bônus como efeito de cena ou dia.
//
// A gravação é a MESMA da API JSON (`applySpellBuffEffect`): duas escritas
// divergiriam no dia em que uma regra nova chegasse, e o escopo padrão de cada
// magia vive no catálogo, não aqui.
func applySpellBuff(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	magia := chi.URLParam(r, "magia")
	if _, _, err := s.applySpellBuffEffect(r.Context(), row.ID, magia, nil); err != nil {
		return err
	}
	return nil
}

// endAppliedEffect encerra um efeito em curso.
func endAppliedEffect(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	id, err := strconv.ParseInt(chi.URLParam(r, "efeito"), 10, 64)
	if err != nil {
		return fmt.Errorf("o efeito %q não é um número", chi.URLParam(r, "efeito"))
	}
	// A POSSE É CONFERIDA ANTES, e a query não a confere por nós: o
	// `DeleteEffectByID` apaga por id e mais nada, então sem esta leitura um
	// pedido montado à mão encerraria o efeito de OUTRO personagem. É a mesma
	// checagem que o `handleDeleteEffect` da API JSON faz.
	meta, err := s.queries.GetActiveEffectMeta(r.Context(), id)
	if err != nil || meta.Characterid != row.ID {
		return fmt.Errorf("o efeito %d não é desta ficha", id)
	}
	return s.queries.DeleteEffectByID(r.Context(), id)
}

// endStance encerra uma postura.
//
// Encerrar apaga a linha da postura E desliga a flag: são duas escritas para uma
// coisa só, e deixar a flag ligada manteria os modificadores em pé numa postura
// que a tela diz encerrada. Entrar continua sendo dos Poderes, onde o PM é
// cobrado — aqui não há como pagar nada.
func endStance(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	flag := chi.URLParam(r, "flag")
	if err := s.queries.RemoveCharacterStance(r.Context(), sqlcgen.RemoveCharacterStanceParams{
		Characterid: row.ID, Flag: flag,
	}); err != nil {
		return err
	}
	// E O QUE A POSTURA CONCEDEU sai junto (fatia 8): a reserva de PV
	// temporários da Alma de Bronze dura "enquanto a Fúria durar" (p41), e
	// deixá-la para trás daria PV que a postura encerrada continua pagando.
	if err := s.removeAsConcessoesDaPostura(r, row, flag); err != nil {
		return err
	}
	return s.removeConditionalsWithFlag(r, row, flag)
}

// toggleSituational liga ou desliga um condicional de contexto.
func toggleSituational(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	chave := ""
	if sinais.Situacao != nil {
		chave = *sinais.Situacao
	}
	if chave == "" {
		return fmt.Errorf("o gesto não disse qual efeito situacional alternar")
	}
	atuais, err := s.queries.ListCharacterConditionals(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, c := range atuais {
		if c == chave {
			return s.queries.RemoveCharacterConditional(r.Context(), sqlcgen.RemoveCharacterConditionalParams{
				Characterid: row.ID, Conditionalid: chave,
			})
		}
	}
	return s.queries.AddCharacterConditional(r.Context(), sqlcgen.AddCharacterConditionalParams{
		Characterid: row.ID, Conditionalid: chave,
	})
}

// removeConditionalsWithFlag desliga todo condicional que a postura acendia.
func (s *Server) removeConditionalsWithFlag(r *http.Request, row sqlcgen.Character, flag string) error {
	dto, err := s.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil || s.catalogs == nil {
		return nil
	}
	for _, c := range engine.ComputeItemEffects(s.catalogs.ActiveItemsFor(ec)).Conditional {
		if c.Flag != flag {
			continue
		}
		_ = s.queries.RemoveCharacterConditional(r.Context(), sqlcgen.RemoveCharacterConditionalParams{
			Characterid: row.ID, Conditionalid: engine.ConditionalID(c),
		})
	}
	return nil
}
