import { useState } from 'react';

interface Props {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDialog({ open, title, message, confirmLabel = '确认', danger, onConfirm, onCancel }: Props) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-lg bg-white p-6 shadow-xl">
        <h3 className="text-lg font-semibold">{title}</h3>
        <p className="mt-2 text-sm text-slate-600">{message}</p>
        <div className="mt-4 flex justify-end gap-2">
          <button onClick={onCancel} className="rounded-md border px-4 py-2 text-sm">取消</button>
          <button
            onClick={onConfirm}
            className={`rounded-md px-4 py-2 text-sm text-white ${danger ? 'bg-red-600 hover:bg-red-700' : 'bg-slate-900 hover:bg-slate-700'}`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}

export function useConfirm() {
  const [state, setState] = useState<{ open: boolean; title: string; message: string; confirmLabel: string; danger: boolean; resolve: ((v: boolean) => void) | null }>({
    open: false, title: '', message: '', confirmLabel: '确认', danger: false, resolve: null,
  });

  const confirm = (opts: { title: string; message: string; confirmLabel?: string; danger?: boolean }): Promise<boolean> =>
    new Promise((resolve) => setState({ ...opts, open: true, confirmLabel: opts.confirmLabel ?? '确认', danger: opts.danger ?? false, resolve }));

  const handleConfirm = () => {
    state.resolve?.(true);
    setState((s) => ({ ...s, open: false }));
  };
  const handleCancel = () => {
    state.resolve?.(false);
    setState((s) => ({ ...s, open: false }));
  };

  return { confirmProps: { ...state, onConfirm: handleConfirm, onCancel: handleCancel }, confirm };
}
