package sheetui

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// As rotas da FICHA no app.
//
// A ficha mora em `/personagens/{id}` — filha do endereço do elenco, e não numa
// raiz própria: ela é o que se abre DE dentro da lista, e o endereço dizer isso
// é o que faz o ‹ Voltar ter para onde voltar.

func Routes(r chi.Router, s Scene) {
	r.Get("/personagens/{id}", s.sheetHandle)
	r.Get("/personagens/{id}/ao-vivo", s.sheetLive)
	// O PASSO no CAMINHO e não no corpo. Quem segura essa escolha é o
	// `sheetCommand` logo abaixo: ele lê os sinais uma vez e os entrega ao
	// `s.Load`, que redesenha a cena INTEIRA. Como o `payload` do Datastar
	// SUBSTITUI os sinais, um corpo com o passo chegaria com os outros campos
	// zerados e a tela voltaria sem o filtro que a pessoa usava.
	r.Post("/personagens/{id}/vitais/{qual}/{passo}", s.sheetCommand(touchesVital))
	r.Post("/personagens/{id}/vitais/{qual}/temporarios/{passo}", s.sheetCommand(touchesTempVital))
	r.Post("/personagens/{id}/vitais/{qual}/gasto/{passo}", s.sheetCommand(spendsVital))
	// A CLASSE vai no caminho porque o nível é dela: o do personagem é a SOMA.
	r.Post("/personagens/{id}/nivel/{classe}/{passo}", s.sheetCommand(mudaONivel))
	r.Post("/personagens/{id}/classes/{classe}/entra", s.sheetCommand(enterClass))
	r.Post("/personagens/{id}/atributos/{atributo}/{valor}", s.sheetCommand(setAttributeValue))
	// DOIS caminhos e não um com "padrao" no lugar da categoria: aí o dia em que
	// uma proficiência se chamasse `padrao` calaria o restaurar, e o `alterna`
	// no meio é o que impede a colisão de existir.
	r.Post("/personagens/{id}/proficiencias/alterna/{categoria}", s.sheetCommand(toggleProficiency))
	r.Post("/personagens/{id}/proficiencias/padrao", s.sheetCommand(restoresDefaultClass))
	// AS PERÍCIAS. O NOME vai no CAMINHO, escapado, e não num sinal: um sinal
	// aqui seria disputado por uma dezena de botões idênticos.
	r.Post("/personagens/{id}/pericias/treino/{nome}", s.sheetCommand(toggleTraining))
	r.Post("/personagens/{id}/pericias/rolar/{nome}", s.sheetCommand(rollExpertise))
	r.Post("/personagens/{id}/pericias/atributo/{nome}/{atributo}", s.sheetCommand(swapAttribute))
	r.Post("/personagens/{id}/pericias/remover/{nome}", s.sheetCommand(removeCraft))
	// A CRIAÇÃO é a única que lê SINAL, porque o nome é texto que a pessoa acabou
	// de digitar e ainda não existe em lugar nenhum para virar caminho.
	r.Post("/personagens/{id}/pericias/nova", s.sheetCommand(criaOOficio))
	// AS AÇÕES: os gestos que saem da superfície Ações e terminam na MESA.
	//
	// ROTA DE FICHA e não de sessão, e a razão é o remendo: estes botões moram
	// dentro do `#actions-scene`, que o gateway desta cena redesenha inteiro —
	// um endereço com campanha e sessão dentro obrigaria a ficha a carregar
	// coordenada de mesa só para se redesenhar. Quem sabe em que sessão este
	// personagem está é o hospedeiro, pela porta, como no teste de perícia.
	r.Post("/personagens/{id}/acoes/atacar/{arma}", s.sheetCommand(rollsAttackOnTheTarget(false)))
	r.Post("/personagens/{id}/acoes/atacar/{arma}/outro-uso", s.sheetCommand(rollsAttackOnTheTarget(true)))
	r.Post("/personagens/{id}/acoes/manobra/{manobra}", s.sheetCommand(rollsManeuverOnTheTarget))
	r.Post("/personagens/{id}/acoes/investida/{arma}", s.sheetCommand(chargesTheTarget(engine.ChargePlain)))
	r.Post("/personagens/{id}/acoes/ataque-extra/{arma}", s.sheetCommand(rollsExtraStrikeOnTheTarget(false)))
	r.Post("/personagens/{id}/acoes/ataque-extra/{arma}/outro-uso", s.sheetCommand(rollsExtraStrikeOnTheTarget(true)))
	r.Post("/personagens/{id}/acoes/mirar", s.sheetCommand(aimsAtTheTarget))
	r.Post("/personagens/{id}/acoes/bola-de-fogo/{item}", s.sheetCommand(firesTheBlastOnTheTarget))
	r.Post("/personagens/{id}/acoes/investida-imprudente/{arma}", s.sheetCommand(chargesTheTarget(engine.ChargeReckless)))
	// OS EFEITOS: quatro donos de estado, quatro caminhos.
	r.Post("/personagens/{id}/efeitos/condicao/{cond}", s.sheetCommand(toggleBookCondition))
	r.Post("/personagens/{id}/efeitos/aplica/{magia}", s.sheetCommand(applySpellBuff))
	r.Post("/personagens/{id}/efeitos/encerra/{efeito}", s.sheetCommand(endAppliedEffect))
	r.Post("/personagens/{id}/efeitos/postura/{flag}", s.sheetCommand(endStance))
	r.Post("/personagens/{id}/efeitos/declaracao/{flag}", s.sheetCommand(endDeclaration))
	r.Post("/personagens/{id}/declara/{flag}", s.sheetCommand(declares))
	r.Post("/personagens/{id}/efeitos/forma-selvagem", s.sheetCommand(leaveWildShape))
	// O SITUACIONAL manda a CHAVE do condicional, que é um encadeado com `::` e
	// texto livre do catálogo dentro — por isso ela vai por SINAL e não no
	// caminho. É a exceção da ficha, e a razão dela é o formato da chave.
	r.Post("/personagens/{id}/efeitos/situacao", s.sheetCommand(toggleSituational))
	// O DEGRAU tem rota PRÓPRIA, e não é o mesmo gesto com um sinal a mais:
	// aquele ALTERNA, e escolher "+6" num Baluarte já ligado o desligaria. Aqui
	// a semântica é "ligue com estes degraus", que é o que o seletor diz.
	r.Post("/personagens/{id}/efeitos/situacao/degraus", s.sheetCommand(setSituationalDegrees))
	// AS MAGIAS.
	r.Post("/personagens/{id}/magias/aprende/{magia}", s.sheetCommand(learnSpell))
	r.Post("/personagens/{id}/magias/esquece/{magia}", s.sheetCommand(forgetSpell))
	r.Post("/personagens/{id}/magias/prepara/{magia}", s.sheetCommand(togglePrepared))
	r.Post("/personagens/{id}/magias/conjura/{magia}", s.sheetCommand(castSpellFromSheet))
	// A MOCHILA.
	r.Post("/personagens/{id}/itens/{item}/guarda", s.sheetCommand(stowItem))
	r.Post("/personagens/{id}/itens/{item}/equipa/{slot}", s.sheetCommand(equipItemFromSheet))
	// O DINHEIRO manda o modo e o valor por SINAL: a conta do saldo depende dos
	// dois juntos, e um caminho com o valor dentro daria um endereço diferente
	// a cada tecla digitada no campo.
	r.Post("/personagens/{id}/dinheiro", s.sheetCommand(changeMoney))
	r.Post("/personagens/{id}/itens/adiciona/{catalogo}", s.sheetCommand(addCatalogItem))
	r.Post("/personagens/{id}/itens/custom", s.sheetCommand(addCustomItem))
	r.Post("/personagens/{id}/itens/{item}/edita", s.sheetCommand(editItem))
	r.Post("/personagens/{id}/itens/{item}/remover", s.sheetCommand(removeItemFromSheet))
	r.Post("/personagens/{id}/itens/{item}/usa", s.sheetCommand(useItem))
	r.Post("/personagens/{id}/itens/{item}/carrega/{arma}", s.sheetCommand(loadDose))
	r.Post("/personagens/{id}/itens/{item}/melhorias", s.sheetCommand(applyOverlays))
	r.Post("/personagens/{id}/itens/{item}/encantos", s.sheetCommand(enchantsTheItem))
	r.Post("/personagens/{id}/itens/{item}/categoria-maior", s.sheetCommand(toggleOversizedWeapon))
	r.Post("/personagens/{id}/itens/{item}/escola/{escola}", s.sheetCommand(choosesTheItemSchool))
	r.Post("/personagens/{id}/itens/{item}/deus/{deus}", s.sheetCommand(choosesTheItemGod))
	r.Post("/personagens/{id}/itens/{item}/melhorias/{melhoria}/magia/{magia}", s.sheetCommand(choosesTheImprovementSpell))
	// OS PODERES.
	r.Post("/personagens/{id}/poderes/usa/{poder}", s.sheetCommand(usePower))
	r.Post("/personagens/{id}/poderes/postura/{flag}/entra", s.sheetCommand(enterStance))
	r.Post("/personagens/{id}/poderes/forma-selvagem", s.sheetCommand(takeWildShape))
	// AS ESCOLHAS do diálogo.
	r.Post("/personagens/{id}/poderes/escolhe/{poder}", s.sheetCommand(pickPower))
	r.Post("/personagens/{id}/poderes/escolha/{poder}/{opcao}", s.sheetCommand(pickPowerOption))
	// TROCAR é o único jeito de mudar uma escolha feita: ela nunca fica vazia.
	r.Post("/personagens/{id}/escolhas/{escolha}/troca/{poder}", s.sheetCommand(swapChoice))
	r.Post("/personagens/{id}/escolhas/{escolha}/troca/{poder}/{opcao}", s.sheetCommand(swapChoice))
	r.Post("/personagens/{id}/escolhas/{escolha}/libera", s.sheetCommand(freeChoice))
	r.Post("/personagens/{id}/poderes/origem/{beneficio}", s.sheetCommand(pickOriginBenefit))
	r.Post("/personagens/{id}/poderes/raca/{habilidade}/{opcao}", s.sheetCommand(pickRaceAbility))
	r.Post("/personagens/{id}/poderes/variante/{variante}", s.sheetCommand(pickRaceVariant))
	r.Post("/personagens/{id}/poderes/classe/{classe}/{escolha}/{valor}", s.sheetCommand(pickClassChoice))
	r.Post("/personagens/{id}/poderes/devocao/{deus}", s.sheetCommand(pickDevotion))
	r.Post("/personagens/{id}/poderes/poder-concedido/{poder}", s.sheetCommand(pickGrantedPower))
	r.Post("/personagens/{id}/poderes/atributos", s.sheetCommand(pickRaceAttributes))
	r.Post("/personagens/{id}/poderes/ascendencia/{ascendencia}", s.sheetCommand(pickRaceAscendencia))
}

