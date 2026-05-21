import { Injectable, OnModuleDestroy, OnModuleInit } from '@nestjs/common';
import { PrismaBetterSqlite3 } from '@prisma/adapter-better-sqlite3';
import { PrismaClient } from '../generated/prisma/client';

function resolveSqlitePath(databaseUrl: string | undefined): string {
  if (!databaseUrl) {
    throw new Error(
      `DATABASE_URL is required (got: ${String(databaseUrl)}). Expected "file:<path>" or ":memory:".`,
    );
  }
  if (databaseUrl === ':memory:') return ':memory:';
  if (databaseUrl.startsWith('file:')) return databaseUrl.slice('file:'.length);
  return databaseUrl;
}

@Injectable()
export class PrismaService extends PrismaClient implements OnModuleInit, OnModuleDestroy {
  constructor() {
    super({
      adapter: new PrismaBetterSqlite3({
        url: resolveSqlitePath(process.env.DATABASE_URL),
      }),
    });
  }

  async onModuleInit(): Promise<void> {
    await this.$connect();
  }

  async onModuleDestroy(): Promise<void> {
    await this.$disconnect();
  }
}
