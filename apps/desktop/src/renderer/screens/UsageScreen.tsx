import { useCallback, useMemo, useState } from 'react';
import type { UsageQuery } from '@calango/sdk-ts';
import { Card, Field, Input, PageHeader, Select, StatCard } from '@';
import { usePoll } from '../usePoll';
import type { MessageKey } from '../i18n';
import type { ScreenProps } from './props';

type GroupBy = NonNullable<UsageQuery['groupBy']>;

const GROUPS: GroupBy[] = ['agent', 'routine', 'project', 'node', 'model'];

const GROUP_KEY: Record<GroupBy, MessageKey> = {
  agent: 'usageGroupAgent',
  routine: 'usageGroupRoutine',
  project: 'usageGroupProject',
  node: 'usageGroupNode',
  model: 'usageGroupModel',
};

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

  const totals = rows.reduce(
    (sum, row) => ({
      tokens: sum.tokens + row.tokens_in + row.tokens_out,
      known: sum.known + row.cost_known_micros,
      estimated: sum.estimated + row.cost_estimated_micros,
      unknown: sum.unknown + row.cost_unknown_rows,
    }),
    { tokens: 0, known: 0, estimated: 0, unknown: 0 },
  );

  return (
    <section className="flex flex-col gap-md">
      <PageHeader
        title={t('navUsage')}
        actions={
          <>
            <Field label={t('usageFrom')}>
              <Input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
            </Field>
            <Field label={t('usageTo')}>
              <Input type="date" value={to} onChange={(event) => setTo(event.target.value)} />
            </Field>
            <Field label={t('usageGroupBy')}>
              <Select value={groupBy} onChange={(event) => setGroupBy(event.target.value as GroupBy)}>
                {GROUPS.map((group) => (
                  <option key={group} value={group}>
                    {t(GROUP_KEY[group])}
                  </option>
                ))}
              </Select>
            </Field>
          </>
        }
      />
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      <div className="grid grid-cols-[repeat(auto-fit,minmax(180px,1fr))] gap-sm">
        <StatCard label={t('usageTokensTotal')} value={totals.tokens} />
        <StatCard label={t('usageCostKnown')} value={totals.known} />
        <StatCard label={t('usageCostEstimated')} value={totals.estimated} />
        <StatCard
          label={t('usageUnknownRows')}
          value={totals.unknown}
          hint={totals.unknown > 0 ? t('usageUnknownHint') : undefined}
          hintTone="warn"
        />
      </div>
      <Card title={t(GROUP_KEY[groupBy])}>
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
      </Card>
    </section>
  );
}