func (s Scene) sheetHandle(w http.ResponseWriter, r *http.Request) {
	id, ok := uRLCharacter(w, r)
	if !ok {
		return
	}
	pageSignals := sheetSignals(r)
	view, status, err := s.Load(
		r.Context(), s.deps.CurrentUserID(r), id, AskedTab(r.URL.Query().Get("tab")),
		pageSignals.term(), pageSignals)
	view.Embedded = insideSessionAsked(r)
	view.ColorMode = ui.ColorModeOf(r)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	// A FICHA DENTRO DA SESSÃO responde só o pedaço: quem pediu foi a aba "Minha
	// ficha" da Mesa, e mandar a página inteira faria o Datastar remendar a cena
	// da sessão com uma ficha de corpo inteiro.
	//
	// O `@get` da ficha solta também: a troca macia de aba e a busca pedem a
	// cena, e a página inteira — a casca, o buscador, o livro — fazia o
	// navegador comparar tudo de novo, ~450ms num celular médio (ALE-520).
	if view.Embedded || datastarAsked(r) {
		fragment, err := ui.RenderFragment(r.Context(), SceneBody(view))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sse := datastar.NewSSE(w, r)
		_ = sse.PatchElements(fragment)
		patchActionsAlong(r, sse, view)
		return
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Title: view.Name + " · Tormenta 20",
		// `ui.ShellBare`: a cena desenha o próprio cabeçalho, com a volta e o nome.
		Shape: ui.ShellBare,
		// O FLUXO AO VIVO: a ficha aberta acompanha o que muda nela por outro caminho (`sheetLive`).
		Init: fmt.Sprintf("@get('/personagens/%d/ao-vivo')", view.ID),
		// Os sinais moram no <body> porque o <body> NUNCA é remendado: declarado
		// dentro de um painel, o `@post` de qualquer gesto redeclararia o sinal a
		// cada toque e fecharia o diálogo que o jogador acabou de abrir.
		Signals: "{sheet_version: '', detail: '', craft: false, new_expertise: '', new_attribute: 'intelligence'," +
			" condition_dialog: false, buff_dialog: false, conditional: ''," +
			" learn_dialog: false, augment0: 0, augment1: 0, augment2: 0, augment3: 0, augment4: 0, augment5: 0, augment_quickened: 0, augment_amplified: 0, augment_discreet: 0," +
			" spell_search: '', spell_circle: '', spell_school: '', grimoire_search: '', grimoire_circle: ''," +
			" condition_search: '', buff_search: '', buff_circle: ''," +
			" item_search: '', item_category: '', tibar_mode: 'receber', tibar_value: 0," +
			" catalog_search: '', catalog_category: '', improvement_spell_search: '', item_qty: 1, item_name: '', item_slots: 1," +
			" item_roll_hp: 0, item_roll_mp: 0, item_improvements: [], item_material: '', item_enchants: [], item_enchant_against: ''," +
			" power_search: '', stance_degrees: 0, power: '', show_passives: false, wild_form: '', wild_tier: '', wild_damage: '', wild_move: ''," +
			" power_source: 'raca', choice_step: '', power_pick_filter: '', race_attributes: [], expertise_d20: 0, expertise_trained_only: false, vital_amount: '', vital_temp_amount: '', " + ExpertiseSpendsDeclaration() + "}",
	}, SceneBody(view))
}

