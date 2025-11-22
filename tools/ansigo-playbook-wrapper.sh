#!/bin/bash
# AnsiGo Playbook Wrapper - 自动预处理 Ansible playbook 并执行
#
# 用法:
#   ./ansigo-playbook-wrapper.sh -i inventory playbook.yml
#   ./ansigo-playbook-wrapper.sh --no-preprocess -i inventory playbook.yml

set -e

# 默认配置
PREPROCESS=true
KEEP_PREPROCESSED=false
VERBOSE=false
PREPROCESSOR=""
ANSIGO_PLAYBOOK=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 帮助信息
usage() {
    cat <<EOF
AnsiGo Playbook Wrapper - 自动预处理并执行 Ansible playbook

用法: $0 [选项] <playbook.yml>

选项:
    -i, --inventory INVENTORY   指定 inventory 文件
    --no-preprocess            跳过预处理步骤
    --keep-preprocessed        保留预处理后的文件
    -v, --verbose              详细输出
    -h, --help                 显示此帮助信息

示例:
    $0 -i hosts.ini deploy.yml
    $0 --no-preprocess -i hosts.ini deploy.yml
    $0 --keep-preprocessed -v -i hosts.ini deploy.yml

环境变量:
    ANSIGO_PREPROCESSOR        预处理器路径（默认: ./bin/playbook-preprocessor）
    ANSIGO_PLAYBOOK           AnsiGo playbook 执行器路径（默认: ./bin/ansigo-playbook）

EOF
    exit 1
}

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 查找可执行文件
find_executable() {
    local name=$1
    local env_var=$2
    local default_path=$3

    # 优先使用环境变量
    if [ -n "${!env_var}" ]; then
        echo "${!env_var}"
        return
    fi

    # 检查默认路径
    if [ -f "$default_path" ] && [ -x "$default_path" ]; then
        echo "$default_path"
        return
    fi

    # 在 PATH 中查找
    if command -v "$name" >/dev/null 2>&1; then
        command -v "$name"
        return
    fi

    # 未找到
    echo ""
}

# 初始化
init() {
    # 查找预处理器
    PREPROCESSOR=$(find_executable "playbook-preprocessor" "ANSIGO_PREPROCESSOR" "./bin/playbook-preprocessor")
    if [ -z "$PREPROCESSOR" ]; then
        log_error "Playbook preprocessor not found"
        log_info "Please build it first: go build -o bin/playbook-preprocessor tools/playbook-preprocessor/main.go"
        exit 1
    fi

    # 查找 ansigo-playbook
    ANSIGO_PLAYBOOK=$(find_executable "ansigo-playbook" "ANSIGO_PLAYBOOK" "./bin/ansigo-playbook")
    if [ -z "$ANSIGO_PLAYBOOK" ]; then
        log_error "ansigo-playbook not found"
        log_info "Please build it first: go build -o bin/ansigo-playbook cmd/ansigo-playbook/main.go"
        exit 1
    fi

    if $VERBOSE; then
        log_info "Using preprocessor: $PREPROCESSOR"
        log_info "Using ansigo-playbook: $ANSIGO_PLAYBOOK"
    fi
}

# 解析参数
PLAYBOOK_FILE=""
ANSIGO_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            ;;
        --no-preprocess)
            PREPROCESS=false
            shift
            ;;
        --keep-preprocessed)
            KEEP_PREPROCESSED=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -i|--inventory)
            ANSIGO_ARGS+=("$1" "$2")
            shift 2
            ;;
        -*)
            # 其他选项传递给 ansigo-playbook
            ANSIGO_ARGS+=("$1")
            shift
            ;;
        *)
            # Playbook 文件
            if [ -z "$PLAYBOOK_FILE" ]; then
                PLAYBOOK_FILE="$1"
            else
                log_error "Multiple playbook files specified"
                exit 1
            fi
            shift
            ;;
    esac
done

# 检查 playbook 文件
if [ -z "$PLAYBOOK_FILE" ]; then
    log_error "No playbook file specified"
    usage
fi

if [ ! -f "$PLAYBOOK_FILE" ]; then
    log_error "Playbook file not found: $PLAYBOOK_FILE"
    exit 1
fi

# 初始化
init

# 执行预处理
FINAL_PLAYBOOK="$PLAYBOOK_FILE"
if $PREPROCESS; then
    log_info "Preprocessing playbook: $PLAYBOOK_FILE"

    # 生成临时文件名
    PREPROCESSED_FILE=$(mktemp "${PLAYBOOK_FILE%.yml}_preprocessed_XXXXXX.yml")

    # 运行预处理器
    PREPROCESS_ARGS="-input $PLAYBOOK_FILE -output $PREPROCESSED_FILE"
    if $VERBOSE; then
        PREPROCESS_ARGS="$PREPROCESS_ARGS -v"
    fi

    if $VERBOSE; then
        log_info "Running: $PREPROCESSOR $PREPROCESS_ARGS"
    fi

    if ! $PREPROCESSOR $PREPROCESS_ARGS; then
        log_error "Preprocessing failed"
        rm -f "$PREPROCESSED_FILE"
        exit 1
    fi

    log_success "Preprocessing completed"
    FINAL_PLAYBOOK="$PREPROCESSED_FILE"
fi

# 执行 ansigo-playbook
log_info "Executing playbook: $FINAL_PLAYBOOK"
if $VERBOSE; then
    log_info "Running: $ANSIGO_PLAYBOOK ${ANSIGO_ARGS[@]} $FINAL_PLAYBOOK"
fi

# 执行并保存退出码
set +e
$ANSIGO_PLAYBOOK "${ANSIGO_ARGS[@]}" "$FINAL_PLAYBOOK"
EXIT_CODE=$?
set -e

# 清理临时文件
if $PREPROCESS && [ "$FINAL_PLAYBOOK" != "$PLAYBOOK_FILE" ]; then
    if $KEEP_PREPROCESSED; then
        log_info "Preprocessed file kept at: $FINAL_PLAYBOOK"
    else
        rm -f "$FINAL_PLAYBOOK"
        if $VERBOSE; then
            log_info "Removed temporary preprocessed file"
        fi
    fi
fi

# 退出
if [ $EXIT_CODE -eq 0 ]; then
    log_success "Playbook execution completed successfully"
else
    log_error "Playbook execution failed with exit code $EXIT_CODE"
fi

exit $EXIT_CODE
