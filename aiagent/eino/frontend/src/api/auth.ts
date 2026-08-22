import { BackendCapabilityError } from '@/api/http'

// 认证后端尚未实现。完成 docs/API_CONTRACT.md 的认证部分后，
// AuthView 应从本地演示方法切换为调用这组 API。
export const reservedAuthAPI = {
  register: () => Promise.reject(new BackendCapabilityError('用户注册')),
  login: () => Promise.reject(new BackendCapabilityError('用户登录')),
  currentUser: () => Promise.reject(new BackendCapabilityError('当前用户信息')),
  logout: () => Promise.reject(new BackendCapabilityError('用户退出')),
}
