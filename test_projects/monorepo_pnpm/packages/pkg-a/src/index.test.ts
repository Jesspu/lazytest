import { describe, expect, it } from 'vitest';
import { greet } from './index';

describe('pkg-a greet function', () => {
  it('greets user properly', () => {
    expect(greet('World')).toBe('Hello, World!');
  });
});
