// 前端 API 客户端：统一请求入口与错误信息提取。
import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 15_000,
  withCredentials: true,
})

let unauthorizedHandler: (() => void) | null = null

api.interceptors.response.use(undefined, (error: unknown) => {
  if (axios.isAxiosError(error) && error.response?.status === 401) unauthorizedHandler?.()
  return Promise.reject(error)
})

export function setUnauthorizedHandler(handler: () => void): void {
  unauthorizedHandler = handler
}

export function isUnauthorized(error: unknown): boolean {
  return axios.isAxiosError(error) && error.response?.status === 401
}

export function getApiError(error: unknown, fallback = '请求失败，请稍后重试'): string {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error || error.message || fallback
  }
  return error instanceof Error ? error.message : fallback
}

export default api
