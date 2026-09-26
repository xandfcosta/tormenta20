#!/usr/bin/env python3
"""Confere `races.json` contra o Capítulo 1 do livro (p19-31). Seção 1 da ALE-391.

O que este auditor mede
-----------------------
Os NÚMEROS da raça — modificadores de atributo, deslocamento, tamanho e as duas
visões —, mais a presença de cada habilidade que o catálogo declara. Não mede a
DESCRIÇÃO da habilidade: o catálogo reescreve as frases do livro em forma curta
de propósito, e comparar prosa reescrita com prosa impressa só produz ruído.

Por que a âncora não é o título nem a página
--------------------------------------------
"Habilidades de Raça" não serve: ele quebra em duas linhas no Elfo e as "Raças
Extras" (p27 em diante) não o imprimem. E o `bookPage` do catálogo também não
serve, porque o bloco de uma raça ATRAVESSA a página — o Hynne tem título e
prosa na p27, o Golem inteiro vem depois, e a linha de atributos do Hynne só
aparece no alto da p28. Ordem de leitura por coluna não desembaralha isso: é o
mesmo interfoliamento que fez o Orc e o Glop se misturarem na ALE-151.

A âncora é a LINHA DE ATRIBUTOS, e a ligação dela com a raça é semântica
-------------------------------------------------------------------------
Toda raça imprime exatamente uma linha de atributos, e ela é a cabeça do bloco.
Ligá-la à raça se faz em três regras, nesta ordem, e o que sobrar REPROVA com
nome:

1. **A linha traz o rótulo da subraça** — `(Aggelus)` e `(Sulfure)` são as
   chaves que o próprio catálogo usa em `variants`. Só o Suraggel, e ela corre
   PRIMEIRO: a habilidade colada na linha dele é "Mau Cheiro", que é do Trog.
2. **Vem colada à primeira habilidade da raça** — ou É ela, que é o caso do
   Humano. Vale para 15 das 17.
3. **Fecho da bijeção:** sobrando exatamente uma raça e exatamente uma linha,
   elas são par. Só dispara com resto 1, e é o que alcança o Trog — cujas
   habilidades são impressas ANTES da linha de atributos dele, porque o Suraggel
   está interfoliado no meio.

Para deslocamento, tamanho e visão a região lida é o TEXTO DAS HABILIDADES DA
PRÓPRIA RAÇA, achado nome a nome — nunca uma faixa de página. Assim o Suraggel
e o Trog se medem certo mesmo entrelaçados.

O denominador
-------------
As 17 raças, e por raça quantas habilidades foram achadas no livro. Raça que não
ligou a uma linha de atributos, e habilidade que não foi achada (ou foi achada
duas vezes), saem NOMEADAS — não somem num total.

Uso: `python3 scripts/audit-races.py`. Precisa do PDF do livro (veja t20pdf).
"""
import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from t20pdf import (  # noqa: E402
    MENOS, RAIZ, chave, junta, linhas_da_pagina, num)

PRIMEIRA, ULTIMA = 25, 37  # PDF
OFFSET_DO_PDF = 6  # livro = PDF - 6
RACAS = RAIZ / 'engine-go/domain/catalog/data/races.json'

ATRIBUTOS = {
    'Força': 'strength', 'Destreza': 'dexterity', 'Constituição': 'constitution',
    'Inteligência': 'intelligence', 'Sabedoria': 'wisdom', 'Carisma': 'charisma',
}
NOMES = '|'.join(ATRIBUTOS)
RE_FIXO = re.compile(rf'(?:(?:{NOMES}) [+{MENOS}-]\d+(?: \(\w+\))?[,;]? ?)+\.?')
RE_PAR = re.compile(rf'({NOMES}) ([+{MENOS}-]\d+)(?: \((\w+)\))?')
# "+1 em três atributos diferentes", com a exceção e a penalidade opcionais do
# Lefou e do Osteon na mesma frase.
RE_LIVRE = re.compile(
    rf'\+1 em [Tt]rês [Aa]tributos [Dd]iferentes(?: \(exceto ({NOMES})\))?'
    rf'(?:, ({NOMES}) ([+{MENOS}-]\d+))?\.?')

