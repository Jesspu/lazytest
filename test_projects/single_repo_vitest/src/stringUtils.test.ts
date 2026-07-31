import { describe, expect, it } from 'vitest';
import { capitalize, reverse } from './stringUtils';

describe('stringUtils module', () => {
  it('capitalizes string', () => {
    expect(capitalize('hello')).toBe('Hello');
  });

  it('reverses string', () => {
    expect(reverse('hello')).toBe('olleh');
  });
});
