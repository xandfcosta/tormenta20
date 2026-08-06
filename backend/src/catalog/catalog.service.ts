import { Injectable, NotFoundException } from '@nestjs/common';
import {
  BESTIARY,
  CATALOG_ITEMS,
  CONDITIONS,
  DEUSES,
  ORIGINS_CATALOG,
  RACAS,
  SPELL_CATALOG,
} from '@tormenta20/t20-data';

/**
 * Read-only reference data (spells, bestiary, items, …) the frontend used to
 * import from `@tormenta20/t20-data` at build time — bundling ~11k LOC of
 * static tables into the browser JS. Serving it as JSON lets the front fetch +
 * cache instead, shrinking the bundle and decoupling data from a front rebuild.
 *
 * Thin wrapper (CLAUDE.md: wrap third-party libs behind a project-owned
 * interface). The data is an in-process constant, not I/O — no DB, no async.
 */
export const CATALOG_RESOURCES = [
  'spells',
  'bestiary',
  'items',
  'conditions',
  'deuses',
  'races',
  'origins',
] as const;

export type CatalogResource = (typeof CATALOG_RESOURCES)[number];

@Injectable()
export class CatalogService {
  private readonly registry: Record<CatalogResource, () => unknown> = {
    spells: () => SPELL_CATALOG,
    bestiary: () => BESTIARY,
    items: () => CATALOG_ITEMS,
    conditions: () => CONDITIONS,
    deuses: () => DEUSES,
    races: () => RACAS,
    origins: () => ORIGINS_CATALOG,
  };

  /** Names the `:resource` route accepts — also the catalog index payload. */
  resources(): readonly CatalogResource[] {
    return CATALOG_RESOURCES;
  }

  /** The static payload for one resource. Unknown name ⇒ 404 with the
   *  offending value + the accepted set (CLAUDE.md exception-message rule). */
  get(resource: string): unknown {
    const fn = this.registry[resource as CatalogResource];
    if (!fn) {
      throw new NotFoundException(
        `unknown catalog resource: "${resource}"; expected one of ${CATALOG_RESOURCES.join(', ')}`,
      );
    }
    return fn();
  }
}