// insideSessionAsked diz se este pedido veio da superfície "Minha ficha" da
// Mesa. A marca viaja na query pela mesma razão que o `?tab=`: o handler
// descobre o que desenhar lendo a requisição, e sinal do cliente sumiria num F5.
func insideSessionAsked(r *http.Request) bool {
	return r.URL.Query().Get("embutida") == "1"
}

// sheetCommand é o gateway das mutações da ficha.
//
// Ele existe pela mesma razão do `boardCommand`: resolver a posse, mutar e
// redesenhar são três passos que toda mutação da ficha faz, e escrevê-los em
// cada handler é como um deles esquece de redesenhar — a pessoa clica, o banco
// muda e a tela fica igual.
//
// A POSSE é conferida aqui, uma vez. A ficha é do dono e de mais ninguém: a
// regra é a mesma da API JSON (`characterFor`), e a cena não ganha uma segunda.
func (s Scene) sheetCommand(
	mutate func(Scene, *http.Request, sqlcgen.Character, Signals) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := uRLCharacter(w, r)
		if !ok {
			return
		}
		// UMA leitura de sinais por requisição, e ela vem ANTES de tudo: o
		// `ReadSignals` consome o corpo do POST, então a segunda chamada
		// receberia vazio sem erro nenhum. Ver `signals.go`.
		signals := sheetSignals(r)
		row, status, err := s.reachableCharacter(r.Context(), id, s.deps.CurrentUserID(r))
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		// A RECUSA VOLTA PELA CENA, e não por um status de erro: o cliente do
		// Datastar NÃO aplica remendo de resposta que não é 2xx, então um
		// `http.Error(400)` aqui deixaria a recusa só como uma linha vermelha no
		// console e o gesto simplesmente não aconteceria na tela.
		//
		// Todas as recusas da ficha são de REGRA — o teto de duas mãos, o PM que
		// falta, a magia que não está preparada —, e regra recusada é informação
		// para quem joga. Ela sobe com a cena inteira redesenhada, que é o que
		// prova que nada mudou, mais a frase. A API JSON continua respondendo os
		// status dela; esta rota desenha página.
		refusal := ""
		if err := mutate(s, r, row, signals); err != nil {
			refusal = err.Error()
		} else {
			// A FICHA MUDOU, e quem avisa é o GATEWAY e não cada comando: são
			// mais de trinta mutações passando por aqui, e uma que esquecesse a
			// linha nasceria sem aviso.
			//
			// Só no caminho SEM recusa: uma regra que barrou o gesto não mudou
			// nada, e avisar ali faria toda tela que escuta refazer a ficha por
			// causa de um clique que não aconteceu.
			s.deps.CharacterChanged(row.ID)
		}
		view, status, err := s.Load(
			r.Context(), s.deps.CurrentUserID(r), id, AskedTab(r.URL.Query().Get("tab")), signals.term(), signals)
		view.Embedded = insideSessionAsked(r)
		view.ColorMode = ui.ColorModeOf(r)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		view.Refusal = refusal
		sse := datastar.NewSSE(w, r)
		fragment, err := ui.RenderFragment(r.Context(), SceneBody(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragment)
		patchActionsAlong(r, sse, view)
	}
}

