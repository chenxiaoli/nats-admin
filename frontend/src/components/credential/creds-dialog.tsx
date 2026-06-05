import { useState } from 'react';

interface Props {
  open: boolean;
  creds: string;
  onClose: () => void;
}

export default function CredsDialog({ open, creds, onClose }: Props) {
  const [copied, setCopied] = useState(false);
  if (!open) return null;

  const handleCopy = async () => {
    await navigator.clipboard.writeText(creds);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    const blob = new Blob([creds], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'nats-user.creds';
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-lg rounded-lg bg-white p-6 shadow-xl">
        <div className="mb-3 rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800">
          凭证仅显示一次，请立即保存
        </div>
        <pre className="max-h-64 overflow-auto rounded bg-slate-100 p-3 text-xs">{creds}</pre>
        <div className="mt-4 flex justify-end gap-2">
          <button onClick={handleCopy} className="rounded-md border px-4 py-2 text-sm">
            {copied ? '已复制' : '复制到剪贴板'}
          </button>
          <button onClick={handleDownload} className="rounded-md border px-4 py-2 text-sm">
            下载 .creds
          </button>
          <button onClick={onClose} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white">
            我已保存
          </button>
        </div>
      </div>
    </div>
  );
}
