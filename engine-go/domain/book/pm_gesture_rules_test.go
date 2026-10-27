package book

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TODO VERBETE QUE FALA EM PM TEM GESTO NA ABA AÇÕES — ou diz por que não
// (ALE-492).
//
// A regra "gaste N PM para…" que não está nas ativações não é oferecida nem
// cobra o PM. A varredura cobre poder de classe, poder geral e da Tormenta,
// poder de deus, habilidade de raça e poder único de origem. É uma lista de
// PERMITIDOS: o verbete novo que fale em PM sem gesto reprova com o nome dele,
// e o permitido que ganhou gesto também — a lista só encolhe.
var notAGesture = map[string]string{
	// o TOTAL de PM
	"class.arcanista.caminho-bruxo": "total de PM", "class.arcanista.caminho-feiticeiro": "total de PM",
	"class.arcanista.caminho-mago": "total de PM", "class.arcanista.magias-1-circulo": "total de PM",
	"class.bardo.magias-1-circulo": "total de PM", "class.clerigo.magias-1-circulo": "total de PM",
	"class.druida.magias-1-circulo": "total de PM", "class.arcanista.poder-magico": "total de PM",
	"class.barbaro.totem-espiritual": "total de PM", "class.paladino.abencoado": "total de PM",
	"divino.bencao-do-mana": "total de PM", "vontade-de-ferro": "total de PM",
	"origin-heroi-campones-unique": "total de PM", "class.bucaneiro.bravata-audaz": "total de PM",
	// o CUSTO de magia ou de habilidade
	"class.arcanista.alta-arcana": "custo", "class.arcanista.mestre-em-escola": "custo",
	"class.bardo.artista-completo": "custo", "class.druida.aspecto-da-primavera": "custo",
	"class.druida.forca-da-natureza": "custo", "class.paladino.virtude-paladinesca-caridade": "custo",
	"foco-em-magia": "custo", "magia-ilimitada": "custo", "qareen-desejos": "custo",
	"celebrar-ritual": "custo", "class.arcanista.escriba-arcano": "custo em dinheiro por PM",
	// RECUPERAR ou ganhar PM
	"class.bucaneiro.panache": "recupera PM", "class.cacador.mestre-cacador": "recupera PM",
	"class.guerreiro.campeao": "recupera PM", "class.ladino.roubo-de-mana": "PM temporário",
	"class.lutador.confianca-dos-ringues": "PM temporário", "class.nobre.general": "PM temporário",
	"fome-de-mana": "PM temporário",
	// o DESCANSO
	"origin-selvagem-unique": "descanso", "goblin-rato-das-ruas": "descanso",
	"osteon-preco-da-nao-vida": "descanso", "sereia-transformacao-anfibia": "descanso",
	// PERDER PM por violar o código
	"class.cavaleiro.codigo-de-honra": "perde PM", "class.paladino.codigo-do-heroi": "perde PM",
	// muda OUTRA habilidade, sem gasto próprio
	"class.barbaro.furia-titanica": "muda a Fúria", "class.paladino.virtude-paladinesca-compaixao": "muda a Cura pelas Mãos",
	"class.inventor.obra-prima": "não gasta PM",
	// já COBERTO por outro mecanismo
	"class.cavaleiro.baluarte": "situacional com degraus", "class.cavaleiro.duelo": "situacional",
	"class.guerreiro.ataque-especial-4": "impulso no ataque", "class.paladino.golpe-divino-1d8": "impulso no ataque",
}

type pmEntry struct{ id, name, description string }

func everyEntryThatMentionsPM() []pmEntry {
	pm := regexp.MustCompile(`\bPM\b`)
	out := []pmEntry{}
	add := func(id, name, description string) {
		if pm.MatchString(description) {
			out = append(out, pmEntry{id, name, description})
		}
	}
	for _, p := range FlattenedPowers() {
		if !strings.HasPrefix(p.Source, "Geral") {
			add(p.ID, p.Name, p.Description)
		}
	}
	for _, p := range GeneralPowers() {
		add(p.ID, p.Name, p.Description)
	}
	for key, race := range RaceTraitsByKey() {
		if key != race.ID {
			continue
		}
		for _, a := range race.Abilities {
			add(a.ID, a.Name, a.Description)
		}
	}
	for _, o := range Origins() {
		add(o.UniquePower.ID, o.UniquePower.Name, o.UniquePower.Description)
	}
	return out
}

func TestEveryEntryThatSpendsPMHasAGesture(t *testing.T) {
	gestures := map[string]bool{}
	for _, a := range Activations() {
		if a.Kind != "passive" {
			gestures[a.ID], gestures["name:"+a.Name] = true, true
		}
	}
	entries := everyEntryThatMentionsPM()
	if len(entries) < 250 {
		t.Fatalf("a varredura achou %d verbetes falando em PM; a medição da ALE-492 achou 272", len(entries))
	}
	seen := []string{}
	for _, e := range entries {
		seen = append(seen, e.id)
		has := gestures[e.id] || gestures["name:"+e.name]
		reason, allowed := notAGesture[e.id]
		switch {
		case !has && !allowed:
			t.Errorf("%s (%s) fala em PM e não tem gesto na aba Ações: escreva a ativação, ou ponha o id em `notAGesture` com o motivo", e.name, e.id)
		case has && allowed:
			t.Errorf("%s (%s) ganhou gesto e continua em `notAGesture` (%q): tire-o da lista", e.name, e.id, reason)
		}
	}
	for id := range notAGesture {
		if !slices.Contains(seen, id) {
			t.Errorf("%s está em `notAGesture` e não é verbete que fala em PM: tire-o da lista", id)
		}
	}
}
