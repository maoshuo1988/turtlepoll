#!/usr/bin/env bash
# 开启严格模式：
# -e：命令失败立即退出
# -u：使用未定义变量时报错
# -o pipefail：管道中任意命令失败都视为失败
set -Eeuo pipefail

# 应用名称和 Go 版本可以通过环境变量覆盖，例如：
# GO_VERSION=1.25.3 APP_NAME=bbs-go ./bash.sh
APP_NAME="${APP_NAME:-bbs-go}"
GO_VERSION="${GO_VERSION:-1.23.0}"
GO_OS="${GO_OS:-linux}"
GO_ARCH="${GO_ARCH:-amd64}"

# 获取脚本所在目录，确保无论从哪里执行脚本，路径都以项目根目录为准。
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Go 会下载安装到项目内的 .tools 目录，避免污染系统级 /usr/local/go。
TOOLS_DIR="${ROOT_DIR}/.tools"
GO_ROOT="${TOOLS_DIR}/go/go${GO_VERSION}"
GO_BIN="${GO_ROOT}/bin/go"

# 缓存、构建产物、运行时 PID 和日志目录。
CACHE_DIR="${ROOT_DIR}/.cache"
BIN_DIR="${ROOT_DIR}/bin"
RUN_DIR="${ROOT_DIR}/run"
LOG_DIR="${ROOT_DIR}/logs"

# 二进制文件、PID 文件和应用日志文件路径。
BINARY_PATH="${BIN_DIR}/${APP_NAME}"
PID_FILE="${RUN_DIR}/${APP_NAME}.pid"
APP_LOG="${LOG_DIR}/${APP_NAME}.log"

# Go 安装包缓存路径和下载地址；GO_URL 可用于切换镜像源。
GO_TARBALL="${CACHE_DIR}/go${GO_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"
GO_URL="${GO_URL:-https://go.dev/dl/go${GO_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz}"

# 统一日志输出格式，方便观察每个步骤的执行时间。
log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

# 输出错误日志并退出脚本。
fail() {
  log "ERROR: $*"
  exit 1
}

# 通用步骤包装函数：执行前后打印日志。
run_step() {
  log "START: $*"
  "$@"
  log "DONE: $*"
}

# 创建脚本需要使用的目录。
ensure_dirs() {
  log "Create runtime directories"
  mkdir -p "${TOOLS_DIR}/go" "${CACHE_DIR}" "${BIN_DIR}" "${RUN_DIR}" "${LOG_DIR}"
}

# 按 PID 停止进程：先优雅终止，等待后仍未退出则强制 kill。
kill_pid() {
  local pid="$1"

  if [[ -z "${pid}" ]] || ! kill -0 "${pid}" 2>/dev/null; then
    return 0
  fi

  log "Stop process pid=${pid}"
  kill "${pid}" 2>/dev/null || true

  for _ in {1..20}; do
    if ! kill -0 "${pid}" 2>/dev/null; then
      log "Process stopped pid=${pid}"
      return 0
    fi
    sleep 0.2
  done

  log "Force kill process pid=${pid}"
  kill -9 "${pid}" 2>/dev/null || true
}

# 停止上一次运行的项目进程，避免端口占用或重复运行。
stop_previous_processes() {
  log "Stop previous ${APP_NAME} processes"

  # 优先根据 PID 文件停止，这是最精确的方式。
  if [[ -f "${PID_FILE}" ]]; then
    kill_pid "$(cat "${PID_FILE}")"
    rm -f "${PID_FILE}"
  fi

  # 兜底清理：如果 PID 文件不存在，也尝试按命令行特征查找旧进程。
  if command -v pgrep >/dev/null 2>&1; then
    # 清理由本脚本生成的二进制进程。
    while IFS= read -r pid; do
      [[ "${pid}" == "$$" ]] && continue
      kill_pid "${pid}"
    done < <(pgrep -f "${BINARY_PATH}" || true)

    # 清理开发时可能留下的 go run main.go 进程。
    while IFS= read -r pid; do
      [[ "${pid}" == "$$" ]] && continue
      kill_pid "${pid}"
    done < <(pgrep -f "go run .*${ROOT_DIR}/main.go|go run main.go" || true)
  fi

  log "Previous process cleanup finished"
}

# 下载并安装指定版本的 Go。如果已经安装过，则直接复用。
download_go() {
  if [[ -x "${GO_BIN}" ]]; then
    log "Go already installed: $(${GO_BIN} version)"
    return 0
  fi

  log "Download Go ${GO_VERSION} from ${GO_URL}"
  # 优先使用 curl；如果环境没有 curl，则尝试 wget。
  if command -v curl >/dev/null 2>&1; then
    curl -fL --retry 3 --connect-timeout 20 -o "${GO_TARBALL}" "${GO_URL}"
  elif command -v wget >/dev/null 2>&1; then
    wget -O "${GO_TARBALL}" "${GO_URL}"
  else
    fail "curl or wget is required to download Go"
  fi

  log "Install Go ${GO_VERSION} into ${GO_ROOT}"
  rm -rf "${GO_ROOT}"
  mkdir -p "${GO_ROOT}"
  tar -C "${GO_ROOT}" --strip-components=1 -xzf "${GO_TARBALL}"
  "${GO_BIN}" version
}

# 下载依赖并生成项目二进制文件。
build_binary() {
  log "Download Go modules"
  # 使用项目内下载的 Go，避免 hook/CI 中调用到系统里的旧版本 Go。
  export PATH="${GO_ROOT}/bin:${PATH}"
  export GOTOOLCHAIN=local
  "${GO_BIN}" mod download

  log "Build binary ${BINARY_PATH}"
  # 默认关闭 CGO，生成更容易部署的 Linux amd64 二进制文件。
  CGO_ENABLED="${CGO_ENABLED:-0}" GOOS="${GO_OS}" GOARCH="${GO_ARCH}" \
    "${GO_BIN}" build -trimpath -ldflags "-s -w" -o "${BINARY_PATH}" "${ROOT_DIR}/main.go"

  chmod +x "${BINARY_PATH}"
  log "Binary generated: ${BINARY_PATH}"
}

# 后台运行新生成的二进制，并记录 PID 和运行日志。
run_app() {
  log "Run ${APP_NAME}; logs: ${APP_LOG}"
  nohup "${BINARY_PATH}" >>"${APP_LOG}" 2>&1 &
  local pid="$!"
  echo "${pid}" >"${PID_FILE}"
  sleep 1

  if kill -0 "${pid}" 2>/dev/null; then
    log "${APP_NAME} started pid=${pid}"
  else
    log "${APP_NAME} failed to start. Last logs:"
    tail -n 80 "${APP_LOG}" || true
    exit 1
  fi
}

# 主流程：清理旧进程 -> 准备 Go -> 构建二进制 -> 启动服务。
main() {
  log "Deploy ${APP_NAME} from ${ROOT_DIR}"
  ensure_dirs
  stop_previous_processes
  download_go
  build_binary
  run_app
  log "All steps finished"
}

main "$@"
