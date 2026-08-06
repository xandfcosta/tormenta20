import { NotFoundException } from '@nestjs/common';
import { CATALOG_RESOURCES, CatalogService } from './catalog.service';

describe('CatalogService', () => {
  const service = new CatalogService();

  it('exposes every declared resource in the index', () => {
    expect(service.resources()).toEqual(CATALOG_RESOURCES);
  });

  it('returns non-empty static data for each resource', () => {
    for (const name of CATALOG_RESOURCES) {
      const data = service.get(name);
      const size = Array.isArray(data)
        ? data.length
        : Object.keys(data as object).length;
      expect(size).toBeGreaterThan(0);
    }
  });

  it('serves the spell catalog as an id-keyed record', () => {
    const spells = service.get('spells') as Record<string, { id: string }>;
    const first = Object.values(spells)[0];
    expect(first).toHaveProperty('id');
    expect(spells[first.id]).toBe(first);
  });

  it('serves the item catalog as an array', () => {
    expect(Array.isArray(service.get('items'))).toBe(true);
  });

  it('throws 404 naming the bad value + accepted set on unknown resource', () => {
    expect(() => service.get('spellz')).toThrow(NotFoundException);
    expect(() => service.get('spellz')).toThrow(/"spellz".*spells, bestiary/s);
  });

  it('is JSON-serializable (no functions/cycles leak into the payload)', () => {
    for (const name of CATALOG_RESOURCES) {
      expect(() => JSON.stringify(service.get(name))).not.toThrow();
    }
  });
});
