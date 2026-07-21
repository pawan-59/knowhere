import { api } from '../lib/api'
import { Card, Stat, Badge, Spinner, ErrorBox, useAsync, fmtDate } from '../components/ui'

export default function Devtron() {
  const summary = useAsync(api.devtronSummary, [])
  const deployments = useAsync(api.devtronDeployments, [])

  const version = summary.data?.version

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-bold text-slate-900">Devtron — Releases & Deployments</h1>
        <p className="text-sm text-slate-500">App rollouts and the Devtron version in production.</p>
      </header>

      {summary.error && <ErrorBox error={summary.error} />}
      {summary.loading ? (
        <Spinner />
      ) : summary.data ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <Stat label="Devtron version" value={version?.version || '—'} hint={version?.gitCommit?.slice(0, 7)} />
          <Stat label="Deployments" value={summary.data.totalDeployments} />
          <Stat label="Healthy" value={summary.data.healthy} tone="good" />
          <Stat label="Degraded" value={summary.data.degraded} tone="bad" />
        </div>
      ) : null}

      <Card title="Deployment status">
        {deployments.error ? (
          <ErrorBox error={deployments.error} />
        ) : deployments.loading ? (
          <Spinner />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs uppercase text-slate-400">
                  <th className="py-2 pr-3">Application</th>
                  <th className="py-2 pr-3">Environment</th>
                  <th className="py-2 pr-3">Status</th>
                  <th className="py-2 pr-3">Version</th>
                  <th className="py-2 pr-3">Deployed by</th>
                  <th className="py-2">Last deployed</th>
                </tr>
              </thead>
              <tbody>
                {(deployments.data.deployments || []).map((d, i) => (
                  <tr key={i} className="border-b border-slate-50 hover:bg-slate-50">
                    <td className="py-2 pr-3 font-medium text-slate-800">{d.appName}</td>
                    <td className="py-2 pr-3 text-slate-600">{d.environmentName}</td>
                    <td className="py-2 pr-3"><Badge>{d.deploymentStatus}</Badge></td>
                    <td className="py-2 pr-3 font-mono text-xs text-slate-500">{d.deployedVersion || '—'}</td>
                    <td className="py-2 pr-3 text-slate-600">{d.deployedBy || '—'}</td>
                    <td className="py-2 text-slate-500">{fmtDate(d.lastDeployedTime)}</td>
                  </tr>
                ))}
                {deployments.data.deployments?.length === 0 && (
                  <tr><td colSpan="6" className="py-6 text-center text-slate-400">No deployments found.</td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  )
}
