import axios, { AxiosError, type AxiosRequestConfig } from 'axios'

export const TOKEN_KEY = 'desklink_admin_token'

export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  list: T[]
  page: number
  page_size: number
  total: number
}

export class ApiError extends Error {
  code: number

  constructor(message: string, code = -1) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

const client = axios.create({
  baseURL: '/api/admin',
  timeout: 30_000,
  withCredentials: true,
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) config.headers['api-token'] = token
  config.headers['Accept-Language'] = 'zh-CN'
  return config
})

client.interceptors.response.use(
  (response) => {
    const payload = response.data as ApiEnvelope<unknown>
    if (payload && typeof payload.code === 'number' && payload.code !== 0) {
      if (payload.code === 403) window.dispatchEvent(new Event('desklink:unauthorized'))
      throw new ApiError(payload.message || '请求失败', payload.code)
    }
    return response
  },
  (error: AxiosError<{ error?: string; message?: string }>) => {
    const message = error.response?.data?.error || error.response?.data?.message || error.message || '网络请求失败'
    throw new ApiError(message, error.response?.status ?? -1)
  },
)

export async function api<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await client.request<ApiEnvelope<T>>(config)
  return response.data.data
}

export function get<T>(url: string, params?: Record<string, unknown>) {
  return api<T>({ url, method: 'GET', params })
}

export function post<T>(url: string, data?: unknown) {
  return api<T>({ url, method: 'POST', data })
}

export function normalizePage<T>(data?: Partial<PageData<T>>): PageData<T> {
  return {
    list: Array.isArray(data?.list) ? data.list : [],
    page: Number(data?.page || 1),
    page_size: Number(data?.page_size || 20),
    total: Number(data?.total || 0),
  }
}

export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : '操作失败'
}
