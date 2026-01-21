import type {ComponentProps} from 'react';

export function Button({children, ...props}: ComponentProps<'button'>) {
  return <button {...props} className="bg-green-400">{children ?? 'Click me'}</button>;
}