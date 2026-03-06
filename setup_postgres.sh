#!/bin/bash

################################################################################
# PostgreSQL 自动化配置脚本
# 功能：修改为密码认证并创建授权数据库用户
# 适用于 WSL/Linux 开发环境
################################################################################

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查是否以 root 或 sudo 运行
check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "请使用 sudo 运行此脚本：sudo $0"
        exit 1
    fi
}

# 检查参数
if [ $# -lt 3 ]; then
    print_info "用法：$0 <数据库名> <用户名> <密码>"
    print_info "示例：$0 appdb appuser mypassword123"
    exit 1
fi

DB_NAME="$1"
DB_USER="$2"
DB_PASS="$3"

print_info "开始配置 PostgreSQL..."
print_info "数据库名：$DB_NAME"
print_info "用户名：$DB_USER"
print_info "密码：$DB_PASS"

# 步骤 1: 查找 pg_hba.conf 文件路径
print_info "步骤 1: 查找 PostgreSQL 配置文件..."
HBA_FILE=$(sudo -u postgres psql -t -c "SHOW hba_file;" | xargs)
print_info "找到配置文件：$HBA_FILE"

if [ ! -f "$HBA_FILE" ]; then
    print_error "配置文件不存在：$HBA_FILE"
    exit 1
fi

# 步骤 2: 备份配置文件
print_info "步骤 2: 备份配置文件..."
BACKUP_FILE="${HBA_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$HBA_FILE" "$BACKUP_FILE"
print_success "配置文件已备份到：$BACKUP_FILE"

# 步骤 3: 修改认证方式为 md5
print_info "步骤 3: 修改认证方式为 md5..."

# 注释掉原有的 local 行，添加新的 md5 认证行
sed -i.bak '/^local.*all.*all.*peer/s/^/# /' "$HBA_FILE"
sed -i.bak '/^local.*all.*all.*md5/s/^/# /' "$HBA_FILE"

# 在文件末尾添加新的认证配置
cat >> "$HBA_FILE" << EOF

# 自定义配置 - 允许所有本地用户使用密码认证
local   all             all                                     md5
host    all             all             127.0.0.1/32            md5
host    all             all             ::1/128                 md5
EOF

print_success "认证配置已修改"

# 步骤 4: 重启 PostgreSQL 服务
print_info "步骤 4: 重启 PostgreSQL 服务..."
systemctl restart postgresql || service postgresql restart || {
    print_warning "systemctl 不可用，尝试直接重启..."
    sudo -u postgres pg_ctlcluster $(ls /etc/postgresql/ | head -1) main reload
}

sleep 2
print_success "PostgreSQL 服务已重启"

# 步骤 5: 创建数据库和用户
print_info "步骤 5: 创建数据库和用户..."

sudo -u postgres psql << EOF
-- 如果数据库不存在则创建
SELECT 'CREATE DATABASE $DB_NAME' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$DB_NAME')\gexec

-- 如果用户不存在则创建
DO \$\$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$DB_USER') THEN
      CREATE ROLE $DB_USER LOGIN PASSWORD '$DB_PASS';
   END IF;
END
\$\$;

-- 授予数据库连接权限
GRANT CONNECT ON DATABASE $DB_NAME TO $DB_USER;

-- 切换到新数据库
\\c $DB_NAME

-- 授予 schema 权限
GRANT CREATE, USAGE ON SCHEMA public TO $DB_USER;

-- 授予所有表权限
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;

-- 设置默认权限（未来创建的表也会授权）
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO $DB_USER;

-- 显示用户信息
\\du $DB_USER
EOF

print_success "数据库和用户已创建并授权"

# 步骤 6: 验证配置
print_info "步骤 6: 验证配置..."

# 测试连接
print_info "测试数据库连接..."
if PGPASSWORD="$DB_PASS" psql -h localhost -U "$DB_USER" -d "$DB_NAME" -c "SELECT 'Connection successful!' AS status;" > /dev/null 2>&1; then
    print_success "✓ 数据库连接测试成功"
else
    print_error "✗ 数据库连接测试失败"
    exit 1
fi

# 测试建表权限
print_info "测试建表权限..."
if PGPASSWORD="$DB_PASS" psql -h localhost -U "$DB_USER" -d "$DB_NAME" -c "
CREATE TABLE IF NOT EXISTS test_permission_check (id SERIAL PRIMARY KEY);
DROP TABLE IF EXISTS test_permission_check;
SELECT 'Create table permission OK' AS status;
" > /dev/null 2>&1; then
    print_success "✓ 建表权限测试成功"
else
    print_error "✗ 建表权限测试失败"
    exit 1
fi

# 步骤 7: 显示连接信息
print_info "=========================================="
print_success "PostgreSQL 配置完成！"
print_info "=========================================="
echo ""
print_info "数据库连接信息："
echo "  主机：localhost"
echo "  端口：5432"
echo "  数据库：$DB_NAME"
echo "  用户名：$DB_USER"
echo "  密码：$DB_PASS"
echo ""
print_info "连接命令："
echo "  psql -h localhost -U $DB_USER -d $DB_NAME"
echo ""
print_info "或使用环境变量："
echo "  export PGHOST=localhost"
echo "  export PGDATABASE=$DB_NAME"
echo "  export PGUSER=$DB_USER"
echo "  export PGPASSWORD=$DB_PASS"
echo "  psql"
echo ""
print_info "=========================================="

# 完成
print_success "所有配置已完成并验证通过！"
