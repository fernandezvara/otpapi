-- Add is_active to customers for lifecycle management
ALTER TABLE customers ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Track used backup codes to enforce one-time use
ALTER TABLE mfa_users ADD COLUMN IF NOT EXISTS used_backup_codes_encrypted TEXT[] DEFAULT '{}';
