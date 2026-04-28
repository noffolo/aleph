import type { ZodSchema } from 'zod';

/**
 * Validate and parse data from an unknown source (e.g., protobuf/gRPC response)
 * using a Zod schema. Throws ZodError if validation fails.
 */
export function fromProto<T>(schema: ZodSchema<T>, data: unknown): T {
  return schema.parse(data);
}

/**
 * Validate and parse a value using a Zod schema.
 * Throws ZodError if validation fails.
 */
export function validateType<T>(schema: ZodSchema<T>, data: unknown): T {
  return schema.parse(data);
}
