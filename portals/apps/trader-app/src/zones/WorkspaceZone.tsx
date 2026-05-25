import { useRef, useState } from 'react'
import { Button } from '@radix-ui/themes'
import type { Action, ActionVariant, ZoneComponent } from './types'
import { renderZoneComponent } from './renderers'
import { FormRenderer, type FormHandle } from './renderers/FormRenderer'
import { getBooleanEnv } from '../runtimeConfig'

type Props = {
  name: string
  component: ZoneComponent
  actions: Action[]
  onSubmitForm?: (command: string, data: Record<string, unknown>) => Promise<void>
}

export function WorkspaceZone({ name, component, actions, onSubmitForm }: Props) {
  const formRef = useRef<FormHandle>(null)
  const [isFormValid, setIsFormValid] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const handleAction = (action: Action) => {
    if (action.kind !== 'submit_form') {
      console.log('[zones] task_action', { action: action.action })
      return
    }
    if (!onSubmitForm) return
    const data = formRef.current?.getData() ?? {}
    setSubmitting(true)
    void onSubmitForm(action.command, data).finally(() => setSubmitting(false))
  }

  const handleAutoFill = () => formRef.current?.autoFill()

  const showAutoFill = component.type === 'FORM' && getBooleanEnv('VITE_SHOW_AUTOFILL_BUTTON', false)

  return (
    <section className="space-y-2">
      <ZoneHeader name={name} type={component.type} />
      <div className="bg-white rounded-lg shadow-sm border border-gray-100">
        <div className="p-6">
          {component.type === 'FORM' ? (
            <FormRenderer ref={formRef} payload={component.payload} onValidityChange={setIsFormValid} />
          ) : (
            renderZoneComponent(component)
          )}
        </div>
        {(actions.length > 0 || showAutoFill) && (
          <WorkspaceActionBar
            actions={actions}
            onAction={handleAction}
            submitting={submitting}
            canSubmit={component.type === 'FORM' ? isFormValid : true}
            onAutoFill={showAutoFill ? handleAutoFill : undefined}
          />
        )}
      </div>
    </section>
  )
}

function WorkspaceActionBar({
  actions,
  onAction,
  submitting,
  canSubmit,
  onAutoFill,
}: {
  actions: Action[]
  onAction: (action: Action) => void
  submitting: boolean
  canSubmit: boolean
  onAutoFill?: () => void
}) {
  return (
    <div className="sticky bottom-0 border-t border-gray-100 bg-white/95 backdrop-blur rounded-b-lg shadow-[0_-4px_12px_-8px_rgba(0,0,0,0.08)]">
      <div className="px-6 py-4 flex items-center gap-3">
        {onAutoFill && (
          <Button type="button" variant="soft" color="purple" size="3" onClick={onAutoFill} disabled={submitting}>
            Demo - Auto Fill
          </Button>
        )}
        <div className="flex-1" />
        {actions.map((action) => (
          <ActionButton
            key={actionKey(action)}
            action={action}
            onAction={onAction}
            submitting={submitting}
            canSubmit={canSubmit}
          />
        ))}
      </div>
    </div>
  )
}

function ActionButton({
  action,
  onAction,
  submitting,
  canSubmit,
}: {
  action: Action
  onAction: (action: Action) => void
  submitting: boolean
  canSubmit: boolean
}) {
  const variant = action.variant ?? 'primary'
  const disabled = submitting || (action.kind === 'submit_form' && !canSubmit)
  return (
    <Button
      onClick={() => onAction(action)}
      size="3"
      variant={buttonVariant(variant)}
      color={buttonColor(variant)}
      disabled={disabled}
    >
      {submitting && action.kind === 'submit_form' ? 'Submitting...' : action.label}
    </Button>
  )
}

function ZoneHeader({ name, type }: { name: string; type: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs font-semibold uppercase tracking-widest text-gray-400">{name}</span>
      <span className="text-[10px] font-medium uppercase tracking-wider text-gray-300">{type}</span>
    </div>
  )
}

function buttonVariant(v: ActionVariant): 'solid' | 'outline' {
  return v === 'outline' ? 'outline' : 'solid'
}

function buttonColor(v: ActionVariant): 'red' | undefined {
  return v === 'danger' ? 'red' : undefined
}

function actionKey(action: Action): string {
  return action.kind === 'submit_form' ? `submit:${action.command}` : `action:${action.action}`
}
