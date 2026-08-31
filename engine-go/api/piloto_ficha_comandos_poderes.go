package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// OS COMANDOS DA ABA PODERES (ALE-272, fatia 8).
//
// Usar um poder e entrar numa postura são as duas escritas que a mesa faz nesta
// aba; encerrar mora nos Efeitos, onde a postura em curso aparece.

// usePower gasta um uso de um poder instantâneo: cobra o PM e soma o contador.
//
// As duas escritas são de coisas diferentes — o PM é da ficha, o contador é do
// estado de jogo — e a ordem importa: o PM primeiro, porque é ele que pode
// faltar. Somar o uso antes deixaria um uso gasto por um poder que não saiu.
func usePower(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	spec := aAtivacaoDe(chi.URLParam(r, "poder"), "")
	if spec == nil {
		return fmt.Errorf("o poder %q não tem ativação no catálogo", chi.URLParam(r, "poder"))
	}
	if spec.Kind != "instant" {
		return fmt.Errorf("%q não é um poder de usar", spec.Name)
	}
	usos := osUsosPorPoder(dto)[spec.ID]
	pode, porque := aDecisaoDoUso(*spec, contextoDoUso{
		PmAtual: int(dto.MpCurrent), UsadoNaCena: usos.Cena, UsadoNoDia: usos.Dia,
		Flags: s.asFlagsAtivas(dto),
	})
	if !pode {
		return fmt.Errorf("%s: %s", spec.Name, porque)
	}
	if err := s.cobraOPm(r, row, oPmDaAtivacao(*spec)); err != nil {
		return err
	}
	escopo := oEscopoCobrado(*spec)
	if escopo == "" {
		return nil
	}
	return s.queries.BumpCharacterPowerUse(r.Context(), sqlcgen.BumpCharacterPowerUseParams{
		Characterid: row.ID, Powerid: spec.ID, Scope: escopo,
	})
}

// enterStance entra numa postura, com os degraus escolhidos.
//
// São QUATRO escritas para um gesto: o PM sai, o pagamento é registrado, os
// condicionais da flag sobem, e o que a postura concede vira efeito. O
// pagamento é registrado para sair não devolver PM — é o que a tabela
// `character_stances` existe para lembrar (ALE-222).
func enterStance(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	flag := chi.URLParam(r, "flag")
	spec := aPosturaDaFlag(flag)
	if spec == nil {
		return fmt.Errorf("%q não é uma postura do livro", flag)
	}
	degraus := 0
	if sinais.PoderDegraus != nil {
		degraus = int(*sinais.PoderDegraus)
	}
	maximo := 0
	if spec.Scaling != nil {
		maximo = osDegrausDoNivel(*spec.Scaling, oNivelNaClasseDoPoder(dto, spec.ID))
	}
	pode, porque := aDecisaoDaPostura(*spec, degraus, maximo, int(dto.MpCurrent))
	if !pode {
		return fmt.Errorf("%s: %s", spec.Name, porque)
	}
	custo := oCustoDaPostura(*spec, degraus)
	if err := s.cobraOPm(r, row, custo); err != nil {
		return err
	}
	if err := s.queries.UpsertCharacterStance(r.Context(), sqlcgen.UpsertCharacterStanceParams{
		Characterid: row.ID, Flag: flag, Steps: int64(degraus), Pmpaid: int64(custo),
	}); err != nil {
		return err
	}
	if err := s.ligaOsCondicionaisDaFlag(r, row, dto, flag); err != nil {
		return err
	}
	return s.aplicaAsConcessoesDaPostura(r, row, flag)
}

// aPosturaDaFlag acha a ativação da postura pela flag que ela acende.
func aPosturaDaFlag(flag string) *activationOfBook {
	postura, tem := stancesFromCatalog()[flag]
	if !tem {
		return nil
	}
	return aAtivacaoDe("", postura.Name)
}

// cobraOPm tira o PM da ficha, sem deixar o saldo abaixo de zero.
//
// O piso existe porque a decisão que autorizou o gasto foi tomada com o saldo
// LIDO antes, e duas requisições na mesma ficha podem se cruzar: cobrar até o
// fundo é melhor que gravar um PM negativo, que a tela desenharia como barra
// para trás.
func (s *Server) cobraOPm(r *http.Request, row sqlcgen.Character, quanto int) error {
	if quanto <= 0 {
		return nil
	}
	depois := row.Mpcurrent - int64(quanto)
	if depois < 0 {
		depois = 0
	}
	return s.queries.SetMpCurrent(r.Context(), sqlcgen.SetMpCurrentParams{
		MpCurrent: depois, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	})
}

// ligaOsCondicionaisDaFlag sobe TODOS os condicionais daquela flag.
//
// São vários por postura — a Fúria mexe em ataque, dano, Defesa e testes de
// Vontade —, e eles sobem juntos: metade ligada é uma ficha que soma metade de
// uma regra do livro.
func (s *Server) ligaOsCondicionaisDaFlag(
	r *http.Request, row sqlcgen.Character, dto CharacterDTO, flag string,
) error {
	if s.catalogs == nil {
		return nil
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		return err
	}
	for _, c := range engine.ComputeItemEffects(s.catalogs.ActiveItemsFor(ec)).Conditional {
		if c.Flag != flag {
			continue
		}
		if err := s.queries.AddCharacterConditional(r.Context(), sqlcgen.AddCharacterConditionalParams{
			Characterid: row.ID, Conditionalid: engine.ConditionalID(c),
		}); err != nil {
			return err
		}
	}
	return nil
}
