import { type RefObject, useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import type { Device, DeviceScope } from '@calango/sdk-ts';
import {
  encodePairingPayload,
  formatCountdown,
  formatPairingCode,
  PAIRING_SCOPES,
  secondsLeft,
} from '../../shared/pairing';
import { qrMatrix } from '../../shared/qr';
import { bridge } from '../bridge';
import type { MessageKey, Translate } from '../i18n';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';

const SCOPE_KEYS: Record<DeviceScope, { label: MessageKey; hint: MessageKey }> = {
  full: { label: 'scopeFull', hint: 'scopeFullHint' },
  approve_only: { label: 'scopeApproveOnly', hint: 'scopeApproveOnlyHint' },
  read: { label: 'scopeRead', hint: 'scopeReadHint' },
};

const PLATFORM_KEYS: Record<Device['platform'], MessageKey> = {
  desktop: 'devicesPlatformDesktop',
  ios: 'devicesPlatformIos',
  android: 'devicesPlatformAndroid',
  web: 'devicesPlatformWeb',
};

interface IssuedCode {
  code: string;
  scope: DeviceScope;
  expiresAtMs: number;
  uri: string;
  /** Devices that existed when the code was issued; a new id afterwards means it was used. */
  knownIds: Set<string>;
}

type Announcement = 'pairingOneMinuteLeft' | 'pairingExpired' | 'pairingPaired' | 'devicesRevokeDone';

export function DevicesScreen({ client, t }: ScreenProps) {
  const load = useCallback(() => client.listDevices(), [client]);
  const { data: devices, error: loadError, refresh } = usePoll(load, 3_000);
  const [scope, setScope] = useState<DeviceScope>('full');
  const [issued, setIssued] = useState<IssuedCode | null>(null);
  const [generating, setGenerating] = useState(false);
  const [generateError, setGenerateError] = useState(false);
  const [now, setNow] = useState(() => Date.now());
  const [announcement, setAnnouncement] = useState<Announcement | null>(null);
  const [nodeLabel, setNodeLabel] = useState<string | undefined>(undefined);
  const [confirmingId, setConfirmingId] = useState<string | null>(null);
  const [revokingId, setRevokingId] = useState<string | null>(null);
  const [revokeError, setRevokeError] = useState(false);
  const confirmRef = useRef<HTMLButtonElement | null>(null);
  const announcedMinute = useRef(false);
  const pairingTitleId = useId();
  const devicesTitleId = useId();

  useEffect(() => {
    let active = true;
    void bridge.getNodeInfo().then(
      (info) => {
        if (active) setNodeLabel(info.label);
      },
      () => undefined,
    );
    return () => {
      active = false;
    };
  }, []);

  const remaining = issued === null ? 0 : secondsLeft(issued.expiresAtMs, now);
  const expired = issued !== null && remaining === 0;

  // The countdown ticks only while a live code is on screen.
  useEffect(() => {
    if (issued === null || expired) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1_000);
    return () => window.clearInterval(timer);
  }, [issued, expired]);

  useEffect(() => {
    if (issued === null) return;
    if (expired) {
      setAnnouncement('pairingExpired');
    } else if (remaining <= 60 && !announcedMinute.current) {
      announcedMinute.current = true;
      setAnnouncement('pairingOneMinuteLeft');
    }
  }, [issued, expired, remaining]);

  // A device that was not there when the code was issued means the code was redeemed.
  useEffect(() => {
    if (issued === null || expired || devices === null) return;
    if (devices.some((device) => !issued.knownIds.has(device.id) && !device.revoked_at)) {
      setIssued(null);
      setAnnouncement('pairingPaired');
    }
  }, [devices, issued, expired]);

  useEffect(() => {
    if (confirmingId !== null) confirmRef.current?.focus();
  }, [confirmingId]);

  const qr = useMemo(() => (issued === null ? null : qrMatrix(issued.uri)), [issued]);

  async function generate() {
    setGenerating(true);
    setGenerateError(false);
    setAnnouncement(null);
    try {
      // A fresh list, so a device paired just before this code is not taken for its redemption.
      const current = await client.listDevices();
      const pairing = await client.createPairingCode(scope);
      const expiresAtMs = Date.parse(pairing.expires_at);
      const uri = encodePairingPayload({
        code: pairing.code,
        scope: pairing.scope,
        expiresAt: Math.floor(expiresAtMs / 1000),
        ...(nodeLabel === undefined ? {} : { node: nodeLabel }),
      });
      announcedMinute.current = false;
      setNow(Date.now());
      setIssued({
        code: pairing.code,
        scope: pairing.scope,
        expiresAtMs,
        uri,
        knownIds: new Set(current.map((device) => device.id)),
      });
    } catch {
      setIssued(null);
      setGenerateError(true);
    } finally {
      setGenerating(false);
    }
  }

  async function revoke(deviceId: string) {
    setRevokingId(deviceId);
    setRevokeError(false);
    setAnnouncement(null);
    try {
      await client.revokeDevice(deviceId);
      setConfirmingId(null);
      setAnnouncement('devicesRevokeDone');
      refresh();
    } catch {
      setRevokeError(true);
    } finally {
      setRevokingId(null);
    }
  }

  return (
    <section className="screen">
      <header className="screen-header">
        <h1>{t('navDevices')}</h1>
      </header>
      <p className="muted">{t('devicesDescription')}</p>

      <section className="card pairing" aria-labelledby={pairingTitleId}>
        <h2 id={pairingTitleId}>{t('pairingTitle')}</h2>
        <p className="muted">{t('pairingDescription')}</p>
        <fieldset className="scope-picker" disabled={generating}>
          <legend>{t('pairingScopeLabel')}</legend>
          {PAIRING_SCOPES.map((value) => (
            <label key={value} className="check-label">
              <input
                type="radio"
                name="pairing-scope"
                value={value}
                checked={scope === value}
                onChange={() => setScope(value)}
              />
              <span>
                {t(SCOPE_KEYS[value].label)}
                <span className="muted"> · {t(SCOPE_KEYS[value].hint)}</span>
              </span>
            </label>
          ))}
        </fieldset>
        <div className="actions">
          <button type="button" onClick={() => void generate()} disabled={generating}>
            {generating ? t('pairingGenerating') : issued === null ? t('pairingGenerate') : t('pairingGenerateAnother')}
          </button>
        </div>
        {generateError && (
          <p className="error" role="alert">
            {t('pairingGenerateError')}
          </p>
        )}
        {issued !== null && qr !== null && !expired && (
          <div className="pairing-code">
            <svg
              className="qr"
              role="img"
              aria-label={t('pairingQrLabel')}
              viewBox={`0 0 ${qr.viewBox} ${qr.viewBox}`}
              shapeRendering="crispEdges"
            >
              <rect className="qr-light" width={qr.viewBox} height={qr.viewBox} />
              <path className="qr-dark" d={qr.path} />
            </svg>
            <div className="pairing-details">
              <p>
                {t('pairingCodeLabel')}
                <br />
                <code className="pairing-code-text">{formatPairingCode(issued.code)}</code>
              </p>
              <p>
                {t('devicesColScope')}: {t(SCOPE_KEYS[issued.scope].label)}
              </p>
              <p role="timer" aria-live="off" aria-atomic="true">
                {t('pairingExpiresIn')} <strong>{formatCountdown(remaining)}</strong>
              </p>
            </div>
          </div>
        )}
        {expired && <p className="error">{t('pairingExpired')}</p>}
        <p className="muted">{t('pairingTransportNote')}</p>
      </section>

      <p className="visually-hidden" aria-live="polite" aria-atomic="true">
        {announcement === null ? '' : t(announcement)}
      </p>

      <section className="card" aria-labelledby={devicesTitleId}>
        <h2 id={devicesTitleId}>{t('devicesListTitle')}</h2>
        {loadError !== null && (
          <p className="error" role="alert">
            {t('devicesLoadError')}
          </p>
        )}
        {revokeError && (
          <p className="error" role="alert">
            {t('devicesRevokeError')}
          </p>
        )}
        {devices === null ? (
          loadError === null && <p className="muted">{t('commonLoading')}</p>
        ) : devices.length === 0 ? (
          <p className="muted">{t('devicesEmpty')}</p>
        ) : (
          <DeviceTable
            devices={devices}
            t={t}
            confirmingId={confirmingId}
            revokingId={revokingId}
            confirmRef={confirmRef}
            onAsk={setConfirmingId}
            onRevoke={(id) => void revoke(id)}
          />
        )}
      </section>
    </section>
  );
}

