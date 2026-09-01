package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
	"t20engine/sheet"
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
	dto, err := s.LoadCharacter(r.Context(), row)
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
	dto, err := s.LoadCharacter(r.Context(), row)
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
	r *http.Request, row sqlcgen.Character, dto sheet.CharacterDTO, flag string,
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

// ── AS ESCOLHAS, e a validação que virou fronteira ───────────────────────────

// pickPower liga ou desliga um poder eletivo.
//
// O comando manda o PODER e não a lista inteira, pela razão de sempre: mandar a
// lista perde para o clique repetido e para a segunda aba aberta. Quem decide o
// estado final é o servidor, que sabe o que está gravado.
func pickPower(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	id := chi.URLParam(r, "poder")
	return s.gravaAsEscolhas(r, row, func(dto *sheet.CharacterDTO) {
		dto.ClassPowers = comOIdAlternado(dto.ClassPowers, id)
	})
}

// pickOriginBenefit liga ou desliga um benefício da origem.
func pickOriginBenefit(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	id := chi.URLParam(r, "beneficio")
	return s.gravaAsEscolhas(r, row, func(dto *sheet.CharacterDTO) {
		dto.OriginChoices = comOIdAlternado(dto.OriginChoices, id)
	})
}

// pickRaceVariant escolhe a variante de uma habilidade de raça.
//
// Ela é EXCLUSIVA dentro da habilidade: escolher "resistência a fogo" tira
// "resistência a frio", porque o qareen tem uma resistência e não seis.
func pickRaceVariant(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	escolhida := chi.URLParam(r, "variante")
	return s.gravaAsEscolhas(r, row, func(dto *sheet.CharacterDTO) {
		dto.RaceAbilityChoices = comAVarianteTrocada(*dto, escolhida)
	})
}

// pickClassChoice grava o caminho ou o devoto de uma classe.
func pickClassChoice(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	escolha, valor := chi.URLParam(r, "escolha"), chi.URLParam(r, "valor")
	if escolha != "caminho" && escolha != "devoto" {
		return fmt.Errorf("%q não é uma escolha de classe", escolha)
	}
	classe, err := url.PathUnescape(chi.URLParam(r, "classe"))
	if err != nil {
		return fmt.Errorf("a classe %q não é um nome", chi.URLParam(r, "classe"))
	}
	return s.gravaAsEscolhas(r, row, func(dto *sheet.CharacterDTO) {
		dto.ClassChoices = comAEscolhaDeClasse(dto.ClassChoices, classe, escolha, valor)
	})
}

// pickRaceAttributes grava a distribuição de atributo da raça.
//
// Os atributos vêm por SINAL porque são uma lista — três chaves de uma vez — e
// um caminho com três pedaços daria uma rota por combinação. Quem recusa a
// combinação inválida é o motor, pelo `aFichaComEscolhasValidas`... e não: a
// distribuição tem regra PRÓPRIA (distintas, count exato, atributo proibido), e
// quem a conhece é o `RaceAttributeChoiceIsComplete`. Gravar e perguntar depois
// seria aceitar uma ficha inválida por um instante.
func pickRaceAttributes(s *Server, r *http.Request, row sqlcgen.Character, sinais fichaSignals) error {
	escolhas := sinais.RacaAtributos
	blob, err := json.Marshal(map[string]any{"floatingPicks": escolhas})
	if err != nil {
		return err
	}
	dto, err := s.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	dto.RaceAttributeChoices = string(blob)
	if s.catalogs != nil {
		for _, raca := range dto.Races {
			if !s.catalogs.RaceAttributeChoiceIsComplete(raca.Race, dto.RaceAttributeChoices) {
				return fmt.Errorf("a distribuição não fecha para %s: ela pede atributos distintos", raca.Race)
			}
		}
	}
	return s.gravaOBlobDaFicha(r, row.ID, "raceAttributeChoices", string(blob))
}

// pickRaceAscendencia grava a metade escolhida de uma raça de ascendência.
func pickRaceAscendencia(s *Server, r *http.Request, row sqlcgen.Character, _ fichaSignals) error {
	blob, err := json.Marshal(map[string]any{"ascendencia": chi.URLParam(r, "ascendencia")})
	if err != nil {
		return err
	}
	return s.gravaOBlobDaFicha(r, row.ID, "raceAttributeChoices", string(blob))
}

