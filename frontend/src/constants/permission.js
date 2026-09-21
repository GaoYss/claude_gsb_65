// 权限点常量。取值必须与后端 internal/modules/auth/permission.go 完全一致,
// 是前端页面、按钮、路由显隐的唯一依据; 真正能否操作仍以后端判定为准。
export const PERMISSIONS = {
  LAMP_VIEW: 'lamp:view',
  LAMP_MANAGE: 'lamp:manage',

  FAULT_VIEW: 'fault:view',
  FAULT_REGISTER: 'fault:register',
  FAULT_UPDATE: 'fault:update',
  FAULT_CLOSE: 'fault:close',
  FAULT_DELETE: 'fault:delete',
  FAULT_REASSIGN: 'fault:reassign',
  FAULT_EXCEPTION: 'fault:exception',

  REPAIR_VIEW: 'repair:view',
  REPAIR_CLAIM: 'repair:claim',
  REPAIR_ADVANCE: 'repair:advance',
  REPAIR_MANAGE: 'repair:manage',
  REPAIR_ASSIGN: 'repair:assign',
  REPAIR_REASSIGN: 'repair:reassign',
  REPAIR_EXCEPTION: 'repair:exception',

  STATUS_VIEW: 'status:view',
  EXPORT_FAULT: 'export:fault',
  EXPORT_REPAIR: 'export:repair',

  AUDIT_VIEW: 'audit:view',
  USER_MANAGE: 'user:manage',
}

// 角色常量。
export const ROLES = {
  REGISTRAR: 'registrar',
  REPAIR: 'repair',
  ADMIN: 'admin',
}
