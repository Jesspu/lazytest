const { add, subtract } = require('./math');

describe('math module', () => {
  test('adds two numbers', () => {
    expect(add(1, 2)).toBe(3);
  });

  test('subtracts two numbers', () => {
    expect(subtract(5, 2)).toBe(3);
  });
});
