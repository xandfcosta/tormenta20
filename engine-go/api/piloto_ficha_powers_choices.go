package api

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"t20engine/sheet"
)

// O DIÁLOGO DE ESCOLHER PODERES como dado (ALE-272, fatia 8).
//
// # Por que é um diálogo, e não meia tela
//
// Ele era um MODO da aba Poderes, e o modo é o que a tornava difícil de usar
// (ALE-217): ele ABRIA sozinho sempre que havia pendência — o estado normal de
// quem acabou de subir de nível —, e o cromo dele comia 44% do painel no
// telefone. Escolher poder acontece uma vez por nível; virar meia tela para isso
// é caro demais.
//
// As três abas de FONTE ficam aqui dentro, e aqui elas fazem sentido: no preparo
// a pergunta é "de onde vem o que eu ainda posso escolher". Na mesa a pergunta é
// outra, e por isso a lista de trás não tem abas.

// choicesPanel é o diálogo inteiro.
type choicesPanel struct {
	Pendencias []pendencia
	Races      []raceChoiceCard
	Origin     *originChoiceCard
	Classes    []classChoiceCard
}

// raceChoiceCard é o que uma raça pede: o bônus de atributo e as variantes.
type raceChoiceCard struct {
	Race string
	// Attribute existe quando a raça distribui bônus — o `+1 ×3` do humano — ou
	// escolhe ascendência, como o suraggel.
	Attribute *attributeChoice
	Variants  []variantChoice
}

// attributeChoice é o bônus de atributo que a raça deixa escolher.
type attributeChoice struct {
	// Kind é `floating` (N atributos distintos) ou `ascendencia`.
	Kind string
	// Count são quantos atributos se escolhe, no `floating`.
	Count int
	Value int
	// Exclude é o atributo PROIBIDO, quando a raça tem um.
	Exclude  string
	Options  []filterOption
	Chosen   []string
	Complete bool
}

// variantChoice é uma habilidade de raça com opções — a Resistência Elemental
// do qareen, a Herança Divina do suraggel.
type variantChoice struct {
	AbilityID string
	Name      string
	Options   []filterOption
	Chosen    string
}

// originChoiceCard são os benefícios da origem, com o teto de dois.
type originChoiceCard struct {
	Name    string
	Options []filterOption
	Chosen  []string
	Left    int
}

// classChoiceCard é o que uma classe pede: as vagas de poder, o caminho e o
// devoto.
type classChoiceCard struct {
	ClassName string
	Level     int64
	// Slots são as vagas do nível e quantas já foram usadas.
	Slots int
	Used  int
	// Powers são os poderes ELETIVOS da classe mais os gerais, já marcados.
	Powers  []powerChoice
	Caminho *pickerChoice
	Devoto  *pickerChoice
}

// powerChoice é um poder que dá para escolher.
type powerChoice struct {
	ID     string
	Name   string
	Detail string
	// Fonte é "Classe" ou "Geral", para a lista dizer de onde o poder vem.
	Fonte  string
	Chosen bool
}

// pickerChoice é uma escolha de valor único — caminho ou devoto.
type pickerChoice struct {
	Options []filterOption
	Chosen  string
}

// choicesPanelOf monta o diálogo.
func (s *Server) choicesPanelOf(dto sheet.CharacterDTO, busca string) choicesPanel {
	panel := choicesPanel{Pendencias: s.asPendenciasDaFicha(dto)}
	panel.Races = s.asEscolhasDeRaca(dto)
	panel.Origin = aEscolhaDeOrigem(dto)
	panel.Classes = asEscolhasDasClasses(dto, busca)
	return panel
}

func (s *Server) asEscolhasDeRaca(dto sheet.CharacterDTO) []raceChoiceCard {
	cartoes := []raceChoiceCard{}
	for _, r := range dto.Races {
		cartao := raceChoiceCard{Race: r.Race, Variants: asVariantesDaRaca(dto, r.Race)}
		cartao.Attribute = s.oBonusDeAtributoDaRaca(dto, r.Race)
		if cartao.Attribute == nil && len(cartao.Variants) == 0 {
			continue
		}
		cartoes = append(cartoes, cartao)
	}
	return cartoes
}

