import { useEffect, useState } from 'react';
import ReauthModal from './reauth-modal';
import { reauthController } from '@/lib/auth-events';

export default function ReauthProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const [resolver, setResolver] = useState<((token: string) => void) | null>(null);
  const [rejecter, setRejecter] = useState<(() => void) | null>(null);

  useEffect(() => {
    return reauthController.onRequest(() => {
      setOpen(true);
      setResolver(() => (token: string) => reauthController.succeed(token));
      setRejecter(() => () => reauthController.cancel());
    });
  }, []);

  return (
    <>
      {children}
      <ReauthModal
        open={open}
        onSolved={(token) => {
          setOpen(false);
          resolver?.(token);
        }}
        onCancelled={() => {
          setOpen(false);
          rejecter?.();
        }}
      />
    </>
  );
}
