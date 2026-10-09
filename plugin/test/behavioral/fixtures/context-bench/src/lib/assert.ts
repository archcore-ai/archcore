export function assertArgs(condition: unknown, message: string): asserts condition {
  if (!condition) {
    throw new TypeError(message);
  }
}
