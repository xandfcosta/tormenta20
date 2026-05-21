import { BadRequestException } from '@nestjs/common';
import type { ValidationError } from 'class-validator';

export type FieldErrorMap = Record<string, string[]>;

function flatten(errors: ValidationError[], path = ''): FieldErrorMap {
  const out: FieldErrorMap = {};
  for (const err of errors) {
    const key = path ? `${path}.${err.property}` : err.property;
    if (err.constraints) {
      out[key] = Object.values(err.constraints);
    }
    if (err.children?.length) {
      Object.assign(out, flatten(err.children, key));
    }
  }
  return out;
}

export function validationExceptionFactory(errors: ValidationError[]): BadRequestException {
  const fieldErrors = flatten(errors);
  return new BadRequestException({
    statusCode: 400,
    error: 'Bad Request',
    message: 'Validation failed',
    fieldErrors,
  });
}
