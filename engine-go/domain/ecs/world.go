// Package ecs é o núcleo de entidade-componente-sistema do motor (ALE-380).
//
// Ele NÃO sabe o que é Tormenta: não há atributo, perícia nem modificador aqui,
// e nenhum import deste projeto — só a stdlib. É a mesma propriedade que o
// `domain/engine` defende, e ela é o que permite exercitar o mecanismo sem
// arranjar uma ficha inteira.
//
// # Por que escrito à mão e não uma biblioteca
//
// As bibliotecas de ECS em Go otimizam arquétipo para localidade de cache em
// milhares de entidades POR QUADRO. Aqui é uma ficha calculada sob demanda:
// pagaríamos a API delas por uma propriedade que não se usa, e o motor perderia
// a pureza de importar só a stdlib. Se um dia entrar uma, este pacote já é a
// interface fina que o guia manda ter na frente de terceiro.
//
// # A ordem é CORREÇÃO, e é a decisão que molda tudo aqui
//
// O oráculo de paridade compara byte a byte, e o coletor que este núcleo vai
// substituir avisa que a ordem é significativa. A iteração de `map` em Go é
// aleatória por construção, então o armazenamento guarda os componentes numa
// FATIA na ordem de inserção e usa o mapa só para achar a posição.
//
// @example
//
//	w := ecs.NewWorld()
//	e := w.Spawn()
//	ecs.Set(w, e, Peso{Gramas: 10})
//	ecs.Each(w, func(e ecs.Entity, p Peso) { … })
package ecs

import "reflect"

// Entity é só um número: quem carrega significado é o componente.
type Entity uint32

// World guarda as entidades vivas e um armazenamento por TIPO de componente.
type World struct {
	next   Entity
	alive  map[Entity]bool
	stores map[reflect.Type]anyStore
}

func NewWorld() *World {
	return &World{alive: map[Entity]bool{}, stores: map[reflect.Type]anyStore{}}
}

// Spawn cunha uma entidade nova. O contador NÃO é reaproveitado depois de um
// Despawn: um id reciclado faria um `Entity` guardado fora do mundo apontar,
// silenciosamente, para outra coisa.
func (w *World) Spawn() Entity {
	w.next++
	w.alive[w.next] = true
	return w.next
}

func (w *World) Alive(e Entity) bool { return w.alive[e] }

// Despawn mata a entidade e apaga TODO componente dela, em todo armazenamento.
// Sem isto o mundo responderia `Alive == false` e a consulta continuaria
// visitando — que é o pior dos dois, porque só aparece na asserção seguinte.
func (w *World) Despawn(e Entity) {
	delete(w.alive, e)
	for _, s := range w.stores {
		s.remove(e)
	}
}

// anyStore é o que o World consegue pedir sem saber o tipo do componente.
type anyStore interface{ remove(e Entity) }

// store guarda os componentes de UM tipo.
//
// A fatia é a fonte da ordem; o mapa é só o índice. Remover COMPACTA a fatia em
// vez de trocar com o último — a troca seria O(1) e reordenaria, que é
// exatamente o que não pode acontecer. O mundo de uma ficha tem dezenas de
// entidades, então o O(n) não é o custo que importa aqui.
type store[C any] struct {
	entities []Entity
	values   []C
	index    map[Entity]int
}

func (s *store[C]) remove(e Entity) {
	at, ok := s.index[e]
	if !ok {
		return
	}
	s.entities = append(s.entities[:at], s.entities[at+1:]...)
	s.values = append(s.values[:at], s.values[at+1:]...)
	delete(s.index, e)
	for _, later := range s.entities[at:] {
		s.index[later]--
	}
}

// storeOf devolve o armazenamento do tipo C, criando-o na primeira escrita.
func storeOf[C any](w *World, create bool) *store[C] {
	key := reflect.TypeFor[C]()
	if found, ok := w.stores[key]; ok {
		return found.(*store[C])
	}
	if !create {
		return nil
	}
	fresh := &store[C]{index: map[Entity]int{}}
	w.stores[key] = fresh
	return fresh
}

// Set escreve o componente na entidade. Escrever de novo SUBSTITUI sem
// reordenar: a ordem da consulta é a de inserção da ENTIDADE, e não a da última
// escrita — senão ela mudaria conforme a ordem das regras que tocam o valor.
//
// Entidade morta é ignorada em silêncio, e isso é deliberado: um sistema que
// escreve em cima de algo que outro sistema matou no mesmo passe não deve
// derrubar o cálculo da ficha.
func Set[C any](w *World, e Entity, c C) {
	if !w.alive[e] {
		return
	}
	s := storeOf[C](w, true)
	if at, ok := s.index[e]; ok {
		s.values[at] = c
		return
	}
	s.index[e] = len(s.entities)
	s.entities = append(s.entities, e)
	s.values = append(s.values, c)
}

// Get devolve o componente e se ele existe. O segundo retorno é a pergunta:
// o valor zero de um componente é um valor legítimo, então `C{}` não distingue
// "não tem" de "tem, zerado".
func Get[C any](w *World, e Entity) (C, bool) {
	s := storeOf[C](w, false)
	if s == nil {
		var zero C
		return zero, false
	}
	at, ok := s.index[e]
	if !ok {
		var zero C
		return zero, false
	}
	return s.values[at], true
}

// Remove tira UM componente sem matar a entidade.
func Remove[C any](w *World, e Entity) {
	if s := storeOf[C](w, false); s != nil {
		s.remove(e)
	}
}

// Each visita quem tem C, na ordem em que o componente foi inserido.
func Each[C any](w *World, visit func(Entity, C)) {
	s := storeOf[C](w, false)
	if s == nil {
		return
	}
	// A VARREDURA É SOBRE UMA CÓPIA, e não sobre a fatia viva.
	//
	// Um sistema que chame Set ou Remove de dentro do `visit` mexe nas fatias no
	// meio do laço — o Remove compacta, e o índice do laço passa a pular um
	// elemento em silêncio. Copiar dezenas de pares é barato; descobrir isso
	// como vermelho intermitente na fatia 2 não é.
	entities := make([]Entity, len(s.entities))
	values := make([]C, len(s.values))
	copy(entities, s.entities)
	copy(values, s.values)
	for at := range entities {
		visit(entities[at], values[at])
	}
}

// Each2 visita quem tem OS DOIS componentes, na ordem de inserção de A.
//
// A ordem é a de A e não a de B, e a escolha tem de estar escrita porque as
// duas são plausíveis: quem chama precisa saber qual componente manda na
// ordem do resultado.
//
// Não há versão variádica, e não pode haver: Go não tem parâmetro de tipo
// variádico. Uma terceira aridade se escreve à mão no dia em que faltar — e
// precisar de três componentes casados costuma ser sinal de que eles deviam
// ser um só.
func Each2[A, B any](w *World, visit func(Entity, A, B)) {
	Each(w, func(e Entity, a A) {
		if b, ok := Get[B](w, e); ok {
			visit(e, a, b)
		}
	})
}

// System é um passo do cálculo. A ORDEM da fatia é o grafo de dependência,
// escrita à mão — defesa depende de Destreza, que depende de carga.
type System func(*World)

func Run(w *World, systems ...System) {
	for _, run := range systems {
		run(w)
	}
}
