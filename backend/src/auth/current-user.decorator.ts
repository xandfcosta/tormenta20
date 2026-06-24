import { ExecutionContext, createParamDecorator } from '@nestjs/common';
import type { Request } from 'express';
import type { AuthUser } from './auth.service';

export function extractCurrentUser(ctx: ExecutionContext): AuthUser {
  const req = ctx.switchToHttp().getRequest<Request & { user?: AuthUser }>();
  if (!req.user) {
    throw new Error('CurrentUser used outside JwtAuthGuard scope');
  }
  return req.user;
}

export const CurrentUser = createParamDecorator(
  (_: unknown, ctx: ExecutionContext): AuthUser => extractCurrentUser(ctx),
);
