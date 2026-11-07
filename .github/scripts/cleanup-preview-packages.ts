import * as core from '@actions/core';
import { context, getOctokit } from '@actions/github';

export type CleanupOptions = {
  github: ReturnType<typeof getOctokit>;
  context: Pick<typeof context, 'repo' | 'payload'>;
  core: Pick<typeof core, 'info'>;
  prNumber: number;
};

const isNotFound = (error: unknown): boolean =>
  typeof error === 'object' && error !== null && 'status' in error && error.status === 404;

export const cleanup = async ({ github, context, core, prNumber }: CleanupOptions) => {
  if (!Number.isSafeInteger(prNumber) || prNumber < 1) {
    throw new Error('Expected a positive PR number');
  }

  const { owner, repo } = context.repo;
  const { data: pr } = await github.rest.pulls.get({ owner, repo, pull_number: prNumber });
  if (pr.state !== 'closed' || pr.head.repo?.full_name !== context.payload.repository?.full_name) {
    core.info('Skipping an open PR or a PR from another repository');
    return;
  }

  const organization = context.payload.repository?.owner?.type === 'Organization';
  const packages = [
    'charts/preview/handlat',
    `charts/preview/pr-${prNumber}/handlat`,
    `${repo.toLowerCase()}/preview/pr-${prNumber}`,
  ];

  for (const packageName of packages) {
    const parameters = { package_type: 'container' as const, package_name: packageName };
    let versions;
    try {
      versions = organization
        ? await github.paginate(github.rest.packages.getAllPackageVersionsForPackageOwnedByOrg, {
          ...parameters, org: owner, per_page: 100,
        })
        : await github.paginate(github.rest.packages.getAllPackageVersionsForPackageOwnedByUser, {
          ...parameters, username: owner, per_page: 100,
        });
    } catch (error) {
      if (!isNotFound(error)) throw error;
      core.info(`No package found: ${packageName}`);
      continue;
    }

    const chartVersion = new RegExp(`^[0-9]+\\.[0-9]+\\.[0-9]+-pr\\.${prNumber}\\.[0-9]{14}$`);
    for (const version of versions) {
      if (packageName === 'charts/preview/handlat') {
        const tags = version.metadata?.container?.tags ?? [];
        if (tags.length === 0 || !tags.every(tag => chartVersion.test(tag))) continue;
      }
      const deletion = { ...parameters, package_version_id: version.id };
      try {
        if (organization) {
          await github.rest.packages.deletePackageVersionForOrg({ ...deletion, org: owner });
        } else {
          await github.rest.packages.deletePackageVersionForUser({ ...deletion, username: owner });
        }
      } catch (error) {
        if (!isNotFound(error)) throw error;
      }
    }
    core.info(`Cleaned up ${versions.length} versions of ${packageName}`);
  }
};

if (import.meta.main) {
  const token = process.env.GITHUB_TOKEN;
  if (!token) throw new Error('GITHUB_TOKEN is required');
  await cleanup({ github: getOctokit(token), context, core, prNumber: Number(process.env.PR_NUMBER) });
}