# O deslocamento de NATAÇÃO, VOO, ESCALADA e ESCAVAÇÃO não é o deslocamento da
# raça: a Sereia imprime "deslocamento de natação 12m" e "(deslocamento 9m)" na
# mesma habilidade, e ler o primeiro número trocaria um pelo outro.
RE_DESLOCAMENTO = re.compile(
    r'deslocamento(?! de (?:natação|voo|escalada|escavação))\s*(?:é|de)?\s*(\d+)m')
RE_TAMANHO = re.compile(r'[Ss]eu tamanho é (Minúsculo|Pequeno|Médio|Grande|Enorme)')
DESLOCAMENTO_PADRAO, TAMANHO_PADRAO = 9, 'Médio'


SECAO_DAS_EXTRAS = 'Raças Extras'
TITULO_MAXIMO = 25  # caracteres; um título de raça é curto, a prosa não


def texto_do_capitulo() -> str:
    linhas: list[str] = []
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        linhas.extend(linhas_da_pagina(pagina))
    return junta(linhas)


def titulos_por_pagina(racas: dict) -> tuple[dict, int | None]:
    """Em que página do LIVRO cada raça abre, e onde abre a seção "Raças Extras".

    O título é a única coisa da raça que fica na página que o catálogo cita: o
    bloco de regras dela pode terminar na página seguinte (o Hynne termina), e
    conferir `bookPage` pela linha de atributos acusaria a p28 para uma raça que
    o livro abre na p27.
    """
    onde, extras = {}, None
    por_chave = {chave(r['name']): id_raca for id_raca, r in racas.items()}
    for pagina in range(PRIMEIRA, ULTIMA + 1):
        for linha in linhas_da_pagina(pagina):
            if linha.strip() == SECAO_DAS_EXTRAS and extras is None:
                extras = pagina - OFFSET_DO_PDF
            if len(linha) > TITULO_MAXIMO:
                continue
            id_raca = por_chave.get(chave(linha))
            if id_raca is not None and id_raca not in onde:
                onde[id_raca] = pagina - OFFSET_DO_PDF
    return onde, extras


def le_linha_de_atributos(bruto: str) -> dict:
    """A linha impressa vira a mesma forma que o catálogo guarda em `atributoMod`.

    O livro escreve o atributo ANTES do sinal ("Constituição +2"), ao contrário
    do bestiário, e marca a subraça entre parênteses ("Sabedoria +2 (Aggelus)").
    """
    if m := RE_LIVRE.match(bruto):
        livre: dict = {'kind': 'floating', 'count': 3, 'value': 1}
        if m.group(1):
            livre['exclude'] = ATRIBUTOS[m.group(1)]
        if m.group(2):
            livre['penalty'] = {'attribute': ATRIBUTOS[m.group(2)], 'value': num(m.group(3))}
        return livre
    # Uma linha ROTULADA carrega as duas subraças de uma vez ("Sabedoria +2,
    # Carisma +1 (Aggelus); Destreza +2, Inteligência +1 (Sulfure)."), e o
    # rótulo vem DEPOIS do grupo que ele nomeia. Acumular até o rótulo aparecer
    # é o que separa os dois conjuntos.
    mods, variantes, pendentes = {}, {}, {}
    for nome, sinal, subraca in RE_PAR.findall(bruto):
        pendentes[ATRIBUTOS[nome]] = num(sinal)
        if subraca:
            variantes[subraca.lower()] = pendentes
            pendentes = {}
    if variantes:
        return {'kind': 'subraca-gated', 'variants': variantes}
    mods.update(pendentes)
    return {'kind': 'fixed', 'mods': mods}


def acha_linhas_de_atributo(texto: str) -> list[tuple[int, int, str]]:
    """(início, fim, texto) de cada linha de atributos do capítulo."""
    achados = [(m.start(), m.end(), m.group(0)) for m in RE_LIVRE.finditer(texto)]
    for m in RE_FIXO.finditer(texto):
        # A penalidade do Lefou ("…Diferentes (exceto Carisma), Carisma –1.")
        # também casa como linha fixa; ela já pertence à linha livre.
        if not any(i <= m.start() < f for i, f, _t in achados):
            achados.append((m.start(), m.end(), m.group(0)))
    return sorted(achados)


