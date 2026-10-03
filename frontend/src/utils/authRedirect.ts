/**
 * 解析登录后的站内跳转地址。
 * 显式回跳仅接受站内绝对路径；没有合法回跳时按账号角色进入对应仪表盘。
 */
export function resolvePostLoginPath(redirect: unknown, isAdmin: boolean): string {
  if (
    typeof redirect === 'string' &&
    redirect.startsWith('/') &&
    !redirect.startsWith('//') &&
    !redirect.includes('://') &&
    !redirect.includes('\n') &&
    !redirect.includes('\r')
  ) {
    return redirect
  }
  return isAdmin ? '/admin/dashboard' : '/dashboard'
}