// patchActionsAlong manda a superfície de Ações junto da ficha, porque ela é um nó
// IRMÃO do `#sheet-scene`. Um remendo só da ficha deixaria o número velho na tela:
// ligar a Fúria pelos chips moveria o `+8` da aba Combate e não o `+8` da linha de
// Atacar, e o aviso do stream deixaria nela a dose que o mestre acabou de gastar.
//
// Só EMBUTIDA: fora da sessão o `#actions-scene` não existe na página, e mandar
// remendo para um id que não está lá é trabalho que ninguém usa.
func patchActionsAlong(r *http.Request, sse *datastar.ServerSentEventGenerator, view View) {
	if !view.Embedded {
		return
	}
	if actions, err := ui.RenderFragment(r.Context(), ActionsSurface(view)); err == nil {
		_ = sse.PatchElements(actions)
	}
}

// touchesVital soma o passo ao PV ou ao PM, PRENDENDO na faixa.
//
// PRENDER e não recusar, e essa é a diferença entre o gesto e a API: o corpo do
// `PATCH /vitals` manda o valor ABSOLUTO e é recusado fora da faixa, o que está
// certo para um cliente que calculou. Aqui o gesto é uma correção do número, e uma
// recusa faria o mestre clicar de um em um para chegar no mesmo lugar. Por isso ele
// não toca na poça temporária: quem paga com ela é o `spendsVital`.
//
// O TETO é o máximo do personagem: curar além do máximo não é PV temporário, que
// é outra regra e tem dono no motor (`TempHpFuria`).
func touchesVital(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	step, err := uRLStep(r)
	if err != nil {
		return err
	}
	return s.plays.TouchVital(r.Context(), row, chi.URLParam(r, "qual"), step)
}

