import axios from 'axios'
import type { ApiEnvelope } from '@/types/chat'

export const apiBaseURL = import.meta.env.VITE_API_BASE_URL ?? ''

export const http = axios.create({ baseURL: apiBaseURL, timeout: 15_000 })

http.interceptors.response.use((response) => {
  const body = response.data as Partial<ApiEnvelope<unknown>>
  if (typeof body.code === 'number' && body.code !== 0) {
    return Promise.reject(new Error(body.message || `请求失败（${body.code}）`))
  }
  return response
})

export class BackendCapabilityError extends Error {
  constructor(public readonly capability: string) {
    super(`后端尚未实现「${capability}」接口，详见 docs/API_CONTRACT.md。`)
  }
}
