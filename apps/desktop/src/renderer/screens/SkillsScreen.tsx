import { useCallback, useEffect, useState } from 'react';
import {
  ApiError,
  type DiscoveredSkill,
  type ImportedSkill,
  type SkillDiscovery,
  type SkillWarning,
  type SkippedSkill,
} from '@calango/sdk-ts';
import type { MessageKey } from '../i18n';
import type { ScreenProps } from './props';

const ORIGIN_KEYS: Record<DiscoveredSkill['origin'], MessageKey> = {
  claude: 'skillsOriginClaude',
  codex: 'skillsOriginCodex',
  project: 'skillsOriginProject',
};

const WARNING_KEYS: Record<SkillWarning, MessageKey> = {
  no_license: 'skillsWarnNoLicense',
  license_restrictive: 'skillsWarnLicenseRestrictive',
  missing_frontmatter: 'skillsWarnMissingFrontmatter',
  invalid_frontmatter: 'skillsWarnInvalidFrontmatter',
  missing_name: 'skillsWarnMissingName',
  invalid_name: 'skillsWarnInvalidName',
  missing_description: 'skillsWarnMissingDescription',
  description_truncated: 'skillsWarnDescriptionTruncated',
  symlink_skipped: 'skillsWarnSymlinkSkipped',
  special_file_skipped: 'skillsWarnSpecialFileSkipped',
};

const SKIP_KEYS: Record<SkippedSkill['reason'], MessageKey> = {
  symlink_outside_root: 'skillsSkipSymlinkOutsideRoot',
  broken_symlink: 'skillsSkipBrokenSymlink',
  no_skill_md: 'skillsSkipNoSkillMd',
  skill_md_too_large: 'skillsSkipSkillMdTooLarge',
  too_many_files: 'skillsSkipTooManyFiles',
  too_large: 'skillsSkipTooLarge',
  too_deep: 'skillsSkipTooDeep',
  unreadable: 'skillsSkipUnreadable',
};

const ROOT_KEYS: Record<SkillDiscovery['roots'][number]['status'], MessageKey> = {
  ok: 'skillsRootOk',
  missing: 'skillsRootMissing',
  not_directory: 'skillsRootNotDirectory',
  unreadable: 'skillsRootUnreadable',
};