// oBonusDeAtributoDaRaca descreve a escolha de atributo, ou nil quando a raça
// não pede nenhuma (as doze de bônus fixo).
func (s *Server) oBonusDeAtributoDaRaca(dto sheet.CharacterDTO, nome string) *attributeChoice {
	mod := oModDeAtributoDaRaca(nome)
	if mod == nil || mod.Kind == "fixed" {
		return nil
	}
	escolha := &attributeChoice{
		Kind: mod.Kind, Count: mod.Count, Value: mod.Value, Exclude: mod.Exclude,
		Chosen: asEscolhasDeAtributoGuardadas(dto.RaceAttributeChoices),
	}
	if s.catalogs != nil {
		escolha.Complete = s.catalogs.RaceAttributeChoiceIsComplete(nome, dto.RaceAttributeChoices)
	}
	if mod.Kind == "floating" {
		escolha.Options = osAtributosQueCabem(mod.Exclude)
		return escolha
	}
	escolha.Kind = "ascendencia"
	escolha.Options = asAscendenciasDaRaca(nome)
	if a := aAscendenciaGuardada(dto.RaceAttributeChoices); a != "" {
		escolha.Chosen = []string{a}
	}
	return escolha
}

// osAtributosQueCabem são os seis do livro, menos o proibido da raça.
func osAtributosQueCabem(proibido string) []filterOption {
	fora := []filterOption{}
	for _, a := range ordemDosAtributos {
		if a.Chave == proibido {
			continue
		}
		fora = append(fora, filterOption{Valor: a.Chave, Rotulo: a.Sigla})
	}
	return fora
}

// asVariantesDaRaca são as habilidades de raça que pedem uma escolha.
func asVariantesDaRaca(dto sheet.CharacterDTO, nome string) []variantChoice {
	raca := aRacaComVariantes(nome)
	if raca == nil {
		return nil
	}
	escolhidas := asEscolhasGuardadas(dto.RaceAbilityChoices)
	fora := []variantChoice{}
	for _, hab := range raca.Habilidades {
		if len(hab.Variantes) == 0 {
			continue
		}
		escolha := variantChoice{AbilityID: hab.ID, Name: hab.Nome}
		for _, v := range hab.Variantes {
			ativa := contemTraco(escolhidas, v.ID)
			if ativa {
				escolha.Chosen = v.ID
			}
			escolha.Options = append(escolha.Options,
				filterOption{Valor: v.ID, Rotulo: v.Nome, Ativo: ativa})
		}
		fora = append(fora, escolha)
	}
	return fora
}

// aEscolhaDeOrigem monta o cartão da origem.
func aEscolhaDeOrigem(dto sheet.CharacterDTO) *originChoiceCard {
	origem, tem := origensDoLivro()[dto.Origin]
	if !tem {
		return nil
	}
	escolhidos := asEscolhasGuardadas(dto.OriginChoices)
	cartao := &originChoiceCard{
		Name: origem.Name, Chosen: escolhidos,
		Left: oLimiteDeBeneficiosDaOrigem - len(escolhidos),
	}
	for _, b := range osBeneficiosQueAOrigemOferece(origem) {
		cartao.Options = append(cartao.Options, filterOption{
			Valor: b.ID, Rotulo: b.Name, Ativo: contemTraco(escolhidos, b.ID),
		})
	}
	return cartao
}

// asEscolhasDasClasses monta um cartão por classe da ficha.
func asEscolhasDasClasses(dto sheet.CharacterDTO, busca string) []classChoiceCard {
	escolhidos := asEscolhasGuardadas(dto.ClassPowers)
	escolhas := asEscolhasDeClasse(dto)
	cartoes := []classChoiceCard{}
	for _, classe := range dto.Classes {
		cartao := classChoiceCard{
			ClassName: classe.ClassName, Level: classe.Level,
			Slots:  asVagasDePoder(classe.Level),
			Used:   len(escolhidos),
			Powers: osPoderesQueDaParaEscolher(classe.ClassName, escolhidos, busca),
		}
		if opcoes := osCaminhosDoNivel(classe.ClassName, classe.Level); len(opcoes) > 0 {
			cartao.Caminho = oSeletorMarcado(opcoes, escolhas[classe.ClassName].Caminho)
		}
		if opcoes := osDevotosDaClasse(classe.ClassName); len(opcoes) > 0 {
			cartao.Devoto = oSeletorMarcado(opcoes, escolhas[classe.ClassName].Devoto)
		}
		cartoes = append(cartoes, cartao)
	}
	return cartoes
}

func oSeletorMarcado(opcoes []filterOption, escolhido string) *pickerChoice {
	marcadas := make([]filterOption, 0, len(opcoes))
	for _, o := range opcoes {
		o.Ativo = o.Valor == escolhido
		marcadas = append(marcadas, o)
	}
	return &pickerChoice{Options: marcadas, Chosen: escolhido}
}

