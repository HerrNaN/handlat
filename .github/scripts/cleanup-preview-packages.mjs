export const cleanup = async ({ github, context, core, prNumber }) => {
  if (!Number.isSafeInteger(prNumber) || prNumber < 1) {
    throw new Error('Expected a positive PR number');
  }

  const { owner, repo } = context.repo;
  const { data: pr } = await github.rest.pulls.get({ owner, repo, pull_number: prNumber });
  if (pr.state !== 'closed' || pr.head.repo?.full_name !== context.payload.repository.full_name) {
    core.info('Skipping an open PR or a PR from another repository');
    return;
  }

  const organization = context.payload.repository.owner.type === 'Organization';
  const scope = organization ? 'orgs/{org}' : 'users/{username}';
  const ownerParameter = organization ? { org: owner } : { username: owner };
  const route = `/${scope}/packages/{package_type}/{package_name}/versions`;
  const packages = [
    `charts/preview/pr-${prNumber}/handlat`,
    `${repo.toLowerCase()}/preview/pr-${prNumber}`,
  ];

  for (const packageName of packages) {
    const parameters = { ...ownerParameter, package_type: 'container', package_name: packageName };
    let versions;
    try {
      versions = await github.paginate(`GET ${route}`, { ...parameters, per_page: 100 });
    } catch (error) {
      if (error.status !== 404) throw error;
      core.info(`No package found: ${packageName}`);
      continue;
    }

    for (const version of versions) {
      try {
        await github.request(`DELETE ${route}/{package_version_id}`, {
          ...parameters, package_version_id: version.id,
        });
      } catch (error) {
        if (error.status !== 404) throw error;
      }
    }
    core.info(`Cleaned up ${versions.length} versions of ${packageName}`);
  }
};
