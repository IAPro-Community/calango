// Internal browser (F5-T01): tabs as WebContentsView, one persistent partition per agent.
// Project preview tabs (F4-T06) live beside them in their own partition, confined to one
// loopback origin. Electron objects come in through BrowserDeps, so tests can run this
// under plain Node.
import { randomUUID } from 'node:crypto';
import type { Session, WebContents, WebContentsView, WebContentsViewConstructorOptions } from 'electron';
import type { BrowserTab, PreviewBlockedEvent } from '../../shared/ipc';
import { hardenSession, hardenWebContents, webPreferencesFor } from './harden';
import { assertAgentId, isNavigableUrl, normalizeAddress, partitionForAgent, type Rect } from './policy';
import { isProjectId, isWithinPreviewOrigin, PREVIEW_OWNER, parsePreviewTarget, partitionForPreview } from './preview';

/** Where tab views are shown: the main window's content view. */
export interface ViewHost {
  add(view: WebContentsView): void;
  remove(view: WebContentsView): void;
}

/**
 * Extension point for later F5 work: F5-T02 (#92) attaches CDP and the take-over flow to a
 * tab's WebContents; F5-T03 (#93) inspects loads to detect barriers. Observers see only tab
 * metadata and the WebContents in the main process; nothing is exposed to the renderer.
 */
export interface BrowserObserver {
  tabCreated?(tab: BrowserTab, contents: WebContents): void;
  tabLoaded?(tab: BrowserTab, contents: WebContents): void;
  tabClosed?(tabId: string, agentId: string): void;
}

export interface BrowserDeps {
  profileKey: Uint8Array;
  fromPartition(partition: string): Session;
  createView(options: WebContentsViewConstructorOptions): WebContentsView;
  host: ViewHost;
  onTabsChanged(agentId: string, tabs: BrowserTab[]): void;
  onBlocked?(agentId: string, url: string): void;
  /** A preview tab tried to leave its origin; the renderer may offer the system browser. */
  onPreviewBlocked?(event: PreviewBlockedEvent): void;
  /** Opens a URL in the system browser (Electron shell.openExternal). */
  openExternal?(url: string): Promise<void> | void;
  /** Loopback ports a preview may never open, such as the Calango daemon's own port. */
  reservedPorts?: readonly number[];
}

export interface OpenPreviewOptions {
  autoReload?: boolean;
  /** Hook for the project registry (#82): each project gets its own preview profile. */
  projectId?: string | null;
}

interface PreviewState {
  origin: string;
  autoReload: boolean;
  projectId: string | null;
  retryTimer: ReturnType<typeof setTimeout> | null;
  retryDelay: number;
}

interface Tab {
  id: string;
  /** Agent id, or PREVIEW_OWNER for project preview tabs. */
  agentId: string;
  partition: string;
  view: WebContentsView;
  preview?: PreviewState;
}

const MAX_TABS_PER_AGENT = 20;
const MAX_PENDING_EXTERNAL = 20;
const RETRY_FIRST_MS = 2_000;
const RETRY_MAX_MS = 10_000;
// Chromium net errors meaning "the dev server is not answering right now" (closed, reset,
// refused, aborted, unreachable, empty response, timed out). Others, such as a load we
// cancelled (ERR_BLOCKED_BY_CLIENT) or one replaced by a newer navigation (ERR_ABORTED),
// never trigger the auto reload.
const RETRYABLE_ERRORS = new Set([-7, -100, -101, -102, -103, -109, -324]);

function assertOwner(value: unknown): string {
  return value === PREVIEW_OWNER ? PREVIEW_OWNER : assertAgentId(value);
}

export class BrowserManager {
  private readonly tabs = new Map<string, Tab>();
  private readonly activeByAgent = new Map<string, string>();
  private readonly hardened = new Set<string>();
  private readonly observers = new Set<BrowserObserver>();
  private readonly previewPartitions = new Set<string>();
  /** URLs a preview tab was kept from; only these may go to the system browser. */
  private readonly pendingExternal = new Set<string>();
  private shownTabId: string | null = null;
  private focusedAgent: string | null = null;
  private bounds: Rect | null = null;

  constructor(private readonly deps: BrowserDeps) {}

  observe(observer: BrowserObserver): () => void {
    this.observers.add(observer);
    return () => this.observers.delete(observer);
  }

  /** The partition name of an agent; used by tests and by later CDP work. */
  partitionFor(agentId: string): string {
    return partitionForAgent(agentId, this.deps.profileKey);
  }

