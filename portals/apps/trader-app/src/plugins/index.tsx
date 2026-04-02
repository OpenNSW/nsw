import SimpleForm, {type SimpleFormConfig} from "./SimpleForm.tsx";
import WaitForEvent, {type WaitForEventConfigs} from "./WaitForEvent.tsx";
import Payment, { type PaymentConfigs } from "./Payment.tsx";

export type TaskType = "SIMPLE_FORM" | "WAIT_FOR_EVENT" | "PAYMENT"

export type RenderInfoTyped<Type extends TaskType, T> = {
  type: Type
  content: T
  state: string
  pluginState: string
}

export type RenderInfo =
  | RenderInfoTyped<"SIMPLE_FORM", SimpleFormConfig>
  | RenderInfoTyped<"WAIT_FOR_EVENT", WaitForEventConfigs>
  | RenderInfoTyped<"PAYMENT", PaymentConfigs>

// Renderer component
export default function PluginRenderer({
  response,
  onTaskUpdated,
}: {
  response: RenderInfo
  onTaskUpdated?: () => Promise<void>
}) {
  const { type, content, pluginState } = response;

  // TypeScript automatically narrows the content type based on type field
  switch (type) {
    case 'SIMPLE_FORM':
      return <SimpleForm configs={content} pluginState={pluginState}  />;
    case 'WAIT_FOR_EVENT':
      return <WaitForEvent configs={content} pluginState={pluginState} />;
    case 'PAYMENT':
      return <Payment configs={content} pluginState={pluginState} onTaskUpdated={onTaskUpdated} />;
    default:
      // Exhaustiveness check - TypeScript will error if you miss a case
      return null;
  }
}