import axios, { type AxiosError } from 'axios'

import type { ApiErrorResponse } from '../types'

export const getErrorMessage = (error: unknown, fallback: string): string => {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ApiErrorResponse>
    const responseMessage = axiosError.response?.data?.error?.message
    if (responseMessage) {
      return responseMessage
    }
    if (axiosError.message) {
      return axiosError.message
    }
  }

  if (error instanceof Error && error.message) {
    return error.message
  }

  return fallback
}
