import { Controller, Get, Header, Param } from '@nestjs/common';
import { CatalogService } from './catalog.service';

/**
 * Public, read-only game reference data. NO JwtAuthGuard on purpose: these are
 * static rulebook tables (spells, bestiary, items…), carry no user data, and
 * are identical for everyone — so they can be cached hard by the browser/CDN.
 * The mutation side of the app stays behind auth; this is pure lookup data.
 */
@Controller('catalog')
export class CatalogController {
  constructor(private readonly catalog: CatalogService) {}

  @Get()
  index() {
    return { resources: this.catalog.resources() };
  }

  @Get(':resource')
  // Static per deploy; safe to cache for an hour and revalidate cheaply.
  @Header('Cache-Control', 'public, max-age=3600')
  resource(@Param('resource') resource: string) {
    return this.catalog.get(resource);
  }
}
