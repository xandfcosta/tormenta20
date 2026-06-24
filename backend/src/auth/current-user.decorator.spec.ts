import type { ExecutionContext } from '@nestjs/common';
import type { Request } from 'express';
import type { AuthUser } from './auth.service';
import { extractCurrentUser } from './current-user.decorator';

/**
 * The decorator itself is a Nest paramDecorator factory whose runtime
 * function is buried under metadata. Test the named `extractCurrentUser`
 * function the decorator delegates to.
 */
function makeCtx(req: Partial<Request>): ExecutionContext {
  return {
    switchToHttp: () => ({
      getRequest: <T = Request>() => req as unknown as T,
    }),
  } as unknown as ExecutionContext;
}

describe('extractCurrentUser', () => {
  it('returns request.user when JwtAuthGuard has populated it', () => {
    const user: AuthUser = { id: 1, email: 'a@b.com', name: 'A' };
    expect(extractCurrentUser(makeCtx({ user } as Partial<Request>))).toEqual(
      user,
    );
  });

  it('throws when used outside the JwtAuthGuard scope (no req.user)', () => {
    expect(() => extractCurrentUser(makeCtx({}))).toThrow(
      /CurrentUser used outside JwtAuthGuard scope/,
    );
  });
});
