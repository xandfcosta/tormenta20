#!/usr/bin/env python3
"""Passa o `id` de raça e origem para a forma de slug, e arrasta quem aponta.

Uso: python3 scripts/slugify-ids.py [--conferir]

O `uid` é a identidade (ALE-402); o `id` é o ENDEREÇO, e endereço não carrega
rótulo. Antes desta passagem `race-defs.json` dizia `Humano` e `races.json`
dizia `humano` para a mesma raça — cinquenta e dois conceitos com duas grafias,
e nada no dado dizendo que eram o mesmo (ALE-404).

O ESCOPO é uma tabela explícita e não um predicado. A cunhagem de uid tentou
predicado duas vezes ("tem nome e descrição") e as duas erraram o conjunto em
silêncio; aqui quem decide o que muda está escrito, arquivo por arquivo.

Rodar é ato deliberado como o `genoracle`: ele reescreve o catálogo no lugar, e
o `--conferir` mostra o que faria sem escrever.
"""

import json
import re
import sys
import unicodedata
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DADOS = RAIZ / "engine-go" / "domain" / "catalog" / "data"
PARIDADE = RAIZ / "engine-go" / "parity"


def slug(texto: str) -> str:
    """`Herói Camponês` -> `heroi-campones`. Sem acento, sem caixa, sem espaço."""
    sem_acento = "".join(
        c for c in unicodedata.normalize("NFD", texto) if unicodedata.category(c) != "Mn"
    )
    return re.sub(r"[^a-z0-9]+", "-", sem_acento.lower()).strip("-")


def le(caminho: Path):
    return json.loads(caminho.read_text(encoding="utf-8"))


def escreve(caminho: Path, dado, original: str) -> None:
    """Escreve preservando o ESTILO do arquivo — compacto ou indentado.

    Os catálogos são compactos e as fichas de paridade são indentadas.
    Reformatar tudo num estilo só produziria um diff de 2.700 linhas onde a
    mudança real são 367 valores, e ninguém revisa um diff desses.
    """
    if "\n" in original.strip():
        texto = json.dumps(dado, ensure_ascii=False, indent=2)
    else:
        texto = json.dumps(dado, ensure_ascii=False, separators=(",", ":"))
    caminho.write_text(texto + ("\n" if original.endswith("\n") else ""), encoding="utf-8")


def mapa_de_renome(entradas) -> dict[str, str]:
    """De `Humano` para `humano`, só para os ids que ainda não são slug."""
    fora = {}
    for e in entradas:
        antigo = e["id"]
        novo = slug(antigo)
        if antigo != novo:
            fora[antigo] = novo
    return fora


def troca_em_toda_arvore(no, campos: set[str], mapa: dict[str, str]) -> int:
    """Troca o VALOR dos campos nomeados, onde quer que eles estejam.

    Os campos são nomeados porque um `id` de benefício e um `raceId` são coisas
    diferentes que podem ter o mesmo texto; trocar por valor arrastaria junto
    uma descrição que por acaso dissesse `Humano`.
    """
    trocas = 0
    if isinstance(no, dict):
        for chave, valor in no.items():
            if chave in campos and isinstance(valor, str) and valor in mapa:
                no[chave] = mapa[valor]
                trocas += 1
            elif isinstance(valor, str):
                dentro, n = troca_em_json_embutido(valor, campos, mapa)
                if n:
                    no[chave] = dentro
                    trocas += n
            else:
                trocas += troca_em_toda_arvore(valor, campos, mapa)
    elif isinstance(no, list):
        for item in no:
            trocas += troca_em_toda_arvore(item, campos, mapa)
    return trocas


def troca_em_json_embutido(texto: str, campos: set[str], mapa: dict[str, str]):
    """A ficha guarda escolha como JSON DENTRO de uma string, e o ponteiro lá
    dentro não se parece com ponteiro.

    `secondaryRaceChoices` é `"[{\\"race\\":\\"Lefou\\",…}]"` — uma string para
    quem lê o arquivo de fora, e o único lugar onde a raça secundária é
    nomeada. A varredura que só olhava valores de campo passou por cima dela, e
    o que apareceu foi o Kharvos perdendo o −1 de Carisma da Deformidade.
    """
    if texto.strip()[:1] not in "[{":
        return texto, 0
    try:
        dentro = json.loads(texto)
    except json.JSONDecodeError:
        return texto, 0
    n = troca_em_toda_arvore(dentro, campos, mapa)
    if n == 0:
        return texto, 0
    return json.dumps(dentro, ensure_ascii=False, separators=(",", ":")), n


def main() -> int:
    conferir = "--conferir" in sys.argv

    racas = le(DADOS / "race-defs.json")
    origens = le(DADOS / "origins.json")

    renome_raca = mapa_de_renome(racas)
    renome_origem = mapa_de_renome(origens)
    renome_beneficio = {}
    for o in origens:
        renome_beneficio.update(mapa_de_renome(o.get("benefits", [])))

    print(f"raças a renomear:      {len(renome_raca)}")
    print(f"origens a renomear:    {len(renome_origem)}")
    print(f"benefícios a renomear: {len(renome_beneficio)}")

    # `id` e `raceId` no catálogo de raça; `id` no de origem. Os campos da
    # FICHA (`race`, `origin`) valem para os arquivos de paridade, onde a ficha
    # guardada mora.
    todos_os_ids = renome_raca | renome_origem | renome_beneficio
    alvos: list[tuple[Path, set[str], dict[str, str]]] = [
        (DADOS / "race-defs.json", {"id", "raceId"}, renome_raca),
        (DADOS / "origins.json", {"id"}, renome_origem | renome_beneficio),
        # O `_catalogs.json` é DESPEJO DE CATÁLOGO e não ficha: ele repete os
        # mesmos verbetes, e o `TestDumpAgreesWithEmbeddedCatalog` os compara
        # campo a campo. Tratá-lo como as fichas deixou os ids de ORIGEM para
        # trás e o guarda acusou — as raças tinham passado e as origens não.
        (PARIDADE / "_catalogs.json", {"id", "raceId"}, todos_os_ids),
    ]
    for fixture in sorted(PARIDADE.glob("*.json")):
        if fixture.name == "_catalogs.json":
            continue
        alvos.append((fixture, {"race", "origin"}, renome_raca | renome_origem))

    total = 0
    por_arquivo: dict[Path, int] = {}
    conteudo: dict[Path, object] = {}
    texto_original: dict[Path, str] = {}
    for caminho, campos, mapa in alvos:
        if caminho not in conteudo:
            texto_original[caminho] = caminho.read_text(encoding="utf-8")
            conteudo[caminho] = json.loads(texto_original[caminho])
        dado = conteudo[caminho]
        n = troca_em_toda_arvore(dado, campos, mapa)
        por_arquivo[caminho] = por_arquivo.get(caminho, 0) + n
        total += n

    for caminho, n in sorted(por_arquivo.items()):
        if n:
            print(f"  {caminho.relative_to(RAIZ)}: {n} trocas")

    if conferir:
        print(f"\n{total} trocas — nada escrito (--conferir)")
        return 0

    for caminho, dado in conteudo.items():
        if por_arquivo.get(caminho):
            escreve(caminho, dado, texto_original[caminho])
    print(f"\n{total} trocas escritas")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