def acha_habilidades(texto: str, racas: dict) -> tuple[dict, list[str]]:
    """Cada habilidade do catálogo → onde ela começa no livro.

    O corte de uma habilidade é a PRÓXIMA habilidade de QUALQUER raça, e não a
    próxima da mesma raça: o Suraggel e o Trog imprimem as suas alternadas, e
    cortar só nas próprias faria o texto de uma engolir o da outra.
    """
    onde, ausentes, ambiguas = {}, [], []
    for id_raca, raca in racas.items():
        for hab in raca['abilities']:
            achados = [m.start() for m in re.finditer(re.escape(hab['name']) + r'\.', texto)]
            if not achados:
                ausentes.append(f"{raca['name']} / {hab['name']}: não achada no livro")
            elif len(achados) == 1:
                onde[(id_raca, hab['name'])] = achados[0]
            else:
                ambiguas.append((id_raca, raca['name'], hab['name'], achados))
    # Nem todo nome é único no capítulo: "+1 em Três Atributos Diferentes" é
    # habilidade do Humano e linha de atributos da Sereia/Tritão, com a mesma
    # grafia. Quem desempata é a distância às habilidades da MESMA raça que já
    # ancoraram — um nome ambíguo sozinho numa raça continua saindo NOMEADO.
    for id_raca, nome_raca, nome_hab, achados in ambiguas:
        ancoras = [pos for (dono, _n), pos in onde.items() if dono == id_raca]
        if not ancoras:
            ausentes.append(f'{nome_raca} / {nome_hab}: achada {len(achados)}x e sem âncora '
                            f'na mesma raça para desempatar')
            continue
        onde[(id_raca, nome_hab)] = min(
            achados, key=lambda p: min(abs(p - a) for a in ancoras))
    return onde, ausentes


def liga(linhas, onde, racas) -> tuple[dict, list[str]]:
    """Linha de atributos → raça, pelas três regras da docstring do módulo.

    O RÓTULO corre antes da colagem, e a ordem é a regra: a linha do Suraggel é
    a que vem colada em "Mau Cheiro", que é habilidade do TROG — as duas raças
    são impressas entrelaçadas no encarte de p30-31. Deixar a colagem correr
    primeiro dá a linha do Suraggel ao Trog, com o resto da bijeção fechando
    por cima e nada reclamando.
    """
    de_raca: dict[str, tuple[int, int, str]] = {}
    tomadas: set[int] = set()
    for inicio, fim, bruto in linhas:
        # 1. rótulo de subraça que o catálogo declara em `variants`
        rotulos = le_linha_de_atributos(bruto).get('variants', {})
        for id_raca, raca in racas.items():
            declaradas = raca['atributoMod'].get('variants', {})
            if rotulos and set(rotulos) <= set(declaradas):
                de_raca.setdefault(id_raca, (inicio, fim, bruto))
                tomadas.add(inicio)
    for inicio, fim, bruto in linhas:
        # 2. colada à primeira habilidade da raça — ou SENDO ela: o Humano tem
        # "+1 em Três Atributos Diferentes" como linha de atributos E como nome
        # da primeira habilidade, e o livro imprime a frase uma vez só.
        if inicio in tomadas:
            continue
        donos = [id_raca for (id_raca, _n), pos in onde.items() if pos in (fim + 1, inicio)]
        if len(set(donos)) == 1 and donos[0] not in de_raca:
            de_raca[donos[0]] = (inicio, fim, bruto)
            tomadas.add(inicio)
    sobram_racas = [r for r in racas if r not in de_raca]
    sobram_linhas = [ln for ln in linhas if ln[0] not in tomadas]
    if len(sobram_racas) == 1 and len(sobram_linhas) == 1:
        # 3. fecho da bijeção — só com resto 1
        de_raca[sobram_racas[0]] = sobram_linhas[0]
        sobram_racas, sobram_linhas = [], []
    queixas = [f'{racas[r]["name"]}: nenhuma linha de atributos ligou' for r in sobram_racas]
    queixas += [f'linha sem dona: {ln[2]!r}' for ln in sobram_linhas]
    return de_raca, queixas


def regiao(texto: str, onde: dict, cortes: list[int], id_raca: str) -> str:
    """O texto das habilidades DESTA raça, cada uma até a próxima de qualquer uma."""
    pedacos = []
    for (dono, _nome), pos in onde.items():
        if dono != id_raca:
            continue
        seguintes = [c for c in cortes if c > pos]
        pedacos.append(texto[pos:min(seguintes) if seguintes else pos + 1200])
    return ' '.join(pedacos)


