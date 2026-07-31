const assert = require('assert');
const { fetchData } = require('../src/api');

describe('API module', () => {
  it('fetches data successfully', () => {
    const res = fetchData();
    assert.strictEqual(res.status, 200);
    assert.strictEqual(res.data, 'ok');
  });
});
