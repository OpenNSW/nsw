import type { ZoneComponent } from './types'
import { renderZoneComponent } from './renderers'

type Props = {
  name: string
  component: ZoneComponent
}

export function PresentationalZone({ name, component }: Props) {
  return (
    <section className="space-y-2">
      <div className="flex items-center gap-2">
        <span className="text-xs font-semibold uppercase tracking-widest text-gray-400">{name}</span>
        <span className="text-[10px] font-medium uppercase tracking-wider text-gray-300">{component.type}</span>
      </div>
      <div className="bg-white rounded-lg shadow-sm border border-gray-100">
        <div className="p-6">{renderZoneComponent(component)}</div>
      </div>
    </section>
  )
}
