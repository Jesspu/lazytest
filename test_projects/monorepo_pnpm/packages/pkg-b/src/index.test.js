const { multiply } = require('./index');

describe('pkg-b multiply function', () => {
  test('multiplies numbers', () => {
    expect(multiply(3, 4)).toBe(12);
  });
});