  /** The preview partition of a project (or the shared one when projectId is null). */
  previewPartitionFor(projectId: string | null = null): string {
    return partitionForPreview(projectId, this.deps.profileKey);
  }

  listTabs(agentId: unknown): BrowserTab[] {
    const id = assertOwner(agentId);
    return [...this.tabs.values()].filter((tab) => tab.agentId === id).map((tab) => this.describe(tab));
  }

  openTab(agentId: unknown, address?: unknown): BrowserTab {
    const id = assertAgentId(agentId);
    const url = address === undefined || address === '' ? null : this.requireUrl(address);
    return this.createTab(id, this.partitionFor(id), undefined, url);
  }

  /**
   * openPreview opens a project preview tab (F4-T06) in the preview partition, never in an
   * agent's. The tab is confined to the chosen loopback origin: navigations, redirects and
   * popups elsewhere are cancelled and reported through onPreviewBlocked. Main-process
   * callers (a future agent tool) use this directly; the renderer goes through IPC.
   */
  openPreview(target: unknown, options: OpenPreviewOptions = {}): BrowserTab {
    const parsed = parsePreviewTarget(target);
    if (parsed === null) throw new Error('preview must be a loopback http address');
    if (this.deps.reservedPorts?.includes(parsed.port)) throw new Error('this port is reserved');
    const projectId = options.projectId ?? null;
    if (projectId !== null && !isProjectId(projectId)) throw new Error('invalid project id');
    const preview: PreviewState = {
      origin: parsed.origin,
      autoReload: options.autoReload === true,
      projectId,
      retryTimer: null,
      retryDelay: RETRY_FIRST_MS,
    };
    return this.createTab(PREVIEW_OWNER, this.previewPartitionFor(projectId), preview, parsed.url);
  }

  /**
   * openExternalBlocked sends to the system browser a URL that a preview tab was just kept
   * from. The person's click is the confirmation; any other URL is refused.
   */
  async openExternalBlocked(url: unknown): Promise<boolean> {
    if (typeof url !== 'string' || !this.pendingExternal.has(url) || !isNavigableUrl(url)) return false;
    if (this.deps.openExternal === undefined) return false;
    this.pendingExternal.delete(url);
    await this.deps.openExternal(url);
    return true;
  }

  private createTab(id: string, partition: string, preview: PreviewState | undefined, url: string | null): BrowserTab {
    if (this.listTabs(id).length >= MAX_TABS_PER_AGENT) {
      throw new Error('too many tabs for this agent');
    }
    const session = this.deps.fromPartition(partition);
    if (!this.hardened.has(partition)) {
      hardenSession(session, (blockedUrl) => this.deps.onBlocked?.(id, blockedUrl));
      if (preview !== undefined) this.confineSession(session);
      this.hardened.add(partition);
    }
    if (preview !== undefined) this.previewPartitions.add(partition);
    const view = this.deps.createView({ webPreferences: webPreferencesFor(partition) });
    const tab: Tab = { id: randomUUID(), agentId: id, partition, view, preview };
    this.tabs.set(tab.id, tab);
    const contents = view.webContents;
    hardenWebContents(contents, {
      openTab: (popupUrl) => {
        if (!this.tabs.has(tab.id)) return;
        const popupPreview = preview === undefined ? undefined : { ...preview, retryTimer: null };
        this.createTab(id, partition, popupPreview, popupUrl);
      },
      blocked: (blockedUrl) => this.deps.onBlocked?.(id, blockedUrl),
      ...(preview === undefined
        ? {}
        : {
            allowNavigation: (next: string) => isWithinPreviewOrigin(next, preview.origin),
            leftOrigin: (next: string) => this.previewLeft(tab, next),
          }),
    });
    if (preview !== undefined) this.watchPreview(tab, preview);
    const changed = () => this.emit(id);
    contents.on('did-start-loading', changed);
    contents.on('did-stop-loading', changed);
    contents.on('did-navigate', changed);
    contents.on('did-navigate-in-page', changed);
    contents.on('page-title-updated', changed);
    contents.on('did-finish-load', () => {
      for (const observer of this.observers) observer.tabLoaded?.(this.describe(tab), contents);
    });
    for (const observer of this.observers) observer.tabCreated?.(this.describe(tab), contents);
    this.activate(tab.id);
    if (url !== null) {
      this.load(tab, url);
    }
    return this.describe(tab);
  }

