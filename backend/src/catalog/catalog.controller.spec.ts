import { NotFoundException } from '@nestjs/common';
import { CatalogController } from './catalog.controller';
import { CatalogService } from './catalog.service';

describe('CatalogController', () => {
  const controller = new CatalogController(new CatalogService());

  it('lists the resource names at the index', () => {
    expect(controller.index()).toEqual({
      resources: expect.arrayContaining(['spells', 'items', 'bestiary']),
    });
  });

  it('returns the resource payload', () => {
    const spells = controller.resource('spells') as Record<string, unknown>;
    expect(Object.keys(spells).length).toBeGreaterThan(0);
  });

  it('propagates 404 for an unknown resource', () => {
    expect(() => controller.resource('nope')).toThrow(NotFoundException);
  });
});
