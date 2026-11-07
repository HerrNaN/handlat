import { test } from 'bun:test';
import assert from 'node:assert/strict';
import { getOctokit } from '@actions/github';
import { cleanup, type CleanupOptions } from './cleanup-preview-packages.ts';

type FixtureOptions = {
  ownerType?: string;
  state?: string;
  deletedFork?: boolean;
  headRepo?: string;
  listStatus?: number;
  deleteStatus?: number;
};

const fixture = (options: FixtureOptions = {}) => {
  const requests: Request[] = [];
  const args: CleanupOptions = {
    prNumber: 42,
    context: {
      repo: { owner: 'HerrNaN', repo: 'handlat' },
      payload: { repository: { name: 'handlat', full_name: 'HerrNaN/handlat', owner: { login: 'HerrNaN', type: options.ownerType || 'User' } } },
    },
    core: { info: () => {} },
    github: getOctokit('test-token', { request: { fetch: async (input: string, init?: RequestInit) => {
      const request = new Request(input, init);
      requests.push(request);
      const url = new URL(request.url);
      if (url.pathname.endsWith('/pulls/42')) {
        return Response.json({
          state: options.state || 'closed',
          head: { repo: options.deletedFork ? null : { full_name: options.headRepo || 'HerrNaN/handlat' } },
        });
      }
      if (request.method === 'DELETE') {
        return options.deleteStatus
          ? Response.json({ message: 'API error' }, { status: options.deleteStatus })
          : new Response(null, { status: 204 });
      }
      if (options.listStatus) return Response.json({ message: 'API error' }, { status: options.listStatus });
      const secondPage = url.searchParams.get('page') === '2';
      const versions = Array.from({ length: secondPage ? 1 : 100 }, (_, index) => ({
        id: secondPage ? 101 : index + 1,
        metadata: { container: { tags: ['0.1.0-pr.42.20261107120000'] } },
      }));
      if (secondPage && decodeURIComponent(url.pathname).includes('/charts/preview/handlat/')) {
        for (const [index, tags] of [
          ['0.1.0-pr.43.20261107120000'],
          ['0.1.0'],
          [],
          ['0.1.0-pr.42.20261107120000', '0.1.0-pr.43.20261107120000'],
        ].entries()) versions.push({ id: 102 + index, metadata: { container: { tags } } });
      }
      url.searchParams.set('page', '2');
      return Response.json(versions, { headers: secondPage ? {} : { link: `<${url}>; rel="next"` } });
    } } }),
  };
  return { args, requests };
};

for (const ownerType of ['User', 'Organization']) {
  test(`deletes every page of only PR 42's artifacts for ${ownerType}`, async () => {
    const { args, requests } = fixture({ ownerType });
    await cleanup(args);
    const deleted = requests.filter(request => request.method === 'DELETE');
    const scope = ownerType === 'User' ? 'users' : 'orgs';
    const paths = deleted.map(request => decodeURIComponent(new URL(request.url).pathname));
    const prefixes = [
      `/${scope}/HerrNaN/packages/container/charts/preview/handlat/versions/`,
      `/${scope}/HerrNaN/packages/container/charts/preview/pr-42/handlat/versions/`,
      `/${scope}/HerrNaN/packages/container/handlat/preview/pr-42/versions/`,
    ];
    assert.equal(deleted.length, 303);
    for (const prefix of prefixes) {
      assert.equal(paths.filter(path => path.startsWith(prefix)).length, 101);
      assert.ok(paths.includes(`${prefix}101`));
    }
    assert.ok(paths.every(path => !/\/versions\/10[2-5]$/.test(path)));
  });
}

for (const options of [{ state: 'open' }, { headRepo: 'other/handlat' }, { deletedFork: true }]) {
  test(`skips ineligible PR: ${JSON.stringify(options)}`, async () => {
    const { args, requests } = fixture(options);
    await cleanup(args);
    assert.equal(requests.length, 1);
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
    const { args, requests } = fixture();
    await assert.rejects(cleanup({ ...args, prNumber: prNumber as number }), /positive PR number/);
    assert.equal(requests.length, 0);
  }
});
