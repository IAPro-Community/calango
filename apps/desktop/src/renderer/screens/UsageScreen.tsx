import { useCallback, useMemo, useState } from 'react';
import type { UsageQuery } from '@calango/sdk-ts';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';

type GroupBy = NonNullable<UsageQuery['groupBy']>;

const GROUPS: GroupBy[] = ['agent', 'routine', 'project', 'node', 'model'];

// Dates from <input type="date"> are local days; the API takes RFC 3339 instants.
// from is inclusive and to is exclusive, as documented in openapi.yaml.
function dayStart(day: string): string | undefined {
  return day === '' ? undefined : new Date(`${day}T00:00:00`).toISOString();
}

export function UsageScreen({ client, t }: ScreenProps) {
  const [groupBy, setGroupBy] = useState<GroupBy>('agent');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');

  const query = useMemo<UsageQuery>(() => ({ groupBy, from: dayStart(from), to: dayStart(to) }), [groupBy, from, to]);
  const load = useCallback(() => client.getUsage(query), [client, query]);
  const { data, error } = usePoll(load, 15_000);
  const rows = data ?? [];

  return (
    <section className="screen">
      <header className="screen-header">
        <h1>{t('navUsage')}</h1>
        <div className="actions">
          <label>
            {t('usageFrom')} <input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
          </label>
          <label>
            {t('usageTo')} <input type="date" value={to} onChange={(event) => setTo(event.target.value)} />
          </label>
          <label>
            {t('usageGroupBy')}{' '}
            <select value={groupBy} onChange={(event) => setGroupBy(event.target.value as GroupBy)}>
              {GROUPS.map((group) => (
                <option key={group} value={group}>
                  {group}
                </option>
              ))}
            </select>
          </label>
        </div>
      </header>
      {error && <p className="error">{error}</p>}
      <div className="card">
        {rows.length === 0 ? (
          <p className="muted">{t('usageEmpty')}</p>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>{t('usageKey')}</th>
                <th>{t('usageTokensIn')}</th>
                <th>{t('usageTokensOut')}</th>
                <th>{t('usageCostKnown')}</th>
                <th>{t('usageCostEstimated')}</th>
                <th>{t('usageUnknownRows')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.key}>
                  <td>{row.key === '' ? '—' : row.key}</td>
                  <td>{row.tokens_in}</td>
                  <td>{row.tokens_out}</td>
                  <td>{row.cost_known_micros}</td>
                  <td>{row.cost_estimated_micros}</td>
                  <td>{row.cost_unknown_rows}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </section>
  );
}