// gravaAsEscolhas aplica a mudança e CONFERE a ficha inteira antes de gravar.
//
// A conferência é sobre o RESULTADO e não sobre a diferença: a escrita tem de
// deixar a ficha válida pelo livro. É a decisão do dono nesta fatia, e ela é
// mais estrita do que "não acrescente além do limite" — uma ficha fora da conta
// não aceita escrita de escolha nenhuma até ser arrumada.
func (s *Server) gravaAsEscolhas(
	r *http.Request, row sqlcgen.Character, muda func(*sheet.CharacterDTO),
) error {
	dto, err := s.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	antes := dto
	muda(&dto)
	if err := aFichaComEscolhasValidas(dto); err != nil {
		return err
	}
	var set setBuilder
	if dto.ClassPowers != antes.ClassPowers {
		set.Add("classPowers = ?", dto.ClassPowers)
	}
	if dto.OriginChoices != antes.OriginChoices {
		set.Add("originChoices = ?", dto.OriginChoices)
	}
	if dto.ClassChoices != antes.ClassChoices {
		set.Add("classChoices = ?", dto.ClassChoices)
	}
	if dto.RaceAbilityChoices != antes.RaceAbilityChoices {
		set.Add("raceAbilityChoices = ?", dto.RaceAbilityChoices)
	}
	if set.empty() {
		return nil
	}
	return set.execTouched(r.Context(), s.db, "UPDATE characters", row.ID)
}

// gravaOBlobDaFicha escreve UMA coluna de escolha.
func (s *Server) gravaOBlobDaFicha(r *http.Request, id int64, coluna, valor string) error {
	var set setBuilder
	set.Add(coluna+" = ?", valor)
	return set.execTouched(r.Context(), s.db, "UPDATE characters", id)
}

// comOIdAlternado liga ou desliga um id numa lista guardada como blob.
func comOIdAlternado(blob, id string) string {
	atuais := asEscolhasGuardadas(blob)
	depois := []string{}
	tinha := false
	for _, atual := range atuais {
		if atual == id {
			tinha = true
			continue
		}
		depois = append(depois, atual)
	}
	if !tinha {
		depois = append(depois, id)
	}
	return marshalStrings(&depois)
}

// comAVarianteTrocada troca a variante escolhida dentro da MESMA habilidade.
func comAVarianteTrocada(dto sheet.CharacterDTO, escolhida string) string {
	irmas := asIrmasDaVariante(dto, escolhida)
	depois := []string{escolhida}
	for _, atual := range asEscolhasGuardadas(dto.RaceAbilityChoices) {
		if !irmas[atual] {
			depois = append(depois, atual)
		}
	}
	return marshalStrings(&depois)
}

// asIrmasDaVariante são todas as opções da habilidade a que a escolhida
// pertence — inclusive ela.
func asIrmasDaVariante(dto sheet.CharacterDTO, escolhida string) map[string]bool {
	fora := map[string]bool{}
	for _, r := range dto.Races {
		for _, hab := range asVariantesDaRaca(dto, r.Race) {
			daHabilidade := map[string]bool{}
			achou := false
			for _, o := range hab.Options {
				daHabilidade[o.Valor] = true
				achou = achou || o.Valor == escolhida
			}
			if achou {
				return daHabilidade
			}
		}
	}
	return fora
}

// comAEscolhaDeClasse escreve caminho ou devoto no blob de escolhas.
func comAEscolhaDeClasse(blob, classe, qual, valor string) string {
	escolhas := map[string]engine.ClassChoiceSelections{}
	_ = json.Unmarshal([]byte(blob), &escolhas)
	daClasse := escolhas[classe]
	if qual == "caminho" {
		daClasse.Caminho = valor
	} else {
		daClasse.Devoto = valor
	}
	escolhas[classe] = daClasse
	depois, err := json.Marshal(escolhas)
	if err != nil {
		return blob
	}
	return string(depois)
}