  navigate(tabId: unknown, address: unknown): BrowserTab {
    const tab = this.requireTab(tabId);
    const url = this.requireUrl(address);
    if (tab.preview !== undefined && !isWithinPreviewOrigin(url, tab.preview.origin)) {
      throw new Error('preview tabs stay on their local address');
    }
    this.load(tab, url);
    return this.describe(tab);
  }

  goBack(tabId: unknown): void {
    const history = this.requireTab(tabId).view.webContents.navigationHistory;
    if (history.canGoBack()) history.goBack();
  }

  goForward(tabId: unknown): void {
    const history = this.requireTab(tabId).view.webContents.navigationHistory;
    if (history.canGoForward()) history.goForward();
  }

  reload(tabId: unknown): void {
    this.requireTab(tabId).view.webContents.reload();
  }

  /** activate shows a tab (and hides any tab of another agent) when the panel is visible. */
  activate(tabId: unknown): BrowserTab {
    const tab = this.requireTab(tabId);
    this.activeByAgent.set(tab.agentId, tab.id);
    // A popup from a background agent must not take over the panel of another agent.
    if (this.focusedAgent === null || this.focusedAgent === tab.agentId) {
      this.focusedAgent = tab.agentId;
      this.show(tab.id);
    }
    this.emit(tab.agentId);
    return this.describe(tab);
  }

  /** focusAgent switches the panel to an agent: shows its active tab, or nothing. */
  focusAgent(agentId: unknown): BrowserTab[] {
    const id = assertOwner(agentId);
    this.focusedAgent = id;
    this.show(this.activeByAgent.get(id) ?? null);
    return this.listTabs(id);
  }

  closeTab(tabId: unknown): void {
    const tab = this.requireTab(tabId);
    const wasShown = this.shownTabId === tab.id;
    this.destroy(tab);
    if (this.activeByAgent.get(tab.agentId) === tab.id) {
      const next = [...this.tabs.values()].find((other) => other.agentId === tab.agentId);
      if (next === undefined) {
        this.activeByAgent.delete(tab.agentId);
      } else {
        this.activeByAgent.set(tab.agentId, next.id);
        if (wasShown) this.show(next.id);
      }
    }
    this.emit(tab.agentId);
  }

  /** setBounds places the visible tab; null hides the browser (panel closed). */
  setBounds(value: Rect | null): void {
    this.bounds = value;
    const shown = this.shownTabId === null ? undefined : this.tabs.get(this.shownTabId);
    if (shown === undefined) return;
    if (value === null || value.width === 0 || value.height === 0) {
      this.deps.host.remove(shown.view);
    } else {
      this.deps.host.add(shown.view);
      shown.view.setBounds(value);
    }
  }

  /**
   * clearProfile closes every tab of the agent and erases its partition: cookies, storage,
   * cache, auth cache. The partition name stays the same, so the next tab starts empty.
   */
  async clearProfile(agentId: unknown): Promise<void> {
    const id = assertOwner(agentId);
    for (const tab of [...this.tabs.values()]) {
      if (tab.agentId === id) this.destroy(tab);
    }
    this.activeByAgent.delete(id);
    const partitions =
      id === PREVIEW_OWNER
        ? new Set([this.previewPartitionFor(null), ...this.previewPartitions])
        : [this.partitionFor(id)];
    for (const partition of partitions) {
      const session = this.deps.fromPartition(partition);
      await session.clearData();
      await session.clearStorageData();
      await session.clearAuthCache();
    }
    this.emit(id);
  }

  /** Main-process only (F5-T02 hook): the WebContents of a tab. Never sent over IPC. */
  webContentsFor(tabId: string): WebContents | null {
    return this.tabs.get(tabId)?.view.webContents ?? null;
  }

  /** dispose closes every tab without waiting for pages. Safe to call more than once. */
  dispose(): void {
    for (const tab of [...this.tabs.values()]) this.destroy(tab);
    this.activeByAgent.clear();
    this.pendingExternal.clear();
    this.focusedAgent = null;
    this.bounds = null;
  }

  private show(tabId: string | null): void {
    if (this.shownTabId !== null && this.shownTabId !== tabId) {
      const previous = this.tabs.get(this.shownTabId);
      if (previous !== undefined) this.deps.host.remove(previous.view);
    }
    this.shownTabId = tabId;
    this.setBounds(this.bounds);
  }

