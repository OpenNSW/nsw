import React from 'react'
import SimpleForm, { type SimpleFormConfig } from './SimpleForm.tsx'
import WaitForEvent, { type WaitForEventConfigs } from './WaitForEvent.tsx'
import Payment, { type PaymentConfigs } from './Payment.tsx'
import FireAndForget, { type FireAndForgetConfig } from './FireAndForget.tsx'
import PluginHeader from './PluginHeader.tsx'

export type TaskType = 'SIMPLE_FORM' | 'WAIT_FOR_EVENT' | 'PAYMENT' | 'FIRE_AND_FORGET'

export type RenderInfoTyped<Type extends TaskType, T> = {
  type: Type
  content: T
  state: string
  pluginState: string
}

export type RenderInfo =
  | RenderInfoTyped<'SIMPLE_FORM', SimpleFormConfig>
  | RenderInfoTyped<'WAIT_FOR_EVENT', WaitForEventConfigs>
  | RenderInfoTyped<'PAYMENT', PaymentConfigs>
  | RenderInfoTyped<'FIRE_AND_FORGET', FireAndForgetConfig>

// Renderer component
export default function PluginRenderer({
  response,
  onTaskUpdated,
}: {
  response: RenderInfo
  onTaskUpdated?: () => Promise<void>
}) {
  const { type, state, content, pluginState } = response

  let plugin: React.ReactNode
  switch (type) {
    case 'SIMPLE_FORM':
      plugin = <SimpleForm configs={content} pluginState={pluginState} />
      break
    case 'WAIT_FOR_EVENT':
      plugin = <WaitForEvent configs={content} pluginState={pluginState} />
      break
    case 'PAYMENT':
      plugin = <Payment configs={content} pluginState={pluginState} onTaskUpdated={onTaskUpdated} />
      break
    case 'FIRE_AND_FORGET':
      plugin = <FireAndForget configs={content} pluginState={pluginState} />
      break
    default:
      return null
  }

  return (
    <div className="space-y-4">
      <PluginHeader type={type} state={state} pluginState={pluginState} />
      {plugin}
    </div>
  )
}
