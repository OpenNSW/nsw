export type FireAndForgetConfig = {
  endpoint?: string
  data?: Record<string, unknown>
}

function formatKey(key: string): string {
  return key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

function DataValue({ value }: { value: unknown }) {
  if (value === null || value === undefined) {
    return <span className="text-gray-400 italic">—</span>
  }
  if (typeof value === 'object') {
    return (
      <pre className="text-xs font-mono bg-gray-100 text-gray-700 rounded px-2 py-1 whitespace-pre-wrap break-all">
        {JSON.stringify(value, null, 2)}
      </pre>
    )
  }
  return <span className="text-sm font-mono text-gray-800 break-all">{String(value)}</span>
}

function SendIcon() {
  return (
    <div className="flex items-center justify-center w-16 h-16 rounded-full bg-emerald-100">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="w-8 h-8 text-emerald-600"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.75}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5"
        />
      </svg>
    </div>
  )
}

function FailedIcon() {
  return (
    <div className="flex items-center justify-center w-16 h-16 rounded-full bg-red-100">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="w-8 h-8 text-red-500"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fillRule="evenodd"
          d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
          clipRule="evenodd"
        />
      </svg>
    </div>
  )
}

export default function FireAndForget({
  configs,
  pluginState,
}: {
  configs: FireAndForgetConfig
  pluginState: string
}) {
  const isFailed = pluginState === 'SUBMISSION_FAILED'
  const entries = configs.data ? Object.entries(configs.data) : []

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl shadow-md overflow-hidden animate-fade-in-up">
        <div className={`h-1 ${isFailed ? 'bg-red-400' : 'bg-emerald-500'}`} />

        <div className="px-8 py-10 flex flex-col items-center text-center gap-3">
          {isFailed ? <FailedIcon /> : <SendIcon />}

          <div className="space-y-1">
            <p className={`text-xs font-semibold uppercase tracking-widest ${isFailed ? 'text-red-500' : 'text-emerald-600'}`}>
              {isFailed ? 'Dispatch failed' : 'Dispatched'}
            </p>
            <h2 className="text-xl font-bold text-gray-800">
              {isFailed ? 'Submission could not be sent' : 'Submission sent successfully'}
            </h2>
            {configs.endpoint && (
              <p className="text-sm text-gray-400 mt-1">
                Sent to{' '}
                <code className="text-xs bg-gray-100 text-gray-600 px-1.5 py-0.5 rounded font-mono">
                  {configs.endpoint}
                </code>
              </p>
            )}
          </div>
        </div>
      </div>

      {entries.length > 0 && (
        <div className="bg-white rounded-xl shadow-md overflow-hidden animate-fade-in-up">
          <div className="bg-gray-50 border-b border-gray-100 px-6 py-4">
            <p className="text-xs font-semibold uppercase tracking-wider text-gray-500">Dispatched data</p>
          </div>
          <div className="divide-y divide-gray-100">
            {entries.map(([key, value]) => (
              <div key={key} className="grid grid-cols-5 gap-4 px-6 py-3 items-start">
                <span className="col-span-2 text-xs font-medium text-gray-500 pt-0.5">{formatKey(key)}</span>
                <div className="col-span-3">
                  <DataValue value={value} />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
