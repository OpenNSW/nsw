import type {ComponentProps} from 'react';

export function Card({children, ...props}: ComponentProps<'div'>) {
  return (
    <div {...props}>{children}</div>
  )
}