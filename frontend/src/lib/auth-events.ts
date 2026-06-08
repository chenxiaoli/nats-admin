let inFlight: Promise<string> | null = null;

export function requestReauth(): Promise<string> {
  if (inFlight) return inFlight;
  inFlight = new Promise<string>((resolve, reject) => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent<{ token?: string; cancelled?: boolean }>).detail;
      bus.removeEventListener('reauth', handler);
      inFlight = null;
      if (detail.cancelled) reject(new Error('reauth cancelled'));
      else if (detail.token) resolve(detail.token);
      else reject(new Error('reauth failed'));
    };
    bus.addEventListener('reauth', handler);
    bus.dispatchEvent(new CustomEvent('reauth-request'));
  });
  return inFlight;
}

function completeReauth(detail: { token?: string; cancelled?: boolean }) {
  bus.dispatchEvent(new CustomEvent('reauth', { detail }));
}

export const reauthController = {
  /** Called by the provider to tell the bus "a modal opened, await completeReauth" */
  onRequest(handler: () => void) {
    bus.addEventListener('reauth-request', handler);
    return () => bus.removeEventListener('reauth-request', handler);
  },
  /** Called by the modal on successful login */
  succeed(token: string) {
    completeReauth({ token });
  },
  /** Called by the modal on cancel */
  cancel() {
    completeReauth({ cancelled: true });
  },
};

const bus = new EventTarget();
