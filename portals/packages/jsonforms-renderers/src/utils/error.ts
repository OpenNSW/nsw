export const getErrorMessage = (errors: string, label?: string): string => {
  if (errors === 'is a required property') {
    return `${label || 'This field'} is required`
  }
  return errors
}