def confere(raca: dict, lido: dict, corpo: str, pagina: int | None,
            pagina_das_extras: int | None, impressos: dict) -> list[str]:
    """As queixas desta raça; `impressos` conta quantas vezes o livro DISSE cada
    coisa, em vez de o auditor ter caído no padrão.

    Sem essa contagem, um regex de deslocamento que parasse de casar devolveria
    "9m" para as dezessete e as raças que andam 9m continuariam verdes — metade
    da tabela passando por medição.
    """
    fora = []
    if raca['atributoMod'] != lido:
        fora.append(f'atributos: catálogo {raca["atributoMod"]} × livro {lido}')

    achados = RE_DESLOCAMENTO.findall(corpo)
    deslocamento = int(achados[0]) if achados else DESLOCAMENTO_PADRAO
    impressos['deslocamento'] += bool(achados)
    if raca['deslocamento'] != deslocamento:
        onde_ = 'impresso' if achados else 'padrão do livro (nenhuma habilidade o muda)'
        fora.append(f'deslocamento: catálogo {raca["deslocamento"]}m × livro {deslocamento}m ({onde_})')

    m = RE_TAMANHO.search(corpo)
    tamanho = m.group(1) if m else TAMANHO_PADRAO
    impressos['tamanho'] += bool(m)
    if raca['tamanho'] != tamanho:
        fora.append(f'tamanho: catálogo {raca["tamanho"]} × livro {tamanho}')

    if pagina is not None:
        if raca['bookPage'] != pagina:
            fora.append(f'bookPage: catálogo {raca["bookPage"]} × livro {pagina}')
        if pagina_das_extras is not None:
            tier = 'extra' if pagina >= pagina_das_extras else 'comum'
            if raca['tier'] != tier:
                fora.append(f'tier: catálogo {raca["tier"]} × livro {tier} '
                            f'("{SECAO_DAS_EXTRAS}" abre na p{pagina_das_extras})')

    for campo, frase in (('visaoNoEscuro', 'visão no escuro'),
                         ('visaoNaPenumbra', 'visão na penumbra')):
        no_livro = frase in corpo
        impressos[campo] += no_livro
        if raca[campo] != no_livro:
            fora.append(f'{campo}: catálogo {raca[campo]} × livro {no_livro}')
    return fora


def main() -> int:
    racas = json.load(open(RACAS, encoding='utf-8'))
    texto = texto_do_capitulo()
    linhas = acha_linhas_de_atributo(texto)
    onde, ausentes = acha_habilidades(texto, racas)
    de_raca, queixas = liga(linhas, onde, racas)
    paginas, pagina_das_extras = titulos_por_pagina(racas)
    queixas += [f'{racas[r]["name"]}: título não achado no capítulo — bookPage e tier '
                f'não medidos' for r in racas if r not in paginas]
    cortes = sorted(set(list(onde.values()) + [ln[0] for ln in linhas]))

    divergem = 0
    impressos = dict.fromkeys(
        ('deslocamento', 'tamanho', 'visaoNoEscuro', 'visaoNaPenumbra'), 0)
    for id_raca, raca in racas.items():
        if id_raca not in de_raca:
            continue
        lido = le_linha_de_atributos(de_raca[id_raca][2])
        for queixa in confere(raca, lido, regiao(texto, onde, cortes, id_raca),
                              paginas.get(id_raca), pagina_das_extras, impressos):
            print(f'  {raca["name"]}: {queixa}')
            divergem += 1

    for queixa in queixas + ausentes:
        print(f'  NÃO MEDIDO {queixa}')

    print(f'\nraças: {len(racas)} | ligadas a uma linha de atributos: {len(de_raca)} '
          f'| habilidades achadas no livro: {len(onde)} '
          f'| divergem do livro: {divergem}')
    print(f'títulos de raça achados no capítulo (bookPage e tier medidos): '
          f'{len(paginas)}/{len(racas)}')
    print('o livro IMPRIMIU (o resto caiu no padrão): ' + ', '.join(
        f'{campo} {n}x' for campo, n in impressos.items()))
    return 1 if divergem or queixas or ausentes else 0


if __name__ == '__main__':
    raise SystemExit(main())