// osPoderesQueDaParaEscolher são os ELETIVOS da classe mais os gerais.
//
// "Você sempre pode substituir um poder de classe por um poder geral" (p33), e
// por isso as duas listas viram uma só — a vaga é a mesma.
func osPoderesQueDaParaEscolher(classe string, escolhidos []string, busca string) []powerChoice {
	termo := foldAccents(strings.TrimSpace(busca))
	fora := []powerChoice{}
	for _, p := range poderesDeClasseDoLivro() {
		if p.ClassName != classe || p.GrantedAtLevel != nil || !casaComABusca(p.Name, termo) {
			continue
		}
		fora = append(fora, powerChoice{
			ID: p.ID, Name: p.Name, Detail: p.Description, Fonte: "Classe",
			Chosen: contemTraco(escolhidos, p.ID),
		})
	}
	for _, p := range poderesGeraisDoLivro() {
		if !casaComABusca(p.Name, termo) {
			continue
		}
		fora = append(fora, powerChoice{
			ID: p.ID, Name: p.Name, Detail: p.Description, Fonte: aFonteDoPoderGeral(p),
			Chosen: contemTraco(escolhidos, p.ID),
		})
	}
	sort.SliceStable(fora, func(a, b int) bool {
		if fora[a].Chosen != fora[b].Chosen {
			return fora[a].Chosen
		}
		return fora[a].Name < fora[b].Name
	})
	return fora
}

func casaComABusca(nome, termo string) bool {
	return termo == "" || strings.Contains(foldAccents(nome), termo)
}

// ── o que a TELA escreve ─────────────────────────────────────────────────────

// asVagasDaClasseEscritas é "3 de 4 vagas".
func asVagasDaClasseEscritas(cartao classChoiceCard) string {
	return strconv.Itoa(cartao.Used) + " de " + strconv.Itoa(cartao.Slots) + " vagas"
}

// aEscolhaDoAtributoEscrita descreve o que a raça pede.
func aEscolhaDoAtributoEscrita(escolha attributeChoice) string {
	if escolha.Kind == "ascendencia" {
		return "Escolha a ascendência"
	}
	texto := "Distribua +" + strconv.Itoa(escolha.Value) + " em " +
		strconv.Itoa(escolha.Count) + " atributos diferentes"
	if escolha.Exclude != "" {
		return texto + " (exceto " + siglaDoAtributo(escolha.Exclude) + ")"
	}
	return texto
}

// asTresFontes são as abas do diálogo, na ordem em que o livro monta um
// personagem: raça, origem, classe.
var asTresFontes = []string{"raca", "origem", "classe"}

// aFonteDaPrimeiraPendencia é a aba em que o diálogo ABRE.
//
// Quem abriu veio pela pendência, e fazê-lo caçar a aba certa é gastar o clique
// que ele acabou de dar. Sem pendência nenhuma, ele abre na Raça.
func aFonteDaPrimeiraPendencia(v fichaView) string {
	if len(v.Choices.Pendencias) > 0 {
		return v.Choices.Pendencias[0].Fonte
	}
	return "raca"
}

// oChipDaEscolha é a classe de um chip que liga e desliga.
func oChipDaEscolha(ativo bool) string {
	base := "rounded-full border px-2 py-0.5 text-3xs uppercase tracking-wider outline-none transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
	if ativo {
		return base + " border-grimorio-gold/60 bg-accent text-grimorio-gold"
	}
	return base + " border-grimorio-iron text-muted-foreground hover:text-foreground"
}

// oComandoDaEscolhaDeClasse escreve o `@post` do caminho ou do devoto.
func oComandoDaEscolhaDeClasse(v fichaView, classe, escolha, valor string) string {
	return oPostDaFicha(v, "/poderes/classe/"+url.PathEscape(classe)+"/"+escolha+"/"+valor)
}

// oGestoQueAlternaOAtributo liga ou desliga um atributo na distribuição.
//
// A lista é COPIADA antes de ser mexida: o sinal é um proxy, e escrever dentro
// dele item a item é a armadilha que o guia do Go registra.
func oGestoQueAlternaOAtributo(atributo string) string {
	return "const escolhidos = [...$racaatributos]; const onde = escolhidos.indexOf('" + atributo + "'); " +
		"if (onde >= 0) { escolhidos.splice(onde, 1) } else { escolhidos.push('" + atributo + "') }; " +
		"$racaatributos = escolhidos"
}