export function SkillsScreen({ client, t }: ScreenProps) {
  const [discovery, setDiscovery] = useState<SkillDiscovery | null>(null);
  const [imported, setImported] = useState<ImportedSkill[] | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [confirmed, setConfirmed] = useState<ReadonlySet<string>>(new Set());

  const load = useCallback(async () => {
    setLoadError(false);
    try {
      const [found, list] = await Promise.all([client.discoverSkills(), client.listImportedSkills()]);
      setDiscovery(found);
      setImported(list);
    } catch {
      setLoadError(true);
    }
  }, [client]);

  useEffect(() => {
    void load();
  }, [load]);

  function toggleConfirm(sourceId: string, value: boolean) {
    setConfirmed((prev) => {
      const next = new Set(prev);
      if (value) {
        next.add(sourceId);
      } else {
        next.delete(sourceId);
      }
      return next;
    });
  }

  async function importSkill(skill: DiscoveredSkill) {
    if (busyId !== null) {
      return;
    }
    setBusyId(skill.source_id);
    setActionError(null);
    try {
      await client.importSkill({
        source_id: skill.source_id,
        hash: skill.hash,
        confirm_license: confirmed.has(skill.source_id),
        replace: skill.import_status === 'update_available',
      });
    } catch (err) {
      if (err instanceof ApiError && err.code === 'skill_changed') {
        setActionError(t('skillsChanged'));
      } else if (err instanceof ApiError && err.code === 'license_confirmation_required') {
        setActionError(t('skillsConfirmLicenseRequired'));
      } else {
        setActionError(err instanceof Error ? err.message : String(err));
      }
    } finally {
      setBusyId(null);
      await load();
    }
  }

  async function removeSkill(skill: ImportedSkill) {
    if (busyId !== null || !window.confirm(t('skillsRemoveConfirm'))) {
      return;
    }
    setBusyId(skill.id);
    setActionError(null);
    try {
      await client.removeImportedSkill(skill.id);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyId(null);
      await load();
    }
  }

  return (
    <section className="screen">
      <header className="screen-header">
        <h1>{t('skillsTitle')}</h1>
        <div className="actions">
          <button type="button" onClick={() => void load()} disabled={busyId !== null}>
            {t('skillsRefresh')}
          </button>
        </div>
      </header>
      <p className="muted">{t('skillsDescription')}</p>
      {loadError && (
        <p className="error" role="alert">
          {t('skillsLoadError')}
        </p>
      )}
      {actionError && (
        <p className="error" role="alert">
          {actionError}
        </p>
      )}
      <section className="card">
        <h2>{t('skillsDiscoveredTitle')}</h2>
        {discovery === null ? (
          loadError ? null : (
            <p className="muted">{t('commonLoading')}</p>
          )
        ) : (
          <>
            <ul className="list">
              {discovery.roots.map((root) => (
                <li key={`${root.origin}:${root.path}`}>
                  <span>
                    {t(ORIGIN_KEYS[root.origin])} · <code>{root.path}</code>
                  </span>
                  <span className="muted">{t(ROOT_KEYS[root.status])}</span>
                </li>
              ))}
            </ul>
            {discovery.roots.length === 0 && <p className="muted">{t('skillsNoRoots')}</p>}
            {discovery.truncated && <p className="muted">{t('skillsTruncated')}</p>}
            {discovery.skills.length === 0 ? (
              <p className="muted">{t('skillsDiscoveredEmpty')}</p>
            ) : (
              <ul className="list">
                {discovery.skills.map((skill) => (
                  <DiscoveredRow
                    key={skill.source_id}
                    skill={skill}
                    t={t}
                    busy={busyId !== null}
                    working={busyId === skill.source_id}
                    confirmed={confirmed.has(skill.source_id)}
                    onConfirm={(value) => toggleConfirm(skill.source_id, value)}
                    onImport={() => void importSkill(skill)}
                  />
                ))}
              </ul>
            )}
            {discovery.skipped.length > 0 && (
              <>
                <h3>{t('skillsSkippedTitle')}</h3>
                <ul className="list">
                  {discovery.skipped.map((item) => (
                    <li key={`${item.origin}:${item.path}`}>
                      <code>{item.path}</code>
                      <span className="muted">{t(SKIP_KEYS[item.reason])}</span>
                    </li>
                  ))}
                </ul>
              </>
            )}
          </>
        )}
      </section>
      <section className="card">
        <h2>{t('skillsImportedTitle')}</h2>
        <p className="muted">{t('skillsNotWired')}</p>
        {imported === null ? null : imported.length === 0 ? (
          <p className="muted">{t('skillsImportedEmpty')}</p>
        ) : (
          <ul className="list">
            {imported.map((skill) => (
              <li key={skill.id}>
                <span>
                  <strong>{skill.name}</strong>
                  <span className="muted">
                    {' '}
                    · {t(ORIGIN_KEYS[skill.origin])} · {skill.license ?? t('skillsNoLicense')}
                    {skill.license_confirmed ? ` · ${t('skillsLicenseConfirmed')}` : ''} ·{' '}
                    {new Date(skill.updated_at).toLocaleString()}
                  </span>
                </span>
                <button
                  type="button"
                  className="danger"
                  onClick={() => void removeSkill(skill)}
                  disabled={busyId !== null}
                >
                  {busyId === skill.id ? t('commonLoading') : t('skillsRemove')}
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </section>
  );
}

interface DiscoveredRowProps {
  skill: DiscoveredSkill;
  t: ScreenProps['t'];
  busy: boolean;
  working: boolean;
  confirmed: boolean;
  onConfirm: (value: boolean) => void;
  onImport: () => void;
}

function DiscoveredRow({ skill, t, busy, working, confirmed, onConfirm, onImport }: DiscoveredRowProps) {
  const needsConfirm = skill.requires_license_confirmation && skill.import_status !== 'imported';
  return (
    <li className="approval">
      <div>
        <strong>{skill.name}</strong>
        <span className="muted"> · {t(ORIGIN_KEYS[skill.origin])}</span>
        {skill.description && <p>{skill.description}</p>}
        <p className="muted">
          {t('skillsLicense')}: {skill.license ?? t('skillsNoLicense')} · <code>{skill.path}</code>
        </p>
        {skill.warnings.length > 0 && (
          <ul className="muted">
            {skill.warnings.map((warning) => (
              <li key={warning}>{t(WARNING_KEYS[warning])}</li>
            ))}
          </ul>
        )}
        {needsConfirm && (
          <label className="check-label">
            <input
              type="checkbox"
              checked={confirmed}
              onChange={(event) => onConfirm(event.target.checked)}
              disabled={busy}
            />
            {t('skillsConfirmLicense')}
          </label>
        )}
      </div>
      {skill.import_status === 'imported' ? (
        <span className="muted">{t('skillsStatusImported')}</span>
      ) : (
        <button type="button" onClick={onImport} disabled={busy || (needsConfirm && !confirmed)}>
          {working
            ? t('commonLoading')
            : skill.import_status === 'update_available'
              ? t('skillsUpdate')
              : t('skillsImport')}
        </button>
      )}
    </li>
  );
}
