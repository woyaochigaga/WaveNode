export default {
  audit: {
    title: '操作日志',
    description: '记录管理员与用户的管理面操作，请求凭证及输入输出均已脱敏。日志无法单条删除，全量清理需二次验证。',
    clearAll: '全部清理',
    empty: '暂无操作日志',
    loadFailed: '加载操作日志失败',
    filters: {
      all: '全部',
      q: '关键字',
      qPlaceholder: '路径 / 动作 / 操作者邮箱',
      actorEmail: '操作者邮箱',
      action: '动作',
      clientIp: '客户端 IP',
      method: '请求方法',
      authMethod: '认证方式',
      result: '结果',
      resultSuccess: '成功',
      resultFailure: '失败',
      advanced: '高级筛选',
      startTime: '开始时间',
      endTime: '结束时间'
    },
    columns: {
      time: '时间',
      actor: '操作者',
      action: '动作',
      method: '方法',
      result: '结果',
      clientIp: '客户端 IP',
      detail: '详情'
    },
    detail: {
      title: '操作日志详情',
      actorRole: '角色',
      methodPath: '方法 / 路径',
      latency: '耗时',
      requestId: '请求 ID',
      credential: '凭证（掩码）',
      userAgent: 'User-Agent',
      requestBody: '请求体（已脱敏）',
      responseBody: '响应体（已脱敏）',
      responseType: '响应类型',
      pathParams: '路径参数',
      queryParams: '查询参数',
      noPathParams: '本次请求没有路径参数',
      noQueryParams: '本次请求没有查询参数',
      noRequestBody: '本次请求没有正文，或该历史记录尚未采集正文',
      noResponseBody: '该历史记录尚未采集响应正文',
      tabs: {
        overview: '概览',
        request: '请求输入',
        response: '响应输出'
      },
      extra: '附加信息'
    },
    clearConfirm: {
      title: '清理全部操作日志',
      message: '此操作将永久删除所有操作日志，且不可恢复。清理动作本身会被留痕记录。确定继续吗？',
      totpTitle: '输入二次验证码',
      totpHint: '清理操作日志需要现场验证 TOTP 验证码。',
      success: '已清理 {count} 条操作日志',
      failed: '清理操作日志失败'
    }
  }
}
