package sheetui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"t20engine/book"
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
func usePower(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	spec := book.ActivationOf(chi.URLParam(r, "poder"), "")
	if spec == nil {
		return fmt.Errorf("o poder %q não tem ativação no catálogo", chi.URLParam(r, "poder"))
	}
	if spec.Kind != "instant" {
		return fmt.Errorf("%q não é um poder de usar", spec.Name)
	}
	usos := powerUses(dto)[spec.ID]
	pode, porque := useDecision(*spec, useContext{
		PmAtual: int(dto.MpCurrent), UsadoNaCena: usos.Cena, UsadoNoDia: usos.Dia,
		Flags: s.activeFlags(dto),
	})
	if !pode {
		return fmt.Errorf("%s: %s", spec.Name, porque)
	}
	if err := s.chargePm(r, row, activationPm(*spec)); err != nil {
		return err
	}
	escopo := chargedScope(*spec)
	if escopo == "" {
		return nil
	}
	return s.deps.Queries().BumpCharacterPowerUse(r.Context(), sqlcgen.BumpCharacterPowerUseParams{
		Characterid: row.ID, Powerid: spec.ID, Scope: escopo,
	})
}

// enterStance entra numa postura, com os degraus escolhidos.
//
// São QUATRO escritas para um gesto: o PM sai, o pagamento é registrado, os
// condicionais da flag sobem, e o que a postura concede vira efeito. O
// pagamento é registrado para sair não devolver PM — é o que a tabela
// `character_stances` existe para lembrar (ALE-222).
func enterStance(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	flag := chi.URLParam(r, "flag")
	spec := flagStance(flag)
	if spec == nil {
		return fmt.Errorf("%q não é uma postura do livro", flag)
	}
	degraus := 0
	if sinais.PoderDegraus != nil {
		degraus = int(*sinais.PoderDegraus)
	}
	maximo := 0
	if spec.Scaling != nil {
		maximo = levelSteps(*spec.Scaling, classPowerLevel(dto, spec.ID))
	}
	pode, porque := stanceDecision(*spec, degraus, maximo, int(dto.MpCurrent))
	if !pode {
		return fmt.Errorf("%s: %s", spec.Name, porque)
	}
	custo := stanceCost(*spec, degraus)
	if err := s.chargePm(r, row, custo); err != nil {
		return err
	}
	if err := s.deps.Queries().UpsertCharacterStance(r.Context(), sqlcgen.UpsertCharacterStanceParams{
		Characterid: row.ID, Flag: flag, Steps: int64(degraus), Pmpaid: int64(custo),
	}); err != nil {
		return err
	}
	if err := s.turnsOnTheConditionalsFlag(r, row, dto, flag); err != nil {
		return err
	}
	return s.applyTheGrantsStance(r, row, flag)
}

// aPosturaDaFlag acha a ativação da postura pela flag que ela acende.
func flagStance(flag string) *book.Activation {
	postura, tem := stancesFromCatalog()[flag]
	if !tem {
		return nil
	}
	return book.ActivationOf("", postura.Name)
}

// cobraOPm tira o PM da ficha, sem deixar o saldo abaixo de zero.
//
// O piso existe porque a decisão que autorizou o gasto foi tomada com o saldo
// LIDO antes, e duas requisições na mesma ficha podem se cruzar: cobrar até o
// fundo é melhor que gravar um PM negativo, que a tela desenharia como barra
// para trás.
func (s Scene) chargePm(r *http.Request, row sqlcgen.Character, quanto int) error {
	if quanto <= 0 {
		return nil
	}
	depois := row.Mpcurrent - int64(quanto)
	if depois < 0 {
		depois = 0
	}
	return s.deps.Queries().SetMpCurrent(r.Context(), sqlcgen.SetMpCurrentParams{
		MpCurrent: depois, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	})
}

