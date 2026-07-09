export interface FolderOpenerOptions {
  /** Hostname of the Folder Opener server. @default "localhost" */
  host?: string;
  /** Port of the Folder Opener server. @default 29101 */
  port?: number;
  /** Protocol to use. @default "http" */
  protocol?: 'http' | 'https';
}

export interface ServerStatus {
  status: string;
  version: string;
}

export interface OpenResult {
  path: string;
  /** "opened" for a directory, "revealed" for a file selected in its parent folder. */
  action: 'opened' | 'revealed';
}

export type ErrorCode = 'bad_request' | 'not_found' | 'access_denied' | 'internal';

export interface WatchOptions {
  /** Polling interval in milliseconds. @default 5000 */
  interval?: number;
}

export class FolderOpener {
  private baseUrl: string;
  private watchers = new Set<(running: boolean) => void>();
  private pollTimer: ReturnType<typeof setInterval> | null = null;
  private pollInterval = 0;
  private lastStatus: boolean | null = null;

  constructor(options?: FolderOpenerOptions) {
    const host = options?.host ?? 'localhost';
    const port = options?.port ?? 29101;
    const protocol = options?.protocol ?? 'http';
    this.baseUrl = `${protocol}://${host}:${port}`;
  }

  /**
   * Open a folder in the system's file browser. If the path is a file, it is
   * revealed (selected) in its parent folder instead.
   *
   * Accepts an absolute local path, a Windows UNC path, or (server v0.2+) an
   * `smb://server/share/...` URL — the server mounts the share on demand
   * where the platform needs it (macOS/Linux), so one payload works on
   * every OS.
   *
   * Throws a `FolderOpenerError` with `code: 'not_found'` when the path does
   * not exist on the machine — unlike the legacy protocol-handler approach,
   * a missing folder is a real, detectable error.
   *
   * Throws `code: 'access_denied'` (server v0.2.3+) when the server process
   * is denied access to the path AND the file browser couldn't take over —
   * typically a sign the server is running under a different account than
   * the desktop user (e.g. launched by an elevated installer), whose
   * network-share permissions differ.
   */
  async open(path: string): Promise<OpenResult> {
    const res = await fetch(`${this.baseUrl}/open`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    });

    if (!res.ok) {
      let message = `Server error: ${res.status}`;
      let code: ErrorCode = 'internal';
      try {
        const body = (await res.json()) as { error?: string; code?: ErrorCode };
        if (body.error) message = body.error;
        if (body.code) code = body.code;
      } catch {
        // non-JSON error body; keep defaults
      }
      throw new FolderOpenerError(message, res.status, code);
    }

    return res.json();
  }

  /**
   * Watch for server status changes. The callback fires immediately with
   * the current status, then again whenever the status changes.
   *
   * Returns an unwatch function. When all watchers unsubscribe, polling stops.
   *
   * ```ts
   * const unwatch = folderOpener.watch((running) => {
   *   banner.classList.toggle('d-none', running);
   * });
   * // later: unwatch();
   * ```
   */
  watch(callback: (running: boolean) => void, options?: WatchOptions): () => void {
    const interval = options?.interval ?? 5000;
    this.watchers.add(callback);

    // Fire immediately with current known status, then poll
    this.isRunning().then((running) => {
      if (this.watchers.has(callback)) {
        callback(running);
        this.lastStatus = running;
      }
    });

    // Start or restart polling at the shortest requested interval
    if (this.pollTimer === null || interval < this.pollInterval) {
      this.startPolling(interval);
    }

    return () => {
      this.watchers.delete(callback);
      if (this.watchers.size === 0) {
        this.stopPolling();
      }
    };
  }

  private startPolling(interval: number): void {
    this.stopPolling();
    this.pollInterval = interval;
    this.pollTimer = setInterval(async () => {
      const running = await this.isRunning();
      if (running !== this.lastStatus) {
        this.lastStatus = running;
        for (const cb of this.watchers) {
          cb(running);
        }
      }
    }, interval);
  }

  private stopPolling(): void {
    if (this.pollTimer !== null) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
      this.pollInterval = 0;
      this.lastStatus = null;
    }
  }

  /**
   * Check if the Folder Opener server is reachable.
   *
   * Uses `no-cors` mode to avoid noisy CORS console errors when
   * the server is not running.
   */
  async isRunning(): Promise<boolean> {
    try {
      // An opaque response (type "opaque") means the server responded.
      // A network error (TypeError) means it's unreachable.
      const res = await fetch(`${this.baseUrl}/status`, { mode: 'no-cors' });
      return res.type === 'opaque' || res.ok;
    } catch {
      return false;
    }
  }

  /** Get server status and version. */
  async status(): Promise<ServerStatus> {
    const res = await fetch(`${this.baseUrl}/status`);
    if (!res.ok) {
      throw new FolderOpenerError(`Server error: ${res.status}`, res.status, 'internal');
    }
    return res.json();
  }
}

export class FolderOpenerError extends Error {
  constructor(
    message: string,
    public readonly statusCode: number,
    public readonly code: ErrorCode
  ) {
    super(message);
    this.name = 'FolderOpenerError';
  }
}