// touchesTempVital soma ou tira os temporários do PV ou do PM. O sinal do passo diz qual.
func touchesTempVital(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	step, err := uRLStep(r)
	if err != nil {
		return err
	}
	return s.plays.TouchTempVital(r.Context(), row, chi.URLParam(r, "qual"), step)
}

// spendsVital é o "Dano" e o "Gastar" do diálogo, que drenam os temporários primeiro.
func spendsVital(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	amount, err := uRLStep(r)
	if err != nil {
		return err
	}
	if amount < 0 {
		return fmt.Errorf("o gasto é um valor positivo, e veio %d", amount)
	}
	return s.plays.SpendVital(r.Context(), row, chi.URLParam(r, "qual"), amount)
}

// enterClass dá o 1º nível numa classe que o personagem ainda não tem (p35).
func enterClass(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	class, err := url.PathUnescape(chi.URLParam(r, "classe"))
	if err != nil {
		return fmt.Errorf("classe %q não é um nome válido", chi.URLParam(r, "classe"))
	}
	return s.plays.EnterClass(r.Context(), row, class)
}

// setAttributeValue grava o valor livre que o diálogo do atributo escreveu. O `−` e o `+` mandam o valor
// já calculado, então o gesto é um só, e a recusa de um POST escrito à mão é a mesma do campo.
func setAttributeValue(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	written, err := url.PathUnescape(chi.URLParam(r, "valor"))
	if err != nil {
		return fmt.Errorf("%q não é um valor válido", chi.URLParam(r, "valor"))
	}
	value, err := sheet.AttributeValueRead(written)
	if err != nil {
		return err
	}
	return s.plays.SetAttribute(r.Context(), row, chi.URLParam(r, "atributo"), value)
}

