const { test } = require('node:test');
const assert = require('node:assert/strict');
const cleanup = require('./cleanup-preview-packages.cjs');

const fixture = (options = {}) => {
  const requests = [];
  const lists = [];
  const args = {
    prNumber: 42,
    context: {
      repo: { owner: 'HerrNaN', repo: 'handlat' },
      payload: { repository: { full_name: 'HerrNaN/handlat', owner: { type: options.ownerType || 'User' } } },
    },
    core: { info: () => {} },
    github: {
      rest: { pulls: { get: async () => ({ data: {
        state: options.state || 'closed',
        head: { repo: options.deletedFork ? null : { full_name: options.headRepo || 'HerrNaN/handlat' } },
      } }) } },
      paginate: async (route, parameters) => {
        lists.push({ route, parameters });
        if (options.listStatus) throw Object.assign(new Error('API error'), { status: options.listStatus });
        return Array.from({ length: 101 }, (_, index) => ({ id: index + 1 }));
      },
      request: async (route, parameters) => {
        requests.push({ route, parameters });
        if (options.deleteStatus) throw Object.assign(new Error('API error'), { status: options.deleteStatus });
      },
    },
  };
  return { args, requests, lists };
};

for (const ownerType of ['User', 'Organization']) {
  test(`deletes every version of only PR 42's packages for ${ownerType}`, async () => {
    const { args, requests, lists } = fixture({ ownerType });
    await cleanup(args);
    assert.deepEqual(lists.map(({ parameters }) => parameters.package_name), [
      'charts/preview/pr-42/handlat', 'handlat/preview/pr-42',
    ]);
    assert.equal(requests.length, 202);
    assert.ok(requests.every(({ route }) => route.startsWith(ownerType === 'User' ? 'DELETE /users/' : 'DELETE /orgs/')));
    assert.ok(requests.every(({ parameters }) => lists.some(({ parameters: listed }) => listed.package_name === parameters.package_name)));
    assert.ok(lists.every(({ parameters }) => parameters.per_page === 100));
  });
}

for (const options of [{ state: 'open' }, { headRepo: 'other/handlat' }, { deletedFork: true }]) {
  test(`skips ineligible PR: ${JSON.stringify(options)}`, async () => {
    const { args, requests, lists } = fixture(options);
    await cleanup(args);
    assert.equal(requests.length, 0);
    assert.equal(lists.length, 0);
  });
}

for (const options of [{ listStatus: 404 }, { deleteStatus: 404 }]) {
  test(`tolerates already absent packages or versions: ${JSON.stringify(options)}`, async () => {
    await cleanup(fixture(options).args);
  });
}

for (const options of [{ listStatus: 403 }, { deleteStatus: 403 }, { deleteStatus: 500 }]) {
  test(`reports API failures: ${JSON.stringify(options)}`, async () => {
    await assert.rejects(cleanup(fixture(options).args), /API error/);
  });
}

test('rejects invalid PR numbers before making API calls', async () => {
  for (const prNumber of [0, -1, NaN, 1.5, '42']) {
    await assert.rejects(cleanup({ ...fixture().args, prNumber }), /positive PR number/);
  }
});
