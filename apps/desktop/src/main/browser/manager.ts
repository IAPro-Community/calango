// Internal browser (F5-T01): tabs as WebContentsView, one persistent partition per agent.
// Electron objects come in through BrowserDeps, so tests can run this under plain Node.
import { randomUUID } from 'node:crypto';
import type { Session, WebContents, WebContentsView, WebContentsViewConstructorOptions } from 'electron';
import type { BrowserTab } from '../../shared/ipc';
import { hardenSession, hardenWebContents, webPreferencesFor } from './harden';
import { assertAgentId, isNavigableUrl, normalizeAddress, partitionForAgent, type Rect } from './policy';

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
}

interface Tab {
  id: string;
  agentId: string;
  view: WebContentsView;
}

const MAX_TABS_PER_AGENT = 20;

export class BrowserManager {
  private readonly tabs = new Map<string, Tab>();
  private readonly activeByAgent = new Map<string, string>();
  private readonly hardened = new Set<string>();
  private readonly observers = new Set<BrowserObserver>();
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

  listTabs(agentId: unknown): BrowserTab[] {
    const id = assertAgentId(agentId);
    return [...this.tabs.values()].filter((tab) => tab.agentId === id).map((tab) => this.describe(tab));
  }

  openTab(agentId: unknown, address?: unknown): BrowserTab {
    const id = assertAgentId(agentId);
    const url = address === undefined || address === '' ? null : this.requireUrl(address);
    if (this.listTabs(id).length >= MAX_TABS_PER_AGENT) {
      throw new Error('too many tabs for this agent');
    }
    const partition = this.partitionFor(id);
    const session = this.deps.fromPartition(partition);
    if (!this.hardened.has(partition)) {
      hardenSession(session, (blockedUrl) => this.deps.onBlocked?.(id, blockedUrl));
      this.hardened.add(partition);
    }
    const view = this.deps.createView({ webPreferences: webPreferencesFor(partition) });
    const tab: Tab = { id: randomUUID(), agentId: id, view };
    this.tabs.set(tab.id, tab);
    const contents = view.webContents;
    hardenWebContents(contents, {
      openTab: (popupUrl) => {
        if (this.tabs.has(tab.id)) {
          this.openTab(id, popupUrl);
        }
      },
      blocked: (blockedUrl) => this.deps.onBlocked?.(id, blockedUrl),
    });
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
    this.load(tab, this.requireUrl(address));
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
    const id = assertAgentId(agentId);
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
    const id = assertAgentId(agentId);
    for (const tab of [...this.tabs.values()]) {
      if (tab.agentId === id) this.destroy(tab);
    }
    this.activeByAgent.delete(id);
    const session = this.deps.fromPartition(this.partitionFor(id));
    await session.clearData();
    await session.clearStorageData();
    await session.clearAuthCache();
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

  private load(tab: Tab, url: string): void {
    // Aborted loads (a newer navigation, a blocked redirect) reject; the tab state still
    // updates through the navigation events.
    tab.view.webContents.loadURL(url).catch(() => undefined);
  }

  private destroy(tab: Tab): void {
    this.tabs.delete(tab.id);
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
    };
  }

  private emit(agentId: string): void {
    this.deps.onTabsChanged(agentId, this.listTabs(agentId));
  }
}
