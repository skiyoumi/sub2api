interface APIErrorLike {
  message?: string
  reason?: string
  response?: {
    data?: {
      detail?: string
      message?: string
    }
  }
}

function extractErrorMessage(error: unknown): string {
  const err = (error || {}) as APIErrorLike
  return err.response?.data?.detail || err.response?.data?.message || err.message || ''
}

export function buildAuthErrorMessage(
  error: unknown,
  options: { fallback: string },
): string {
  const { fallback } = options
  if ((error as APIErrorLike)?.reason === 'REGISTRATION_IP_LIMIT') {
    return '\u5f53\u524d IP \u4e0d\u5141\u8bb8\u6ce8\u518c'
  }
  return extractErrorMessage(error) || fallback
}
