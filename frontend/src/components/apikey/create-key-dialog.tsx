import { useState } from 'react';
import { useCreateAPIKey } from '@/api/apikeys';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export default function CreateKeyDialog({ open, onOpenChange }: Props) {
  const [name, setName] = useState('');
  const [created, setCreated] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const create = useCreateAPIKey();

  function reset() {
    setName('');
    setCreated(null);
    setCopied(false);
  }

  async function handleSubmit() {
    if (!name.trim()) return;
    const resp = await create.mutateAsync(name.trim());
    setCreated(resp.key);
  }

  async function copy() {
    if (!created) return;
    await navigator.clipboard.writeText(created);
    setCopied(true);
  }

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h3 className="text-lg font-semibold">
          {created ? '保存你的 API Key' : '创建 API Key'}
        </h3>
        {created ? (
          <div className="mt-4">
            <p className="mb-2 text-sm text-amber-700">
              ⚠ 这个 key 只会显示一次。请立即复制并妥善保存。
            </p>
            <div className="flex items-center gap-2">
              <code className="flex-1 break-all rounded bg-slate-100 p-2 text-xs">
                {created}
              </code>
              <button
                onClick={copy}
                className="rounded bg-slate-900 px-3 py-1 text-sm text-white"
              >
                {copied ? '已复制' : '复制'}
              </button>
            </div>
          </div>
        ) : (
          <div className="mt-4">
            <label className="mb-1 block text-sm font-medium">名称</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：ci-pipeline"
              className="w-full rounded border border-slate-300 px-3 py-2 text-sm"
            />
          </div>
        )}
        <div className="mt-6 flex justify-end gap-2">
          <button
            onClick={() => {
              onOpenChange(false);
              reset();
            }}
            className="rounded-md border border-slate-300 px-4 py-2 text-sm"
          >
            取消
          </button>
          <button
            onClick={created ? () => { onOpenChange(false); reset(); } : handleSubmit}
            disabled={!created && (!name.trim() || create.isPending)}
            className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-50"
          >
            {created ? '完成' : create.isPending ? '创建中…' : '创建'}
          </button>
        </div>
      </div>
    </div>
  );
}