  /**
   * confineSession is the second containment layer of a preview partition: a top-level
   * request whose WebContents is not a preview tab, or that leaves the tab's origin, is
   * cancelled in the network stack, whatever started it (history, a missed event...).
   * Subresources (scripts, fonts, images, iframes) are not filtered.
   */
  private confineSession(session: Session): void {
    session.webRequest.onBeforeRequest((details, callback) => {
      if (details.resourceType !== 'mainFrame') {
        callback({});
        return;
      }
      const tab = [...this.tabs.values()].find((candidate) => candidate.view.webContents.id === details.webContentsId);
      const allowed = tab?.preview !== undefined && isWithinPreviewOrigin(details.url, tab.preview.origin);
      callback(allowed ? {} : { cancel: true });
    });
  }

  private previewLeft(tab: Tab, url: string): void {
    if (this.pendingExternal.size >= MAX_PENDING_EXTERNAL) {
      const oldest = this.pendingExternal.values().next().value;
      if (oldest !== undefined) this.pendingExternal.delete(oldest);
    }
    this.pendingExternal.add(url);
    this.deps.onPreviewBlocked?.({ tabId: tab.id, url });
  }

  /**
   * watchPreview implements the optional auto reload: when the dev server is down (restart,
   * rebuild) the main-frame load fails, and the tab retries with backoff until it answers.
   */
  private watchPreview(tab: Tab, preview: PreviewState): void {
    const contents = tab.view.webContents;
    contents.on('did-finish-load', () => {
      preview.retryDelay = RETRY_FIRST_MS;
    });
    contents.on(
      'did-fail-load',
      (_event: unknown, errorCode: number, _description: string, validatedURL: string, isMainFrame: boolean) => {
        if (!preview.autoReload || !isMainFrame || !RETRYABLE_ERRORS.has(errorCode) || preview.retryTimer !== null)
          return;
        const retryUrl = isWithinPreviewOrigin(validatedURL, preview.origin) ? validatedURL : `${preview.origin}/`;
        preview.retryTimer = setTimeout(() => {
          preview.retryTimer = null;
          if (this.tabs.has(tab.id)) this.load(tab, retryUrl);
        }, preview.retryDelay);
        preview.retryDelay = Math.min(preview.retryDelay * 2, RETRY_MAX_MS);
      },
    );
  }

  private load(tab: Tab, url: string): void {
    // Aborted loads (a newer navigation, a blocked redirect) reject; the tab state still
    // updates through the navigation events.
    tab.view.webContents.loadURL(url).catch(() => undefined);
  }

  private destroy(tab: Tab): void {
    this.tabs.delete(tab.id);
    if (tab.preview?.retryTimer) {
      clearTimeout(tab.preview.retryTimer);
      tab.preview.retryTimer = null;
    }
    if (this.shownTabId === tab.id) {
      this.shownTabId = null;
      try {
        this.deps.host.remove(tab.view);
      } catch {
        // The window may already be destroyed during quit; the WebContents still must close.
      }
    }
    const contents = tab.view.webContents;
    // A live tab WebContents keeps the app process alive after quit, and a page's
    // beforeunload must not be able to delay closing it.
    if (!contents.isDestroyed()) contents.close({ waitForBeforeUnload: false });
    for (const observer of this.observers) observer.tabClosed?.(tab.id, tab.agentId);
  }

  private requireTab(tabId: unknown): Tab {
    const tab = typeof tabId === 'string' ? this.tabs.get(tabId) : undefined;
    if (tab === undefined) throw new Error('unknown tab');
    return tab;
  }

  private requireUrl(address: unknown): string {
    const url = normalizeAddress(address);
    if (url === null || !isNavigableUrl(url)) throw new Error('only http and https addresses can be opened');
    return url;
  }

  private describe(tab: Tab): BrowserTab {
    const contents = tab.view.webContents;
    const destroyed = contents.isDestroyed();
    return {
      id: tab.id,
      agentId: tab.agentId,
      url: destroyed ? '' : contents.getURL(),
      title: destroyed ? '' : contents.getTitle(),
      loading: !destroyed && contents.isLoading(),
      canGoBack: !destroyed && contents.navigationHistory.canGoBack(),
      canGoForward: !destroyed && contents.navigationHistory.canGoForward(),
      active: this.activeByAgent.get(tab.agentId) === tab.id,
      ...(tab.preview === undefined
        ? {}
        : { preview: { origin: tab.preview.origin, autoReload: tab.preview.autoReload } }),
    };
  }

  private emit(agentId: string): void {
    this.deps.onTabsChanged(agentId, this.listTabs(agentId));
  }
}