// ligaOsCondicionaisDaFlag sobe TODOS os condicionais daquela flag.
//
// São vários por postura — a Fúria mexe em ataque, dano, Defesa e testes de
// Vontade —, e eles sobem juntos: metade ligada é uma ficha que soma metade de
// uma regra do livro.
func (s Scene) turnsOnTheConditionalsFlag(
	r *http.Request, row sqlcgen.Character, dto sheet.CharacterDTO, flag string,
) error {
	if s.deps.Catalogs() == nil {
		return nil
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return err
	}
	for _, c := range engine.ComputeItemEffects(s.deps.Catalogs().ActiveItemsFor(ec)).Conditional {
		if c.Flag != flag {
			continue
		}
		if err := s.deps.Queries().AddCharacterConditional(r.Context(), sqlcgen.AddCharacterConditionalParams{
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
func pickPower(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	id := chi.URLParam(r, "poder")
	return s.saveTheChoices(r, row, func(dto *sheet.CharacterDTO) {
		dto.ClassPowers = idToggledCom(dto.ClassPowers, id)
	})
}

// pickOriginBenefit liga ou desliga um benefício da origem.
func pickOriginBenefit(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	id := chi.URLParam(r, "beneficio")
	return s.saveTheChoices(r, row, func(dto *sheet.CharacterDTO) {
		dto.OriginChoices = idToggledCom(dto.OriginChoices, id)
	})
}

// pickRaceVariant escolhe a variante de uma habilidade de raça.
//
// Ela é EXCLUSIVA dentro da habilidade: escolher "resistência a fogo" tira
// "resistência a frio", porque o qareen tem uma resistência e não seis.
func pickRaceVariant(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	escolhida := chi.URLParam(r, "variante")
	return s.saveTheChoices(r, row, func(dto *sheet.CharacterDTO) {
		dto.RaceAbilityChoices = variantSwappedCom(*dto, escolhida)
	})
}

// pickClassChoice grava o caminho ou o devoto de uma classe.
func pickClassChoice(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	escolha, valor := chi.URLParam(r, "escolha"), chi.URLParam(r, "valor")
	if escolha != "caminho" && escolha != "devoto" {
		return fmt.Errorf("%q não é uma escolha de classe", escolha)
	}
	classe, err := url.PathUnescape(chi.URLParam(r, "classe"))
	if err != nil {
		return fmt.Errorf("a classe %q não é um nome", chi.URLParam(r, "classe"))
	}
	return s.saveTheChoices(r, row, func(dto *sheet.CharacterDTO) {
		dto.ClassChoices = choiceClassCom(dto.ClassChoices, classe, escolha, valor)
	})
}

// pickRaceAttributes grava a distribuição de atributo da raça.
//
// Os atributos vêm por SINAL porque são uma lista — três chaves de uma vez — e
// um caminho com três pedaços daria uma rota por combinação. Quem recusa a
// combinação inválida é o motor, pelo `sheet.WithChoicesValid`... e não: a
// distribuição tem regra PRÓPRIA (distintas, count exato, atributo proibido), e
// quem a conhece é o `RaceAttributeChoiceIsComplete`. Gravar e perguntar depois
// seria aceitar uma ficha inválida por um instante.
func pickRaceAttributes(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	escolhas := sinais.RacaAtributos
	blob, err := json.Marshal(map[string]any{"floatingPicks": escolhas})
	if err != nil {
		return err
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	dto.RaceAttributeChoices = string(blob)
	if s.deps.Catalogs() != nil {
		for _, raca := range dto.Races {
			if !s.deps.Catalogs().RaceAttributeChoiceIsComplete(raca.Race, dto.RaceAttributeChoices) {
				return fmt.Errorf("a distribuição não fecha para %s: ela pede atributos distintos", raca.Race)
			}
		}
	}
	return s.saveRaceAttributeChoice(r, row.ID, string(blob))
}

// pickRaceAscendencia grava a metade escolhida de uma raça de ascendência.
func pickRaceAscendencia(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	blob, err := json.Marshal(map[string]any{"ascendencia": chi.URLParam(r, "ascendencia")})
	if err != nil {
		return err
	}
	return s.saveRaceAttributeChoice(r, row.ID, string(blob))
}

// gravaAsEscolhas aplica a mudança e CONFERE a ficha inteira antes de gravar.
//
// A conferência é sobre o RESULTADO e não sobre a diferença: a escrita tem de
// deixar a ficha válida pelo livro. É a decisão do dono nesta fatia, e ela é
// mais estrita do que "não acrescente além do limite" — uma ficha fora da conta
// não aceita escrita de escolha nenhuma até ser arrumada.
func (s Scene) saveTheChoices(
	r *http.Request, row sqlcgen.Character, muda func(*sheet.CharacterDTO),
) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	antes := dto
	muda(&dto)
	if err := sheet.WithChoicesValid(dto); err != nil {
		return err
	}
	var escreve ChoiceWrite
	mexeu := false
	if dto.ClassPowers != antes.ClassPowers {
		escreve.ClassPowers, mexeu = &dto.ClassPowers, true
	}
	if dto.OriginChoices != antes.OriginChoices {
		escreve.OriginChoices, mexeu = &dto.OriginChoices, true
	}
	if dto.ClassChoices != antes.ClassChoices {
		escreve.ClassChoices, mexeu = &dto.ClassChoices, true
	}
	if dto.RaceAbilityChoices != antes.RaceAbilityChoices {
		escreve.RaceAbilityChoices, mexeu = &dto.RaceAbilityChoices, true
	}
	if !mexeu {
		return nil
	}
	return s.deps.SaveChoices(r.Context(), row.ID, escreve)
}

// gravaAEscolhaDeAtributoDaRaca escreve o blob de `raceAttributeChoices`.
//
// Ela é a única escolha que se grava SOZINHA — as outras quatro passam pelo
// `saveTheChoices`, que confere a ficha inteira antes.
func (s Scene) saveRaceAttributeChoice(r *http.Request, id int64, valor string) error {
	return s.deps.SaveChoices(r.Context(), id, ChoiceWrite{RaceAttributeChoices: &valor})
}

// comOIdAlternado liga ou desliga um id numa lista guardada como blob.
func idToggledCom(blob, id string) string {
	atuais := sheet.UnmarshalStrings(blob)
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
	return sheet.MarshalStrings(&depois)
}

// comAVarianteTrocada troca a variante escolhida dentro da MESMA habilidade.
func variantSwappedCom(dto sheet.CharacterDTO, escolhida string) string {
	irmas := variantSibling(dto, escolhida)
	depois := []string{escolhida}
	for _, atual := range sheet.UnmarshalStrings(dto.RaceAbilityChoices) {
		if !irmas[atual] {
			depois = append(depois, atual)
		}
	}
	return sheet.MarshalStrings(&depois)
}

// asIrmasDaVariante são todas as opções da habilidade a que a escolhida
// pertence — inclusive ela.
func variantSibling(dto sheet.CharacterDTO, escolhida string) map[string]bool {
	fora := map[string]bool{}
	for _, r := range dto.Races {
		for _, hab := range raceVariants(dto, r.Race) {
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
func choiceClassCom(blob, classe, qual, valor string) string {
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
