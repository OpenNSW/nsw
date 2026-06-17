export const getErrorMessage = (errors: string, label?: string): string => {
  if (!errors) return ''
  return errors
    .split('\n')
    .map((err) => {
      const trimmed = err.trim()
      if (trimmed === 'is a required property') {
        return `${label || 'This field'} is required`
      }
      return err
    })
    .join('\n')
}