// lowerLastLevel é o `−`: só o último nível sai, e o que foi escolhido nele sai junto.
func (s Scene) lowerLastLevel(
	r *http.Request, row sqlcgen.Character, dto sheet.CharacterDTO, class string, target int64,
) error {
	loss, known := sheet.LastLevelOf(dto)
	if known && loss.ClassName != class {
		return fmt.Errorf("o nível %d é de %s: só o último nível desce", loss.Level, loss.ClassName)
	}
	if target < 1 {
		if err := s.plays.LeaveClass(r.Context(), row, class); err != nil {
			return err
		}
		return s.plays.FollowMaximumChange(r.Context(), dto, withClassAt(dto, class, 0))
	}
	if !known {
		return fmt.Errorf("a ficha %d não tem nível guardado para descer", row.ID)
	}
	// A ficha antiga pode ter o mesmo atributo duas vezes no mesmo patamar, e o `−` não a conserta
	// sozinho: quem resolve é o Trocar.
	if err := sheet.AttributeIncreasesFit(sheet.LoweredBy(dto, loss)); err != nil {
		return err
	}
	if err := s.plays.LowerLastLevel(r.Context(), row, loss); err != nil {
		return err
	}
	if err := s.settlePickEffects(r, row, dto, loss.Removed, nil); err != nil {
		return err
	}
	return s.plays.FollowMaximumChange(r.Context(), dto, sheet.LoweredBy(dto, loss))
}

// withClassAt é a ficha simulada com a classe naquele nível. Nível zero tira a classe.
func withClassAt(dto sheet.CharacterDTO, class string, level int64) sheet.CharacterDTO {
	classes := make([]sheet.ClassDTO, 0, len(dto.Classes))
	for _, entry := range dto.Classes {
		if entry.ClassName == class {
			entry.Level = level
		}
		if entry.Level > 0 {
			classes = append(classes, entry)
		}
	}
	dto.Classes = classes
	return dto
}

// mudaONivel sobe ou desce UMA CLASSE, e o nível do personagem acompanha.
//
// NUNCA escreva o nível do personagem direto: ele é a SOMA dos níveis de classe,
// e escrever o total deixa a ficha dizendo 13 com as classes somando 12 — os
// pools de PV e PM derivam das CLASSES e não se mexem, então o número sobe e o
// personagem não fica mais forte.
//
// A regra mora no caso de uso (`character.Plays.LevelClass`): a classe tem de
// ser do personagem, o total para em 20, e os poços sincronizam.
func mudaONivel(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	step, err := uRLStep(r)
	if err != nil {
		return err
	}
	class, err := url.PathUnescape(chi.URLParam(r, "classe"))
	if err != nil {
		return fmt.Errorf("classe %q não é um nome válido", chi.URLParam(r, "classe"))
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	for _, cl := range dto.Classes {
		if cl.ClassName != class {
			continue
		}
		target := cl.Level + int64(step)
		if step > 0 {
			if err := s.plays.LevelClass(r.Context(), row, class, target); err != nil {
				return err
			}
			return s.plays.FollowMaximumChange(r.Context(), dto, withClassAt(dto, class, target))
		}
		return s.lowerLastLevel(r, row, dto, class, target)
	}
	return fmt.Errorf("%s não é uma classe deste personagem", class)
}

// uRLCharacter lê o id do caminho. Erro aqui é URL digitada errada, e a
// resposta é uma frase: quem está do outro lado é um navegador mostrando página.
func uRLCharacter(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "o personagem precisa ser um número", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// uRLStep aceita o sinal de menos: o passo é para os dois lados.
func uRLStep(r *http.Request) (int, error) {
	raw := chi.URLParam(r, "passo")
	step, err := strconv.Atoi(raw)
	if err != nil || step == 0 {
		return 0, fmt.Errorf("passo %q não é um número diferente de zero", raw)
	}
	return step, nil
}