interface DeviceTableProps {
  devices: Device[];
  t: Translate;
  confirmingId: string | null;
  revokingId: string | null;
  confirmRef: RefObject<HTMLButtonElement | null>;
  onAsk: (id: string | null) => void;
  onRevoke: (id: string) => void;
}

function formatDate(value: string | undefined, fallback: string): string {
  if (value === undefined || value === '') return fallback;
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? fallback : parsed.toLocaleString();
}

function DeviceTable({ devices, t, confirmingId, revokingId, confirmRef, onAsk, onRevoke }: DeviceTableProps) {
  // Active devices first, then by pairing date, newest first.
  const sorted = [...devices].sort((a, b) => {
    const revoked = Number(Boolean(a.revoked_at)) - Number(Boolean(b.revoked_at));
    return revoked !== 0 ? revoked : b.created_at.localeCompare(a.created_at);
  });
  return (
    <table className="table">
      <thead>
        <tr>
          <th scope="col">{t('devicesColName')}</th>
          <th scope="col">{t('devicesColPlatform')}</th>
          <th scope="col">{t('devicesColScope')}</th>
          <th scope="col">{t('devicesColCreated')}</th>
          <th scope="col">{t('devicesColLastSeen')}</th>
          <th scope="col">{t('devicesColStatus')}</th>
          <th scope="col">{t('devicesColActions')}</th>
        </tr>
      </thead>
      <tbody>
        {sorted.map((device) => {
          const revoked = Boolean(device.revoked_at);
          return (
            <tr key={device.id} className={revoked ? 'revoked' : undefined}>
              <th scope="row">{device.name}</th>
              <td>{t(PLATFORM_KEYS[device.platform])}</td>
              <td>{t(SCOPE_KEYS[device.scope].label)}</td>
              <td>{formatDate(device.created_at, '')}</td>
              <td>{formatDate(device.last_seen_at, t('devicesNeverSeen'))}</td>
              <td>{revoked ? t('devicesStatusRevoked') : t('devicesStatusActive')}</td>
              <td>
                {revoked ? null : confirmingId === device.id ? (
                  <fieldset className="confirm">
                    <legend className="visually-hidden">{`${t('devicesRevoke')}: ${device.name}`}</legend>
                    <p>{t('devicesRevokeConfirm')}</p>
                    <div className="actions">
                      <button
                        ref={confirmRef}
                        type="button"
                        className="danger"
                        disabled={revokingId !== null}
                        onClick={() => onRevoke(device.id)}
                      >
                        {t('devicesRevokeConfirmAction')}
                      </button>
                      <button type="button" disabled={revokingId !== null} onClick={() => onAsk(null)}>
                        {t('commonCancel')}
                      </button>
                    </div>
                  </fieldset>
                ) : (
                  <button
                    type="button"
                    className="danger"
                    aria-label={`${t('devicesRevoke')}: ${device.name}`}
                    disabled={revokingId !== null}
                    onClick={() => onAsk(device.id)}
                  >
                    {t('devicesRevoke')}
                  </button>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
