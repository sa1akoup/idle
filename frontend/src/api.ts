// 前端 API 客户端：统一请求入口与错误信息提取。
import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 15_000,
})

export function getApiError(error: unknown, fallback = '请求失败，请稍后重试'): string {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error || error.message || fallback
  }
  return error instanceof Error ? error.message : fallback
}

export default api
