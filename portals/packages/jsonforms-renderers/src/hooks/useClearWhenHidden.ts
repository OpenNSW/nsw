import { useEffect } from 'react'

export const useClearWhenHidden = (
  visible: boolean | undefined,
  path: string,
  handleChange: (path: string, value: any) => void,
  emptyValue: any = undefined,
) => {
  useEffect(() => {
    if (visible === false) {
      handleChange(path, emptyValue)
    }
  }, [visible, path, handleChange, emptyValue])
}
